package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yshujie/miniblog/internal/miniblog/notionsync"
	"github.com/yshujie/miniblog/internal/miniblog/source"
)

// A full scan always visits five known sources. Only the owned fixture source
// may contribute this page or a topic option; foreign sources remain valid empty
// libraries rather than projecting duplicate pages or misleading catalog plans.
func TestLocalNotionSeparatesAllFiveSources(t *testing.T) {
	page := source.NotionPage{ID: adminFixturePage, Object: "page"}
	page.Parent.Type = "data_source_id"
	page.Parent.DataSourceID = adminFixtureSource
	fixture := &adminNotionHTTP{page: page}
	for _, id := range notionsync.AllowedSources() {
		schema := adminNotionSchema(id)
		options := schema.Properties["主题"].Select.Options
		expected := 0
		if id == adminFixtureSource {
			expected = 1
		}
		if len(options) != expected {
			t.Fatalf("source %s has %d topic options; want %d", id, len(options), expected)
		}
		for _, archived := range []bool{false, true} {
			body := `{"is_archived":false}`
			if archived {
				body = `{"is_archived":true}`
			}
			req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1/v1/data_sources/"+id+"/query", strings.NewReader(body))
			response := httptest.NewRecorder()
			fixture.serve(response, req)
			var result source.NotionQueryResult
			if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
				t.Fatal("query must match real Notion HTTP envelope", err)
			}
			expectedPages := 0
			if id == adminFixtureSource && !archived {
				expectedPages = 1
			}
			if len(result.Results) != expectedPages {
				t.Fatalf("source %s archived=%v returned %d pages", id, archived, len(result.Results))
			}
			for _, p := range result.Results {
				if p.Parent.DataSourceID != id {
					t.Fatal("fixture page crossed source boundary")
				}
			}
		}
	}
	if fixture.writes != 0 {
		t.Fatal("query fixture attempted provider writes")
	}
}
