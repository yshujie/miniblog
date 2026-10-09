package article

import (
	"context"
	"database/sql"
	"os"
	"regexp"
	"testing"
	"time"

	drivermysql "github.com/go-sql-driver/mysql"
	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Never share content-integration's resettable tables or the NULL-repair test
// namespace. Creation must succeed without replacing an existing database.
func projectionMySQLDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("MINIBLOG_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("dedicated loopback MySQL fixture not configured")
	}
	cfg, err := drivermysql.ParseDSN(dsn)
	if err != nil || cfg.Net != "tcp" || !regexp.MustCompile(`^(127\.0\.0\.1|localhost|\[::1\]):[1-9][0-9]{0,4}$`).MatchString(cfg.Addr) || !regexp.MustCompile(`^miniblog_refactor_test_[A-Za-z0-9_]{1,30}$`).MatchString(cfg.DBName) {
		t.Fatal("refusing non-isolated projection MySQL fixture")
	}
	namespace := cfg.DBName + "_projection"
	adminConfig := *cfg
	adminConfig.DBName = ""
	admin, err := sql.Open("mysql", adminConfig.FormatDSN())
	if err != nil {
		t.Fatal("projection fixture connection unavailable")
	}
	t.Cleanup(func() { admin.Close() })
	if _, err = admin.Exec("CREATE DATABASE `" + namespace + "` CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci"); err != nil {
		t.Fatal("projection namespace creation failed; existing databases are never replaced")
	}
	t.Cleanup(func() {
		if _, err := admin.Exec("DROP DATABASE `" + namespace + "`"); err != nil {
			t.Error("projection namespace cleanup failed")
		}
	})
	cfg.DBName = namespace
	cfg.ParseTime = true
	db, err := gorm.Open(mysql.Open(cfg.FormatDSN()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal("projection fixture connection unavailable")
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal("projection fixture pool unavailable")
	}
	pool.SetMaxOpenConns(4)
	t.Cleanup(func() { pool.Close() })
	if err = db.AutoMigrate(&model.Module{}, &model.Section{}, &model.Subsection{}, &model.Article{}, &model.NotionSyncControl{}, &model.NotionSyncSource{}, &model.NotionCatalogBinding{}, &model.NotionPageBinding{}, &model.NotionSyncRun{}, &model.NotionSyncRunItem{}); err != nil {
		t.Fatal("projection fixture schema unavailable")
	}
	for _, row := range []interface{}{
		&model.Module{Code: "m1", Title: "M1", Status: 1}, &model.Module{Code: "m2", Title: "M2", Status: 1},
		&model.Section{Code: "s1", Title: "S1", ModuleCode: "m1", Status: 1}, &model.Section{Code: "s2", Title: "S2", ModuleCode: "m2", Status: 1},
		&model.NotionSyncControl{ID: 1},
		&model.NotionSyncSource{ID: "a", DataSourceID: syncDataSourceA, ModuleCode: "m1", Enabled: true, ConfigRevision: 1},
		&model.NotionSyncSource{ID: "b", DataSourceID: syncDataSourceB, ModuleCode: "m2", Enabled: true, ConfigRevision: 1},
		&model.NotionCatalogBinding{SourceID: "a", DataSourceID: syncDataSourceA, ThemePropertyID: "topic", OptionID: "o1", OptionName: "S1", SectionCode: syncString("s1"), Status: model.NotionCatalogBound},
		&model.NotionCatalogBinding{SourceID: "b", DataSourceID: syncDataSourceB, ThemePropertyID: "topic", OptionID: "o2", OptionName: "S2", SectionCode: syncString("s2"), Status: model.NotionCatalogBound},
	} {
		if err = db.Create(row).Error; err != nil {
			t.Fatal("projection fixture seed failed")
		}
	}
	return db
}

func TestMySQLSyncedProjectionPreservesPhysicalNull(t *testing.T) {
	db := projectionMySQLDB(t)
	ctx := context.Background()
	lease, ok, err := store.NewNotionSyncRepository(db).AcquireLease(ctx, "projection-test", time.Hour)
	if err != nil || !ok {
		t.Fatal("projection fixture lease unavailable")
	}
	b := NewForSync(store.NewStore(db), "Default author")
	r := SyncInput{Lease: lease, RunID: "projection-run", SourceID: "a", DataSourceID: syncDataSourceA, PageID: syncPageID, ThemePropertyID: "topic", ThemeOptionID: "o1", ThemeOptionName: "S1", Title: "Original title", PageURL: "https://www.notion.so/" + syncPageID, PublicURL: syncString("https://public.notion.site/" + syncPageID), DesiredState: model.ArticleStatusPublished, MetadataComplete: true, ExpectedConfigRevision: 1}
	first := syncApply(t, db, b, r)
	if err = db.Exec("UPDATE article SET subsection_code = NULL, content = NULL, pos = 29 WHERE id = ?", first.ArticleID).Error; err != nil {
		t.Fatal("legacy NULL fixture setup failed")
	}
	before := syncArticle(t, db, first.ArticleID)
	assertNullLocalFields := func() {
		t.Helper()
		requireRawSubsection(t, db, first.ArticleID, sql.NullString{})
		var content sql.NullString
		if err := db.Raw("SELECT content FROM article WHERE id = ?", first.ArticleID).Row().Scan(&content); err != nil {
			t.Fatal("projection physical content readback failed")
		}
		if content.Valid {
			t.Fatal("projection rewrote local NULL content")
		}
		a := syncArticle(t, db, first.ArticleID)
		if a.ID != before.ID || a.SectionCode != before.SectionCode || a.Pos != 29 || a.Author != before.Author || a.ExternalLink != before.ExternalLink || !a.CreatedAt.Equal(before.CreatedAt) {
			t.Fatal("projection changed preserved local fields or placement")
		}
	}
	assertNullLocalFields()
	r.Title = "Updated title"
	r.Tags = []string{"comma,tag", "second"}
	syncApply(t, db, b, r)
	assertNullLocalFields()
	updated := syncArticle(t, db, first.ArticleID)
	if updated.Title != r.Title || updated.Status != model.ArticleStatusPublished {
		t.Fatal("same-group metadata was not applied")
	}
	tags, err := model.ArticleTags(&updated)
	if err != nil || len(tags) != 2 || tags[0] != "comma,tag" {
		t.Fatal("same-group tags were not applied")
	}
	for _, state := range []int{model.ArticleStatusDraft, model.ArticleStatusUnpublished} {
		withdrawn := r
		withdrawn.MetadataComplete = false
		withdrawn.MetadataError = "invalid_field"
		withdrawn.DesiredState = state
		out := syncApply(t, db, b, withdrawn)
		if out.Outcome != "state_only" || out.AppliedState != state {
			t.Fatal("state-only withdrawal not applied")
		}
		assertNullLocalFields()
	}
	syncApply(t, db, b, r) // Fresh complete metadata restores publication without normalizing placement.
	assertNullLocalFields()
	beforeNoop := syncArticle(t, db, first.ArticleID)
	out := syncApply(t, db, b, r)
	afterNoop := syncArticle(t, db, first.ArticleID)
	if out.Outcome != "unchanged" || !afterNoop.UpdatedAt.Equal(beforeNoop.UpdatedAt) {
		t.Fatal("no-op changed MySQL update timestamp")
	}
	r.SourceID = "b"
	r.DataSourceID = syncDataSourceB
	r.ThemeOptionID = "o2"
	r.ThemeOptionName = "S2"
	out = syncApply(t, db, b, r)
	requireRawSubsection(t, db, out.ArticleID, sql.NullString{Valid: true})
	moved := syncArticle(t, db, out.ArticleID)
	if moved.ID != before.ID || moved.SectionCode != "s2" || moved.Pos != 1 || moved.Author != before.Author || moved.ExternalLink != before.ExternalLink {
		t.Fatal("real move did not preserve identity and append placement")
	}
	var content sql.NullString
	if err := db.Raw("SELECT content FROM article WHERE id = ?", out.ArticleID).Row().Scan(&content); err != nil || content.Valid {
		t.Fatal("real move rewrote local NULL content")
	}
}
