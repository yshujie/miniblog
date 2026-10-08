package main

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/yshujie/miniblog/internal/miniblog/biz/reading"
	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/source"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"github.com/yshujie/miniblog/internal/pkg/errno"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func auditDB(t *testing.T) *gorm.DB {
	t.Helper()
	gdb, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err = gdb.AutoMigrate(&model.Module{}, &model.Section{}, &model.Subsection{}, &model.Article{}); err != nil {
		t.Fatal(err)
	}
	m := model.Module{Code: "m", Title: "Module", Status: 1}
	if err = gdb.Create(&m).Error; err != nil {
		t.Fatal(err)
	}
	s := model.Section{Code: "s", ModuleCode: "m", Title: "Section", Status: 1}
	if err = gdb.Create(&s).Error; err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := gdb.DB()
	t.Cleanup(func() { sqlDB.Close() })
	return gdb
}

func auditArticle(t *testing.T, gdb *gorm.DB, id uint64, link string) {
	t.Helper()
	row := model.Article{ID: id, Title: "Historical title", ExternalLink: link, Content: "historical content", SectionCode: "s", Status: 2, Pos: 3}
	if err := gdb.Create(&row).Error; err != nil {
		t.Fatal(err)
	}
}

func TestAuditAndBackfillPreserveHistoricalData(t *testing.T) {
	gdb := auditDB(t)
	link := "https://www.notion.so/Article-1234567890abcdef1234567890abcdef?pvs=4"
	auditArticle(t, gdb, 9007199254740993, link)
	stamp := time.Date(2024, 2, 1, 12, 0, 0, 0, time.UTC)
	if err := gdb.Model(&model.Article{}).Where("id = ?", uint64(9007199254740993)).UpdateColumns(map[string]interface{}{"created_at": stamp, "updated_at": stamp}).Error; err != nil {
		t.Fatal(err)
	}
	state, err := inspect(context.Background(), gdb)
	if err != nil {
		t.Fatal(err)
	}
	if state.report.blocked() || state.report.WithSource != 1 {
		t.Fatalf("unexpected audit: %+v", state.report)
	}
	var before model.Article
	gdb.First(&before)
	if before.SourceKey != nil {
		t.Fatal("read-only audit wrote identity")
	}
	if err = backfill(context.Background(), gdb, state); err != nil {
		t.Fatal(err)
	}
	var after model.Article
	gdb.First(&after)
	if after.ID != before.ID || after.Title != before.Title || after.Content != before.Content || after.ExternalLink != link || after.Status != before.Status || after.Pos != before.Pos {
		t.Fatalf("historical fields changed: %+v", after)
	}
	if !after.CreatedAt.Equal(stamp) || !after.UpdatedAt.Equal(stamp) {
		t.Fatal("backfill changed historical timestamps")
	}
	if after.SourceKey == nil || after.Provider == nil || *after.Provider != "notion" {
		t.Fatal("source was not backfilled")
	}
	rerun, err := inspect(context.Background(), gdb)
	if err != nil {
		t.Fatal(err)
	}
	if err = backfill(context.Background(), gdb, rerun); err != nil {
		t.Fatalf("backfill was not repeatable: %v", err)
	}
}

func TestDuplicatesBlockAllBackfillWithoutMerging(t *testing.T) {
	gdb := auditDB(t)
	auditArticle(t, gdb, 11, "https://www.notion.so/One-1234567890abcdef1234567890abcdef")
	auditArticle(t, gdb, 12, "https://example.notion.site/Two-1234567890abcdef1234567890abcdef?pvs=4")
	auditArticle(t, gdb, 13, "https://example.com/unique")
	state, err := inspect(context.Background(), gdb)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.report.Duplicates) != 1 || len(state.report.Duplicates[0].IDs) != 2 {
		t.Fatalf("duplicate not reported: %+v", state.report)
	}
	if err = backfill(context.Background(), gdb, state); err == nil {
		t.Fatal("backfilled unresolved duplicates")
	}
	var changed int64
	gdb.Model(&model.Article{}).Where("source_key IS NOT NULL").Count(&changed)
	if changed != 0 {
		t.Fatal("partial backfill occurred despite blocker")
	}
	var count int64
	gdb.Model(&model.Article{}).Count(&count)
	if count != 3 {
		t.Fatal("audit merged/deleted history")
	}
}

func TestInvalidPlacementAndLinksAreBlocking(t *testing.T) {
	gdb := auditDB(t)
	auditArticle(t, gdb, 21, "javascript:alert(1)")
	if err := gdb.Model(&model.Article{}).Where("id = ?", 21).UpdateColumns(map[string]interface{}{"section_code": "missing", "status": 9}).Error; err != nil {
		t.Fatal(err)
	}
	state, err := inspect(context.Background(), gdb)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.report.Issues) != 3 || !state.report.blocked() {
		t.Fatalf("missing issues: %+v", state.report)
	}
}

