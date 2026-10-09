package contentintegration

// This acceptance layer uses the existing Go controllers, business services and
// MySQL repositories over loopback HTTP. It does not replace production routing,
// assert that existing authorization is secure, or contact a real provider.
import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	articlebiz "github.com/yshujie/miniblog/internal/miniblog/biz/article"
	articlecontroller "github.com/yshujie/miniblog/internal/miniblog/controller/v1/article"
	authcontroller "github.com/yshujie/miniblog/internal/miniblog/controller/v1/auth"
	blogcontroller "github.com/yshujie/miniblog/internal/miniblog/controller/v1/blog"
	catalogcontroller "github.com/yshujie/miniblog/internal/miniblog/controller/v1/catalog"
	modulecontroller "github.com/yshujie/miniblog/internal/miniblog/controller/v1/module"
	synccontroller "github.com/yshujie/miniblog/internal/miniblog/controller/v1/notionsync"
	sectioncontroller "github.com/yshujie/miniblog/internal/miniblog/controller/v1/section"
	usercontroller "github.com/yshujie/miniblog/internal/miniblog/controller/v1/user"
	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/notionsync"
	"github.com/yshujie/miniblog/internal/miniblog/source"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"github.com/yshujie/miniblog/internal/pkg/middleware"
	"github.com/yshujie/miniblog/pkg/auth"
	"gorm.io/gorm"
)

const adminFixturePassword = "LocalFixture12"
const adminFixtureSource = "2bf330bd-ddf1-80a6-aa49-000bbd1e154b"
const adminFixturePage = "abcdefabcdef4abc8abcabcdefabcdef"

type adminAPI struct {
	t      *testing.T
	db     *gorm.DB
	server *httptest.Server
	jwt    string
	authz  *auth.Authz
	sync   *notionsync.Service
}
type adminResponse struct {
	Code    string          `json:"code"`
	Payload json.RawMessage `json:"payload"`
}

