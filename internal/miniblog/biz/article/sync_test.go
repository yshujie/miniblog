package article

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/yshujie/miniblog/internal/miniblog/biz/catalog"
	"github.com/yshujie/miniblog/internal/miniblog/biz/reading"
	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"gorm.io/gorm"
	"strings"
	"testing"
	"time"
)

const syncPageID = "aabbccddeeff00112233445566778899"
const syncDataSourceA = "11111111-1111-1111-1111-111111111111"
const syncDataSourceB = "22222222-2222-2222-2222-222222222222"

func syncFixture(t *testing.T) (*gorm.DB, *articleBiz, SyncInput) {
	t.Helper()
	db := contentDB(t)
	if e := db.AutoMigrate(&model.NotionSyncControl{}, &model.NotionSyncSource{}, &model.NotionCatalogBinding{}, &model.NotionPageBinding{}, &model.NotionSyncRun{}, &model.NotionSyncRunItem{}); e != nil {
		t.Fatal(e)
	}
	if e := db.Create(&model.NotionSyncControl{ID: 1, Paused: false}).Error; e != nil {
		t.Fatal(e)
	}
	for _, src := range []model.NotionSyncSource{{ID: "a", DataSourceID: syncDataSourceA, ModuleCode: "m1", Enabled: true, ConfigRevision: 1}, {ID: "b", DataSourceID: syncDataSourceB, ModuleCode: "m2", Enabled: true, ConfigRevision: 1}} {
		if e := db.Create(&src).Error; e != nil {
			t.Fatal(e)
		}
	}
	for _, binding := range []model.NotionCatalogBinding{{SourceID: "a", DataSourceID: syncDataSourceA, ThemePropertyID: "topic", OptionID: "o1", OptionName: "S1", SectionCode: syncString("s1"), Status: model.NotionCatalogBound}, {SourceID: "b", DataSourceID: syncDataSourceB, ThemePropertyID: "topic", OptionID: "o2", OptionName: "S2", SectionCode: syncString("s2"), Status: model.NotionCatalogBound}} {
		if e := db.Create(&binding).Error; e != nil {
			t.Fatal(e)
		}
	}
	token, ok, e := store.NewNotionSyncRepository(db).AcquireLease(context.Background(), "worker", time.Hour)
	if e != nil || !ok {
		t.Fatal(token, ok, e)
	}
	b := NewForSync(store.NewStore(db), "Default author")
	return db, b, SyncInput{Lease: token, RunID: "run", SourceID: "a", DataSourceID: syncDataSourceA, PageID: syncPageID, ThemePropertyID: "topic", ThemeOptionID: "o1", ThemeOptionName: "S1", Title: "Source title", PageURL: "https://www.notion.so/" + syncPageID, PublicURL: syncString("https://public.notion.site/" + syncPageID), DesiredState: 2, MetadataComplete: true, ExpectedConfigRevision: 1}
}
func syncString(s string) *string { return &s }
func syncBinding(t *testing.T, db *gorm.DB) model.NotionPageBinding {
	t.Helper()
	var p model.NotionPageBinding
	if e := db.First(&p, "page_id = ?", syncPageID).Error; e != nil {
		t.Fatal(e)
	}
	return p
}
func syncApply(t *testing.T, db *gorm.DB, b *articleBiz, r SyncInput) *SyncResult {
	t.Helper()
	var p model.NotionPageBinding
	if db.First(&p, "page_id = ?", r.PageID).Error == nil {
		r.ExpectedBindingRevision = p.Revision
	}
	out, e := b.ApplySyncedSource(context.Background(), r)
	if e != nil {
		t.Fatal(e)
	}
	return out
}
func syncArticle(t *testing.T, db *gorm.DB, id uint64) model.Article {
	t.Helper()
	var a model.Article
	if e := db.First(&a, id).Error; e != nil {
		t.Fatal(e)
	}
	return a
}
func requirePublic(t *testing.T, db *gorm.DB, id uint64, want bool) {
	t.Helper()
	_, e := reading.New(store.NewStore(db)).Article(context.Background(), id)
	if want && e != nil {
		t.Fatal(e)
	}
	if !want {
		errorHTTP(t, e, 404)
	}
}
func TestSyncAllFourStateTransitions(t *testing.T) {
	for from := 1; from <= 4; from++ {
		for to := 1; to <= 4; to++ {
			t.Run(fmt.Sprintf("%d_to_%d", from, to), func(t *testing.T) {
				db, b, r := syncFixture(t)
				first := syncApply(t, db, b, r)
				db.Model(&model.Article{}).Where("id = ?", first.ArticleID).Updates(map[string]interface{}{"status": from, "content": "legacy body", "author": "local author", "pos": 42})
				r.DesiredState = to
				r.Title = "Updated title"
				r.Tags = []string{"A,B", strings.Repeat("中", 300)}
				out := syncApply(t, db, b, r)
				a := syncArticle(t, db, out.ArticleID)
				if a.Status != to || a.ID != first.ArticleID || a.Content != "legacy body" || a.Author != "local author" || a.Pos != 42 {
					t.Fatalf("lost state/legacy fields: %+v", a)
				}
				if to == 2 {
					requirePublic(t, db, a.ID, true)
				} else {
					requirePublic(t, db, a.ID, false)
				}
				if from == 4 && to == 4 && a.Title == "Updated title" {
					t.Fatal("archive metadata edited without restore")
				}
			})
		}
	}
}
func TestSyncNewPagesStayPendingUntilEligiblePublication(t *testing.T) {
	db, b, r := syncFixture(t)
	for _, state := range []int{1, 3, 4} {
		r.DesiredState = state
		out := syncApply(t, db, b, r)
		if out.ArticleID != 0 {
			t.Fatal("non-published page created blog record", out)
		}
	}
	r.DesiredState = 2
	r.PublicURL = nil
	out := syncApply(t, db, b, r)
	if out.ArticleID != 0 {
		t.Fatal(out)
	}
	r.PublicURL = syncString("https://public.notion.site/" + syncPageID)
	r.Title = ""
	out = syncApply(t, db, b, r)
	if out.ArticleID != 0 {
		t.Fatal("invalid metadata created record", out)
	}
	r.Title = "Valid"
	out = syncApply(t, db, b, r)
	if out.ArticleID == 0 || out.Outcome != "created" {
		t.Fatal(out)
	}
	var count int64
	db.Model(&model.Article{}).Count(&count)
	if count != 1 {
		t.Fatal(count)
	}
	a := syncArticle(t, db, out.ArticleID)
	if a.Author != "Default author" || a.Content != "" {
		t.Fatal(a)
	}
}
func TestSyncPreservesHistorySubsectionAndMovesAcrossModules(t *testing.T) {
	db, b, r := syncFixture(t)
	first := syncApply(t, db, b, r)
	a := syncArticle(t, db, first.ArticleID)
	originalLink := a.ExternalLink
	db.Model(&model.Article{}).Where("id = ?", a.ID).Updates(map[string]interface{}{"content": "legacy markdown", "author": "Local", "subsection_code": "sub1", "pos": 19})
	r.Title = "New title"
	r.Tags = []string{"tag,with,commas", strings.Repeat("标", 300)}
	r.PublicURL = syncString("https://new.notion.site/" + syncPageID)
	out := syncApply(t, db, b, r)
	a = syncArticle(t, db, out.ArticleID)
	if a.SubsectionCode != "sub1" || a.Pos != 19 || a.ExternalLink != originalLink || a.Content != "legacy markdown" || a.Author != "Local" {
		t.Fatal(a)
	}
	detail, e := reading.New(store.NewStore(db)).Article(context.Background(), a.ID)
	if e != nil || detail.ArticleDetail.ReadingURL != *r.PublicURL || detail.ArticleDetail.ExternalLink != originalLink || len(detail.ArticleDetail.Tags) != 2 || detail.ArticleDetail.Tags[0] != "tag,with,commas" {
		t.Fatal(detail, e)
	}
	r.SourceID = "b"
	r.DataSourceID = syncDataSourceB
	r.ThemeOptionID = "o2"
	r.ThemeOptionName = "S2"
	out = syncApply(t, db, b, r)
	moved := syncArticle(t, db, out.ArticleID)
	if moved.ID != a.ID || moved.SectionCode != "s2" || moved.SubsectionCode != "" || moved.ExternalLink != originalLink || *moved.SourceKey != *a.SourceKey || moved.Content != a.Content || moved.Author != a.Author {
		t.Fatal(moved)
	}
}
func TestSyncReadFailuresPreserveVisibilityButConfirmedInvalidityBlocks(t *testing.T) {
	db, b, r := syncFixture(t)
	first := syncApply(t, db, b, r)
	requirePublic(t, db, first.ArticleID, true)
	failed := r
	failed.MetadataComplete = false
	failed.MetadataError = "source_unavailable"
	failed.PublicURL = nil
	failed.Title = ""
	syncApply(t, db, b, failed)
	requirePublic(t, db, first.ArticleID, true)
	bad := r
	bad.MetadataComplete = false
	bad.MetadataError = "invalid_field"
	bad.PublicURLObserved = true
	bad.Title = ""
	syncApply(t, db, b, bad)
	a := syncArticle(t, db, first.ArticleID)
	if a.Status != 2 || a.Title != r.Title {
		t.Fatal(a)
	}
	requirePublic(t, db, a.ID, false)
	syncApply(t, db, b, r)
	requirePublic(t, db, a.ID, true)
	unknown := r
	unknown.MetadataComplete = false
	unknown.MetadataError = "unknown_state"
	unknown.DesiredState = 0
	unknown.PublicURLObserved = true
	unknown.PublicURL = nil
	syncApply(t, db, b, unknown)
	requirePublic(t, db, a.ID, false)
	a = syncArticle(t, db, a.ID)
	if a.Status != 2 {
		t.Fatal(a.Status)
	}
	syncApply(t, db, b, r)
	requirePublic(t, db, a.ID, true)
	bad.DesiredState = 1
	syncApply(t, db, b, bad)
	a = syncArticle(t, db, a.ID)
	if a.Status != 1 || a.Title != r.Title {
		t.Fatal(a)
	}
	requirePublic(t, db, a.ID, false)
}
func TestSyncNativeAndParentWithdrawalAreIndependent(t *testing.T) {
	for _, reason := range []string{"native", "trash", "out_of_scope"} {
		t.Run(reason, func(t *testing.T) {
			db, b, r := syncFixture(t)
			first := syncApply(t, db, b, r)
			r.MetadataComplete = false
			if reason == "native" {
				r.NativeArchived = true
			}
			if reason == "trash" {
				r.InTrash = true
			}
			if reason == "out_of_scope" {
				r.MetadataError = reason
			}
			syncApply(t, db, b, r)
			requirePublic(t, db, first.ArticleID, false)
			if a := syncArticle(t, db, first.ArticleID); a.Status != 2 {
				t.Fatal("native withdrawal fabricated business state", a.Status)
			}
		})
	}
}
func TestManagedLegacyNoopLocalFieldsAndHoldRevalidation(t *testing.T) {
	db, b, r := syncFixture(t)
	first := syncApply(t, db, b, r)
	a := syncArticle(t, db, first.ArticleID)
	tags, _ := model.ArticleTags(&a)
	request := UpdateInput{ID: a.ID, Title: a.Title, Author: a.Author, SectionCode: "s1", ModuleCode: "m1", ExternalLink: a.ExternalLink, Tags: tags}
	if _, e := b.Edit(context.Background(), request); e != nil {
		t.Fatal(e)
	}
	if current := syncArticle(t, db, a.ID); !current.UpdatedAt.Equal(a.UpdatedAt) {
		t.Fatal("noop wrote updated_at")
	}
	request.Title = "local overwrite"
	_, e := b.Edit(context.Background(), request)
	errorHTTP(t, e, 409)
	if _, e = b.PatchLocal(context.Background(), a.ID, LocalPatchInput{Author: "Local patched"}); e != nil {
		t.Fatal(e)
	}
	if _, e = b.SetPublicationHold(context.Background(), a.ID, HoldInput{Held: true, Reason: "Pause"}); e != nil {
		t.Fatal(e)
	}
	requirePublic(t, db, a.ID, false)
	rev := syncBinding(t, db).Revision
	b.SetPublicationHold(context.Background(), a.ID, HoldInput{Held: true, Reason: "Pause"})
	if syncBinding(t, db).Revision != rev {
		t.Fatal("repeated hold wrote")
	}
	if _, e = b.SetPublicationHold(context.Background(), a.ID, HoldInput{Held: false}); e != nil {
		t.Fatal(e)
	}
	requirePublic(t, db, a.ID, false)
	syncApply(t, db, b, r)
	requirePublic(t, db, a.ID, true)
	rev = syncBinding(t, db).Revision
	b.SetPublicationHold(context.Background(), a.ID, HoldInput{Held: false})
	if syncBinding(t, db).Revision != rev {
		t.Fatal("repeated release reblocked")
	}
	requirePublic(t, db, a.ID, true)
	if got := syncArticle(t, db, a.ID); got.Author != "Local patched" {
		t.Fatal(got)
	}
	info, e := b.GetOne(context.Background(), a.ID)
	if e != nil || info.Article.Management.Mode != "notion_sync" || !info.Article.EffectiveVisibility {
		t.Fatal(info, e)
	}
}
func TestSyncCASAndArchiveFailureDoNotOverwrite(t *testing.T) {
	db, b, r := syncFixture(t)
	first := syncApply(t, db, b, r)
	p := syncBinding(t, db)
	r.ExpectedBindingRevision = p.Revision - 1
	r.RunID = "different-run"
	r.Title = "stale overwrite"
	_, e := b.ApplySyncedSource(context.Background(), r)
	errorHTTP(t, e, 409)
	a := syncArticle(t, db, first.ArticleID)
	if a.Title == r.Title {
		t.Fatal(a)
	}
	db.Model(&model.Article{}).Where("id = ?", a.ID).UpdateColumn("status", 4)
	r.ExpectedBindingRevision = p.Revision
	r.ThemeOptionID = "duplicate"
	r.ThemeOptionName = "S1"
	r.DesiredState = 2
	out, e := b.ApplySyncedSource(context.Background(), r)
	if e != nil {
		t.Fatal(e)
	}
	if out.AppliedState != 4 {
		t.Fatal("failed archive recovery became live", out)
	}
	a = syncArticle(t, db, a.ID)
	if a.Status != 4 || a.SectionCode != "s1" {
		t.Fatal(a)
	}
	var blocked model.NotionCatalogBinding
	if e = db.First(&blocked, "option_id = ?", "duplicate").Error; e != nil || blocked.Status != model.NotionCatalogBlocked || blocked.SectionCode != nil {
		t.Fatal(blocked, e)
	}
}
func TestEnsureTopicsCreatesEmptyUnusedAndBlocksSameName(t *testing.T) {
	db, b, r := syncFixture(t)
	results, e := b.EnsureSyncedTopics(context.Background(), r.Lease, "a", 1, "topic", []TopicOption{{ID: "unused", Name: "Unused"}, {ID: "collision", Name: "S1"}})
	if e != nil {
		t.Fatal(e)
	}
	var unused, empty, blocked model.NotionCatalogBinding
	db.First(&unused, "option_id = ?", "unused")
	db.First(&empty, "option_id = ?", "")
	db.First(&blocked, "option_id = ?", "collision")
	if unused.SectionCode == nil || empty.SectionCode == nil || blocked.Status != model.NotionCatalogBlocked || len(results) != 3 {
		t.Fatal(results)
	}
	var s1, s2 model.Section
	db.First(&s1, "code = ?", *unused.SectionCode)
	db.First(&s2, "code = ?", *empty.SectionCode)
	if s1.Sort <= 0 || s2.Sort <= s1.Sort {
		t.Fatal(s1, s2)
	}
}
func TestAdoptAliasPreservesHistoryAndAllDuplicateWriters(t *testing.T) {
	db, b, r := syncFixture(t)
	legacyURL := "https://www.notion.so/ffeeddccbbaa00998877665544332211?v=view&p=" + syncPageID
	registered := register(t, b, legacyURL, "s1", "sub1", true)
	id, _ := ParseID(registered.Article.ID)
	db.Model(&model.Article{}).Where("id = ?", id).Updates(map[string]interface{}{"content": "historical body", "author": "Historical", "pos": 71})
	a := syncArticle(t, db, id)
	beforeJSON, e := ArticleBeforeJSON(&a)
	if e != nil {
		t.Fatal(e)
	}
	state := a.Status
	p := model.NotionPageBinding{PageID: syncPageID, SourceID: "a", ArticleID: &id, ManagementState: model.NotionManagementBaselinePending, BootstrapState: "verified", BootstrapExpectedState: &state, BootstrapExpectedFingerprint: "confirmed", LocalBeforeJSON: beforeJSON, DesiredState: state, PageURL: r.PageURL, PublicURL: r.PublicURL, SnapshotJSON: reviewedAdoptSnapshot(r)}
	db.Model(&model.NotionSyncSource{}).Where("source_id = ?", "a").Update("property_mapping_json", `{"topic_property_id":"topic"}`)
	if e = db.Create(&p).Error; e != nil {
		t.Fatal(e)
	}
	db.Model(&model.NotionSyncControl{}).Where("id=1").Updates(map[string]interface{}{"source_writes_paused": true, "baseline_frozen": true, "paused": true})
	out, e := b.AdoptSyncedSource(context.Background(), AdoptInput{Lease: r.Lease, PageID: syncPageID, SourceID: "a", ExpectedArticleID: id, ExpectedConfigRevision: 1, ExpectedFingerprint: "confirmed"})
	if e != nil {
		t.Fatal(e)
	}
	adopted := syncArticle(t, db, id)
	if out.ArticleID != id || adopted.ExternalLink != a.ExternalLink || adopted.Content != a.Content || adopted.Author != a.Author || adopted.Pos != a.Pos || !adopted.UpdatedAt.Equal(a.UpdatedAt) || !adopted.CreatedAt.Equal(a.CreatedAt) {
		t.Fatal(adopted)
	}
	if *adopted.SourceKey == *a.SourceKey || *syncBinding(t, db).LegacySourceKey != *a.SourceKey {
		t.Fatal("canonical key/alias not migrated")
	}
	db.Model(&model.NotionSyncControl{}).Where("id=1").Update("source_writes_paused", false)
	dup := register(t, b, legacyURL, "s2", "", false)
	if dup.Outcome != "already_registered" || dup.Article.ID != registered.Article.ID {
		t.Fatal(dup)
	}
	imported, e := b.Import(context.Background(), ImportRequest{ExternalLink: legacyURL, Title: "overwrite", SectionCode: "s2", Content: importString("bad")}, false)
	if e != nil || imported.Outcome != "already_registered" || imported.ID != id {
		t.Fatal(imported, e)
	}
	tags, _ := model.ArticleTags(&adopted)
	_, e = b.Edit(context.Background(), UpdateInput{ID: id, Title: adopted.Title, Author: adopted.Author, Tags: tags, ExternalLink: legacyURL, SectionCode: "s1", SubsectionCode: "sub1", ModuleCode: "m1"})
	if e != nil {
		t.Fatal(e)
	}
	again := syncArticle(t, db, id)
	if *again.SourceKey != *adopted.SourceKey {
		t.Fatal("legacy noop reverted canonical identity")
	}
	db.Model(&model.NotionPageBinding{}).Where("page_id = ?", syncPageID).Update("management_state", model.NotionManagementDetached)
	dup = register(t, b, legacyURL, "s2", "", false)
	if dup.Article.ID != registered.Article.ID {
		t.Fatal("detached alias forgotten")
	}
}
func TestInternalBeforeFingerprintProtectsJSONTagsAndSourceKey(t *testing.T) {
	key := "key"
	a := &model.Article{ID: 1, SourceKey: &key, Tags: "shadow"}
	model.SetArticleTags(a, []string{"A,B"})
	raw, e := ArticleBeforeJSON(a)
	if e != nil {
		t.Fatal(e)
	}
	restored, e := ParseArticleBeforeJSON(raw)
	if e != nil || ArticleFingerprint(a) != ArticleFingerprint(restored) {
		t.Fatal(raw, e)
	}
	first := ArticleFingerprint(a)
	model.SetArticleTags(a, []string{"C,D"})
	if ArticleFingerprint(a) == first {
		t.Fatal("JSON tags absent from fingerprint")
	}
	first = ArticleFingerprint(a)
	newKey := "new"
	a.SourceKey = &newKey
	if ArticleFingerprint(a) == first {
		t.Fatal("source identity absent from fingerprint")
	}
}

