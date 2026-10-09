package notionsync

import (
	"context"
	"errors"
	"fmt"
	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/source"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"math"
	"strconv"
	"sync"
	"testing"
	"time"
)

type fixtureNotion struct {
	mu           sync.Mutex
	pages        map[string]source.NotionPage
	queryError   string
	incomplete   string
	getError     map[string]error
	queries      int
	writes       int
	writeErr     error
	writeThenErr bool
}

func (f *fixtureNotion) RetrieveDataSource(_ context.Context, id string) (source.NotionDataSource, error) {
	var schema source.NotionDataSource
	schema.ID = id
	schema.Properties = map[string]source.NotionSchemaProperty{}
	for name, p := range map[string]source.NotionSchemaProperty{"标题": {ID: "title", Type: "title"}, "博客状态": {ID: "state%3Aid", Type: "select"}, "主题": {ID: "topic", Type: "select"}, "知识点": {ID: "tags", Type: "multi_select"}} {
		p.Name = name
		if name == "博客状态" {
			p.Select.Options = []source.NotionOption{{ID: "draft", Name: "草稿"}, {ID: "published", Name: "已发布"}, {ID: "unpublished", Name: "已下架"}, {ID: "archived", Name: "归档"}}
		}
		if name == "主题" {
			p.Select.Options = []source.NotionOption{{ID: "opaque-option", Name: "主题一"}}
		}
		schema.Properties[name] = p
	}
	return schema, nil
}
func (f *fixtureNotion) QueryDataSource(_ context.Context, id string, archived bool, cursor string) (source.NotionQueryResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.queries++
	var result source.NotionQueryResult
	result.RequestStatus.Type = "complete"
	if id == f.queryError {
		return result, errors.New("fixture unavailable")
	}
	if id == f.incomplete {
		result.RequestStatus.Type = "incomplete"
	}
	for _, p := range f.pages {
		if normalizeID(p.Parent.DataSourceID) == id && p.IsArchived == archived {
			result.Results = append(result.Results, p)
		}
	}
	return result, nil
}
func (f *fixtureNotion) RetrievePage(_ context.Context, id string) (source.NotionPage, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if e := f.getError[normalizePageID(id)]; e != nil {
		return source.NotionPage{}, e
	}
	p, ok := f.pages[normalizePageID(id)]
	if !ok {
		return p, &source.NotionAPIError{StatusCode: 404, Code: "object_not_found"}
	}
	return p, nil
}
func (f *fixtureNotion) UpdateBlogState(_ context.Context, id, property, option string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.writes++
	if f.writeErr != nil && !f.writeThenErr {
		return f.writeErr
	}
	p := f.pages[normalizePageID(id)]
	v := p.Properties["博客状态"]
	v.Select = &source.NotionOption{ID: option, Name: option}
	p.Properties["博客状态"] = v
	p.LastEditedTime = p.LastEditedTime.Add(time.Second)
	f.pages[normalizePageID(id)] = p
	return f.writeErr
}
func fixturePage(id, srcID, state string) source.NotionPage {
	var p source.NotionPage
	p.ID = id
	p.Object = "page"
	p.Parent.Type = "data_source_id"
	p.Parent.DataSourceID = srcID
	p.LastEditedTime = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	p.URL = "https://app.notion.com/p/" + id
	public := "https://fixture.notion.site/" + id
	p.PublicURL = &public
	title := source.NotionRichText{PlainText: "Fixture title"}
	p.Properties = map[string]source.NotionProperty{"标题": {ID: "title", Type: "title", Title: []source.NotionRichText{title}}, "博客状态": {ID: "state%3Aid", Type: "select"}, "主题": {ID: "topic", Type: "select", Select: &source.NotionOption{ID: "opaque-option", Name: "主题一"}}, "知识点": {ID: "tags", Type: "multi_select", MultiSelect: []source.NotionOption{{ID: "tag-one", Name: "知识点一"}}}}
	if state != "" {
		v := p.Properties["博客状态"]
		v.Select = &source.NotionOption{ID: state, Name: state}
		p.Properties["博客状态"] = v
	}
	return p
}

