package source

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

type syncRoundTrip func(*http.Request) (*http.Response, error)

func (f syncRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func syncResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}
}
func testSyncClient(f syncRoundTrip) *HTTPNotionSyncClient {
	c := NewNotionSyncClient("fixture-token")
	c.client.Transport = f
	c.spacing = time.Millisecond
	return c
}

const fixturePageID = "3c90c3cc-0d44-4b50-8888-8dd25736052a"

func TestSyncClientOfficialEndpointPartitionAndVersion(t *testing.T) {
	c := testSyncClient(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host != "api.notion.com" || r.URL.Scheme != "https" || r.Header.Get("Notion-Version") != NotionSyncVersion {
			t.Fatalf("unexpected request boundary: %s", r.URL)
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"is_archived":true`) || !strings.Contains(string(body), `"page_size":100`) || !strings.Contains(string(body), `"start_cursor":"opaque"`) {
			t.Fatalf("query body %s", body)
		}
		return syncResponse(200, `{"results":[],"has_more":false,"request_status":{"type":"complete"}}`), nil
	})
	if _, e := c.QueryDataSource(context.Background(), fixturePageID, true, "opaque"); e != nil {
		t.Fatal(e)
	}
}
func TestSyncClientRejectsIncompleteAndMissingCursor(t *testing.T) {
	for _, body := range []string{`{"results":[],"has_more":false,"request_status":{"type":"incomplete"}}`, `{"results":[],"has_more":false}`, `{"results":[],"has_more":true,"request_status":{"type":"complete"}}`} {
		c := testSyncClient(func(*http.Request) (*http.Response, error) { return syncResponse(200, body), nil })
		if _, e := c.QueryDataSource(context.Background(), fixturePageID, false, ""); e == nil {
			t.Fatalf("accepted %s", body)
		}
	}
}
func TestSyncClientReadRetriesBut404AndWritesDoNot(t *testing.T) {
	count := 0
	c := testSyncClient(func(*http.Request) (*http.Response, error) {
		count++
		return syncResponse(500, `{"code":"internal_server_error"}`), nil
	})
	if _, e := c.RetrievePage(context.Background(), fixturePageID); e == nil || count != 3 {
		t.Fatalf("read attempts=%d error=%v", count, e)
	}
	count = 0
	c = testSyncClient(func(*http.Request) (*http.Response, error) {
		count++
		return syncResponse(404, `{"code":"object_not_found"}`), nil
	})
	if _, e := c.RetrievePage(context.Background(), fixturePageID); !IsNotionUnavailable(e) || count != 1 {
		t.Fatalf("404 attempts=%d error=%v", count, e)
	}
	count = 0
	c = testSyncClient(func(*http.Request) (*http.Response, error) {
		count++
		return syncResponse(503, `{"code":"service_unavailable"}`), nil
	})
	if e := c.UpdateBlogState(context.Background(), fixturePageID, "state-property", "option"); e == nil || count != 1 {
		t.Fatalf("write attempts=%d error=%v", count, e)
	}
}
func TestSyncClientCooldownCanBeCancelled(t *testing.T) {
	count := 0
	c := testSyncClient(func(*http.Request) (*http.Response, error) {
		count++
		r := syncResponse(429, `{"code":"rate_limited"}`)
		r.Header.Set("Retry-After", "60")
		return r, nil
	})
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, e := c.RetrievePage(ctx, fixturePageID); !errors.Is(e, context.DeadlineExceeded) || count != 1 {
		t.Fatalf("cooldown attempts=%d error=%v", count, e)
	}
}
func TestSyncClientSerializesAndUsesPageIdentity(t *testing.T) {
	var mu sync.Mutex
	inflight, max := 0, 0
	c := testSyncClient(func(*http.Request) (*http.Response, error) {
		mu.Lock()
		inflight++
		if inflight > max {
			max = inflight
		}
		mu.Unlock()
		time.Sleep(5 * time.Millisecond)
		mu.Lock()
		inflight--
		mu.Unlock()
		return syncResponse(200, `{"id":"`+fixturePageID+`"}`), nil
	})
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, e := c.RetrievePage(context.Background(), fixturePageID); e != nil {
				t.Error(e)
			}
		}()
	}
	wg.Wait()
	if max != 1 {
		t.Fatalf("concurrent requests=%d", max)
	}
	identity, e := NotionIdentity(fixturePageID)
	if e != nil || identity.PageID != "3c90c3cc0d444b5088888dd25736052a" {
		t.Fatalf("identity %+v %v", identity, e)
	}
}
func TestSyncClientWithoutTokenHasNoNetwork(t *testing.T) {
	c := NewNotionSyncClient("")
	c.client.Transport = syncRoundTrip(func(*http.Request) (*http.Response, error) { t.Fatal("unexpected network"); return nil, nil })
	if _, e := c.RetrievePage(context.Background(), fixturePageID); e == nil {
		t.Fatal("expected missing configuration")
	}
}

func TestSyncClientReportsCooldownBeforeRetry(t *testing.T) {
	c := testSyncClient(func(*http.Request) (*http.Response, error) {
		r := syncResponse(429, `{"code":"rate_limited"}`)
		r.Header.Set("Retry-After", "60")
		return r, nil
	})
	var observed time.Time
	c.SetCooldownObserver(func(_ context.Context, until time.Time) error { observed = until; return nil })
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, _ = c.RetrievePage(ctx, fixturePageID)
	if observed.Before(time.Now().Add(50 * time.Second)) {
		t.Fatalf("cooldown not reported: %v", observed)
	}
}