func TestBaselineJournalKeepsHistoricalReadingURLUntilAdopt(t *testing.T) {
	db, b, r := syncFixture(t)
	created := register(t, b, "https://example.com/history", "s1", "", true)
	id, _ := ParseID(created.Article.ID)
	p := model.NotionPageBinding{PageID: syncPageID, ArticleID: &id, SourceID: "a", ManagementState: model.NotionManagementBaselinePending, PageURL: r.PageURL, PublicURL: r.PublicURL}
	if e := db.Create(&p).Error; e != nil {
		t.Fatal(e)
	}
	info, e := b.GetOne(context.Background(), id)
	if e != nil || info.Article.Management.Mode != "manual" || info.Article.ReadingURL != "https://example.com/history" || !info.Article.EffectiveVisibility {
		t.Fatal(info, e)
	}
	detail, e := reading.New(store.NewStore(db)).Article(context.Background(), id)
	if e != nil || detail.ArticleDetail.ReadingURL != "https://example.com/history" {
		t.Fatal(detail, e)
	}
}

func TestAdoptRequiresPauseAndFreshSyncForPublication(t *testing.T) {
	db, b, r := syncFixture(t)
	created := register(t, b, "https://www.notion.so/"+syncPageID, "s1", "", true)
	id, _ := ParseID(created.Article.ID)
	a := syncArticle(t, db, id)
	raw, _ := ArticleBeforeJSON(&a)
	state := a.Status
	p := model.NotionPageBinding{PageID: syncPageID, ArticleID: &id, SourceID: "a", ManagementState: model.NotionManagementBaselinePending, LocalBeforeJSON: raw, BootstrapState: "verified", BootstrapExpectedState: &state, BootstrapExpectedFingerprint: "confirmed", DesiredState: state, PageURL: r.PageURL, PublicURL: r.PublicURL, SnapshotJSON: reviewedAdoptSnapshot(r)}
	db.Model(&model.NotionSyncSource{}).Where("source_id = ?", "a").Update("property_mapping_json", `{"topic_property_id":"topic"}`)
	if e := db.Create(&p).Error; e != nil {
		t.Fatal(e)
	}
	db.Model(&model.NotionSyncControl{}).Where("id=1").Updates(map[string]interface{}{"source_writes_paused": true, "baseline_frozen": true})
	in := AdoptInput{Lease: r.Lease, PageID: syncPageID, SourceID: "a", ExpectedArticleID: id, ExpectedConfigRevision: 1, ExpectedFingerprint: "confirmed"}
	_, e := b.AdoptSyncedSource(context.Background(), in)
	errorHTTP(t, e, 409)
	requirePublic(t, db, id, true)
	db.Model(&model.NotionSyncControl{}).Where("id=1").UpdateColumn("paused", true)
	if _, e = b.AdoptSyncedSource(context.Background(), in); e != nil {
		t.Fatal(e)
	}
	if !syncBinding(t, db).NeedsRevalidation {
		t.Fatal("takeover skipped publication revalidation")
	}
	requirePublic(t, db, id, false)
	db.Model(&model.NotionSyncControl{}).Where("id=1").Updates(map[string]interface{}{"source_writes_paused": false, "paused": false})
	syncApply(t, db, b, r)
	requirePublic(t, db, id, true)
}

