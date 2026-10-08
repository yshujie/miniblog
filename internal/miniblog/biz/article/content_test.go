package article

import (
	"context"
	"errors"
	"fmt"
	"github.com/yshujie/miniblog/internal/miniblog/biz/blog"
	"github.com/yshujie/miniblog/internal/miniblog/biz/catalog"
	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/source"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"github.com/yshujie/miniblog/internal/pkg/errno"
	v1 "github.com/yshujie/miniblog/pkg/api/miniblog/v1"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"strconv"
	"testing"
)

func contentDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sql, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sql.SetMaxOpenConns(1)
	t.Cleanup(func() { sql.Close() })
	if err = db.AutoMigrate(&model.Module{}, &model.Section{}, &model.Subsection{}, &model.Article{}); err != nil {
		t.Fatal(err)
	}
	for _, m := range []*model.Module{{Code: "m1", Title: "M1", Status: 1}, {Code: "m2", Title: "M2", Status: 1}} {
		if err = db.Create(m).Error; err != nil {
			t.Fatal(err)
		}
	}
	for _, s := range []*model.Section{{Code: "s1", Title: "S1", ModuleCode: "m1", Status: 1}, {Code: "s2", Title: "S2", ModuleCode: "m2", Status: 1}} {
		if err = db.Create(s).Error; err != nil {
			t.Fatal(err)
		}
	}
	for _, s := range []*model.Subsection{{Code: "sub1", Title: "Sub1", SectionCode: "s1", Status: 1}, {Code: "sub2", Title: "Sub2", SectionCode: "s2", Status: 1}} {
		if err = db.Create(s).Error; err != nil {
			t.Fatal(err)
		}
	}
	return db
}
func register(t *testing.T, b *articleBiz, url, section, sub string, publish bool) *v1.RegisterArticleResponse {
	t.Helper()
	r, err := b.Register(context.Background(), &v1.RegisterArticleRequest{Title: "Title " + url, ExternalLink: url, SectionCode: section, SubsectionCode: sub, Publish: publish})
	if err != nil {
		t.Fatal(err)
	}
	return r
}
func errorHTTP(t *testing.T, err error, status int) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error")
	}
	http, _, _ := errno.Decode(err)
	if http != status {
		t.Fatalf("want HTTP %d got %d: %v", status, http, err)
	}
}