func (a *adminAPI) send(method, path string, body any, authenticated bool) (int, adminResponse) {
	a.t.Helper()
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			a.t.Fatal(err)
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, a.server.URL+path, reader)
	if err != nil {
		a.t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	if authenticated {
		req.Header.Set("Authorization", "Bearer "+a.jwt)
	}
	resp, err := a.server.Client().Do(req)
	if err != nil {
		a.t.Fatal("loopback API request failed", err)
	}
	defer resp.Body.Close()
	var out adminResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		a.t.Fatal("invalid API response", err)
	}
	return resp.StatusCode, out
}
func adminPayload[T any](t *testing.T, response adminResponse) T {
	t.Helper()
	var out T
	if err := json.Unmarshal(response.Payload, &out); err != nil {
		t.Fatal("payload shape", err)
	}
	return out
}
func (a *adminAPI) ok(method, path string, body any) adminResponse {
	a.t.Helper()
	status, out := a.send(method, path, body, true)
	if status < 200 || status >= 300 || out.Code != "ok" {
		a.t.Fatalf("%s %s: HTTP %d / %s", method, path, status, out.Code)
	}
	return out
}
func (a *adminAPI) rejected(method, path string, body any) {
	a.t.Helper()
	status, out := a.send(method, path, body, true)
	if status < 400 || out.Code == "ok" {
		a.t.Fatalf("%s %s was accepted: HTTP %d / %s", method, path, status, out.Code)
	}
}
func adminFixture(t *testing.T, notionClient source.SyncNotionClient) *adminAPI {
	t.Helper()
	db, _ := scratchDB(t)
	migration(t, db, "000005_source_uniqueness.up.sql", "")
	t.Setenv("MINIBLOG_CONTENT_REGISTER_ENABLED", "true")
	t.Setenv("MINIBLOG_NOTION_TOKEN", "local-fixture-token")
	gin.SetMode(gin.TestMode)
	ds := store.NewStore(db)
	if err := db.Create(&model.UserM{Username: "fixtureadmin", Password: adminFixturePassword, Nickname: "Local author", Introduction: "Isolated fixture", Email: "fixture@example.invalid"}).Error; err != nil {
		t.Fatal(err)
	}
	authz, err := auth.NewAuthz(db)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = authz.AddPolicy("fixtureadmin", "/v1/admin/*", ".*"); err != nil {
		t.Fatal(err)
	}
	service := notionsync.New(ds, notionsync.Options{Enabled: true, Client: notionClient, Author: "Fixture author", OwnerID: "admin-ui-acceptance", Interval: time.Hour})
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := service.Stop(ctx); err != nil {
			t.Error("fixture service shutdown", err)
		}
	})
	router := gin.New()
	router.Use(gin.Recovery())
	ac := authcontroller.New(ds)
	router.POST("/v1/auth/login", ac.Login)
	router.POST("/v1/auth/logout", ac.Logout)
	bc := blogcontroller.New(ds)
	router.GET("/v1/blog/articleDetail", bc.GetArticleDetail)
	protected := router.Group("/v1/admin", middleware.Authn(), middleware.Authz(authz))
	uc := usercontroller.New(ds, authz)
	protected.GET("/users/myinfo", uc.GetMyInfo)
	ar := articlecontroller.New(ds)
	protected.POST("/article-sources/preview", ar.Preview)
	protected.POST("/articles/register", ar.Register)
	protected.GET("/articles", ar.GetList)
	protected.GET("/articles/:id", ar.GetOne)
	protected.PUT("/articles/:id", ar.Update)
	protected.PUT("/articles/:id/move", ar.Move)
	protected.PUT("/articles/:id/publish", ar.Publish)
	protected.PUT("/articles/:id/unpublish", ar.Unpublish)
	protected.PUT("/articles/:id/archive", ar.Archive)
	protected.PUT("/articles/:id/restore", ar.Restore)
	protected.PUT("/articles/reorder", ar.Reorder)
	protected.PATCH("/articles/:id/local-fields", ar.PatchLocal)
	protected.PUT("/articles/:id/publication-hold", ar.PublicationHold)
	mc := modulecontroller.New(ds)
	protected.GET("/modules", mc.GetAll)
	sc := sectioncontroller.New(ds)
	protected.POST("/sections", sc.Create)
	protected.PUT("/sections/:code", sc.Update)
	protected.PUT("/sections/:code/unpublish", sc.Unpublish)
	protected.PUT("/sections/:code/publish", sc.Publish)
	protected.DELETE("/sections/:code", sc.Delete)
	cc := catalogcontroller.New(ds)
	protected.POST("/catalog/reorder", cc.Reorder)
	nc := synccontroller.New(service)
	protected.GET("/notion-sync/status", nc.Status)
	protected.GET("/notion-sync/sources", nc.Sources)
	protected.GET("/notion-sync/pages", nc.Pages)
	protected.GET("/notion-sync/runs", nc.Runs)
	protected.GET("/notion-sync/runs/:run_id", nc.Run)
	protected.GET("/notion-sync/runs/:run_id/items", nc.Items)
	protected.POST("/notion-sync/runs", nc.Trigger)
	protected.PATCH("/notion-sync/control", nc.UpdateControl)
	protected.PATCH("/notion-sync/sources/:source_id", nc.UpdateSource)
	protected.PUT("/notion-sync/catalog-bindings/:binding_id", nc.BindCatalog)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	a := &adminAPI{t: t, db: db, server: server, authz: authz, sync: service}
	status, login := a.send("POST", "/v1/auth/login", map[string]string{"username": "fixtureadmin", "password": adminFixturePassword}, false)
	if status != 200 || login.Code != "ok" {
		t.Fatalf("fixture login: HTTP %d / %s", status, login.Code)
	}
	a.jwt = adminPayload[struct {
		Token string `json:"token"`
	}](t, login).Token
	if a.jwt == "" {
		t.Fatal("login did not return a token")
	}
	return a
}