func TestSyncConfigAndObservedRevisionFenceInvalidFreshSnapshots(t *testing.T) {
	db, b, r := syncFixture(t)
	earlier := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	later := earlier.Add(time.Hour)
	r.NotionLastEditedAt = &earlier
	first := syncApply(t, db, b, r)
	bad := r
	bad.MetadataComplete = false
	bad.MetadataError = "invalid_field"
	bad.PublicURLObserved = true
	bad.NotionLastEditedAt = &later
	syncApply(t, db, b, bad)
	r.ExpectedBindingRevision = syncBinding(t, db).Revision
	_, e := b.ApplySyncedSource(context.Background(), r)
	errorHTTP(t, e, 409)
	requirePublic(t, db, first.ArticleID, false)
	r.NotionLastEditedAt = &later
	db.Model(&model.NotionSyncSource{}).Where("source_id = ?", "a").Update("config_revision", 2)
	_, e = b.ApplySyncedSource(context.Background(), r)
	errorHTTP(t, e, 409)
}

func TestSyncPublicationChecksRetainedSubsection(t *testing.T) {
	db, b, r := syncFixture(t)
	first := syncApply(t, db, b, r)
	db.Model(&model.Article{}).Where("id = ?", first.ArticleID).UpdateColumn("subsection_code", "sub1")
	db.Model(&model.Subsection{}).Where("code = ?", "sub1").UpdateColumn("status", model.SubsectionStatusDeleted)
	out := syncApply(t, db, b, r)
	if out.Outcome != "state_only" || syncBinding(t, db).PublishBlockReason == "" {
		t.Fatal(out)
	}
	a := syncArticle(t, db, first.ArticleID)
	if a.SubsectionCode != "sub1" || a.Status != model.ArticleStatusPublished {
		t.Fatal(a)
	}
	requirePublic(t, db, first.ArticleID, false)
}

