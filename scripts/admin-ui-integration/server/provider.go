package main

import (
	"encoding/json"
	"errors"
	"github.com/yshujie/miniblog/internal/miniblog/source"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// The real Notion HTTP client is redirected exclusively to this loopback
// replacement. A guard rejects every other non-loopback outgoing request.
type adminLocalTransport struct {
	target   *url.URL
	next     http.RoundTripper
	rejected atomic.Int64
}

func (tr *adminLocalTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	copyURL := *req.URL
	clone.URL = &copyURL
	if req.URL.Hostname() == "api.notion.com" {
		clone.URL.Scheme = tr.target.Scheme
		clone.URL.Host = tr.target.Host
		clone.Host = tr.target.Host
		clone.Header.Del("Authorization")
	} else if req.URL.Hostname() != "127.0.0.1" && req.URL.Hostname() != "localhost" && req.URL.Hostname() != "::1" {
		tr.rejected.Add(1)
		return nil, errors.New("external requests forbidden by local acceptance fixture")
	}
	return tr.next.RoundTrip(clone)
}

type adminNotionHTTP struct {
	mu       sync.Mutex
	page     source.NotionPage
	requests int
	writes   int
	failPage bool
}

func adminNotionSchema(id string) source.NotionDataSource {
	out := source.NotionDataSource{ID: id, Properties: map[string]source.NotionSchemaProperty{}}
	for name, p := range map[string]source.NotionSchemaProperty{"标题": {ID: "title", Type: "title"}, "博客状态": {ID: "state", Type: "select"}, "主题": {ID: "topic", Type: "select"}, "知识点": {ID: "tags", Type: "multi_select"}} {
		p.Name = name
		if p.ID == "state" {
			p.Select.Options = []source.NotionOption{{ID: "draft", Name: "草稿"}, {ID: "published", Name: "已发布"}, {ID: "unpublished", Name: "已下架"}, {ID: "archived", Name: "归档"}}
		}
		if p.ID == "topic" {
			p.Select.Options = []source.NotionOption{}
			if id == adminFixtureSource {
				p.Select.Options = []source.NotionOption{{ID: "topic-one", Name: "接口章节"}}
			}
		}
		out.Properties[name] = p
	}
	return out
}
func (f *adminNotionHTTP) serve(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.requests++
	w.Header().Set("Content-Type", "application/json")
	path := strings.TrimPrefix(r.URL.Path, "/v1/")
	if r.Method == "PATCH" {
		f.writes++
		w.WriteHeader(405)
		return
	}
	if strings.HasPrefix(path, "data_sources/") {
		parts := strings.Split(path, "/")
		id := parts[1]
		if len(parts) == 2 {
			json.NewEncoder(w).Encode(adminNotionSchema(id))
			return
		}
		var body struct {
			Archived bool `json:"is_archived"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		result := source.NotionQueryResult{Object: "list", Type: "page_or_data_source", Results: []source.NotionPage{}}
		result.RequestStatus.Type = "complete"
		if id == adminFixtureSource && !body.Archived {
			result.Results = append(result.Results, f.page)
		}
		json.NewEncoder(w).Encode(map[string]any{"object": "list", "type": "page_or_data_source", "page_or_data_source": map[string]any{}, "results": result.Results, "has_more": false, "next_cursor": nil, "request_status": map[string]string{"type": "complete"}})
		return
	}
	if path == "pages/"+adminFixturePage {
		if f.failPage {
			w.WriteHeader(404)
			io.WriteString(w, `{"code":"object_not_found"}`)
			return
		}
		json.NewEncoder(w).Encode(f.page)
		return
	}
	w.WriteHeader(404)
}

func localNotion() (*adminNotionHTTP, *source.HTTPNotionSyncClient, func()) {
	page := source.NotionPage{ID: adminFixturePage, Object: "page", LastEditedTime: time.Now().UTC(), URL: "https://notion.so/" + adminFixturePage, Properties: map[string]source.NotionProperty{"标题": {ID: "title", Type: "title", Title: []source.NotionRichText{{PlainText: "真实API托管文章"}}}, "博客状态": {ID: "state", Type: "select", Select: &source.NotionOption{ID: "published", Name: "已发布"}}, "主题": {ID: "topic", Type: "select", Select: &source.NotionOption{ID: "topic-one", Name: "接口章节"}}, "知识点": {ID: "tags", Type: "multi_select", MultiSelect: []source.NotionOption{}}}}
	page.Parent.Type = "data_source_id"
	page.Parent.DataSourceID = adminFixtureSource
	public := "https://fixture.notion.site/" + adminFixturePage
	page.PublicURL = &public
	fake := &adminNotionHTTP{page: page}
	server := httptest.NewServer(http.HandlerFunc(fake.serve))
	target, _ := url.Parse(server.URL)
	old := http.DefaultTransport
	guard := &adminLocalTransport{target: target, next: old}
	http.DefaultTransport = guard
	return fake, source.NewNotionSyncClient("local-fixture-token"), func() { server.Close(); http.DefaultTransport = old }
}
