package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/yshujie/miniblog/internal/miniblog/model"
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