func TestBackfillRejectsConcurrentContentChanges(t *testing.T) {
	gdb := auditDB(t)
	auditArticle(t, gdb, 31, "https://example.com/one")
	state, err := inspect(context.Background(), gdb)
	if err != nil {
		t.Fatal(err)
	}
	if err = gdb.Model(&model.Article{}).Where("id = ?", 31).UpdateColumn("external_link", "https://example.com/two").Error; err != nil {
		t.Fatal(err)
	}
	if err = backfill(context.Background(), gdb, state); err == nil {
		t.Fatal("accepted changed source identity")
	}
	var row model.Article
	gdb.First(&row)
	if row.SourceKey != nil {
		t.Fatal("failed backfill persisted identity")
	}
}

func TestContentOnlyHistoryRemainsUnbound(t *testing.T) {
	gdb := auditDB(t)
	auditArticle(t, gdb, 41, "")
	state, err := inspect(context.Background(), gdb)
	if err != nil {
		t.Fatal(err)
	}
	if state.report.blocked() || state.report.LegacyContentOnly != 1 {
		t.Fatalf("content-only history blocked: %+v", state.report)
	}
	if err = backfill(context.Background(), gdb, state); err != nil {
		t.Fatal(err)
	}
	var row model.Article
	gdb.First(&row)
	if row.SourceKey != nil || row.Provider != nil || row.CanonicalURL != nil {
		t.Fatal("invented external identity for historical content")
	}
}

