package source

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// NotionClient is the only network boundary used by metadata preview.
type NotionClient interface {
	GetTitle(context.Context, string) (string, error)
}

type MetadataError struct{ Reason string }

func (e *MetadataError) Error() string { return e.Reason }

type HTTPNotionClient struct {
	token  string
	client *http.Client
}

func NewNotionClient(token string) *HTTPNotionClient {
	return &HTTPNotionClient{token: strings.TrimSpace(token), client: &http.Client{Timeout: 5 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
}

func FromEnvironment() NotionClient { return NewNotionClient(os.Getenv("MINIBLOG_NOTION_TOKEN")) }

func (c *HTTPNotionClient) GetTitle(ctx context.Context, pageID string) (string, error) {
	if !exactNotionID.MatchString(pageID) {
		return "", &MetadataError{Reason: "notion_invalid_page"}
	}
	if c.token == "" {
		return "", &MetadataError{Reason: "notion_not_configured"}
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.notion.com/v1/pages/"+pageID, nil)
	if err != nil {
		return "", &MetadataError{Reason: "notion_invalid_page"}
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Notion-Version", "2026-03-11")
	resp, err := c.client.Do(req)
	if err != nil {
		reason := "notion_unavailable"
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			reason = "notion_timeout"
		}
		return "", &MetadataError{Reason: reason}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		reason := "notion_unavailable"
		switch resp.StatusCode {
		case 401, 403, 404:
			reason = "notion_not_shared"
		case 429:
			reason = "notion_rate_limited"
		}
		return "", &MetadataError{Reason: reason}
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", &MetadataError{Reason: "notion_invalid_response"}
	}
	var page struct {
		ID         string `json:"id"`
		Properties map[string]struct {
			Type  string `json:"type"`
			Title []struct {
				PlainText string `json:"plain_text"`
				Text      struct {
					Content string `json:"content"`
				} `json:"text"`
			} `json:"title"`
		} `json:"properties"`
	}
	if json.Unmarshal(data, &page) != nil || normalizePageID(page.ID) != normalizePageID(pageID) {
		return "", &MetadataError{Reason: "notion_invalid_response"}
	}
	for _, property := range page.Properties {
		if property.Type != "title" {
			continue
		}
		var title strings.Builder
		for _, part := range property.Title {
			if part.PlainText != "" {
				title.WriteString(part.PlainText)
			} else {
				title.WriteString(part.Text.Content)
			}
		}
		if result := strings.TrimSpace(title.String()); result != "" {
			return result, nil
		}
	}
	return "", &MetadataError{Reason: "notion_title_missing"}
}
