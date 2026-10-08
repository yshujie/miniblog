package notionsync

import (
	"context"
	"errors"
	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/source"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"gorm.io/gorm"
	"strings"
	"sync"
	"testing"
	"time"
)

func managedFixture(t *testing.T) (*Service, *gorm.DB, *fixtureNotion, model.Article, model.NotionPageBinding) {
	t.Helper()
	s, db, f := syncFixture(t)
	waitFixtureRun(t, s, "dry_run")
	srcID := AllowedSources()[0]
	enableFixtureSource(t, s, srcID, "m1")
	page := fixturePage(firstPage, srcID, "published")
	f.pages[firstPage] = page
	identity, _ := source.NotionIdentity(firstPage)
	a := model.Article{Title: "Last successful title", Content: "Preserved body", Author: "Local author", ExternalLink: identity.CanonicalURL, Provider: &identity.Provider, CanonicalURL: &identity.CanonicalURL, SourceKey: &identity.SourceKey, SectionCode: "s1", Status: 2, Pos: 11}
	if e := db.Create(&a).Error; e != nil {
		t.Fatal(e)
	}
	db.Model(&model.Section{}).Where("code = ?", "s1").Update("title", "主题一")
	section := "s1"
	if e := db.Create(&model.NotionCatalogBinding{SourceID: srcID, DataSourceID: srcID, ThemePropertyID: "topic", OptionID: "opaque-option", OptionName: "主题一", SectionCode: &section, Status: "bound"}).Error; e != nil {
		t.Fatal(e)
	}
	var row model.NotionSyncSource
	db.Where("source_id = ?", srcID).First(&row)
	snap, e := snapshotOf(page, srcID, configOf(row))
	if e != nil {
		t.Fatal(e)
	}
	p := model.NotionPageBinding{PageID: firstPage, ArticleID: &a.ID, SourceID: srcID, ManagementState: "managed", Revision: 1, DesiredState: 2, PageURL: page.URL, PublicURL: page.PublicURL, SnapshotJSON: jsonText(snap), MetadataHash: metadataHash(snap), AppliedHash: metadataHash(snap), NotionLastEditedAt: &snap.LastEditedAt}
	if e := db.Create(&p).Error; e != nil {
		t.Fatal(e)
	}
	return s, db, f, a, p
}
func TestManagedDryRunNeverChangesPublicationSnapshot(t *testing.T) {
	s, db, f, _, before := managedFixture(t)
	page := f.pages[firstPage]
	page.PublicURL = nil
	page.IsArchived = true
	page.InTrash = true
	f.pages[firstPage] = page
	run := waitFixtureRun(t, s, "dry_run")
	if run.Status == "failed" {
		t.Fatal(run)
	}
	var after model.NotionPageBinding
	db.Where("page_id = ?", firstPage).First(&after)
	if after.NativeArchived || after.InTrash || after.PublicURL == nil || *after.PublicURL != *before.PublicURL || after.SnapshotJSON != before.SnapshotJSON || after.DesiredState != before.DesiredState {
		t.Fatalf("preview changed published snapshot %+v", after)
	}
}
func TestMissingPage404PreservesStatusAndVisibility(t *testing.T) {
	s, db, f, a, before := managedFixture(t)
	delete(f.pages, firstPage)
	run := waitFixtureRun(t, s, "sync")
	if run.Counts.Failed != 1 {
		t.Fatal(run)
	}
	var after model.NotionPageBinding
	db.Where("page_id = ?", firstPage).First(&after)
	db.First(&a, a.ID)
	var visible int64
	store.PublishedArticles(context.Background(), db).Count(&visible)
	if a.Status != 2 || after.PublicURL == nil || *after.PublicURL != *before.PublicURL || after.PublishBlockReason != "" || after.LastError == "" || visible != 1 {
		t.Fatalf("404 changed visible snapshot %+v article=%+v count=%d", after, a, visible)
	}
}
func TestKnownWithdrawalSurvivesMissingTitle(t *testing.T) {
	s, db, f, a, _ := managedFixture(t)
	page := fixturePage(firstPage, AllowedSources()[0], "unpublished")
	delete(page.Properties, "标题")
	page.PublicURL = nil
	f.pages[firstPage] = page
	run := waitFixtureRun(t, s, "sync")
	if run.Counts.Updated != 1 {
		t.Fatal(run)
	}
	db.First(&a, a.ID)
	if a.Status != 3 || a.Title != "Last successful title" || a.Content != "Preserved body" || a.Author != "Local author" || a.Pos != 11 {
		t.Fatalf("withdrawal lost historical values %+v", a)
	}
}
func TestUnknownStateStillObservesWithdrawnPublicURL(t *testing.T) {
	s, db, f, a, _ := managedFixture(t)
	page := fixturePage(firstPage, AllowedSources()[0], "")
	page.PublicURL = nil
	f.pages[firstPage] = page
	waitFixtureRun(t, s, "sync")
	var p model.NotionPageBinding
	db.Where("page_id = ?", firstPage).First(&p)
	db.First(&a, a.ID)
	var visible int64
	store.PublishedArticles(context.Background(), db).Count(&visible)
	if a.Status != 2 || p.PublicURL != nil || p.PublishBlockReason == "" || visible != 0 {
		t.Fatalf("known public URL withdrawal ignored %+v visible=%d", p, visible)
	}
}
func TestPostBaselineNewPageCreatesOnceAndKeepsOrder(t *testing.T) {
	s, db, f := syncFixture(t)
	srcID := AllowedSources()[0]
	f.pages[firstPage] = fixturePage(firstPage, srcID, "")
	waitFixtureRun(t, s, "dry_run")
	enableFixtureSource(t, s, srcID, "m1")
	f.pages[secondPage] = fixturePage(secondPage, srcID, "published")
	first := waitFixtureRun(t, s, "sync")
	if first.Counts.Created != 1 {
		t.Fatal(first)
	}
	var articles []model.Article
	db.Find(&articles)
	if len(articles) != 1 || articles[0].Author != "Default author" {
		t.Fatal(articles)
	}
	id := articles[0].ID
	db.Model(&articles[0]).UpdateColumn("pos", 51)
	second := waitFixtureRun(t, s, "sync")
	if second.Counts.Created != 0 {
		t.Fatal(second)
	}
	db.Find(&articles)
	if len(articles) != 1 || articles[0].ID != id || articles[0].Pos != 51 {
		t.Fatal(articles)
	}
	var pending model.NotionPageBinding
	db.Where("page_id = ?", firstPage).First(&pending)
	if pending.ManagementState != "baseline_pending" {
		t.Fatal(pending)
	}
}
func TestBootstrapExplicitNewPageThenSyncCreates(t *testing.T) {
	s, db, f := syncFixture(t)
	srcID := AllowedSources()[0]
	s.UpdateSource(context.Background(), srcID, SourceInput{ModuleCode: "m1"})
	f.pages[firstPage] = fixturePage(firstPage, srcID, "")
	preview, e := s.BootstrapPreview(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	c := preview.Items[0]
	if !c.NewPage || c.ExpectedFingerprint == "" || c.ArticleID != "" {
		t.Fatal(c)
	}
	yes := true
	s.UpdateControl(context.Background(), ControlInput{Paused: &yes, SourceWritesPaused: &yes})
	result, e := s.BootstrapApply(context.Background(), BootstrapInput{Confirmations: []BootstrapConfirm{{NewPage: true, PageID: firstPage, ExpectedState: "draft", ExpectedFingerprint: c.ExpectedFingerprint, ConfirmedBy: "fixture operator"}}}, f)
	if e != nil || result.Items[0].Outcome != "adopted" {
		t.Fatalf("result %+v error=%v", result, e)
	}
	var n int64
	db.Model(&model.Article{}).Count(&n)
	if n != 0 || f.writes != 1 {
		t.Fatalf("adoption prematurely created article=%d writes=%d", n, f.writes)
	}
	no := false
	s.UpdateControl(context.Background(), ControlInput{Paused: &no, SourceWritesPaused: &no})
	enableFixtureSource(t, s, srcID, "m1")
	draftRun := waitFixtureRun(t, s, "sync")
	db.Model(&model.Article{}).Count(&n)
	if n != 0 || draftRun.Counts.Created != 0 {
		t.Fatalf("draft created article: %d %+v", n, draftRun)
	}
	page := f.pages[firstPage]
	state := page.Properties["博客状态"]
	state.Select = &source.NotionOption{ID: "published", Name: "已发布"}
	page.Properties["博客状态"] = state
	page.LastEditedTime = page.LastEditedTime.Add(time.Second)
	f.pages[firstPage] = page
	run := waitFixtureRun(t, s, "sync")
	if run.Counts.Created != 1 {
		t.Fatal(run)
	}
	db.Model(&model.Article{}).Count(&n)
	if n != 1 {
		t.Fatal(n)
	}
}
func TestTagJSONAndEmptyTopicAreValid(t *testing.T) {
	f := &fixtureNotion{}
	schema, _ := f.RetrieveDataSource(context.Background(), AllowedSources()[0])
	cfg, e := discoverConfig(schema)
	if e != nil {
		t.Fatal(e)
	}
	page := fixturePage(firstPage, AllowedSources()[0], "published")
	topic := page.Properties["主题"]
	topic.Select = nil
	page.Properties["主题"] = topic
	tags := page.Properties["知识点"]
	tags.MultiSelect = []source.NotionOption{{ID: "one", Name: "comma,in,name"}, {ID: "long", Name: strings.Repeat("字", 1000)}}
	page.Properties["知识点"] = tags
	snap, e := snapshotOf(page, AllowedSources()[0], cfg)
	if e != nil || snap.Reason != "" || snap.TopicOptionName != "未分类" || len(snap.Tags) != 2 {
		t.Fatalf("snapshot %+v error=%v", snap, e)
	}
}

type waitingNotion struct {
	*fixtureNotion
	once    sync.Once
	entered chan struct{}
}

func (f *waitingNotion) RetrieveDataSource(ctx context.Context, id string) (source.NotionDataSource, error) {
	f.once.Do(func() { close(f.entered) })
	<-ctx.Done()
	return source.NotionDataSource{}, ctx.Err()
}
func TestStopCancelsWorkerAndRejectsLaterTrigger(t *testing.T) {
	s, _, f := syncFixture(t)
	blocking := &waitingNotion{fixtureNotion: f, entered: make(chan struct{})}
	s.client = blocking
	if _, e := s.Trigger(context.Background(), TriggerInput{Mode: "dry_run"}); e != nil {
		t.Fatal(e)
	}
	select {
	case <-blocking.entered:
	case <-time.After(time.Second):
		t.Fatal("worker did not start")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if e := s.Stop(ctx); e != nil {
		t.Fatal(e)
	}
	if _, e := s.Trigger(context.Background(), TriggerInput{Mode: "dry_run"}); e == nil {
		t.Fatal("trigger accepted after stop")
	}
	s.mu.Lock()
	n := len(s.active)
	s.mu.Unlock()
	if n != 0 {
		t.Fatalf("workers remain=%d", n)
	}
}
func TestBusyTriggerReturnsCurrentRun(t *testing.T) {
	s, db, _ := syncFixture(t)
	repo := store.NewNotionSyncRepository(db)
	token, ok, e := repo.AcquireLease(context.Background(), "other fixture", time.Minute)
	if e != nil || !ok {
		t.Fatal(e)
	}
	defer repo.ReleaseLease(context.Background(), token)
	db.Model(&model.NotionSyncControl{}).Where("id = 1").Update("current_run_id", "existing-fixture-run")
	r, e := s.Trigger(context.Background(), TriggerInput{Mode: "dry_run"})
	if e != nil || r.RunID != "existing-fixture-run" {
		t.Fatalf("trigger %+v %v", r, e)
	}
}
func TestRetentionPreservesUnresolvedAndBootstrap(t *testing.T) {
	s, db, _ := syncFixture(t)
	old := time.Now().Add(-91 * 24 * time.Hour)
	for _, r := range []model.NotionSyncRun{{ID: "resolved", Mode: "sync", Status: "completed", StartedAt: old}, {ID: "unresolved", Mode: "sync", Status: "failed", StartedAt: old}, {ID: "journal", Mode: "bootstrap_apply", Status: "completed", StartedAt: old}} {
		db.Create(&r)
	}
	repo := store.NewNotionSyncRepository(db)
	token, _, e := repo.AcquireLease(context.Background(), "retention fixture", time.Minute)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.pruneResolvedRuns(context.Background(), token); e != nil {
		t.Fatal(e)
	}
	repo.ReleaseLease(context.Background(), token)
	var n int64
	db.Model(&model.NotionSyncRun{}).Count(&n)
	if n != 2 {
		t.Fatal(n)
	}
	if e = db.Where("run_id = ?", "resolved").First(&model.NotionSyncRun{}).Error; !errors.Is(e, gorm.ErrRecordNotFound) {
		t.Fatal(e)
	}
}

type schemaOverride struct {
	*fixtureNotion
	sourceID string
	problem  string
	hook     func(string)
}

func (f *schemaOverride) RetrieveDataSource(ctx context.Context, id string) (source.NotionDataSource, error) {
	if f.hook != nil {
		f.hook(id)
	}
	schema, e := f.fixtureNotion.RetrieveDataSource(ctx, id)
	if id != f.sourceID {
		return schema, e
	}
	switch f.problem {
	case "403":
		return source.NotionDataSource{}, &source.NotionAPIError{StatusCode: 403, Code: "restricted_resource"}
	case "missing":
		delete(schema.Properties, "标题")
	case "trash":
		schema.InTrash = true
	}
	return schema, e
}
func TestSchemaDriftPreviewIsReadOnlyButSyncBlocksPublication(t *testing.T) {
	for _, problem := range []string{"missing", "trash", "403"} {
		t.Run(problem, func(t *testing.T) {
			s, db, f, a, before := managedFixture(t)
			s.client = &schemaOverride{fixtureNotion: f, sourceID: AllowedSources()[0], problem: problem}
			preview := waitFixtureRun(t, s, "dry_run")
			if preview.Status != "failed" {
				t.Fatal(preview)
			}
			var p model.NotionPageBinding
			db.Where("page_id = ?", firstPage).First(&p)
			var visible int64
			store.PublishedArticles(context.Background(), db).Count(&visible)
			if p.Revision != before.Revision || p.NeedsRevalidation || p.PublishBlockReason != "" || visible != 1 {
				t.Fatalf("preview changed publication %+v visible=%d", p, visible)
			}
			waitFixtureRun(t, s, "sync")
			db.Where("page_id = ?", firstPage).First(&p)
			db.First(&a, a.ID)
			store.PublishedArticles(context.Background(), db).Count(&visible)
			if a.Status != 2 || a.Title != "Last successful title" {
				t.Fatalf("schema failure changed article %+v", a)
			}
			if problem == "403" {
				if visible != 1 || p.NeedsRevalidation || p.PublishBlockReason != "" {
					t.Fatalf("403 changed visibility %+v", p)
				}
			} else if visible != 0 || !p.NeedsRevalidation || p.PublishBlockReason == "" {
				t.Fatalf("confirmed drift stayed visible %+v", p)
			}
		})
	}
}
func TestPartialScanStillAppliesKnownPage(t *testing.T) {
	s, db, f, a, _ := managedFixture(t)
	f.queryError = AllowedSources()[1]
	page := f.pages[firstPage]
	title := page.Properties["标题"]
	title.Title[0].PlainText = "Freshly verified"
	page.Properties["标题"] = title
	page.LastEditedTime = page.LastEditedTime.Add(time.Second)
	f.pages[firstPage] = page
	run := waitFixtureRun(t, s, "sync")
	db.First(&a, a.ID)
	if run.Status != "failed" || run.Counts.Updated != 1 || a.Title != "Freshly verified" {
		t.Fatalf("known page skipped %+v %+v", run, a)
	}
}
func TestAutoMappingCASRejectsConcurrentConfiguration(t *testing.T) {
	s, db, f := syncFixture(t)
	id := AllowedSources()[0]
	f.pages[firstPage] = fixturePage(firstPage, id, "")
	fired := false
	s.client = &schemaOverride{fixtureNotion: f, hook: func(current string) {
		if current == id && !fired {
			fired = true
			if _, e := s.UpdateSource(context.Background(), id, SourceInput{Label: "Concurrent revision"}); e != nil {
				t.Error(e)
			}
		}
	}}
	run := waitFixtureRun(t, s, "dry_run")
	var row model.NotionSyncSource
	db.Where("source_id = ?", id).First(&row)
	var c model.NotionSyncControl
	db.First(&c, 1)
	if run.Status != "failed" || row.ConfigRevision != 1 || row.PropertyMappingJSON != "" || c.BaselineFrozen {
		t.Fatalf("CAS used stale mapping %+v %+v %+v", run, row, c)
	}
}
func TestBaselineRevisionFenceRejectsConcurrentConfig(t *testing.T) {
	s, db, f := syncFixture(t)
	ids := AllowedSources()
	f.pages[firstPage] = fixturePage(firstPage, ids[0], "")
	fired := false
	s.client = &schemaOverride{fixtureNotion: f, hook: func(current string) {
		if current == ids[len(ids)-1] && !fired {
			fired = true
			if _, e := s.UpdateSource(context.Background(), ids[0], SourceInput{Label: "Changed mid scan"}); e != nil {
				t.Error(e)
			}
		}
	}}
	run := waitFixtureRun(t, s, "dry_run")
	var c model.NotionSyncControl
	db.First(&c, 1)
	var n int64
	db.Model(&model.NotionPageBinding{}).Count(&n)
	if run.Status != "failed" || c.BaselineFrozen || n != 0 {
		t.Fatalf("baseline mixed revisions %+v count=%d", c, n)
	}
}
func TestBootstrapNewConfirmationCannotWriteAnAlreadyRegisteredSource(t *testing.T) {
	s, db, f := syncFixture(t)
	id := AllowedSources()[0]
	s.UpdateSource(context.Background(), id, SourceInput{ModuleCode: "m1"})
	f.pages[firstPage] = fixturePage(firstPage, id, "")
	preview, e := s.BootstrapPreview(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	identity, _ := source.NotionIdentity(firstPage)
	a := model.Article{Title: "Concurrent registration", SectionCode: "s1", Status: 2, Provider: &identity.Provider, CanonicalURL: &identity.CanonicalURL, SourceKey: &identity.SourceKey, ExternalLink: identity.CanonicalURL}
	if e := db.Create(&a).Error; e != nil {
		t.Fatal(e)
	}
	yes := true
	s.UpdateControl(context.Background(), ControlInput{Paused: &yes, SourceWritesPaused: &yes})
	result, e := s.BootstrapApply(context.Background(), BootstrapInput{Confirmations: []BootstrapConfirm{{PageID: firstPage, NewPage: true, ExpectedState: "draft", ExpectedFingerprint: preview.Items[0].ExpectedFingerprint, ConfirmedBy: "fixture"}}}, f)
	if e != nil || result.Items[0].Outcome != "failed" || f.writes != 0 {
		t.Fatalf("unsafe prewrite %+v error=%v writes=%d", result, e, f.writes)
	}
	var p model.NotionPageBinding
	db.Where("page_id = ?", firstPage).First(&p)
	if p.ManagementState != "baseline_pending" || p.LastError == "" {
		t.Fatal(p)
	}
}
func TestConfigurationRejectedDuringBootstrapWrite(t *testing.T) {
	s, db, _ := syncFixture(t)
	id := AllowedSources()[0]
	repo := store.NewNotionSyncRepository(db)
	token, ok, e := repo.AcquireLease(context.Background(), "fixture", time.Minute)
	if e != nil || !ok {
		t.Fatal(e)
	}
	defer repo.ReleaseLease(context.Background(), token)
	r := model.NotionSyncRun{ID: "bootstrap-fixture", Mode: "bootstrap_apply", Status: "running"}
	db.Create(&r)
	db.Model(&model.NotionSyncControl{}).Where("id = 1").Update("current_run_id", r.ID)
	if _, e := s.UpdateSource(context.Background(), id, SourceInput{ModuleCode: "m1"}); fixtureStatus(e) != 409 {
		t.Fatalf("configuration changed during bootstrap %v", e)
	}
	db.Model(&model.NotionSyncControl{}).Where("id = 1").Updates(map[string]interface{}{"paused": true, "source_writes_paused": true})
	no := false
	if _, e := s.UpdateControl(context.Background(), ControlInput{Paused: &no, SourceWritesPaused: &no}); fixtureStatus(e) != 409 {
		t.Fatalf("maintenance ended during remote write %v", e)
	}
}
func TestPersistentCooldownPreventsRequests(t *testing.T) {
	s, db, f := syncFixture(t)
	until := time.Now().Add(time.Minute)
	db.Model(&model.NotionSyncControl{}).Where("id = 1").Update("cooldown_until", until)
	if _, e := s.Trigger(context.Background(), TriggerInput{Mode: "dry_run"}); fixtureStatus(e) != 429 || f.queries != 0 {
		t.Fatalf("cooldown not enforced %v queries=%d", e, f.queries)
	}
}

func fixtureStatus(e error) int {
	var typed *Error
	if errors.As(e, &typed) {
		return typed.StatusCode()
	}
	return 0
}
func TestInvalidSchemaNeverProjectsCatalog(t *testing.T) {
	s, db, f, _, _ := managedFixture(t)
	id := AllowedSources()[0]
	s.client = &schemaOverride{fixtureNotion: f, sourceID: id, problem: "missing"}
	var before int64
	db.Model(&model.Section{}).Count(&before)
	waitFixtureRun(t, s, "sync")
	var after int64
	db.Model(&model.Section{}).Count(&after)
	var section model.Section
	db.Where("code = ?", "s1").First(&section)
	if after != before || section.Title != "主题一" {
		t.Fatalf("invalid source projected catalog before=%d after=%d %+v", before, after, section)
	}
}
func TestLeaseExpiryFinishesAuditWithoutBusinessWrites(t *testing.T) {
	s, db, _ := syncFixture(t)
	repo := store.NewNotionSyncRepository(db)
	token, ok, e := repo.AcquireLease(context.Background(), "expired fixture", time.Minute)
	if e != nil || !ok {
		t.Fatal(e)
	}
	run := model.NotionSyncRun{ID: "expired-run", Mode: "dry_run", Status: "running", LeaseEpoch: token.Epoch, StartedAt: time.Now()}
	db.Create(&run)
	expired := time.Now().Add(-time.Minute)
	db.Model(&model.NotionSyncControl{}).Where("id = 1").Updates(map[string]interface{}{"current_run_id": run.ID, "lease_until": expired})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s.execute(ctx, run, token)
	db.Where("run_id = ?", run.ID).First(&run)
	var c model.NotionSyncControl
	db.First(&c, 1)
	if run.Status != "abandoned" || run.FinishedAt == nil || c.CurrentRunID != "" {
		t.Fatalf("expired worker left running %+v %+v", run, c)
	}
	fresh := waitFixtureRun(t, s, "dry_run")
	if fresh.Status == "running" {
		t.Fatal(fresh)
	}
}

func TestConfigShapeRejectsExtraStatesAndOversizedOpaqueIDs(t *testing.T) {
	f := fixtureNotion{}
	schema, _ := f.RetrieveDataSource(context.Background(), AllowedSources()[0])
	base, e := discoverConfig(schema)
	if e != nil {
		t.Fatal(e)
	}
	for _, kind := range []string{"extra_state", "state_id", "description_id", "difficulty_id", "date_id", "optional_duplicate"} {
		t.Run(kind, func(t *testing.T) {
			cfg := base
			cfg.StateOptionIDs = map[string]string{}
			for k, v := range base.StateOptionIDs {
				cfg.StateOptionIDs[k] = v
			}
			switch kind {
			case "extra_state":
				cfg.StateOptionIDs["foo"] = "published"
			case "state_id":
				cfg.StateOptionIDs["draft"] = strings.Repeat("x", 129)
			case "description_id":
				cfg.DescriptionPropertyID = strings.Repeat("x", 129)
			case "difficulty_id":
				cfg.DifficultyPropertyID = strings.Repeat("x", 129)
			case "date_id":
				cfg.DatePropertyID = strings.Repeat("x", 129)
			case "optional_duplicate":
				cfg.DatePropertyID = cfg.TitlePropertyID
			}
			if e := validateConfigShape(cfg); e == nil {
				t.Fatal("invalid mapping accepted")
			}
		})
	}
}

func TestBootstrapKnownPublicSlugIsReviewedBeforeWrite(t *testing.T) {
	s, db, f := syncFixture(t)
	id := AllowedSources()[0]
	s.UpdateSource(context.Background(), id, SourceInput{ModuleCode: "m1"})
	page := fixturePage(firstPage, id, "")
	public := "https://fixture.notion.site/CaseSensitiveSlug"
	page.PublicURL = &public
	f.pages[firstPage] = page
	identity, _ := source.Parse(public)
	a := model.Article{Title: "Slug article", SectionCode: "s1", Status: 3, Provider: &identity.Provider, CanonicalURL: &identity.CanonicalURL, SourceKey: &identity.SourceKey, ExternalLink: public}
	db.Create(&a)
	preview, e := s.BootstrapPreview(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	c := preview.Items[0]
	if c.NewPage || c.MatchMethod != "known_reading_url" || c.ArticleID == "" || !c.RequiresLegacyAlias || c.LocalState != "unpublished" {
		t.Fatalf("slug omitted from review %+v", c)
	}
	yes := true
	s.UpdateControl(context.Background(), ControlInput{Paused: &yes, SourceWritesPaused: &yes})
	failed, e := s.BootstrapApply(context.Background(), BootstrapInput{Confirmations: []BootstrapConfirm{{PageID: firstPage, NewPage: true, ExpectedState: "published", ExpectedFingerprint: c.ExpectedFingerprint, ConfirmedBy: "fixture"}}}, f)
	if e != nil || failed.Items[0].Outcome != "failed" || f.writes != 0 {
		t.Fatalf("duplicate slug written %+v error=%v writes=%d", failed, e, f.writes)
	}
	result, e := s.BootstrapApply(context.Background(), BootstrapInput{Confirmations: []BootstrapConfirm{{PageID: firstPage, ArticleID: c.ArticleID, AllowLegacyAlias: true, ExpectedState: c.LocalState, ExpectedFingerprint: c.ExpectedFingerprint, ConfirmedBy: "fixture"}}}, f)
	if e != nil || result.Items[0].Outcome != "adopted" || f.writes != 1 {
		t.Fatalf("reviewed slug adoption %+v error=%v writes=%d", result, e, f.writes)
	}
}
func TestBootstrapPreviewShowsCatalogConflictAndPublicCondition(t *testing.T) {
	s, db, f := syncFixture(t)
	id := AllowedSources()[0]
	s.UpdateSource(context.Background(), id, SourceInput{ModuleCode: "m1"})
	f.pages[firstPage] = fixturePage(firstPage, id, "")
	db.Model(&model.Section{}).Where("code = ?", "s1").Update("title", "主题一")
	preview, e := s.BootstrapPreview(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	c := preview.Items[0]
	if c.PublishBlockReason != "catalog_same_name_conflict" || c.ProposedSectionTitle != "主题一" || c.PublicCondition != "public_url_available" {
		t.Fatalf("incomplete review %+v", c)
	}
}

func TestInterruptedBootstrapRequiresNewReviewAfterConfigurationChanges(t *testing.T) {
	s, _, f := syncFixture(t)
	id := AllowedSources()[0]
	s.UpdateSource(context.Background(), id, SourceInput{ModuleCode: "m1"})
	f.pages[firstPage] = fixturePage(firstPage, id, "")
	preview, e := s.BootstrapPreview(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	confirm := BootstrapConfirm{PageID: firstPage, NewPage: true, ExpectedState: "draft", ExpectedFingerprint: preview.Items[0].ExpectedFingerprint, ConfirmedBy: "fixture"}
	yes := true
	s.UpdateControl(context.Background(), ControlInput{Paused: &yes, SourceWritesPaused: &yes})
	f.writeErr = &source.NotionAPIError{StatusCode: 503, Code: "service_unavailable"}
	first, e := s.BootstrapApply(context.Background(), BootstrapInput{Confirmations: []BootstrapConfirm{confirm}}, f)
	if e != nil || first.Items[0].Outcome != "failed" || f.writes != 1 {
		t.Fatalf("expected interrupted journal %+v %v writes=%d", first, e, f.writes)
	}
	f.writeErr = nil
	if _, e := s.UpdateSource(context.Background(), id, SourceInput{Label: "Reconfirmed mapping revision"}); e != nil {
		t.Fatal(e)
	}
	stale, e := s.BootstrapApply(context.Background(), BootstrapInput{Confirmations: []BootstrapConfirm{confirm}}, f)
	if e != nil || stale.Items[0].Outcome != "failed" || f.writes != 1 {
		t.Fatalf("stale confirmation proceeded %+v %v writes=%d", stale, e, f.writes)
	}
	reviewed, e := s.BootstrapPreview(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	confirm.ExpectedFingerprint = reviewed.Items[0].ExpectedFingerprint
	final, e := s.BootstrapApply(context.Background(), BootstrapInput{Confirmations: []BootstrapConfirm{confirm}}, f)
	if e != nil || final.Items[0].Outcome != "adopted" || f.writes != 2 {
		t.Fatalf("explicit new review cannot recover %+v %v writes=%d", final, e, f.writes)
	}
}