// Preserve and report the existing contract, including known security gaps.
func TestAdminMySQLAuthenticationAndExistingLimits(t *testing.T) {
	a := adminFixture(t, nil)
	for _, path := range []string{"/v1/admin/users/myinfo", "/v1/admin/articles?page=1&limit=1", "/v1/admin/notion-sync/status"} {
		status, out := a.send("GET", path, nil, false)
		if status != 401 || out.Code == "ok" {
			t.Fatalf("anonymous %s: HTTP %d / %s", path, status, out.Code)
		}
	}
	status, bad := a.send("POST", "/v1/auth/login", map[string]string{"username": "fixtureadmin", "password": "WrongPassword"}, false)
	if status < 400 || bad.Code == "ok" {
		t.Fatal("incorrect password accepted")
	}
	info := adminPayload[struct {
		User struct {
			Nickname     string   `json:"nickname"`
			Introduction string   `json:"introduction"`
			Roles        []string `json:"roles"`
		} `json:"user"`
	}](t, a.ok("GET", "/v1/admin/users/myinfo", nil))
	if info.User.Nickname != "Local author" || info.User.Introduction != "Isolated fixture" || len(info.User.Roles) != 1 || info.User.Roles[0] != "admin" {
		t.Fatal("current account fields changed")
	}
	a.ok("POST", "/v1/auth/logout", map[string]string{"token": a.jwt})
	a.ok("GET", "/v1/admin/users/myinfo", nil)
	if _, err := a.authz.RemovePolicy("fixtureadmin", "/v1/admin/*", ".*"); err != nil {
		t.Fatal(err)
	}
	allowed, err := a.authz.Authorize("fixtureadmin", "/v1/admin/users/myinfo", "GET")
	if err != nil || allowed {
		t.Fatal("deny policy fixture did not deny")
	}
	a.ok("GET", "/v1/admin/users/myinfo", nil)
	t.Log("EXISTING LIMITS: logout does not revoke JWT; myinfo returns fixed admin role; Authz deny does not abort handler. These are observations, not security acceptance.")
}

func TestAdminMySQLManualCollectionCatalogAndCompleteOrdering(t *testing.T) {
	a := adminFixture(t, nil)
	register := func(link, title, sub string) string {
		response := a.ok("POST", "/v1/admin/articles/register", map[string]any{"external_link": link, "title": title, "section_code": "s1", "subsection_code": sub, "publish": true})
		payload := adminPayload[struct {
			Outcome string `json:"outcome"`
			Article struct {
				ID     string `json:"id"`
				Status string `json:"status"`
			} `json:"article"`
		}](t, response)
		if payload.Outcome != "created" || payload.Article.ID == "" || payload.Article.Status != "Published" {
			t.Fatal("registration response contract")
		}
		return payload.Article.ID
	}
	id := register("https://example.invalid/manual-one", "Manual one", "")
	second := register("https://example.invalid/manual-two", "Manual two", "")
	child := register("https://example.invalid/manual-child", "Child", "sub1")
	dup := adminPayload[struct {
		Outcome string                             `json:"outcome"`
		Article struct{ ID, Title, Status string } `json:"article"`
	}](t, a.ok("POST", "/v1/admin/articles/register", map[string]any{"external_link": "https://example.invalid/manual-one", "title": "Must not overwrite", "section_code": "s2", "publish": false}))
	if dup.Outcome != "already_registered" || dup.Article.ID != id || dup.Article.Title != "Manual one" || dup.Article.Status != "Published" {
		t.Fatal("duplicate changed original record")
	}
	direct := adminPayload[struct {
		Total    int               `json:"total"`
		Articles []json.RawMessage `json:"articles"`
	}](t, a.ok("GET", "/v1/admin/articles?section_code=s1&direct_only=true&page=1&limit=1", nil))
	if direct.Total != 2 || len(direct.Articles) != 1 {
		t.Fatal("direct-only pagination mixed child or page count", direct.Total, len(direct.Articles))
	}
	a.rejected("PUT", "/v1/admin/articles/reorder", map[string]any{"section_code": "s1", "article_ids": []string{id}})
	a.rejected("PUT", "/v1/admin/articles/reorder", map[string]any{"section_code": "s1", "article_ids": []string{id, second, child}})
	a.ok("PUT", "/v1/admin/articles/reorder", map[string]any{"section_code": "s1", "article_ids": []string{second, id}})
	a.ok("PUT", "/v1/admin/articles/"+id, map[string]any{"title": "Edited", "external_link": "https://example.invalid/manual-one", "module_code": "m1", "section_code": "s1", "author": "Local", "tags": []string{"tag"}})
	for _, step := range []struct{ Action, Status string }{{"unpublish", "Unpublished"}, {"publish", "Published"}, {"archive", "Deleted"}, {"restore", "Draft"}} {
		a.ok("PUT", "/v1/admin/articles/"+id+"/"+step.Action, nil)
		detail := adminPayload[struct {
			Article struct {
				Status string `json:"status"`
			} `json:"article"`
		}](t, a.ok("GET", "/v1/admin/articles/"+id, nil))
		if detail.Article.Status != step.Status {
			t.Fatal(step.Action, detail.Article.Status)
		}
	}
	a.ok("POST", "/v1/admin/sections", map[string]any{"code": "new-section", "title": "New section", "module_code": "m1", "sort": 2})
	a.ok("PUT", "/v1/admin/sections/new-section/unpublish", nil)
	a.rejected("POST", "/v1/admin/catalog/reorder", map[string]any{"kind": "section", "parent_code": "m1", "codes": []string{"s1"}})
	a.ok("POST", "/v1/admin/catalog/reorder", map[string]any{"kind": "section", "parent_code": "m1", "codes": []string{"new-section", "s1"}})
	a.rejected("DELETE", "/v1/admin/sections/s1", nil)
	a.ok("DELETE", "/v1/admin/sections/new-section", nil)
	t.Setenv("MINIBLOG_CONTENT_REGISTER_ENABLED", "false")
	a.rejected("POST", "/v1/admin/articles/register", map[string]any{"external_link": "https://example.invalid/disabled", "title": "No write", "section_code": "s1", "publish": true})
}

