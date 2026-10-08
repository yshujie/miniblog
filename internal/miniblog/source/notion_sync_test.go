package source

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"
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
		return syncResponse(200, queryEnvelope(`[]`, false, `null`, `{"type":"complete"}`)), nil
	})
	if _, e := c.QueryDataSource(context.Background(), fixturePageID, true, "opaque"); e != nil {
		t.Fatal(e)
	}
}
func TestSyncClientRejectsIncompleteAndMissingCursor(t *testing.T) {
	for _, body := range []string{queryEnvelope(`[]`, false, `null`, `{"type":"incomplete"}`), queryEnvelope(`[]`, true, `null`, `{"type":"complete"}`)} {
		c := testSyncClient(func(*http.Request) (*http.Response, error) { return syncResponse(200, body), nil })
		if _, e := c.QueryDataSource(context.Background(), fixturePageID, false, ""); e == nil {
			t.Fatalf("accepted %s", body)
		}
	}
}

func queryEnvelope(results string, more bool, cursor, status string) string {
	body := `{"object":"list","type":"page_or_data_source","page_or_data_source":{},"results":` + results + `,"has_more":` + strconv.FormatBool(more) + `,"next_cursor":` + cursor
	if status != "" {
		body += `,"request_status":` + status
	}
	return body + "}"
}

func TestSyncClientAllowsOnlyGenuinelyOmittedQueryStatus(t *testing.T) {
	for _, status := range []string{"", `{"type":"complete"}`} {
		c := testSyncClient(func(*http.Request) (*http.Response, error) {
			return syncResponse(200, queryEnvelope(`[]`, false, `null`, status)), nil
		})
		result, err := c.QueryDataSource(context.Background(), fixturePageID, false, "")
		if err != nil || result.HasMore || result.NextCursor != nil || len(result.Results) != 0 {
			t.Fatalf("valid empty query envelope rejected: %v", err)
		}
	}
	for _, status := range []string{`null`, `{}`, `{"type":null}`, `{"type":""}`, `{"type":"future"}`, `{"type":true}`, `[]`, `"complete"`, `{"type":"complete","incomplete_reason":null}`, `{"type":"complete","incomplete_reason":"future"}`, `{"type":"incomplete"}`, `{"type":"incomplete","incomplete_reason":"query_result_limit_reached"}`} {
		c := testSyncClient(func(*http.Request) (*http.Response, error) {
			return syncResponse(200, queryEnvelope(`[]`, false, `null`, status)), nil
		})
		if _, err := c.QueryDataSource(context.Background(), fixturePageID, false, ""); err == nil {
			t.Fatalf("invalid/explicit incomplete status accepted: %s", status)
		}
	}
}

func TestSyncClientMalformedQueryEnvelopeCannotContributePages(t *testing.T) {
	page := `[{"object":"page","id":"` + fixturePageID + `"}]`
	valid := queryEnvelope(page, false, `null`, "")
	tooMany := "[" + strings.TrimSuffix(strings.Repeat(strings.Trim(page, "[]")+",", 101), ",") + "]"
	for name, body := range map[string]string{
		"empty":                   `{}`,
		"null envelope":           `null`,
		"not list":                strings.Replace(valid, `"object":"list"`, `"object":"error"`, 1),
		"missing object":          strings.Replace(valid, `"object":"list",`, "", 1),
		"missing type":            strings.Replace(valid, `"type":"page_or_data_source",`, "", 1),
		"wrong type":              strings.Replace(valid, `"type":"page_or_data_source"`, `"type":"future"`, 1),
		"missing type metadata":   strings.Replace(valid, `"page_or_data_source":{},`, "", 1),
		"null type metadata":      strings.Replace(valid, `"page_or_data_source":{}`, `"page_or_data_source":null`, 1),
		"nonempty type metadata":  strings.Replace(valid, `"page_or_data_source":{}`, `"page_or_data_source":{"future":1}`, 1),
		"missing results":         strings.Replace(valid, `"results":`+page+`,`, "", 1),
		"null results":            queryEnvelope(`null`, false, `null`, ""),
		"object results":          queryEnvelope(`{}`, false, `null`, ""),
		"overlarge result page":   queryEnvelope(tooMany, false, `null`, ""),
		"null result item":        queryEnvelope(`[null]`, false, `null`, ""),
		"empty result item":       queryEnvelope(`[{}]`, false, `null`, ""),
		"unknown result object":   queryEnvelope(`[{"object":"future","id":"`+fixturePageID+`"}]`, false, `null`, ""),
		"invalid result identity": queryEnvelope(`[{"object":"page","id":"invalid"}]`, false, `null`, ""),
		"missing has more":        strings.Replace(valid, `,"has_more":false`, "", 1),
		"null has more":           strings.Replace(valid, `"has_more":false`, `"has_more":null`, 1),
		"string has more":         strings.Replace(valid, `"has_more":false`, `"has_more":"false"`, 1),
		"missing cursor":          strings.Replace(valid, `,"next_cursor":null`, "", 1),
		"more without cursor":     queryEnvelope(page, true, `null`, ""),
		"more empty cursor":       queryEnvelope(page, true, `""`, ""),
		"more blank cursor":       queryEnvelope(page, true, `" "`, ""),
		"terminal cursor":         queryEnvelope(page, false, `"opaque"`, ""),
		"terminal empty cursor":   queryEnvelope(page, false, `""`, ""),
		"number cursor":           queryEnvelope(page, true, `1`, ""),
		"null status":             queryEnvelope(page, false, `null`, `null`),
		"unknown status":          queryEnvelope(page, false, `null`, `{"type":"future"}`),
		"duplicate status":        strings.TrimSuffix(valid, "}") + `,"request_status":{"type":"incomplete"},"request_status":{"type":"complete"}}`,
		"duplicate status type":   queryEnvelope(page, false, `null`, `{"type":"incomplete","type":"complete"}`),
		"trailing JSON":           valid + `{}`,
	} {
		t.Run(name, func(t *testing.T) {
			c := testSyncClient(func(*http.Request) (*http.Response, error) { return syncResponse(200, body), nil })
			result, err := c.QueryDataSource(context.Background(), fixturePageID, false, "")
			if err == nil || len(result.Results) != 0 {
				t.Fatal("malformed envelope supplied trusted pages")
			}
		})
	}
	c := testSyncClient(func(*http.Request) (*http.Response, error) {
		return syncResponse(200, queryEnvelope(page, false, `null`, `{"type":"incomplete"}`)), nil
	})
	if result, err := c.QueryDataSource(context.Background(), fixturePageID, false, ""); err == nil || len(result.Results) != 1 {
		t.Fatal("reliable incomplete response lost useful pages or its error")
	}
}

func TestSyncClientOmittedStatusWithOpaqueCursor(t *testing.T) {
	count := 0
	c := testSyncClient(func(r *http.Request) (*http.Response, error) {
		count++
		if count == 1 {
			return syncResponse(200, queryEnvelope(`[]`, true, `"opaque/value%3A"`, "")), nil
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"start_cursor":"opaque/value%3A"`) {
			t.Fatal("opaque cursor was rewritten")
		}
		return syncResponse(200, queryEnvelope(`[]`, false, `null`, "")), nil
	})
	first, err := c.QueryDataSource(context.Background(), fixturePageID, false, "")
	if err != nil || !first.HasMore || first.NextCursor == nil {
		t.Fatal("valid missing-status continuation rejected", err)
	}
	if _, err := c.QueryDataSource(context.Background(), fixturePageID, false, *first.NextCursor); err != nil {
		t.Fatal("valid missing-status terminal rejected", err)
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
