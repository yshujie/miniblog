package contentintegration

import (
	"context"
	"errors"
	articlebiz "github.com/yshujie/miniblog/internal/miniblog/biz/article"
	"github.com/yshujie/miniblog/internal/miniblog/biz/catalog"
	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"github.com/yshujie/miniblog/internal/pkg/errno"
	"gorm.io/gorm"
	"reflect"
	"testing"
	"time"
)

const rolloutSourceA = "11111111-1111-1111-1111-111111111111"
const rolloutSourceB = "22222222-2222-2222-2222-222222222222"
const rolloutPageID = "00112233445566778899aabbccddeeff"

func rolloutMySQLFixture(t *testing.T) (*gorm.DB, store.IStore, articlebiz.SyncInput, uint64) {
	t.Helper()
	db, _ := scratchDB(t)
	migration(t, db, "000005_source_uniqueness.up.sql", "")
	ds := store.NewStore(db)
	if e := db.Model(&model.NotionSyncControl{}).Where("id=1").Updates(map[string]interface{}{"paused": false, "baseline_frozen": true}).Error; e != nil {
		t.Fatal(e)
	}
	for _, src := range []model.NotionSyncSource{{ID: rolloutSourceA, DataSourceID: rolloutSourceA, ModuleCode: "m1", Enabled: true, ConfigRevision: 1}, {ID: rolloutSourceB, DataSourceID: rolloutSourceB, ModuleCode: "m2", Enabled: false, ConfigRevision: 1}} {
		if e := db.Create(&src).Error; e != nil {
			t.Fatal(e)
		}
	}
	section := "s1"
	if e := db.Create(&model.NotionCatalogBinding{SourceID: rolloutSourceA, DataSourceID: rolloutSourceA, ThemePropertyID: "topic", OptionID: "one", OptionName: "One", SectionCode: &section, Status: model.NotionCatalogBound}).Error; e != nil {
		t.Fatal(e)
	}
	token, ok, e := ds.NotionSync().AcquireLease(context.Background(), "rollout-worker", time.Minute)
	if e != nil || !ok {
		t.Fatal(token, ok, e)
	}
	url := "https://fixture.notion.site/current-public-slug"
	edited := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	input := articlebiz.SyncInput{Lease: token, RunID: "initial", SourceID: rolloutSourceA, DataSourceID: rolloutSourceA, PageID: rolloutPageID, ThemePropertyID: "topic", ThemeOptionID: "one", ThemeOptionName: "One", Title: "Last successful title", PageURL: "https://notion.so/" + rolloutPageID, PublicURL: &url, DesiredState: 2, MetadataComplete: true, MetadataHash: "last-good", SnapshotJSON: "last-good-snapshot", NotionLastEditedAt: &edited, ExpectedConfigRevision: 1}
	out, e := articlebiz.NewForSync(ds, "Local author").ApplySyncedSource(context.Background(), input)
	if e != nil {
		t.Fatal(e)
	}
	if e = db.Model(&model.Article{}).Where("id = ?", out.ArticleID).Updates(map[string]interface{}{"content": "Preserved body", "author": "Local author", "subsection_code": "sub1", "pos": 71}).Error; e != nil {
		t.Fatal(e)
	}
	return db, ds, input, out.ArticleID
}
func rolloutLoadArticle(t *testing.T, db *gorm.DB, id uint64) model.Article {
	t.Helper()
	var a model.Article
	if e := db.First(&a, id).Error; e != nil {
		t.Fatal(e)
	}
	return a
}
func rolloutLoadPage(t *testing.T, db *gorm.DB) model.NotionPageBinding {
	t.Helper()
	var p model.NotionPageBinding
	if e := db.First(&p, "page_id = ?", rolloutPageID).Error; e != nil {
		t.Fatal(e)
	}
	return p
}
func TestMySQLRolloutPrepareTopicsMaintenanceRevisionAndIdempotency(t *testing.T) {
	db, ds, r, id := rolloutMySQLFixture(t)
	b := articlebiz.NewForSync(ds, "")
	before := rolloutLoadArticle(t, db, id)
	beforePage := rolloutLoadPage(t, db)
	options := []articlebiz.TopicOption{{ID: "empty-topic", Name: "Empty topic"}}
	if _, _, e := b.PrepareSyncedTopics(context.Background(), r.Lease, rolloutSourceB, 1, "topic", options); e == nil {
		t.Fatal("prepare accepted missing dual pause")
	}
	db.Model(&model.NotionSyncControl{}).Where("id=1").Update("paused", true)
	if _, _, e := b.PrepareSyncedTopics(context.Background(), r.Lease, rolloutSourceB, 1, "topic", options); e == nil {
		t.Fatal("prepare accepted source writes active")
	}
	db.Model(&model.NotionSyncControl{}).Where("id=1").Updates(map[string]interface{}{"source_writes_paused": true, "baseline_frozen": false})
	if _, _, e := b.PrepareSyncedTopics(context.Background(), r.Lease, rolloutSourceB, 1, "topic", options); e == nil {
		t.Fatal("prepare accepted an unfrozen baseline")
	}
	db.Model(&model.NotionSyncControl{}).Where("id=1").Update("baseline_frozen", true)
	results, revision, e := b.PrepareSyncedTopics(context.Background(), r.Lease, rolloutSourceB, 1, "topic", options)
	if e != nil || revision != 2 || len(results) != 2 || results[0].BindingID == 0 || results[0].Outcome != "created" {
		t.Fatal(results, revision, e)
	}
	var section model.Section
	db.First(&section, "code = ?", results[0].SectionCode)
	db.Model(&section).Update("sort", 97)
	again, next, e := b.PrepareSyncedTopics(context.Background(), r.Lease, rolloutSourceB, revision, "topic", options)
	if e != nil || next != revision || again[0].BindingID != results[0].BindingID || again[0].Outcome != "bound" {
		t.Fatal(again, next, e)
	}
	db.First(&section, section.ID)
	if section.Sort != 97 {
		t.Fatal("directory prepare changed local order")
	}
	if _, _, e = b.PrepareSyncedTopics(context.Background(), r.Lease, rolloutSourceB, 1, "topic", options); e == nil {
		t.Fatal("stale config succeeded")
	}
	if !reflect.DeepEqual(before, rolloutLoadArticle(t, db, id)) || !reflect.DeepEqual(beforePage, rolloutLoadPage(t, db)) {
		t.Fatal("prepare changed an article or page")
	}
}
func TestMySQLRolloutCatalogOccupancyUsesCanonicalCollation(t *testing.T) {
	db, ds, _, _ := rolloutMySQLFixture(t)
	var binding model.NotionCatalogBinding
	db.First(&binding, "source_id = ?", rolloutSourceA)
	if e := db.Model(&binding).Update("section_code", "S1").Error; e != nil {
		t.Fatal(e)
	}
	e := store.InTransaction(context.Background(), ds, func(tx store.IStore) error {
		if e := catalog.LockModules(tx, "m1"); e != nil {
			return e
		}
		if e := catalog.CheckSyncedThemeSectionAvailable(tx, "s1", binding.ID); e != nil {
			t.Fatalf("self alias rejected: %v", e)
		}
		return catalog.CheckSyncedThemeSectionAvailable(tx, "s1", binding.ID+1)
	})
	var coded *errno.Errno
	if !errors.As(e, &coded) || coded.HTTP != 409 || coded.Code != "CatalogBindingConflict" {
		t.Fatal("case variant did not return the intended occupancy conflict", e)
	}
	var section model.Section
	db.First(&section, "code = ?", "s1")
	if section.Title != "One" {
		t.Fatal("occupancy check changed title")
	}
}
func TestMySQLRolloutIsolationPreservesContentAndWithdrawsAtomically(t *testing.T) {
	db, ds, r, id := rolloutMySQLFixture(t)
	b := articlebiz.NewForSync(ds, "")
	before := rolloutLoadArticle(t, db, id)
	beforePage := rolloutLoadPage(t, db)
	edited := r.NotionLastEditedAt.Add(time.Hour)
	input := articlebiz.TargetIsolationInput{Lease: r.Lease, RunID: "isolation", PageID: rolloutPageID, TargetSourceID: rolloutSourceB, DataSourceID: rolloutSourceB, Reason: articlebiz.TargetDisabled, DesiredState: 2, NotionLastEditedAt: &edited, ExpectedConfigRevision: 1, ExpectedBindingRevision: beforePage.Revision}
	result, e := b.IsolateSyncedTarget(context.Background(), input)
	if e != nil || result.Outcome != "blocked" {
		t.Fatal(result, e)
	}
	after := rolloutLoadArticle(t, db, id)
	page := rolloutLoadPage(t, db)
	var visible int64
	store.PublishedArticles(context.Background(), db).Count(&visible)
	if !reflect.DeepEqual(before, after) || page.SourceID != beforePage.SourceID || page.PublicURL == nil || *page.PublicURL != *beforePage.PublicURL || page.PageURL != beforePage.PageURL || page.MetadataHash != beforePage.MetadataHash || page.SnapshotJSON != beforePage.SnapshotJSON || !page.NeedsRevalidation || page.PublishBlockReason != "target_disabled" || page.LastError != "" || visible != 0 {
		t.Fatalf("bad isolation %+v %+v", after, page)
	}
	input.DesiredState = 1
	input.PublicURLWithdrawn = true
	input.ExpectedBindingRevision = page.Revision
	input.RunID = "withdrawal"
	edited = edited.Add(time.Hour)
	input.NotionLastEditedAt = &edited
	_, e = b.IsolateSyncedTarget(context.Background(), input)
	if e != nil {
		t.Fatal(e)
	}
	after = rolloutLoadArticle(t, db, id)
	page = rolloutLoadPage(t, db)
	expected := before
	expected.Status = 1
	expected.UpdatedAt = after.UpdatedAt
	if !reflect.DeepEqual(expected, after) || page.PublicURL != nil || page.SourceID != beforePage.SourceID || page.PageURL != beforePage.PageURL {
		t.Fatalf("withdrawal overwrote last good metadata %+v %+v", after, page)
	}
	before = after
	beforePage = page
	if e = db.Exec("CREATE TRIGGER rollout_page_failure BEFORE UPDATE ON notion_page_bindings FOR EACH ROW SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT='forced rollback'").Error; e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { db.Exec("DROP TRIGGER IF EXISTS rollout_page_failure") })
	input.DesiredState = 4
	input.ExpectedBindingRevision = page.Revision
	input.RunID = "rollback"
	if _, e = b.IsolateSyncedTarget(context.Background(), input); e == nil {
		t.Fatal("trigger failure ignored")
	}
	if !reflect.DeepEqual(before, rolloutLoadArticle(t, db, id)) || !reflect.DeepEqual(beforePage, rolloutLoadPage(t, db)) {
		t.Fatal("failed binding persistence committed article state")
	}
	input.Lease.Epoch++
	if _, e = b.IsolateSyncedTarget(context.Background(), input); !errors.Is(e, store.ErrLeaseLost) {
		t.Fatal(e)
	}
}