func seedAdminSource(t *testing.T, db *gorm.DB) {
	t.Helper()
	config := notionsync.SourceConfig{TitlePropertyID: "title", StatePropertyID: "state", TopicPropertyID: "topic", TagsPropertyID: "tags", StateOptionIDs: map[string]string{"draft": "draft", "published": "published", "unpublished": "unpublished", "archived": "archived"}}
	configJSON, _ := json.Marshal(config)
	stateJSON, _ := json.Marshal(config.StateOptionIDs)
	if err := db.Create(&model.NotionSyncSource{ID: adminFixtureSource, DataSourceID: adminFixtureSource, ModuleCode: "m1", Enabled: true, ConfigRevision: 1, PropertyMappingJSON: string(configJSON), StatusMappingJSON: string(stateJSON)}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.NotionSyncControl{}).Where("id=1").Updates(map[string]any{"paused": false, "source_writes_paused": false, "baseline_frozen": true}).Error; err != nil {
		t.Fatal(err)
	}
	section := "s1"
	if err := db.Create(&model.NotionCatalogBinding{SourceID: adminFixtureSource, DataSourceID: adminFixtureSource, ThemePropertyID: "topic", OptionID: "topic-one", OptionName: "One", SectionCode: &section, Status: "bound"}).Error; err != nil {
		t.Fatal(err)
	}
}
func TestAdminMySQLManagedAuthorHoldAndSourceRevision(t *testing.T) {
	a := adminFixture(t, nil)
	seedAdminSource(t, a.db)
	ds := store.NewStore(a.db)
	b := articlebiz.New(ds)
	ctx := context.Background()
	lease, ok, err := ds.NotionSync().AcquireLease(ctx, "seed", time.Minute)
	if err != nil || !ok {
		t.Fatal("seed lease")
	}
	public := "https://fixture.notion.site/" + adminFixturePage
	input := articlebiz.SyncInput{Lease: lease, RunID: "fixture-seed", SourceID: adminFixtureSource, DataSourceID: adminFixtureSource, PageID: adminFixturePage, ThemePropertyID: "topic", ThemeOptionID: "topic-one", ThemeOptionName: "One", Title: "Managed", Tags: []string{"Notion tag"}, PageURL: "https://notion.so/" + adminFixturePage, PublicURL: &public, DesiredState: 2, MetadataComplete: true, ExpectedConfigRevision: 1, MetadataHash: "initial"}
	applied, err := b.ApplySyncedSource(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	id := fmt.Sprint(applied.ArticleID)
	if err := ds.NotionSync().ReleaseLease(ctx, lease); err != nil {
		t.Fatal(err)
	}
	a.ok("PATCH", "/v1/admin/articles/"+id+"/local-fields", map[string]string{"author": "Changed local author"})
	a.rejected("PATCH", "/v1/admin/articles/"+id+"/local-fields", map[string]string{"author": "No overwrite", "title": "Wrong ownership"})
	a.rejected("PUT", "/v1/admin/articles/"+id, map[string]any{"title": "Wrong ownership", "section_code": "s1", "external_link": "https://notion.so/" + adminFixturePage})
	a.rejected("PUT", "/v1/admin/articles/"+id+"/unpublish", nil)
	before := adminPayload[struct {
		Article struct {
			Title, Author, Status string
			Management            struct{ Mode string } `json:"management"`
			Visible               bool                  `json:"effective_visibility"`
		} `json:"article"`
	}](t, a.ok("GET", "/v1/admin/articles/"+id, nil))
	if before.Article.Title != "Managed" || before.Article.Author != "Changed local author" || before.Article.Management.Mode != "notion_sync" || !before.Article.Visible {
		t.Fatal("managed ownership/local fields contract")
	}
	a.ok("PUT", "/v1/admin/articles/"+id+"/publication-hold", map[string]any{"held": true, "reason": "Local review"})
	a.rejected("GET", "/v1/blog/articleDetail?article_id="+id, nil)
	a.ok("PUT", "/v1/admin/articles/"+id+"/publication-hold", map[string]any{"held": false})
	a.rejected("GET", "/v1/blog/articleDetail?article_id="+id, nil)
	var page model.NotionPageBinding
	a.db.First(&page, "page_id=?", adminFixturePage)
	input.ExpectedBindingRevision = page.Revision
	lease, ok, err = ds.NotionSync().AcquireLease(ctx, "fresh", time.Minute)
	if err != nil || !ok {
		t.Fatal("fresh lease")
	}
	input.Lease = lease
	input.RunID = "fresh-validation"
	if _, err = b.ApplySyncedSource(ctx, input); err != nil {
		t.Fatal(err)
	}
	if err = ds.NotionSync().ReleaseLease(ctx, lease); err != nil {
		t.Fatal(err)
	}
	a.ok("GET", "/v1/blog/articleDetail?article_id="+id, nil)
	a.ok("PATCH", "/v1/admin/notion-sync/sources/"+adminFixtureSource, map[string]any{"enabled": false, "expected_config_revision": 1})
	a.ok("GET", "/v1/blog/articleDetail?article_id="+id, nil)
	status, _ := a.send("PATCH", "/v1/admin/notion-sync/sources/"+adminFixtureSource, map[string]any{"label": "Stale overwrite", "expected_config_revision": 1}, true)
	if status != 409 {
		t.Fatal("stale source revision did not conflict", status)
	}
	a.rejected("PATCH", "/v1/admin/notion-sync/control", map[string]any{"enabled": false})
	var binding model.NotionCatalogBinding
	if err := a.db.First(&binding, "source_id=?", adminFixtureSource).Error; err != nil {
		t.Fatal(err)
	}
	bindingPath := "/v1/admin/notion-sync/catalog-bindings/" + fmt.Sprint(binding.ID)
	a.rejected("PUT", bindingPath, map[string]any{"source_id": adminFixtureSource, "option_id": "topic-one", "section_code": "s2", "expected_config_revision": 2})
	a.ok("POST", "/v1/admin/sections", map[string]any{"code": "rebound", "title": "Rebound", "module_code": "m1"})
	a.ok("PUT", bindingPath, map[string]any{"source_id": adminFixtureSource, "option_id": "topic-one", "section_code": "rebound", "expected_config_revision": 2})
	status, _ = a.send("PUT", bindingPath, map[string]any{"source_id": adminFixtureSource, "option_id": "topic-one", "section_code": "s1", "expected_config_revision": 2}, true)
	if status != 409 {
		t.Fatal("stale binding revision did not conflict", status)
	}
	a.db.First(&page, "page_id=?", adminFixturePage)
	if !page.NeedsRevalidation {
		t.Fatal("binding change did not request revalidation")
	}
	a.rejected("GET", "/v1/blog/articleDetail?article_id="+id, nil)
	a.ok("PATCH", "/v1/admin/notion-sync/control", map[string]bool{"source_writes_paused": true})
	a.rejected("POST", "/v1/admin/articles/register", map[string]any{"external_link": "https://example.invalid/maintenance", "title": "Blocked collection", "section_code": "s1", "publish": true})
	a.ok("PATCH", "/v1/admin/articles/"+id+"/local-fields", map[string]string{"author": "Changed local author"})
	var final model.Article
	a.db.First(&final, applied.ArticleID)
	if final.Status != 2 || final.Author != "Changed local author" || final.Title != "Managed" {
		t.Fatal("source pause changed article ownership/publication")
	}
}

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
			p.Select.Options = []source.NotionOption{{ID: "topic-one", Name: "One"}}
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
func newAdminLocalNotion(t *testing.T) (*adminNotionHTTP, *source.HTTPNotionSyncClient) {
	t.Helper()
	page := source.NotionPage{ID: adminFixturePage, Object: "page", LastEditedTime: time.Now().UTC(), URL: "https://notion.so/" + adminFixturePage, Properties: map[string]source.NotionProperty{"标题": {ID: "title", Type: "title", Title: []source.NotionRichText{{PlainText: "Current Notion title"}}}, "博客状态": {ID: "state", Type: "select", Select: &source.NotionOption{ID: "published", Name: "已发布"}}, "主题": {ID: "topic", Type: "select", Select: &source.NotionOption{ID: "topic-one", Name: "One"}}, "知识点": {ID: "tags", Type: "multi_select", MultiSelect: []source.NotionOption{}}}}
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
	t.Cleanup(func() {
		http.DefaultTransport = old
		server.Close()
		if guard.rejected.Load() != 0 {
			t.Error("fixture attempted forbidden outbound network")
		}
	})
	return fake, source.NewNotionSyncClient("local-fixture-token")
}
func (a *adminAPI) run(mode string) notionsync.RunDTO {
	a.t.Helper()
	status, out := a.send("POST", "/v1/admin/notion-sync/runs", map[string]string{"mode": mode}, true)
	if status != 202 || out.Code != "ok" {
		a.t.Fatalf("trigger %s: HTTP %d / %s", mode, status, out.Code)
	}
	trigger := adminPayload[notionsync.TriggerResult](a.t, out)
	if trigger.RunID == "" {
		a.t.Fatal("run ID missing")
	}
	deadline := time.Now().Add(75 * time.Second)
	for time.Now().Before(deadline) {
		run := adminPayload[notionsync.RunDTO](a.t, a.ok("GET", "/v1/admin/notion-sync/runs/"+trigger.RunID, nil))
		if run.Status != "running" {
			if run.RunID != trigger.RunID || run.Mode != mode {
				a.t.Fatal("run identity or mode changed")
			}
			return run
		}
		time.Sleep(100 * time.Millisecond)
	}
	a.t.Fatal("local run timed out")
	return notionsync.RunDTO{}
}
func TestAdminMySQLNotionHTTPPreviewFreshExecutionAndMetadataFailure(t *testing.T) {
	fake, client := newAdminLocalNotion(t)
	a := adminFixture(t, client)
	seedAdminSource(t, a.db)
	title := adminPayload[map[string]any](t, a.ok("POST", "/v1/admin/article-sources/preview", map[string]string{"external_link": "https://notion.so/" + adminFixturePage}))
	if title["title"] != "Current Notion title" || title["metadata_status"] != "resolved" {
		t.Fatal("Notion metadata preview contract", title["metadata_status"])
	}
	var beforeSections, beforeArticles int64
	a.db.Model(&model.Section{}).Count(&beforeSections)
	a.db.Model(&model.Article{}).Count(&beforeArticles)
	dry := a.run("dry_run")
	if dry.Status == "failed" {
		t.Fatal("local preview failed", dry.Error)
	}
	var n int64
	a.db.Model(&model.Article{}).Count(&n)
	if n != beforeArticles {
		t.Fatal("preview wrote articles")
	}
	a.db.Model(&model.Section{}).Count(&n)
	if n != beforeSections {
		t.Fatal("preview wrote catalog")
	}
	fake.mu.Lock()
	fake.page.Properties["标题"] = source.NotionProperty{ID: "title", Type: "title", Title: []source.NotionRichText{{PlainText: "Fresh title after preview"}}}
	fake.page.LastEditedTime = fake.page.LastEditedTime.Add(time.Second)
	fake.mu.Unlock()
	actual := a.run("sync")
	if actual.Status == "failed" || actual.Counts.Created != 1 {
		t.Fatalf("fresh sync status=%s created=%d", actual.Status, actual.Counts.Created)
	}
	var article model.Article
	if err := a.db.First(&article).Error; err != nil {
		t.Fatal(err)
	}
	if article.Title != "Fresh title after preview" {
		t.Fatal("sync replayed old preview")
	}
	items := adminPayload[notionsync.PageResult[notionsync.ItemDTO]](t, a.ok("GET", "/v1/admin/notion-sync/runs/"+actual.RunID+"/items?page=1&limit=20", nil))
	if items.Total < 1 {
		t.Fatal("run items missing")
	}
	fake.mu.Lock()
	fake.failPage = true
	fake.mu.Unlock()
	unavailable := adminPayload[map[string]any](t, a.ok("POST", "/v1/admin/article-sources/preview", map[string]string{"external_link": "https://notion.so/" + adminFixturePage}))
	if unavailable["metadata_status"] != "manual_required" {
		t.Fatal("failed title read reported available")
	}
	fallbackPage := "22222222222242228222222222222222"
	fallback := adminPayload[map[string]any](t, a.ok("POST", "/v1/admin/article-sources/preview", map[string]string{"external_link": "https://notion.so/" + fallbackPage}))
	if fallback["metadata_status"] != "manual_required" {
		t.Fatal("missing metadata should allow a manual title")
	}
	manual := adminPayload[map[string]any](t, a.ok("POST", "/v1/admin/articles/register", map[string]any{"external_link": "https://notion.so/" + fallbackPage, "title": "Manual title despite metadata failure", "section_code": "s1", "publish": true}))
	if manual["outcome"] != "created" {
		t.Fatal("metadata failure prevented manual collection")
	}
	a.rejected("POST", "/v1/admin/notion-sync/runs", map[string]string{"mode": "bootstrap_apply"})
	a.rejected("POST", "/v1/admin/notion-sync/runs", map[string]any{"mode": "sync", "confirmations": []string{}})
	a.ok("PATCH", "/v1/admin/notion-sync/control", map[string]bool{"paused": true})
	a.rejected("POST", "/v1/admin/notion-sync/runs", map[string]string{"mode": "sync"})
	fake.mu.Lock()
	requests, writes := fake.requests, fake.writes
	fake.mu.Unlock()
	if requests == 0 || writes != 0 {
		t.Fatal("provider boundary was not read-only", requests, writes)
	}
	t.Logf("Real Go HTTP/MySQL, local Notion HTTP substitute: %d provider reads, %d provider writes; no real Notion acceptance.", requests, writes)
}