func TestRegisterDuplicatePreservesPlacementAndState(t *testing.T) {
	db := contentDB(t)
	b := NewWithNotionClient(store.NewStore(db), nil)
	ctx := context.Background()
	first := register(t, b, "https://www.notion.so/Title-aabbccddeeff00112233445566778899?pvs=4", "s1", "sub1", true)
	id, _ := ParseID(first.Article.ID)
	if first.Outcome != "created" || first.Article.Status != "Published" {
		t.Fatal(first)
	}
	if _, err := b.Archive(ctx, id); err != nil {
		t.Fatal(err)
	}
	duplicate, err := b.Register(ctx, &v1.RegisterArticleRequest{ExternalLink: "https://space.notion.site/aabbccdd-eeff-0011-2233-445566778899", Title: "Overwritten", SectionCode: "s2", Publish: true})
	if err != nil || duplicate.Outcome != "already_registered" || duplicate.Article.ID != first.Article.ID || duplicate.Article.Status != "Deleted" || duplicate.Article.Section.Code != "s1" || duplicate.Article.Title == "Overwritten" {
		t.Fatalf("%+v %v", duplicate, err)
	}
	var count int64
	db.Model(&model.Article{}).Count(&count)
	if count != 1 {
		t.Fatal(count)
	}
	errorHTTP(t, b.Publish(ctx, id), 409)
	restored, err := b.Restore(ctx, id)
	if err != nil || restored.Article.Status != "Draft" {
		t.Fatalf("%+v %v", restored, err)
	}
}
func TestRegisterIsAtomicAndPublishedRequiresVisiblePath(t *testing.T) {
	db := contentDB(t)
	b := NewWithNotionClient(store.NewStore(db), nil)
	db.Model(&model.Module{}).Where("code = ?", "m1").Update("status", 2)
	_, err := b.Register(context.Background(), &v1.RegisterArticleRequest{ExternalLink: "https://example.com/hidden", Title: "Hidden", SectionCode: "s1", Publish: true})
	errorHTTP(t, err, 400)
	var count int64
	db.Model(&model.Article{}).Count(&count)
	if count != 0 {
		t.Fatal("failed publication persisted")
	}
	stop := errors.New("after insert")
	db.Callback().Create().After("gorm:create").Register("test:rollback", func(tx *gorm.DB) {
		if tx.Statement.Table == "article" {
			tx.AddError(stop)
		}
	})
	_, err = b.Register(context.Background(), &v1.RegisterArticleRequest{ExternalLink: "https://example.com/atomic", Title: "Atomic", SectionCode: "s2", Publish: true})
	if !errors.Is(err, stop) {
		t.Fatal(err)
	}
	db.Callback().Create().Remove("test:rollback")
	db.Model(&model.Article{}).Count(&count)
	if count != 0 {
		t.Fatal("failed create escaped transaction")
	}
}
func TestMetadataEditMoveAndPublicVisibility(t *testing.T) {
	db := contentDB(t)
	ds := store.NewStore(db)
	b := NewWithNotionClient(ds, nil)
	ctx := context.Background()
	first := register(t, b, "https://example.com/one", "s1", "", true)
	id, _ := ParseID(first.Article.ID)
	db.Model(&model.Article{}).Where("id = ?", id).Update("content", "legacy body")
	edited, err := b.Update(ctx, &v1.UpdateArticleRequest{ID: first.Article.ID, Title: "Edited", ExternalLink: first.Article.ExternalLink, SectionCode: "s1", ModuleCode: "m1"})
	if err != nil || edited.Article.Status != "Published" || edited.Article.Content != "legacy body" {
		t.Fatalf("%+v %v", edited, err)
	}
	_, err = b.Move(ctx, id, &v1.MoveArticleRequest{SectionCode: "s1", SubsectionCode: "sub2"})
	errorHTTP(t, err, 400)
	db.Model(&model.Module{}).Where("code = ?", "m2").Update("status", 2)
	_, err = b.Move(ctx, id, &v1.MoveArticleRequest{SectionCode: "s2"})
	errorHTTP(t, err, 400)
	read := blog.New(ds)
	detail, err := read.GetArticleDetail(ctx, &v1.GetArticleDetailRequest{ArticleID: id})
	if err != nil || detail.ArticleDetail.ModuleCode != "m1" {
		t.Fatalf("%+v %v", detail, err)
	}
	for _, target := range []struct{ table, code string }{{"module", "m1"}, {"section", "s1"}} {
		db.Table(target.table).Where("code = ?", target.code).Update("status", 2)
		_, err = read.GetArticleDetail(ctx, &v1.GetArticleDetailRequest{ArticleID: id})
		errorHTTP(t, err, 404)
		db.Table(target.table).Where("code = ?", target.code).Update("status", 1)
	}
	if err = b.Unpublish(ctx, id); err != nil {
		t.Fatal(err)
	}
	_, err = read.GetArticleDetail(ctx, &v1.GetArticleDetailRequest{ArticleID: id})
	errorHTTP(t, err, 404)
	moved, err := b.Move(ctx, id, &v1.MoveArticleRequest{SectionCode: "s2"})
	if err != nil || moved.Article.Status != "Unpublished" || moved.Article.Section.Code != "s2" || moved.Article.Pos != 1 {
		t.Fatalf("%+v %v", moved, err)
	}
}
func TestReorderCountAndDirectoryDependencies(t *testing.T) {
	db := contentDB(t)
	ds := store.NewStore(db)
	b := NewWithNotionClient(ds, nil)
	ctx := context.Background()
	a := register(t, b, "https://example.com/a", "s1", "", false)
	c := register(t, b, "https://example.com/c", "s1", "", false)
	archived := register(t, b, "https://example.com/archived", "s1", "", false)
	aid, _ := ParseID(archived.Article.ID)
	b.Archive(ctx, aid)
	register(t, b, "https://example.com/sub", "s1", "sub1", false)
	errorHTTP(t, b.Reorder(ctx, &v1.ReorderArticlesRequest{SectionCode: "s1", ArticleIDs: []string{a.Article.ID}}), 409)
	if err := b.Reorder(ctx, &v1.ReorderArticlesRequest{SectionCode: "s1", ArticleIDs: []string{c.Article.ID, a.Article.ID}}); err != nil {
		t.Fatal(err)
	}
	list, err := b.GetList(ctx, &v1.ArticleListRequest{ModuleCode: "m1", SectionCode: "s1", DirectOnly: true, Status: "Draft", Limit: 1, Page: 2})
	if err != nil || list.Total != 2 || len(list.Articles) != 1 || list.Articles[0].IDText != a.Article.ID {
		t.Fatalf("%+v %v", list, err)
	}
	list, err = b.GetList(ctx, &v1.ArticleListRequest{Title: "/sub"})
	if err != nil || list.Total != 1 {
		t.Fatalf("%+v %v", list, err)
	}
	cats := catalog.New(ds)
	errorHTTP(t, cats.DeleteSection(ctx, "s1"), 400)
	errorHTTP(t, cats.DeleteSubsection(ctx, "sub1"), 400)
	errorHTTP(t, cats.DeleteModule(ctx, "m1"), 400)
	errorHTTP(t, cats.Reorder(ctx, catalog.ReorderInput{Kind: "module", Codes: []string{"m1"}}), 400)
	if err = cats.Reorder(ctx, catalog.ReorderInput{Kind: "module", Codes: []string{"m2", "m1"}}); err != nil {
		t.Fatal(err)
	}
	modules, err := ds.Modules().GetAll()
	if err != nil || modules[0].Code != "m2" {
		t.Fatal(modules, err)
	}
}
func TestImportPreservesExplicitIDAndDryRun(t *testing.T) {
	db := contentDB(t)
	b := NewWithNotionClient(store.NewStore(db), nil)
	ctx := context.Background()
	const id uint64 = 9007199254740993
	r := ImportRequest{ID: id, Title: "Legacy", Content: importString("Body"), SectionCode: "s1", Status: importInt(2)}
	dry, err := b.Import(ctx, r, true)
	if err != nil || dry.ID != id {
		t.Fatalf("%+v %v", dry, err)
	}
	var count int64
	db.Model(&model.Article{}).Count(&count)
	if count != 0 {
		t.Fatal("dry run wrote data")
	}
	result, err := b.Import(ctx, r, false)
	if err != nil || result.ID != id {
		t.Fatalf("%+v %v", result, err)
	}
	var a model.Article
	db.First(&a, id)
	if a.ID != id || a.SourceKey != nil || a.Provider != nil {
		t.Fatal(a)
	}
	edited, err := b.Update(ctx, &v1.UpdateArticleRequest{ID: strconv.FormatUint(id, 10), Title: "Metadata", SectionCode: "s1"})
	if err != nil || edited.Article.Content != "Body" || edited.Article.Status != "Published" {
		t.Fatalf("%+v %v", edited, err)
	}
	r.Title = "Should not write"
	b.Import(ctx, r, true)
	db.First(&a, id)
	if a.Title != "Metadata" {
		t.Fatal(a.Title)
	}
}

