package contentintegration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	articlebiz "github.com/yshujie/miniblog/internal/miniblog/biz/article"
	"github.com/yshujie/miniblog/internal/miniblog/biz/blog"
	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/source"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	v1 "github.com/yshujie/miniblog/pkg/api/miniblog/v1"
)

func TestMySQLNotionLeaseFencingAndConcurrentAcquisition(t *testing.T) {
	db, _ := scratchDB(t)
	ctx := context.Background()
	repo := store.NewNotionSyncRepository(db)
	var wg sync.WaitGroup
	results := make(chan store.LeaseToken, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			token, ok, err := repo.AcquireLease(ctx, "worker", time.Minute)
			if err != nil {
				t.Error(err)
			}
			if ok {
				results <- token
			}
		}()
	}
	wg.Wait()
	close(results)
	var token store.LeaseToken
	count := 0
	for v := range results {
		token = v
		count++
	}
	if count != 1 {
		t.Fatalf("lease owners=%d", count)
	}
	ok, err := repo.RenewLease(ctx, token, time.Minute)
	if err != nil || !ok {
		t.Fatal(ok, err)
	}
	if err = db.Exec("UPDATE notion_sync_control SET lease_until=DATE_SUB(UTC_TIMESTAMP(6),INTERVAL 1 SECOND) WHERE id=1").Error; err != nil {
		t.Fatal(err)
	}
	fresh, ok, err := repo.AcquireLease(ctx, "fresh", time.Minute)
	if err != nil || !ok || fresh.Epoch <= token.Epoch {
		t.Fatal(fresh, ok, err)
	}
	err = store.InTransaction(ctx, store.NewStore(db), func(ds store.IStore) error { _, e := store.LockLease(ds, token); return e })
	if !errors.Is(err, store.ErrLeaseLost) {
		t.Fatal("expired owner committed", err)
	}
	if err = repo.ReleaseLease(ctx, token); err != nil {
		t.Fatal(err)
	}
	c, err := repo.Control(ctx)
	if err != nil || c.LeaseOwner != fresh.Owner || c.LeaseEpoch != fresh.Epoch {
		t.Fatal("old release removed new owner", c, err)
	}
	// Compatibility rollback must preserve the six tables and JSON field.
	migration(t, db, "000006_notion_sync.down.sql", "")
	if !db.Migrator().HasTable(&model.NotionPageBinding{}) || !db.Migrator().HasColumn(&model.Article{}, "tags_json") {
		t.Fatal("rollback removed retained data")
	}
	migration(t, db, "000006_notion_sync.up.sql", "")
}