func TestSyncSQLFailureRollsBackArticleAndCatalog(t *testing.T) {
	db, b, r := syncFixture(t)
	r.ThemeOptionID, r.ThemeOptionName = "new", "New topic"
	if e := db.Exec("CREATE TRIGGER fail_sync_binding BEFORE INSERT ON notion_page_bindings BEGIN SELECT RAISE(FAIL, 'fixture binding failure'); END").Error; e != nil {
		t.Fatal(e)
	}
	var beforeSections int64
	db.Model(&model.Section{}).Count(&beforeSections)
	if _, e := b.ApplySyncedSource(context.Background(), r); e == nil {
		t.Fatal("SQL failure swallowed")
	}
	var articles, sections, bindings int64
	db.Model(&model.Article{}).Count(&articles)
	db.Model(&model.Section{}).Count(&sections)
	db.Model(&model.NotionCatalogBinding{}).Where("option_id = ?", "new").Count(&bindings)
	if articles != 0 || sections != beforeSections || bindings != 0 {
		t.Fatal("partial transaction persisted", articles, sections, bindings)
	}
}

func TestTopicCodesUseUnambiguousOpaqueIdentityTuple(t *testing.T) {
	db, b, r := syncFixture(t)
	left, e := b.EnsureSyncedTopics(context.Background(), r.Lease, "a", 1, "a:b", []TopicOption{{ID: "c", Name: "Left"}})
	if e != nil {
		t.Fatal(e)
	}
	right, e := b.EnsureSyncedTopics(context.Background(), r.Lease, "a", 1, "a", []TopicOption{{ID: "b:c", Name: "Right"}})
	if e != nil {
		t.Fatal(e)
	}
	if left[0].SectionCode == "" || right[0].SectionCode == "" || left[0].SectionCode == right[0].SectionCode {
		t.Fatal(left, right)
	}
	old := left[0].SectionCode
	again, e := b.EnsureSyncedTopics(context.Background(), r.Lease, "a", 1, "a:b", []TopicOption{{ID: "c", Name: "Renamed"}})
	if e != nil || again[0].SectionCode != old {
		t.Fatal(again, e)
	}
	if e = catalog.CheckSectionBindingDependency(store.NewStore(db), old); e == nil {
		t.Fatal("mapped chapter deletion allowed")
	}
}

