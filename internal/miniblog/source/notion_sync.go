package source

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const NotionSyncVersion = "2026-03-11"
const notionSyncBase = "https://api.notion.com/v1"

// SyncNotionClient is read-only. The separate writer is only accepted by bootstrap.
type SyncNotionClient interface {
	RetrieveDataSource(context.Context, string) (NotionDataSource, error)
	QueryDataSource(context.Context, string, bool, string) (NotionQueryResult, error)
	RetrievePage(context.Context, string) (NotionPage, error)
}
type BootstrapNotionWriter interface {
	UpdateBlogState(context.Context, string, string, string) error
}

type NotionOption struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color,omitempty"`
}
type NotionSchemaProperty struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Type   string `json:"type"`
	Select struct {
		Options []NotionOption `json:"options"`
	} `json:"select"`
	MultiSelect struct {
		Options []NotionOption `json:"options"`
	} `json:"multi_select"`
}
type NotionDataSource struct {
	ID         string                          `json:"id"`
	Properties map[string]NotionSchemaProperty `json:"properties"`
	InTrash    bool                            `json:"in_trash"`
}
type NotionRichText struct {
	PlainText string `json:"plain_text"`
	Text      struct {
		Content string `json:"content"`
	} `json:"text"`
}
type NotionProperty struct {
	ID          string           `json:"id"`
	Type        string           `json:"type"`
	Title       []NotionRichText `json:"title"`
	RichText    []NotionRichText `json:"rich_text"`
	Select      *NotionOption    `json:"select"`
	MultiSelect []NotionOption   `json:"multi_select"`
	Date        *struct {
		Start string  `json:"start"`
		End   *string `json:"end"`
	} `json:"date"`
}
type NotionPage struct {
	ID             string    `json:"id"`
	Object         string    `json:"object"`
	LastEditedTime time.Time `json:"last_edited_time"`
	CreatedTime    time.Time `json:"created_time"`
	URL            string    `json:"url"`
	PublicURL      *string   `json:"public_url"`
	IsArchived     bool      `json:"is_archived"`
	InTrash        bool      `json:"in_trash"`
	Parent         struct {
		Type         string `json:"type"`
		DataSourceID string `json:"data_source_id"`
		DatabaseID   string `json:"database_id"`
	} `json:"parent"`
	Properties map[string]NotionProperty `json:"properties"`
}
type NotionQueryResult struct {
	Object        string       `json:"object"`
	Type          string       `json:"type"`
	Results       []NotionPage `json:"results"`
	HasMore       bool         `json:"has_more"`
	NextCursor    *string      `json:"next_cursor"`
	RequestStatus struct {
		Type             string `json:"type"`
		IncompleteReason string `json:"incomplete_reason"`
	} `json:"request_status"`
	requestStatusOmitted bool
}

var ErrNotionQueryIncomplete = errors.New("notion query incomplete")

// QueryStatusError allows a genuinely omitted status only after wire validation.
// Explicit complete/incomplete also support injected clients returning typed results.
func (r NotionQueryResult) QueryStatusError() error {
	switch r.RequestStatus.Type {
	case "complete":
		return nil
	case "incomplete":
		return ErrNotionQueryIncomplete
	case "":
		if r.requestStatusOmitted {
			return nil
		}
	}
	return errors.New("invalid notion query status")
}

// UnmarshalJSON validates the list envelope before exposing any page. JSON null
// and missing fields are distinct; a pointer alone cannot preserve that distinction.
func (r *NotionQueryResult) UnmarshalJSON(data []byte) error {
	*r = NotionQueryResult{}
	invalid := errors.New("invalid notion query envelope")
	fields, err := notionQueryObject(data)
	if err != nil {
		return invalid
	}
	for _, key := range []string{"object", "type", "page_or_data_source", "results", "has_more", "next_cursor"} {
		if _, exists := fields[key]; !exists {
			return invalid
		}
	}
	metadata, err := notionQueryObject(fields["page_or_data_source"])
	if err != nil || len(metadata) != 0 || bytes.Equal(bytes.TrimSpace(fields["has_more"]), []byte("null")) {
		return invalid
	}
	type plainResult NotionQueryResult
	var decoded plainResult
	if json.Unmarshal(data, &decoded) != nil || decoded.Object != "list" || decoded.Type != "page_or_data_source" || decoded.Results == nil || len(decoded.Results) > 100 {
		return invalid
	}
	for _, page := range decoded.Results {
		if page.Object != "page" && page.Object != "data_source" {
			return invalid
		}
		if _, err := NotionIdentity(page.ID); err != nil {
			return invalid
		}
	}
	if decoded.HasMore {
		if decoded.NextCursor == nil || strings.TrimSpace(*decoded.NextCursor) == "" {
			return invalid
		}
	} else if decoded.NextCursor != nil {
		return invalid
	}
	status, exists := fields["request_status"]
	if exists {
		statusFields, err := notionQueryObject(status)
		if err != nil || (decoded.RequestStatus.Type != "complete" && decoded.RequestStatus.Type != "incomplete") {
			return invalid
		}
		if reason, exists := statusFields["incomplete_reason"]; exists {
			if bytes.Equal(bytes.TrimSpace(reason), []byte("null")) || decoded.RequestStatus.IncompleteReason != "query_result_limit_reached" {
				return invalid
			}
		}
	} else {
		decoded.requestStatusOmitted = true
	}
	*r = NotionQueryResult(decoded)
	return nil
}