func TestMySQLSyncedStatesHoldAndIdempotency(t *testing.T) {
	db, _ := scratchDB(t)
	migration(t, db, "000005_source_uniqueness.up.sql", "")
	ctx := context.Background()
	ds := store.NewStore(db)
	b := articlebiz.New(ds)
	const sourceID = "2bf330bd-ddf1-80a6-aa49-000bbd1e154b"
	if err := db.Model(&model.NotionSyncControl{}).Where("id=1").Updates(map[string]interface{}{"paused": false, "baseline_frozen": true}).Error; err != nil {
		t.Fatal(err)
	}
	src := model.NotionSyncSource{ID: sourceID, DataSourceID: sourceID, ModuleCode: "m1", ConfigRevision: 1, Enabled: true}
	if err := db.Create(&src).Error; err != nil {
		t.Fatal(err)
	}
	token, ok, err := ds.NotionSync().AcquireLease(ctx, "test", time.Minute)
	if err != nil || !ok {
		t.Fatal(ok, err)
	}
	url := "https://example.notion.site/Public-Slug?section=Go#Reading"
	in := articlebiz.SyncInput{Lease: token, RunID: "test-run", SourceID: sourceID, DataSourceID: sourceID, PageID: "1234567890abcdef1234567890abcdef", ThemePropertyID: "topic", ThemeOptionID: "topic-one", ThemeOptionName: "New topic", Title: "Synced", Tags: []string{"comma,tag", "标签"}, PageURL: "https://notion.so/1234567890abcdef1234567890abcdef", PublicURL: &url, DesiredState: 2, MetadataComplete: true, ExpectedConfigRevision: 1, MetadataHash: "first"}
	r, err := b.ApplySyncedSource(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	id := r.ArticleID
	if id == 0 {
		t.Fatal("no article created")
	}
	var first model.Article
	if err = db.First(&first, id).Error; err != nil {
		t.Fatal(err)
	}
	var p model.NotionPageBinding
	db.First(&p, "page_id=?", in.PageID)
	in.ExpectedBindingRevision = p.Revision
	r, err = b.ApplySyncedSource(ctx, in)
	if err != nil || r.ArticleID != id {
		t.Fatal(r, err)
	}
	var second model.Article
	db.First(&second, id)
	if second.Pos != first.Pos || !second.UpdatedAt.Equal(first.UpdatedAt) {
		t.Fatal("repeated sync reordered or changed update time")
	}
	tags, err := model.ArticleTags(&second)
	if err != nil || len(tags) != 2 || tags[0] != "comma,tag" {
		t.Fatal(tags, err)
	}
	for _, state := range []int{4, 1, 3, 2} {
		db.First(&p, "page_id=?", in.PageID)
		in.ExpectedBindingRevision = p.Revision
		in.DesiredState = state
		r, err = b.ApplySyncedSource(ctx, in)
		if err != nil || r.ArticleID != id {
			t.Fatal(state, r, err)
		}
		db.First(&second, id)
		if second.Status != state {
			t.Fatal(state, second.Status)
		}
	}
	if _, err = b.SetPublicationHold(ctx, id, articlebiz.HoldInput{Held: true, Reason: "local safety"}); err != nil {
		t.Fatal(err)
	}
	// A known Notion Site slug has no PageID. Re-registering it must not create
	// a manual article that escapes the original local publication hold.
	duplicate, duplicateErr := b.RegisterSource(ctx, articlebiz.RegisterInput{ExternalLink: url, Title: "Must not duplicate", SectionCode: "s2", Publish: true})
	if duplicateErr != nil || duplicate.Outcome != "already_registered" || duplicate.Article.ID != strconv.FormatUint(id, 10) {
		t.Fatal("known slug escaped existing ownership", duplicate, duplicateErr)
	}
	var articleCount int64
	db.Model(&model.Article{}).Count(&articleCount)
	if articleCount != 1 {
		t.Fatal("known reading address created duplicate", articleCount)
	}
	reader := blog.New(ds)
	if _, err = reader.GetArticleDetail(ctx, &v1.GetArticleDetailRequest{ArticleID: id}); err == nil {
		t.Fatal("held article visible")
	}
	if _, err = b.SetPublicationHold(ctx, id, articlebiz.HoldInput{Held: false}); err != nil {
		t.Fatal(err)
	}
	if _, err = reader.GetArticleDetail(ctx, &v1.GetArticleDetailRequest{ArticleID: id}); err == nil {
		t.Fatal("released hold visible before source validation")
	}
	db.First(&p, "page_id=?", in.PageID)
	in.ExpectedBindingRevision = p.Revision
	if _, err = b.ApplySyncedSource(ctx, in); err != nil {
		t.Fatal(err)
	}
	if _, err = reader.GetArticleDetail(ctx, &v1.GetArticleDetailRequest{ArticleID: id}); err != nil {
		t.Fatal("fresh validation did not restore", err)
	}
	// A stale projection is rejected atomically.
	in.ExpectedBindingRevision = 0
	in.Title = "must not overwrite"
	in.RunID = "stale-other-run"
	in.MetadataHash = "stale-different-snapshot"
	if _, err = b.ApplySyncedSource(ctx, in); err == nil {
		t.Fatal("stale page revision accepted")
	}
	db.First(&second, id)
	if second.Title != "Synced" {
		t.Fatal(second.Title)
	}
	// Actual SQL failure after article/catalog creation must roll the whole projection back.
	var beforeArticles, beforeSections int64
	db.Model(&model.Article{}).Count(&beforeArticles)
	db.Model(&model.Section{}).Count(&beforeSections)
	if err = db.Exec("CREATE TRIGGER notion_test_binding_failure BEFORE INSERT ON notion_page_bindings FOR EACH ROW SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'fixture page binding failure'").Error; err != nil {
		t.Fatal(err)
	}
	failing := in
	failing.PageID = "aabbccddeeff00112233445566778899"
	failing.PageURL = "https://notion.so/aabbccddeeff00112233445566778899"
	failingURL := "https://example.notion.site/SQL-Failure-Slug"
	failing.PublicURL = &failingURL
	failing.ThemeOptionID = "failure-option"
	failing.ThemeOptionName = "Must roll back"
	failing.RunID = "failed-new-page"
	failing.ExpectedBindingRevision = 0
	failing.MetadataHash = "rollback"
	if _, err = b.ApplySyncedSource(ctx, failing); err == nil || !strings.Contains(err.Error(), "fixture page binding failure") {
		t.Fatal("expected actual SQL trigger failure", err)
	}
	if err = db.Exec("DROP TRIGGER notion_test_binding_failure").Error; err != nil {
		t.Fatal(err)
	}
	var afterArticles, afterSections int64
	db.Model(&model.Article{}).Count(&afterArticles)
	db.Model(&model.Section{}).Count(&afterSections)
	if afterArticles != beforeArticles || afterSections != beforeSections {
		t.Fatal("partial projection persisted", afterArticles, afterSections)
	}
	// Long comma-bearing tags use JSON and preserve the historical CSV shadow.
	db.First(&p, "page_id=?", in.PageID)
	in.ExpectedBindingRevision = p.Revision
	in.RunID = "long-tags"
	in.Title = "Synced"
	in.Tags = []string{strings.Repeat("tag,", 200)}
	in.MetadataHash = "long-tags"
	if _, err = b.ApplySyncedSource(ctx, in); err != nil {
		t.Fatal(err)
	}
	db.First(&second, id)
	tags, err = model.ArticleTags(&second)
	if err != nil || len(tags) != 1 || tags[0] != in.Tags[0] {
		t.Fatal("JSON tags truncated", tags, err)
	}
	// A disabled target module blocks a cross-module move and keeps the stable ID.
	const sourceTwo = "d579f619-4f1c-4e7e-9593-ab6529fc63d0"
	srcTwo := model.NotionSyncSource{ID: sourceTwo, DataSourceID: sourceTwo, ModuleCode: "m2", ConfigRevision: 1, Enabled: true}
	if err = db.Create(&srcTwo).Error; err != nil {
		t.Fatal(err)
	}
	db.Model(&model.Module{}).Where("code='m2'").Update("status", 2)
	db.First(&p, "page_id=?", in.PageID)
	in.ExpectedBindingRevision = p.Revision
	in.SourceID = sourceTwo
	in.DataSourceID = sourceTwo
	in.ThemeOptionID = "new-source-option"
	in.ThemeOptionName = "Other source chapter"
	in.RunID = "cross-module"
	in.MetadataHash = "cross-module"
	if _, err = b.ApplySyncedSource(ctx, in); err != nil {
		t.Fatal(err)
	}
	if _, err = reader.GetArticleDetail(ctx, &v1.GetArticleDetailRequest{ArticleID: id}); err == nil {
		t.Fatal("blocked projection remained public")
	}
	db.Model(&model.Module{}).Where("code='m2'").Update("status", 1)
	db.First(&p, "page_id=?", in.PageID)
	in.ExpectedBindingRevision = p.Revision
	in.RunID = "cross-module-revalidated"
	if _, err = b.ApplySyncedSource(ctx, in); err != nil {
		t.Fatal(err)
	}
	db.First(&second, id)
	if second.ID != id || second.ExternalLink != first.ExternalLink || second.Content != first.Content || second.SectionCode == first.SectionCode {
		t.Fatal("move failed historical preservation", second)
	}
	detail, err := reader.GetArticleDetail(ctx, &v1.GetArticleDetailRequest{ArticleID: id})
	if err != nil || detail.ArticleDetail.ModuleCode != "m2" {
		t.Fatal("old ID did not resolve current module", detail, err)
	}

}

func TestMySQLVerifiedLegacyAliasResolvesOriginalArticle(t *testing.T) {
	db, _ := scratchDB(t)
	migration(t, db, "000005_source_uniqueness.up.sql", "")
	const id uint64 = 9007199254740993
	canonical, _ := source.NotionIdentity("1234567890abcdef1234567890abcdef")
	legacy, _ := source.Parse("https://example.com/legacy?id=unchanged#original")
	a := model.Article{ID: id, Title: "History", ExternalLink: legacy.CanonicalURL, SectionCode: "s1", Status: 2, Pos: 1, SourceKey: &canonical.SourceKey, Provider: &canonical.Provider, CanonicalURL: &canonical.CanonicalURL}
	if err := db.Create(&a).Error; err != nil {
		t.Fatal(err)
	}
	binding := model.NotionPageBinding{PageID: "1234567890abcdef1234567890abcdef", ArticleID: &a.ID, LegacySourceKey: &legacy.SourceKey, ManagementState: model.NotionManagementManaged}
	if err := db.Create(&binding).Error; err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	ds := store.NewStore(db)
	for _, key := range []string{canonical.SourceKey, legacy.SourceKey} {
		owner, err := store.FindSourceOwner(ds, key)
		if err != nil || owner == nil || owner.ID != id {
			t.Fatal(owner, err)
		}
	}
	r, err := articlebiz.New(ds).Register(ctx, &v1.RegisterArticleRequest{ExternalLink: a.ExternalLink, Title: "Must not replace", SectionCode: "s2", Publish: true})
	if err != nil || r.Outcome != "already_registered" || r.Article.ID != "9007199254740993" {
		t.Fatal(r, err)
	}
	var actual model.Article
	db.First(&actual, id)
	if actual.Title != "History" || actual.SectionCode != "s1" || *actual.SourceKey != canonical.SourceKey {
		t.Fatal("duplicate overwrote managed history")
	}
	if err = store.AuditSourceAliases(ctx, db); err != nil {
		t.Fatal(err)
	}
}

func TestMySQLHistoricalTakeoverPreservesHistoryAndFencesWriters(t *testing.T) {
	db, _ := scratchDB(t)
	migration(t, db, "000005_source_uniqueness.up.sql", "")
	const pageID = "1234567890abcdef1234567890abcdef"
	const sourceID = "2bf330bd-ddf1-80a6-aa49-000bbd1e154b"
	const id uint64 = 9007199254740993
	link := "https://app.notion.com/p/Title-" + pageID + "?source=copy_link#keep"
	digest := sha256.Sum256([]byte("url:" + link))
	legacy := hex.EncodeToString(digest[:])
	provider := "other"
	row := model.Article{ID: id, Title: "History", Content: "Original body", ExternalLink: link, Author: "Local author", Tags: "Go,旧标签", SectionCode: "s1", SubsectionCode: "sub1", Status: 2, Pos: 7, Provider: &provider, SourceKey: &legacy, CanonicalURL: &link}
	if err := db.Create(&row).Error; err != nil {
		t.Fatal(err)
	}
	stamp := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	db.Model(&model.Article{}).Where("id=?", id).UpdateColumns(map[string]interface{}{"created_at": stamp, "updated_at": stamp})
	db.First(&row, id)
	src := model.NotionSyncSource{ID: sourceID, DataSourceID: sourceID, ModuleCode: "m1", ConfigRevision: 1}
	if err := db.Create(&src).Error; err != nil {
		t.Fatal(err)
	}
	beforeJSON, err := articlebiz.ArticleBeforeJSON(&row)
	if err != nil {
		t.Fatal(err)
	}
	state := 2
	public := "https://fixture.notion.site/" + pageID
	binding := model.NotionPageBinding{PageID: pageID, ArticleID: &row.ID, SourceID: sourceID, ManagementState: model.NotionManagementBaselinePending, Revision: 3, DesiredState: 2, PublicURL: &public, BootstrapState: "verified", BootstrapExpectedState: &state, BootstrapExpectedFingerprint: "approved-review", LocalBeforeJSON: beforeJSON}
	if err := db.Create(&binding).Error; err != nil {
		t.Fatal(err)
	}
	db.Model(&model.NotionSyncControl{}).Where("id=1").Updates(map[string]interface{}{"paused": true, "source_writes_paused": true, "baseline_frozen": true})
	ds := store.NewStore(db)
	ctx := context.Background()
	b := articlebiz.New(ds)
	token, ok, err := ds.NotionSync().AcquireLease(ctx, "takeover", time.Minute)
	if err != nil || !ok {
		t.Fatal(ok, err)
	}
	in := articlebiz.AdoptInput{Lease: token, PageID: pageID, SourceID: sourceID, ExpectedArticleID: id, ExpectedBindingRevision: 3, ExpectedConfigRevision: 1, ExpectedFingerprint: "approved-review"}
	// A concurrent administrator cannot write during the drained maintenance window.
	if _, err = b.Update(ctx, &v1.UpdateArticleRequest{ID: "9007199254740993", Title: "Unexpected", ModuleCode: "m1", SectionCode: "s1", SubsectionCode: "sub1", ExternalLink: link}); err == nil {
		t.Fatal("maintenance write was accepted")
	}
	if _, err = b.AdoptSyncedSource(ctx, in); err != nil {
		t.Fatal(err)
	}
	var actual model.Article
	db.First(&actual, id)
	db.First(&binding, "page_id=?", pageID)
	canonical, _ := source.NotionIdentity(pageID)
	if actual.Title != row.Title || actual.Content != row.Content || actual.Author != row.Author || actual.ExternalLink != link || actual.SubsectionCode != "sub1" || actual.Pos != 7 || actual.Status != 2 || !actual.UpdatedAt.Equal(stamp) || *actual.SourceKey != canonical.SourceKey || binding.LegacySourceKey == nil || *binding.LegacySourceKey != legacy {
		t.Fatal("takeover changed history", actual, binding)
	}
	if _, err = b.AdoptSyncedSource(ctx, in); err != nil {
		t.Fatal("takeover retry not idempotent", err)
	}
	// Pause revocation invalidates an already prepared projection/confirmation.
	db.Model(&model.NotionSyncControl{}).Where("id=1").UpdateColumn("lease_epoch", token.Epoch+1)
	if _, err = b.AdoptSyncedSource(ctx, in); !errors.Is(err, store.ErrLeaseLost) {
		t.Fatal("revoked takeover committed", err)
	}
	if err = store.AuditSourceAliases(ctx, db); err != nil {
		t.Fatal(err)
	}
}