const firstPage = "11111111111141118111111111111111"
const secondPage = "22222222222242228222222222222222"

func syncFixture(t *testing.T) (*Service, *gorm.DB, *fixtureNotion) {
	t.Helper()
	db, e := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if e != nil {
		t.Fatal(e)
	}
	sql, e := db.DB()
	if e != nil {
		t.Fatal(e)
	}
	sql.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sql.Close() })
	if e = db.AutoMigrate(&model.Module{}, &model.Section{}, &model.Subsection{}, &model.Article{}, &model.NotionSyncControl{}, &model.NotionSyncSource{}, &model.NotionCatalogBinding{}, &model.NotionPageBinding{}, &model.NotionSyncRun{}, &model.NotionSyncRunItem{}); e != nil {
		t.Fatal(e)
	}
	if e = db.Create(&model.NotionSyncControl{ID: 1}).Error; e != nil {
		t.Fatal(e)
	}
	for _, m := range []model.Module{{Code: "m1", Title: "Module 1", Status: 1}, {Code: "m2", Title: "Module 2", Status: 1}} {
		if e = db.Create(&m).Error; e != nil {
			t.Fatal(e)
		}
	}
	if e = db.Create(&model.Section{Code: "s1", Title: "Historical section", ModuleCode: "m1", Status: 1}).Error; e != nil {
		t.Fatal(e)
	}
	f := &fixtureNotion{pages: map[string]source.NotionPage{}, getError: map[string]error{}}
	s := New(store.NewStore(db), Options{Enabled: true, Author: "Default author", Client: f})
	return s, db, f
}
func waitFixtureRun(t *testing.T, s *Service, mode string) *RunDTO {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	r, e := s.Trigger(ctx, TriggerInput{Mode: mode})
	if e != nil {
		t.Fatal(e)
	}
	for {
		run, e := s.Run(ctx, r.RunID)
		if e != nil {
			t.Fatal(e)
		}
		if run.Status != "running" {
			var c model.NotionSyncControl
			s.ds.DB().First(&c, 1)
			if c.CurrentRunID == "" {
				return run
			}
		}
		select {
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		case <-time.After(5 * time.Millisecond):
		}
	}
}
func enableFixtureSource(t *testing.T, s *Service, id, module string) {
	t.Helper()
	on := true
	_, e := s.UpdateSource(context.Background(), id, SourceInput{ModuleCode: module, Enabled: &on})
	if e != nil {
		t.Fatal(e)
	}
}
func TestCompleteBaselineFreezesEveryPageWithoutPublishing(t *testing.T) {
	s, db, f := syncFixture(t)
	ids := AllowedSources()
	f.pages[firstPage] = fixturePage(firstPage, ids[0], "")
	f.pages[secondPage] = fixturePage(secondPage, ids[1], "published")
	run := waitFixtureRun(t, s, "dry_run")
	if run.Status != "completed" || run.Counts.Pending != 2 || run.Counts.Blocked != 0 || run.Counts.Failed != 0 {
		t.Fatal(run)
	}
	var c model.NotionSyncControl
	db.First(&c, 1)
	var pages []model.NotionPageBinding
	db.Find(&pages)
	var articles int64
	db.Model(&model.Article{}).Count(&articles)
	if !c.BaselineFrozen || len(pages) != 2 || articles != 0 || f.queries != 10 || f.writes != 0 {
		t.Fatalf("baseline %+v pages=%d articles=%d queries=%d writes=%d", c, len(pages), articles, f.queries, f.writes)
	}
	for _, p := range pages {
		if p.ManagementState != "baseline_pending" {
			t.Fatal(p)
		}
	}
}
func TestIncompleteScanCannotFreezeOrApply(t *testing.T) {
	s, db, f := syncFixture(t)
	f.pages[firstPage] = fixturePage(firstPage, AllowedSources()[0], "published")
	f.incomplete = AllowedSources()[4]
	run := waitFixtureRun(t, s, "dry_run")
	if run.Status != "failed" {
		t.Fatal(run)
	}
	var c model.NotionSyncControl
	db.First(&c, 1)
	var n int64
	db.Model(&model.NotionPageBinding{}).Count(&n)
	if c.BaselineFrozen || n != 1 {
		t.Fatalf("partial baseline committed %+v pages=%d", c, n)
	}
}
func TestBaselineFreezeAndPendingRowsAreAtomic(t *testing.T) {
	s, db, f := syncFixture(t)
	f.pages[firstPage] = fixturePage(firstPage, AllowedSources()[0], "")
	f.pages[secondPage] = fixturePage(secondPage, AllowedSources()[0], "")
	e := db.Callback().Create().Before("gorm:create").Register("fixture_baseline_failure", func(tx *gorm.DB) {
		if p, ok := tx.Statement.Dest.(*model.NotionPageBinding); ok && p.PageID == secondPage {
			tx.AddError(errors.New("fixture insertion failed"))
		}
	})
	if e != nil {
		t.Fatal(e)
	}
	run := waitFixtureRun(t, s, "dry_run")
	if run.Status != "failed" {
		t.Fatal(run)
	}
	var c model.NotionSyncControl
	db.First(&c, 1)
	var n int64
	db.Model(&model.NotionPageBinding{}).Count(&n)
	if c.BaselineFrozen || n != 0 {
		t.Fatalf("non-atomic baseline %+v pages=%d", c, n)
	}
}
func TestBootstrapUsesExistingStatusAndPreservesHistoricalFields(t *testing.T) {
	s, db, f := syncFixture(t)
	srcID := AllowedSources()[0]
	f.pages[firstPage] = fixturePage(firstPage, srcID, "")
	a := model.Article{Title: "Local title", Content: "Historical body", ExternalLink: "https://app.notion.com/p/" + firstPage, SectionCode: "s1", Author: "Local author", Tags: "Local tag", Status: 2, Pos: 17}
	if e := db.Create(&a).Error; e != nil {
		t.Fatal(e)
	}
	before := a
	if _, e := s.UpdateSource(context.Background(), srcID, SourceInput{ModuleCode: "m1"}); e != nil {
		t.Fatal(e)
	}
	preparePublishedFixtureCatalog(t, s, db, srcID)
	preview, e := s.BootstrapPreview(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	if len(preview.Items) != 1 || preview.Items[0].LocalState != "published" || preview.Items[0].Reason != "" {
		t.Fatal(preview)
	}
	yes := true
	if _, e = s.UpdateControl(context.Background(), ControlInput{Paused: &yes, SourceWritesPaused: &yes}); e != nil {
		t.Fatal(e)
	}
	c := preview.Items[0]
	result, e := s.BootstrapApply(context.Background(), BootstrapInput{Confirmations: []BootstrapConfirm{{PageID: firstPage, ArticleID: c.ArticleID, ExpectedState: c.LocalState, ExpectedFingerprint: c.ExpectedFingerprint, ConfirmedBy: "fixture operator"}}}, f)
	if e != nil {
		t.Fatal(e)
	}
	if len(result.Items) != 1 || result.Items[0].Outcome != "adopted" {
		t.Fatal(result)
	}
	db.First(&a, a.ID)
	if a.Status != before.Status || a.Content != before.Content || a.Author != before.Author || a.Pos != before.Pos || !a.CreatedAt.Equal(before.CreatedAt) || !a.UpdatedAt.Equal(before.UpdatedAt) {
		t.Fatalf("historical fields changed %+v %+v", before, a)
	}
	var p model.NotionPageBinding
	db.First(&p, "page_id = ?", firstPage)
	if p.ManagementState != "managed" || p.BootstrapState != "adopted" || p.ArticleID == nil || f.writes != 1 {
		t.Fatalf("adoption %+v writes=%d", p, f.writes)
	}
}
func TestBootstrapConflictingFreshStateDoesNotWriteOrAdopt(t *testing.T) {
	s, db, f := syncFixture(t)
	f.pages[firstPage] = fixturePage(firstPage, AllowedSources()[0], "")
	a := model.Article{Title: "Local title", ExternalLink: "https://www.notion.so/" + firstPage, SectionCode: "s1", Status: 2}
	db.Create(&a)
	preview, e := s.BootstrapPreview(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	f.pages[firstPage] = fixturePage(firstPage, AllowedSources()[0], "unpublished")
	yes := true
	s.UpdateControl(context.Background(), ControlInput{Paused: &yes, SourceWritesPaused: &yes})
	c := preview.Items[0]
	result, e := s.BootstrapApply(context.Background(), BootstrapInput{Confirmations: []BootstrapConfirm{{PageID: firstPage, ArticleID: strconv.FormatUint(a.ID, 10), ExpectedState: "published", ExpectedFingerprint: c.ExpectedFingerprint, ConfirmedBy: "fixture"}}}, f)
	if e != nil {
		t.Fatal(e)
	}
	if result.Items[0].Outcome != "failed" || f.writes != 0 {
		t.Fatal(result)
	}
	var p model.NotionPageBinding
	db.First(&p, "page_id = ?", firstPage)
	if p.ManagementState != "baseline_pending" {
		t.Fatal(p)
	}
}
func TestPagination404AndStaleStatus(t *testing.T) {
	s, db, _ := syncFixture(t)
	q := normalizedQuery(ListQuery{Page: math.MaxInt, Limit: math.MaxInt})
	if q.Page != 1000000 || q.Limit != 100 {
		t.Fatal(q)
	}
	if _, e := s.Sources(context.Background(), ListQuery{Page: math.MaxInt, Limit: 100}); e != nil {
		t.Fatal(e)
	}
	if _, e := s.Run(context.Background(), "missing"); e == nil {
		t.Fatal("expected 404")
	} else {
		var typed *Error
		if !errors.As(e, &typed) || typed.HTTPStatus != 404 {
			t.Fatal(e)
		}
	}
	old := time.Now().Add(-16 * time.Minute)
	for id, label := range allowedSources {
		db.Create(&model.NotionSyncSource{ID: id, DataSourceID: id, Label: label, LastCompleteScanAt: &old, Health: "complete"})
	}
	db.Model(&model.NotionSyncControl{}).Where("id = 1").Update("last_complete_scan_at", old)
	status, e := s.Status(context.Background())
	if e != nil || status.Health != "stale" {
		t.Fatalf("status %+v %v", status, e)
	}
}
func TestPauseRevokesLease(t *testing.T) {
	s, db, _ := syncFixture(t)
	repo := store.NewNotionSyncRepository(db)
	token, ok, e := repo.AcquireLease(context.Background(), "fixture owner", time.Minute)
	if e != nil || !ok {
		t.Fatal(e)
	}
	yes := true
	if _, e = s.UpdateControl(context.Background(), ControlInput{Paused: &yes}); e != nil {
		t.Fatal(e)
	}
	e = store.InTransaction(context.Background(), s.ds, func(ds store.IStore) error { _, e := store.LockLease(ds, token); return e })
	if !errors.Is(e, store.ErrLeaseLost) {
		t.Fatalf("old token still valid %v", e)
	}
}
func TestOpaquePropertyAndOptionIDsSurviveNames(t *testing.T) {
	f := &fixtureNotion{}
	schema, _ := f.RetrieveDataSource(context.Background(), AllowedSources()[0])
	cfg, e := discoverConfig(schema)
	if e != nil {
		t.Fatal(e)
	}
	p := schema.Properties["博客状态"]
	delete(schema.Properties, "博客状态")
	p.Name = "Renamed blog state"
	schema.Properties[p.Name] = p
	if e = validateConfig(schema, cfg); e != nil {
		t.Fatal(e)
	}
	page := fixturePage(firstPage, AllowedSources()[0], "draft")
	v := page.Properties["博客状态"]
	v.ID = "state:id"
	delete(page.Properties, "博客状态")
	page.Properties["Renamed"] = v
	snap, e := snapshotOf(page, AllowedSources()[0], cfg)
	if e != nil || snap.DesiredState != 1 {
		t.Fatalf("snapshot %+v %v", snap, e)
	}
	_ = fmt.Sprintf("%v", snap)
}