func TestAuditPreservesVerifiedIdentityAndImmutableLegacyAlias(t *testing.T) {
	db := auditDB(t)
	if err := db.AutoMigrate(&model.NotionPageBinding{}, &model.NotionSyncControl{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.NotionSyncControl{ID: 1}).Error; err != nil {
		t.Fatal(err)
	}
	const id uint64 = 9007199254740993
	const pageID = "1234567890abcdef1234567890abcdef"
	const link = "https://example.com/historical-alias#keep"
	auditArticle(t, db, id, link)
	legacy, _ := source.Parse(link)
	canonical, _ := source.NotionIdentity(pageID)
	if err := db.Model(&model.Article{}).Where("id=?", id).UpdateColumns(map[string]interface{}{"provider": canonical.Provider, "canonical_url": canonical.CanonicalURL, "source_key": canonical.SourceKey}).Error; err != nil {
		t.Fatal(err)
	}
	binding := model.NotionPageBinding{PageID: pageID, ArticleID: newUint(id), LegacySourceKey: &legacy.SourceKey, ManagementState: model.NotionManagementManaged, Revision: 7}
	if err := db.Create(&binding).Error; err != nil {
		t.Fatal(err)
	}
	state, err := inspect(context.Background(), db)
	if err != nil || state.report.blocked() {
		t.Fatal(err, state.report)
	}
	if err = backfill(context.Background(), db, state); err != nil {
		t.Fatal(err)
	}
	var row model.Article
	db.First(&row, id)
	if *row.SourceKey != canonical.SourceKey || row.ExternalLink != link || row.ID != id {
		t.Fatal("verified identity regressed")
	}
	db.Model(&binding).Update("revision", 8)
	if err = backfill(context.Background(), db, state); err == nil {
		t.Fatal("stale binding audit accepted")
	}
}
func newUint(v uint64) *uint64 { return &v }

func TestJSONTagBackfillPreservesCSVAndRequiresMaintenance(t *testing.T) {
	db := auditDB(t)
	if err := db.AutoMigrate(&model.NotionPageBinding{}, &model.NotionSyncControl{}); err != nil {
		t.Fatal(err)
	}
	db.Create(&model.NotionSyncControl{ID: 1})
	auditArticle(t, db, 51, "https://example.com/tags")
	db.Model(&model.Article{}).Where("id=?", 51).UpdateColumn("tags", "Go,复杂标签")
	state, err := inspect(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	if err = backfill(context.Background(), db, state, true); err == nil {
		t.Fatal("unpaused migration accepted")
	}
	db.Model(&model.NotionSyncControl{}).Where("id=1").Updates(map[string]interface{}{"paused": true, "source_writes_paused": true})
	if err = backfill(context.Background(), db, state, true); err != nil {
		t.Fatal(err)
	}
	var row model.Article
	db.First(&row, 51)
	tags, err := model.ArticleTags(&row)
	if err != nil || len(tags) != 2 || tags[1] != "复杂标签" || row.Tags != "Go,复杂标签" || row.TagsJSON == nil {
		t.Fatal(row.TagsJSON, tags, err)
	}
	if !row.UpdatedAt.Equal(state.candidates[0].before.UpdatedAt) {
		t.Fatal("tag migration changed historical timestamp")
	}
}

// Historical SQL seeds wrote status 0 directly, bypassing the create hooks.
// Accepting these inactive module/section rows must never publish or normalize them.
func TestAuditLegacyInactiveCatalogPreservesHistoryAndVisibility(t *testing.T) {
	for _, kind := range []string{"module", "section"} {
		t.Run(kind, func(t *testing.T) {
			gdb := auditDB(t)
			const articleID uint64 = 9007199254740993
			auditArticle(t, gdb, articleID, "https://example.com/legacy-inactive")
			created := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
			updated := created.Add(24 * time.Hour)
			for _, table := range []string{"module", "section", "article"} {
				changes := map[string]interface{}{"created_at": created, "updated_at": updated}
				if table == kind {
					changes["status"] = 0
				}
				if err := gdb.Table(table).Where("1 = 1").UpdateColumns(changes).Error; err != nil {
					t.Fatal(err)
				}
			}
			var beforeModule model.Module
			var beforeSection model.Section
			var beforeArticle model.Article
			for _, row := range []interface{}{&beforeModule, &beforeSection, &beforeArticle} {
				if err := gdb.First(row).Error; err != nil {
					t.Fatal(err)
				}
			}
			state, err := inspect(context.Background(), gdb)
			if err != nil {
				t.Fatal(err)
			}
			if state.report.blocked() {
				t.Fatalf("historical inactive %s blocked: %+v", kind, state.report)
			}
			if err = backfill(context.Background(), gdb, state); err != nil {
				t.Fatal(err)
			}
			var afterModule model.Module
			var afterSection model.Section
			var afterArticle model.Article
			for _, row := range []interface{}{&afterModule, &afterSection, &afterArticle} {
				if err := gdb.First(row).Error; err != nil {
					t.Fatal(err)
				}
			}
			if afterModule.Status != beforeModule.Status || afterSection.Status != beforeSection.Status || afterArticle.Status != beforeArticle.Status {
				t.Fatal("backfill normalized historical status")
			}
			for _, stamps := range [][2]time.Time{{afterModule.CreatedAt, afterModule.UpdatedAt}, {afterSection.CreatedAt, afterSection.UpdatedAt}, {afterArticle.CreatedAt, afterArticle.UpdatedAt}} {
				if !stamps[0].Equal(created) || !stamps[1].Equal(updated) {
					t.Fatal("backfill changed historical timestamps")
				}
			}
			if afterArticle.ID != beforeArticle.ID || afterArticle.Content != beforeArticle.Content || afterArticle.ExternalLink != beforeArticle.ExternalLink || afterArticle.Pos != beforeArticle.Pos || afterArticle.SourceKey == nil {
				t.Fatal("backfill changed article history or omitted source identity")
			}
			public := reading.New(store.NewStore(gdb))
			modules, err := public.Modules(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if kind == "module" {
				if len(modules.Modules) != 0 {
					t.Fatal("legacy inactive module became public")
				}
				if _, err = public.Module(context.Background(), "m"); !errors.Is(err, errno.ErrModuleNotFound) {
					t.Fatalf("legacy inactive module detail is public: %v", err)
				}
			} else {
				detail, err := public.Module(context.Background(), "m")
				if err != nil || len(detail.ModuleDetail.Sections) != 0 {
					t.Fatalf("legacy inactive section became public: %v", err)
				}
			}
			if _, err = public.Article(context.Background(), articleID); !errors.Is(err, errno.ErrArticleNotFound) {
				t.Fatalf("published article under inactive %s is public: %v", kind, err)
			}
		})
	}
}

func TestAuditStillBlocksUnsupportedStatusesWithoutBackfill(t *testing.T) {
	for _, tc := range []struct {
		kind   string
		status int
	}{
		{"module", -1}, {"module", 3}, {"section", -1}, {"section", 3},
		{"subsection", 0}, {"subsection", 3}, {"article", 0}, {"article", 5},
	} {
		t.Run(fmt.Sprintf("%s_%d", tc.kind, tc.status), func(t *testing.T) {
			gdb := auditDB(t)
			auditArticle(t, gdb, 91, "https://example.com/invalid-status")
			if tc.kind == "subsection" {
				if err := gdb.Create(&model.Subsection{Code: "child", SectionCode: "s", Title: "Child", Status: model.SubsectionStatusNormal}).Error; err != nil {
					t.Fatal(err)
				}
			}
			if err := gdb.Table(tc.kind).Where("1 = 1").UpdateColumn("status", tc.status).Error; err != nil {
				t.Fatal(err)
			}
			state, err := inspect(context.Background(), gdb)
			if err != nil {
				t.Fatal(err)
			}
			if !state.report.blocked() || len(state.report.Issues) != 1 || state.report.Issues[0].Kind != tc.kind || state.report.Issues[0].Reason != "invalid_status" {
				t.Fatalf("unsupported status was accepted: %+v", state.report)
			}
			if err = backfill(context.Background(), gdb, state); err == nil {
				t.Fatal("backfill accepted unsupported status")
			}
			var written int64
			if err = gdb.Model(&model.Article{}).Where("source_key IS NOT NULL").Count(&written).Error; err != nil || written != 0 {
				t.Fatal("blocked audit performed partial source backfill", err)
			}
		})
	}
}
