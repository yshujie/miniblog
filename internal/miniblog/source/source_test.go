package source

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestParseConservativeIdentity(t *testing.T) {
	page := "aabbccddeeff00112233445566778899"
	urls := []string{"https://www.notion.so/Title-" + page + "?pvs=4", "https://space.notion.site/" + page, "https://notion.so/aabbccdd-eeff-0011-2233-445566778899", "https://app.notion.com/p/Title-" + page + "?source=copy_link", "https://app.notion.com/p/shujie-blog/" + page}
	first, err := Parse(urls[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, raw := range urls {
		got, err := Parse(raw)
		if err != nil || got.Provider != "notion" || got.PageID != page || got.SourceKey != first.SourceKey {
			t.Fatalf("page alias: %+v %v", got, err)
		}
	}
	a, err := Parse("HTTPS://EXAMPLE.COM:443/path?q=1#anchor")
	if err != nil || a.Provider != "other" || a.CanonicalURL != "https://example.com/path?q=1#anchor" {
		t.Fatalf("%+v %v", a, err)
	}
	b, _ := Parse("https://example.com/path?q=2#anchor")
	if a.SourceKey == b.SourceKey {
		t.Fatal("query identity was discarded")
	}
	for _, query := range []string{"?p=" + page, "?v=11223344556677889900aabbccddeeff", "?p=&pvs=4"} {
		view, err := Parse("https://www.notion.so/" + page + query)
		if err != nil || view.PageID != "" || view.SourceKey == first.SourceKey {
			t.Fatalf("view merged on database path: %+v %v", view, err)
		}
	}
	for _, raw := range []string{"ftp://example.com/x", "https://user:pass@example.com/x", "/relative", "https://", "https://example.com:bad/x"} {
		if _, err := Parse(raw); err == nil {
			t.Fatalf("accepted %s", raw)
		}
	}
	feishu, _ := Parse("https://team.feishu.cn/docx/abc")
	if feishu.Provider != "feishu" {
		t.Fatal(feishu)
	}
	spoof, _ := Parse("https://notion.so.example.com/" + page)
	if spoof.Provider != "other" || spoof.PageID != "" {
		t.Fatal(spoof)
	}
	for _, segment := range []string{"a" + page, "other" + page, "slug-a" + page, page + "f"} {
		got, err := Parse("https://notion.so/" + segment + "?pvs=4#block")
		if err != nil || got.PageID != "" || got.CanonicalURL != "https://notion.so/"+segment+"?pvs=4#block" {
			t.Fatalf("guessed ID for %s: %+v %v", segment, got, err)
		}
	}
	fragmentA, _ := Parse("https://example.com/page?q=1#first")
	fragmentB, _ := Parse("https://example.com/page?q=1#second")
	if fragmentA.SourceKey == fragmentB.SourceKey {
		t.Fatal("generic fragments were discarded")
	}
}

type transport func(*http.Request) (*http.Response, error)

func (f transport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestNotionClientKnownEndpointAndNoRetry(t *testing.T) {
	page := "aabbccddeeff00112233445566778899"
	client := NewNotionClient("test-token")
	calls := 0
	client.client.Transport = transport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Method != "GET" || r.URL.String() != "https://api.notion.com/v1/pages/"+page || r.Header.Get("Notion-Version") != "2026-03-11" || r.Header.Get("Authorization") != "Bearer test-token" {
			t.Fatal("wrong API contract")
		}
		if deadline, ok := r.Context().Deadline(); !ok || time.Until(deadline) > 5*time.Second {
			t.Fatal("missing bounded timeout")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"id":"aabbccdd-eeff-0011-2233-445566778899","properties":{"Name":{"type":"title","title":[{"plain_text":"外部"},{"text":{"content":"标题"}}]}}}`))}, nil
	})
	title, err := client.GetTitle(context.Background(), page)
	if err != nil || title != "外部标题" || calls != 1 {
		t.Fatalf("%q %v %d", title, err, calls)
	}
	client.client.Transport = transport(func(r *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 429, Body: io.NopCloser(strings.NewReader("private provider message"))}, nil
	})
	_, err = client.GetTitle(context.Background(), page)
	var metadata *MetadataError
	if !errors.As(err, &metadata) || metadata.Reason != "notion_rate_limited" || calls != 2 {
		t.Fatalf("%v %d", err, calls)
	}
	_, err = client.GetTitle(context.Background(), "../users")
	if err == nil || calls != 2 {
		t.Fatal("invalid ID made an API call")
	}
}
func TestNotionUnavailableHasSanitizedReason(t *testing.T) {
	page := "aabbccddeeff00112233445566778899"
	for _, status := range []int{401, 403, 404, 500, 302} {
		c := NewNotionClient("test-token")
		calls := 0
		c.client.Transport = transport(func(r *http.Request) (*http.Response, error) {
			calls++
			return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader("private response"))}, nil
		})
		_, err := c.GetTitle(context.Background(), page)
		if err == nil || strings.Contains(err.Error(), "private") || strings.Contains(err.Error(), "test-token") || calls != 1 {
			t.Fatalf("%v", err)
		}
	}
	c := NewNotionClient("")
	_, err := c.GetTitle(context.Background(), page)
	if err == nil || err.Error() != "notion_not_configured" {
		t.Fatal(err)
	}
}

func TestAppNotionViewsAndNonPagePathsRemainConservative(t *testing.T) {
	page := "aabbccddeeff00112233445566778899"
	canonical, _ := Parse("https://www.notion.so/" + page)
	for _, raw := range []string{
		"https://app.notion.com/p/" + page + "?v=11223344556677889900aabbccddeeff",
		"https://app.notion.com/p/" + page + "?p=" + page,
		"https://app.notion.com/settings/" + page,
		"https://app.notion.com.example.com/p/" + page,
	} {
		identity, err := Parse(raw)
		if err != nil || identity.PageID != "" || identity.SourceKey == canonical.SourceKey {
			t.Fatalf("guessed page identity: %+v %v", identity, err)
		}
	}
}
