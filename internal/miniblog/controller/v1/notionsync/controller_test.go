package notionsync

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	syncbiz "github.com/yshujie/miniblog/internal/miniblog/notionsync"
)

type fixtureAPI struct {
	syncbiz.API
	err     error
	calls   int
	query   syncbiz.ListQuery
	binding syncbiz.BindingInput
}

func (f *fixtureAPI) Trigger(_ context.Context, _ syncbiz.TriggerInput) (*syncbiz.TriggerResult, error) {
	f.calls++
	return &syncbiz.TriggerResult{RunID: "9007199254740993"}, f.err
}
func (f *fixtureAPI) Status(context.Context) (*syncbiz.StatusDTO, error) {
	return &syncbiz.StatusDTO{}, f.err
}
func (f *fixtureAPI) Sources(_ context.Context, q syncbiz.ListQuery) (*syncbiz.PageResult[syncbiz.SourceDTO], error) {
	f.calls++
	f.query = q
	return &syncbiz.PageResult[syncbiz.SourceDTO]{Items: []syncbiz.SourceDTO{}, Page: q.Page, Limit: q.Limit}, f.err
}
func (f *fixtureAPI) BindCatalog(_ context.Context, in syncbiz.BindingInput) (*syncbiz.TopicBindingDTO, error) {
	f.calls++
	f.binding = in
	return &syncbiz.TopicBindingDTO{ID: in.BindingID}, f.err
}
func request(t *testing.T, f *fixtureAPI, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	c := New(f)
	router.GET("/status", c.Status)
	router.GET("/sources", c.Sources)
	router.POST("/runs", c.Trigger)
	router.PUT("/catalog-bindings/:binding_id", c.BindCatalog)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	return w
}
func TestTriggerReturnsAcceptedStringID(t *testing.T) {
	for _, mode := range []string{"dry_run", "sync"} {
		f := &fixtureAPI{}
		w := request(t, f, "POST", "/runs", `{"mode":"`+mode+`"}`)
		if w.Code != http.StatusAccepted || !strings.Contains(w.Body.String(), `"run_id":"9007199254740993"`) || f.calls != 1 {
			t.Fatalf("%d %s", w.Code, w.Body.String())
		}
	}
}
func TestTriggerRejectsMalformedOrBootstrapCommands(t *testing.T) {
	for _, body := range []string{`{"mode":"bootstrap"}`, `{"mode":"preview"}`, `{"mode":"sync","confirmations":[]}`, `{"mode":"sync"} {}`, `{}`, `null`} {
		f := &fixtureAPI{}
		w := request(t, f, "POST", "/runs", body)
		if w.Code != 400 || f.calls != 0 {
			t.Fatalf("%s: %d %s", body, w.Code, w.Body.String())
		}
	}
}
func TestApplicationErrorsAndUnknownFailureSanitization(t *testing.T) {
	for _, test := range []struct {
		err      error
		status   int
		contains string
	}{
		{&syncbiz.Error{HTTPStatus: 409, Code: "conflict", Message: "stale configuration"}, 409, "stale configuration"},
		{&syncbiz.Error{HTTPStatus: 503, Code: "not_configured", Message: "sync disabled"}, 503, "sync disabled"},
		{errors.New("mysql secret-password and internal path"), 500, ""},
	} {
		w := request(t, &fixtureAPI{err: test.err}, "GET", "/status", "")
		if w.Code != test.status || !strings.Contains(w.Body.String(), test.contains) || strings.Contains(w.Body.String(), "secret-password") {
			t.Fatalf("%d %s", w.Code, w.Body.String())
		}
	}
}
func TestListBoundsAndCatalogPathIdentity(t *testing.T) {
	f := &fixtureAPI{}
	w := request(t, f, "GET", "/sources", "")
	if w.Code != 200 || f.query.Page != 1 || f.query.Limit != 20 {
		t.Fatal(f.query, w.Body.String())
	}
	for _, q := range []string{"page=0", "limit=0", "limit=101", "page=bad"} {
		f := &fixtureAPI{}
		w := request(t, f, "GET", "/sources?"+q, "")
		if w.Code != 400 || f.calls != 0 {
			t.Fatal(q, w.Code)
		}
	}
	f = &fixtureAPI{}
	w = request(t, f, "PUT", "/catalog-bindings/9007199254740993", `{"source_id":"go","option_id":"opaque%3A","section_code":"s1"}`)
	if w.Code != 200 || f.binding.BindingID != "9007199254740993" || f.binding.OptionID != "opaque%3A" {
		t.Fatal(f.binding, w.Body.String())
	}
	w = request(t, &fixtureAPI{}, "PUT", "/catalog-bindings/1", `{"binding_id":"2"}`)
	if w.Code != 400 {
		t.Fatal("body must not replace path identity", w.Code)
	}
}
