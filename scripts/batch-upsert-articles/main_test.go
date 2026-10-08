package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/yshujie/miniblog/internal/miniblog/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestImportJSONIDsAndContentPresence(t *testing.T) {
	for _, input := range []string{`{"id":"9007199254740993","title":"a"}`, `{"id":9007199254740993,"title":"a"}`} {
		var item articleInput
		if err := json.Unmarshal([]byte(input), &item); err != nil || item.ID != 9007199254740993 || item.Content != nil {
			t.Fatalf("%+v %v", item, err)
		}
	}
	for _, input := range []string{`{"id":-1}`, `{"id":"9223372036854775808"}`, `{"id":1e3}`} {
		var item articleInput
		if json.Unmarshal([]byte(input), &item) == nil {
			t.Fatalf("accepted %s", input)
		}
	}
	var item articleInput
	if err := json.Unmarshal([]byte(`{"content":""}`), &item); err != nil || item.Content == nil || *item.Content != "" {
		t.Fatal(item, err)
	}
}

func TestCLIUsesIdentityAndPreservesHistory(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sql, _ := db.DB()
	defer sql.Close()
	if err = db.AutoMigrate(&model.Module{}, &model.Section{}, &model.Subsection{}, &model.Article{}); err != nil {
		t.Fatal(err)
	}
	db.Create(&model.Module{Code: "m", Title: "Module", Status: 1})
	db.Create(&model.Section{Code: "s", ModuleCode: "m", Title: "Section", Status: 1})
	body := "Historical body"
	item := articleInput{ID: 9007199254740993, Title: "Legacy", SectionCode: "s", ExternalLink: "https://example.com/legacy", Content: &body, Status: "published"}
	action, id, err := upsertArticle(db, t.TempDir(), item, true)
	if err != nil || action != "dry-run-create" || id != item.ID {
		t.Fatal(action, id, err)
	}
	var count int64
	db.Model(&model.Article{}).Count(&count)
	if count != 0 {
		t.Fatal("dry-run wrote")
	}
	action, id, err = upsertArticle(db, "", item, false)
	if err != nil || action != "create" || id != item.ID {
		t.Fatal(action, id, err)
	}
	item.Content = nil
	item.Status = ""
	item.Title = "Edited"
	if _, _, err = upsertArticle(db, "", item, false); err != nil {
		t.Fatal(err)
	}
	var row model.Article
	db.First(&row, item.ID)
	if row.Content != body || row.Status != 2 || row.ID != item.ID {
		t.Fatalf("%+v", row)
	}
	item.ID = 0
	item.ExternalLink = "https://example.com/different"
	item.Title = "Edited"
	if action, _, err = upsertArticle(db, "", item, false); err != nil || action != "create" {
		t.Fatal(action, err)
	}
	db.Model(&model.Article{}).Count(&count)
	if count != 2 {
		t.Fatal("title treated as identity", count)
	}
	item.ExternalLink = "https://example.com/legacy"
	item.Title = "Must not overwrite"
	item.Status = "unpublished"
	if action, _, err = upsertArticle(db, "", item, false); err != nil || action != "already_registered" {
		t.Fatal(action, err)
	}
	db.First(&row, uint64(9007199254740993))
	if row.Title != "Edited" || row.Status != 2 || row.Content != body {
		t.Fatalf("duplicate changed history: %+v", row)
	}
}

func TestContentFileAndExternalOnly(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "body.md"), []byte("# body"), 0600); err != nil {
		t.Fatal(err)
	}
	content, err := resolveContent(dir, articleInput{ContentFile: "body.md"})
	if err != nil || content == nil || *content != "# body" {
		t.Fatal(content, err)
	}
	content, err = resolveContent(dir, articleInput{ExternalLink: "https://example.com/a"})
	if err != nil || content != nil {
		t.Fatal(content, err)
	}
	if _, err = resolveContent(dir, articleInput{}); err == nil {
		t.Fatal("empty content accepted")
	}
}