func TestManagedPendingSourceRejectsLegacyManualWriters(t *testing.T) {
	db, b, r := syncFixture(t)
	r.DesiredState = model.ArticleStatusDraft
	syncApply(t, db, b, r)
	_, e := b.RegisterSource(context.Background(), RegisterInput{ExternalLink: r.PageURL, Title: "Manual", SectionCode: "s1"})
	errorHTTP(t, e, 409)
	_, e = b.Import(context.Background(), ImportRequest{ExternalLink: r.PageURL, Title: "Manual", SectionCode: "s1"}, false)
	errorHTTP(t, e, 409)
	var count int64
	db.Model(&model.Article{}).Count(&count)
	if count != 0 {
		t.Fatal("manual writer created pending-owned source", count)
	}
	r.DesiredState = model.ArticleStatusPublished
	first := syncApply(t, db, b, r)
	if first.ArticleID == 0 {
		t.Fatal(first)
	}
	duplicate, e := b.RegisterSource(context.Background(), RegisterInput{ExternalLink: r.PageURL, Title: "Overwrite", SectionCode: "s2"})
	if e != nil || duplicate.Outcome != "already_registered" {
		t.Fatal(duplicate, e)
	}
}

func TestKnownNotionSlugRegistrationAndImportReturnManagedArticle(t *testing.T) {
	db, b, r := syncFixture(t)
	slug := "https://TEAM.notion.site:443/my-slug?view=public#intro"
	r.PublicURL = &slug
	first := syncApply(t, db, b, r)
	readURL := "https://team.notion.site/my-slug?view=public#intro"
	registered, e := b.RegisterSource(context.Background(), RegisterInput{ExternalLink: readURL, Title: "Overwrite", SectionCode: "s2", Publish: false})
	if e != nil || registered.Outcome != "already_registered" || registered.Article.ID != fmt.Sprint(first.ArticleID) {
		t.Fatal(registered, e)
	}
	imported, e := b.Import(context.Background(), ImportRequest{ExternalLink: readURL, Title: "Overwrite", SectionCode: "s2", Content: importString("bad")}, false)
	if e != nil || imported.Outcome != "already_registered" || imported.ID != first.ArticleID {
		t.Fatal(imported, e)
	}
	a := syncArticle(t, db, first.ArticleID)
	if a.Title != r.Title || a.Status != 2 || a.SectionCode != "s1" || a.ExternalLink != r.PageURL {
		t.Fatal("duplicate overwrote source projection", a)
	}
	var count int64
	db.Model(&model.Article{}).Count(&count)
	if count != 1 {
		t.Fatal(count)
	}
}