// Duplicate envelope/status keys cannot erase an earlier incomplete marker.
func notionQueryObject(data []byte) (map[string]json.RawMessage, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	token, err := decoder.Token()
	if err != nil || token != json.Delim('{') {
		return nil, errors.New("invalid notion query object")
	}
	fields := map[string]json.RawMessage{}
	for decoder.More() {
		token, err = decoder.Token()
		if err != nil {
			return nil, err
		}
		key, ok := token.(string)
		if _, duplicate := fields[key]; !ok || duplicate {
			return nil, errors.New("invalid notion query object key")
		}
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return nil, err
		}
		fields[key] = value
	}
	if token, err = decoder.Token(); err != nil || token != json.Delim('}') {
		return nil, errors.New("invalid notion query object")
	}
	var trailing interface{}
	if decoder.Decode(&trailing) != io.EOF {
		return nil, errors.New("invalid notion query trailing data")
	}
	return fields, nil
}

type NotionAPIError struct {
	StatusCode int
	Code       string
	Retryable  bool
}

func (e *NotionAPIError) Error() string {
	return fmt.Sprintf("notion: %s (HTTP %d)", e.Code, e.StatusCode)
}
func IsNotionUnavailable(err error) bool {
	var e *NotionAPIError
	return errors.As(err, &e) && (e.StatusCode == 403 || e.StatusCode == 404)
}

// HTTPNotionSyncClient serializes requests and applies one-second spacing and cooldowns.
// There is deliberately no configurable base URL or redirect following.
type HTTPNotionSyncClient struct {
	token            string
	client           *http.Client
	mu               sync.Mutex
	next             time.Time
	spacing          time.Duration
	cooldownObserver func(context.Context, time.Time) error
}