type previewClient struct {
	calls int
	err   error
}

func (f *previewClient) GetTitle(context.Context, string) (string, error) {
	f.calls++
	return "Notion title", f.err
}
func TestPreviewFailureDoesNotBlockRegister(t *testing.T) {
	db := contentDB(t)
	client := &previewClient{err: &source.MetadataError{Reason: "notion_not_shared"}}
	b := NewWithNotionClient(store.NewStore(db), client)
	url := "https://notion.so/aabbccddeeff00112233445566778899"
	out, err := b.Preview(context.Background(), &v1.PreviewSourceRequest{ExternalLink: url})
	if err != nil || out.MetadataStatus != "manual_required" || out.Reason != "notion_not_shared" || client.calls != 1 {
		t.Fatalf("%+v %v", out, err)
	}
	register(t, b, url, "s1", "", true)
	client.err = nil
	out, err = b.Preview(context.Background(), &v1.PreviewSourceRequest{ExternalLink: url})
	if err != nil || out.ExistingArticle == nil || out.Title != "Notion title" || out.MetadataStatus != "resolved" {
		t.Fatalf("%+v %v", out, err)
	}
	b.Preview(context.Background(), &v1.PreviewSourceRequest{ExternalLink: "https://example.com/manual"})
	if client.calls != 2 {
		t.Fatal("arbitrary URL was fetched")
	}
}
func TestPublicModuleIncludesAllArticlesAndDirectGroupOnce(t *testing.T) {
	db := contentDB(t)
	ds := store.NewStore(db)
	b := NewWithNotionClient(ds, nil)
	ctx := context.Background()
	for i := 0; i < 105; i++ {
		_, err := b.Import(ctx, ImportRequest{ID: uint64(i + 1), Title: fmt.Sprint("A", i), ExternalLink: fmt.Sprintf("https://example.com/%d", i), SectionCode: "s1", Status: importInt(2)}, false)
		if err != nil {
			t.Fatal(err)
		}
	}
	register(t, b, "https://example.com/sub", "s1", "sub1", true)
	db.Model(&model.Subsection{}).Where("code = ?", "sub1").Update("status", 2)
	queries := 0
	db.Callback().Query().After("gorm:query").Register("test:reading_queries", func(*gorm.DB) { queries++ })
	out, err := blog.New(ds).GetModuleDetail(ctx, &v1.GetModuleDetailRequest{ModuleCode: "m1"})
	if err != nil || len(out.ModuleDetail.Sections) != 1 || len(out.ModuleDetail.Sections[0].Articles) != 105 || len(out.ModuleDetail.Sections[0].Subsections) != 0 {
		t.Fatalf("%+v %v", out, err)
	}
	if queries != 4 {
		t.Fatalf("module used %d queries for 106 articles, want 4", queries)
	}
}