func TestKnownPendingNotionSlugRejectsManualButAllowsOwnFirstPublication(t *testing.T) {
	db, b, r := syncFixture(t)
	slug := "https://team.notion.site/my-slug"
	r.PublicURL = &slug
	r.DesiredState = 1
	pending := syncApply(t, db, b, r)
	if pending.ArticleID != 0 {
		t.Fatal(pending)
	}
	_, e := b.RegisterSource(context.Background(), RegisterInput{ExternalLink: slug, Title: "Manual", SectionCode: "s1"})
	errorHTTP(t, e, 409)
	_, e = b.Import(context.Background(), ImportRequest{ExternalLink: slug, Title: "Manual", SectionCode: "s1"}, false)
	errorHTTP(t, e, 409)
	r.DesiredState = 2
	created := syncApply(t, db, b, r)
	if created.ArticleID == 0 || created.Outcome != "created" {
		t.Fatal(created)
	}
	requirePublic(t, db, created.ArticleID, true)
}

func TestManualNotionSlugRequiresReviewBeforeSyncWithoutAnyOverwrite(t *testing.T) {
	db, b, r := syncFixture(t)
	slug := "https://team.notion.site/my-slug"
	registered := register(t, b, slug, "s2", "", false)
	id, _ := ParseID(registered.Article.ID)
	before := syncArticle(t, db, id)
	r.PublicURL = &slug
	_, e := b.ApplySyncedSource(context.Background(), r)
	errorHTTP(t, e, 409)
	after := syncArticle(t, db, id)
	if ArticleFingerprint(&after) != ArticleFingerprint(&before) || !after.UpdatedAt.Equal(before.UpdatedAt) {
		t.Fatal("sync overwrote manual slug owner", after)
	}
	var articles, pages int64
	db.Model(&model.Article{}).Count(&articles)
	db.Model(&model.NotionPageBinding{}).Count(&pages)
	if articles != 1 || pages != 0 {
		t.Fatal("conflict created another identity", articles, pages)
	}
}