func (c *HTTPNotionSyncClient) SetCooldownObserver(fn func(context.Context, time.Time) error) {
	c.cooldownObserver = fn
}
func NewNotionSyncClient(token string) *HTTPNotionSyncClient {
	return &HTTPNotionSyncClient{token: strings.TrimSpace(token), client: &http.Client{Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}, spacing: time.Second}
}
func NotionIdentity(pageID string) (Identity, error) {
	if !exactNotionID.MatchString(pageID) {
		return Identity{}, errors.New("invalid notion page id")
	}
	return Parse("https://www.notion.so/" + normalizePageID(pageID))
}
func (c *HTTPNotionSyncClient) RetrieveDataSource(ctx context.Context, id string) (NotionDataSource, error) {
	var result NotionDataSource
	if !exactNotionID.MatchString(id) {
		return result, errors.New("invalid notion data source id")
	}
	err := c.request(ctx, http.MethodGet, "/data_sources/"+id, nil, &result, true)
	if err == nil && normalizePageID(result.ID) != normalizePageID(id) {
		err = errors.New("notion data source identity mismatch")
	}
	return result, err
}
func (c *HTTPNotionSyncClient) RetrievePage(ctx context.Context, id string) (NotionPage, error) {
	var result NotionPage
	if !exactNotionID.MatchString(id) {
		return result, errors.New("invalid notion page id")
	}
	err := c.request(ctx, http.MethodGet, "/pages/"+id, nil, &result, true)
	if err == nil && normalizePageID(result.ID) != normalizePageID(id) {
		err = errors.New("notion page identity mismatch")
	}
	return result, err
}
func (c *HTTPNotionSyncClient) QueryDataSource(ctx context.Context, id string, archived bool, cursor string) (NotionQueryResult, error) {
	var result NotionQueryResult
	if !exactNotionID.MatchString(id) {
		return result, errors.New("invalid notion data source id")
	}
	body := map[string]interface{}{"page_size": 100, "is_archived": archived}
	if cursor != "" {
		body["start_cursor"] = cursor
	}
	err := c.request(ctx, http.MethodPost, "/data_sources/"+id+"/query", body, &result, true)
	if err == nil {
		err = result.QueryStatusError()
	}
	return result, err
}
func (c *HTTPNotionSyncClient) UpdateBlogState(ctx context.Context, pageID, propertyID, optionID string) error {
	if !exactNotionID.MatchString(pageID) || propertyID == "" || optionID == "" {
		return errors.New("invalid notion bootstrap state")
	}
	body := map[string]interface{}{"properties": map[string]interface{}{propertyID: map[string]interface{}{"select": map[string]interface{}{"id": optionID}}}}
	// A write can succeed despite a timeout/503. The journal must reread before retrying.
	return c.request(ctx, http.MethodPatch, "/pages/"+pageID, body, nil, false)
}
func (c *HTTPNotionSyncClient) request(ctx context.Context, method, path string, body interface{}, target interface{}, retry bool) error {
	if c.token == "" {
		return &NotionAPIError{Code: "notion_not_configured"}
	}
	var encoded []byte
	var err error
	if body != nil {
		encoded, err = json.Marshal(body)
		if err != nil {
			return err
		}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	attempts := 1
	if retry {
		attempts = 3
	}
	for attempt := 0; attempt < attempts; attempt++ {
		if err = waitNotion(ctx, time.Until(c.next)); err != nil {
			return err
		}
		req, e := http.NewRequestWithContext(ctx, method, notionSyncBase+path, bytes.NewReader(encoded))
		if e != nil {
			return e
		}
		req.Header.Set("Authorization", "Bearer "+c.token)
		req.Header.Set("Notion-Version", NotionSyncVersion)
		req.Header.Set("Content-Type", "application/json")
		c.next = time.Now().Add(c.spacing)
		resp, e := c.client.Do(req)
		if e != nil {
			err = &NotionAPIError{Code: "transport_unavailable", Retryable: true}
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if attempt+1 < attempts {
				c.next = maxNotionTime(c.next, time.Now().Add(time.Duration(attempt+1)*time.Second))
				continue
			}
			return err
		}
		data, readErr := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
		resp.Body.Close()
		if readErr != nil {
			err = &NotionAPIError{Code: "invalid_response", Retryable: true}
		} else if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			if target != nil && json.Unmarshal(data, target) != nil {
				return &NotionAPIError{StatusCode: resp.StatusCode, Code: "invalid_response"}
			}
			return nil
		} else {
			var apiErr struct {
				Code string `json:"code"`
			}
			_ = json.Unmarshal(data, &apiErr)
			if apiErr.Code == "" {
				apiErr.Code = "request_failed"
			}
			isRetry := resp.StatusCode == 429 || resp.StatusCode == 529 || resp.StatusCode >= 500
			err = &NotionAPIError{StatusCode: resp.StatusCode, Code: apiErr.Code, Retryable: isRetry}
			var cooldown time.Duration
			if s := resp.Header.Get("Retry-After"); s != "" {
				if seconds, e := strconv.ParseFloat(s, 64); e == nil && seconds > 0 {
					cooldown = time.Duration(seconds * float64(time.Second))
				} else if date, e := http.ParseTime(s); e == nil {
					cooldown = time.Until(date)
				}
			}
			if cooldown > 0 {
				c.next = maxNotionTime(c.next, time.Now().Add(cooldown))
				if c.cooldownObserver != nil {
					if e := c.cooldownObserver(ctx, c.next); e != nil {
						return e
					}
				}
			}
			if !isRetry {
				return err
			}
		}
		if attempt+1 < attempts {
			c.next = maxNotionTime(c.next, time.Now().Add(time.Duration(attempt+1)*time.Second))
		}
	}
	return err
}
func waitNotion(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
func maxNotionTime(a, b time.Time) time.Time {
	if b.After(a) {
		return b
	}
	return a
}

// MigrationNotionPageID accepts exact official page links for a reviewed one-time match.
// Database/view links with p/v stay ambiguous; callers must never match on title.
func MigrationNotionPageID(rawURL string) (string, error) {
	identity, err := Parse(rawURL)
	if err != nil {
		return "", err
	}
	if identity.PageID != "" {
		return identity.PageID, nil
	}
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return "", err
	}
	if strings.EqualFold(u.Hostname(), "app.notion.com") && strings.HasPrefix(u.Path, "/p/") && !u.Query().Has("p") && !u.Query().Has("v") {
		segment := strings.TrimSuffix(strings.TrimPrefix(u.Path, "/p/"), "/")
		if !strings.Contains(segment, "/") {
			if match := notionID.FindStringSubmatch(segment); len(match) > 1 {
				return normalizePageID(match[1]), nil
			}
		}
	}
	return "", errors.New("notion page identity requires review")
}