func TestImportOmissionAndArchiveRequireExplicitRestore(t *testing.T) {
	db := contentDB(t)
	b := NewWithNotionClient(store.NewStore(db), nil)
	ctx := context.Background()
	r := ImportRequest{ID: 42, Title: "Original", ExternalLink: "https://example.com/import", Content: importString("Body"), SectionCode: "s1", Status: importInt(2)}
	if _, err := b.Import(ctx, r, false); err != nil {
		t.Fatal(err)
	}
	r.Content = nil
	r.Status = nil
	r.Title = "Metadata"
	if _, err := b.Import(ctx, r, false); err != nil {
		t.Fatal(err)
	}
	var a model.Article
	db.First(&a, 42)
	if a.Content != "Body" || a.Status != 2 {
		t.Fatal(a)
	}
	r.Content = importString("")
	if _, err := b.Import(ctx, r, false); err != nil {
		t.Fatal(err)
	}
	db.First(&a, 42)
	if a.Content != "" {
		t.Fatal("explicit empty content did not clear")
	}
	if _, err := b.Archive(ctx, 42); err != nil {
		t.Fatal(err)
	}
	r.Content = nil
	r.Status = nil
	duplicate, err := b.Import(ctx, r, false)
	if err != nil || duplicate.Outcome != "already_registered" {
		t.Fatal(duplicate, err)
	}
	r.Title = "Try edit"
	_, err = b.Import(ctx, r, false)
	errorHTTP(t, err, 409)
	r.Title = "Metadata"
	r.Status = importInt(1)
	_, err = b.Import(ctx, r, false)
	errorHTTP(t, err, 409)
	r.Status = importInt(2)
	_, err = b.Import(ctx, r, false)
	errorHTTP(t, err, 409)
	if _, err = b.Restore(ctx, 42); err != nil {
		t.Fatal(err)
	}
	r.Status = nil
	r.Title = "Restored edit"
	if _, err = b.Import(ctx, r, false); err != nil {
		t.Fatal(err)
	}
	db.First(&a, 42)
	if a.Status != 1 || a.Title != "Restored edit" {
		t.Fatal(a)
	}
	other := ImportRequest{ID: 43, Title: "Other ID", ExternalLink: r.ExternalLink, SectionCode: "s2"}
	_, err = b.Import(ctx, other, false)
	errorHTTP(t, err, 409)
}
func TestImportWithoutIDNeverUpdatesExistingSource(t *testing.T) {
	db := contentDB(t)
	b := NewWithNotionClient(store.NewStore(db), nil)
	ctx := context.Background()
	first := register(t, b, "https://example.com/duplicate-import", "s1", "sub1", true)
	id, _ := ParseID(first.Article.ID)
	db.Model(&model.Article{}).Where("id = ?", id).Update("content", "Preserve")
	r := ImportRequest{Title: "Different", ExternalLink: first.Article.ExternalLink, SectionCode: "s2", Content: importString("Replace"), Status: importInt(1)}
	result, err := b.Import(ctx, r, false)
	if err != nil || result.Outcome != "already_registered" || result.ID != id {
		t.Fatal(result, err)
	}
	var a model.Article
	db.First(&a, id)
	if a.Title != first.Article.Title || a.Content != "Preserve" || a.Status != 2 || a.SectionCode != "s1" || a.SubsectionCode != "sub1" {
		t.Fatal(a)
	}
	b.Archive(ctx, id)
	r.Status = importInt(2)
	r.SectionCode = "nonexistent"
	result, err = b.Import(ctx, r, false)
	if err != nil || result.Outcome != "already_registered" {
		t.Fatal(result, err)
	}
	db.First(&a, id)
	if a.Status != 4 {
		t.Fatal("duplicate import restored archive")
	}
}

func importInt(v int) *int          { return &v }
func importString(v string) *string { return &v }