func TestManagedReaderURLConflictWithdrawsOrBlocksWithoutOverwritingOwners(t *testing.T) {
	for _, state := range []int{model.ArticleStatusDraft, model.ArticleStatusPublished} {
		t.Run(fmt.Sprint(state), func(t *testing.T) {
			db, b, r := syncFixture(t)
			first := syncApply(t, db, b, r)
			ownBefore := syncArticle(t, db, first.ArticleID)
			bindingBefore := syncBinding(t, db)
			slug := "https://team.notion.site/manual-owner"
			manual := register(t, b, slug, "s2", "", true)
			manualID, _ := ParseID(manual.Article.ID)
			otherBefore := syncArticle(t, db, manualID)
			r.PublicURL = &slug
			r.Title = "must not overwrite last good title"
			r.DesiredState = state
			out := syncApply(t, db, b, r)
			ownAfter := syncArticle(t, db, first.ArticleID)
			otherAfter := syncArticle(t, db, manualID)
			bindingAfter := syncBinding(t, db)
			if out.Outcome != "state_only" || ownAfter.Status != state || ownAfter.Title != ownBefore.Title || ownAfter.SectionCode != ownBefore.SectionCode || ownAfter.ExternalLink != ownBefore.ExternalLink {
				t.Fatal(out, ownAfter)
			}
			if ArticleFingerprint(&otherBefore) != ArticleFingerprint(&otherAfter) || !otherBefore.UpdatedAt.Equal(otherAfter.UpdatedAt) {
				t.Fatal("changed other URL owner", otherAfter)
			}
			if bindingAfter.PublicURL == nil || *bindingAfter.PublicURL != *bindingBefore.PublicURL || !bindingAfter.NeedsRevalidation || bindingAfter.PublishBlockReason != "source_identity_conflict" {
				t.Fatal(bindingAfter)
			}
			requirePublic(t, db, first.ArticleID, false)
			requirePublic(t, db, manualID, true)
		})
	}
}

func TestAdoptCannotConfirmNewPageOverKnownManualSlugOwner(t *testing.T) {
	db, b, r := syncFixture(t)
	slug := "https://team.notion.site/manual-owner"
	manual := register(t, b, slug, "s2", "", false)
	id, _ := ParseID(manual.Article.ID)
	before := syncArticle(t, db, id)
	p := model.NotionPageBinding{PageID: syncPageID, SourceID: "a", ManagementState: model.NotionManagementBaselinePending, BootstrapState: "verified", BootstrapExpectedFingerprint: "confirmed", PageURL: r.PageURL, PublicURL: &slug, DesiredState: 2}
	if e := db.Create(&p).Error; e != nil {
		t.Fatal(e)
	}
	db.Model(&model.NotionSyncControl{}).Where("id=1").Updates(map[string]interface{}{"paused": true, "source_writes_paused": true, "baseline_frozen": true})
	_, e := b.AdoptSyncedSource(context.Background(), AdoptInput{Lease: r.Lease, PageID: syncPageID, SourceID: "a", ExpectedConfigRevision: 1, ExpectedFingerprint: "confirmed"})
	errorHTTP(t, e, 409)
	after := syncArticle(t, db, id)
	p = syncBinding(t, db)
	if ArticleFingerprint(&before) != ArticleFingerprint(&after) || p.ManagementState != model.NotionManagementBaselinePending || p.BootstrapState != "verified" {
		t.Fatal(after, p)
	}
}

func reviewedAdoptSnapshot(r SyncInput) string {
	raw, _ := json.Marshal(map[string]interface{}{"page_id": r.PageID, "source_id": r.SourceID, "desired_state": r.DesiredState, "title": r.Title, "tags": r.Tags, "topic_option_id": r.ThemeOptionID, "topic_option_name": r.ThemeOptionName, "public_url": r.PublicURL, "native_archived": r.NativeArchived, "in_trash": r.InTrash})
	return string(raw)
}
