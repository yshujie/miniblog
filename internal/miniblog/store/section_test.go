package store

import (
	"testing"

	"github.com/yshujie/miniblog/internal/miniblog/model"
	gormsqlite "gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupSectionTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(gormsqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite database: %v", err)
	}
	if err := db.AutoMigrate(&model.Section{}); err != nil {
		t.Fatalf("failed to migrate section model: %v", err)
	}
	return db
}

func TestSectionsStore_Getters(t *testing.T) {
	db := setupSectionTestDB(t)
	store := newSections(db)

	sections := []*model.Section{
		{Code: "s1", Title: "Section 1", ModuleCode: "moduleA", Sort: 1, Status: model.SectionStatusNormal},
		{Code: "s2", Title: "Section 2", ModuleCode: "moduleA", Sort: 2, Status: model.SectionStatusDeleted},
		{Code: "s3", Title: "Section 3", ModuleCode: "moduleB", Sort: 1, Status: model.SectionStatusNormal},
	}
	for _, s := range sections {
		if err := store.Create(s); err != nil {
			t.Fatalf("failed to create section %s: %v", s.Code, err)
		}
	}

	allModuleA, err := store.GetSections("moduleA")
	if err != nil {
		t.Fatalf("GetSections returned error: %v", err)
	}
	if len(allModuleA) != 2 {
		t.Fatalf("expected 2 sections for moduleA, got %d", len(allModuleA))
	}
	// Ensure ordering by sort asc
	if allModuleA[0].Code != "s1" || allModuleA[1].Code != "s2" {
		t.Fatalf("unexpected ordering: got %s then %s", allModuleA[0].Code, allModuleA[1].Code)
	}

	normalModuleA, err := store.GetNormalSections("moduleA")
	if err != nil {
		t.Fatalf("GetNormalSections returned error: %v", err)
	}
	if len(normalModuleA) != 1 {
		t.Fatalf("expected 1 normal section for moduleA, got %d", len(normalModuleA))
	}
	if normalModuleA[0].Code != "s1" || normalModuleA[0].Status != model.SectionStatusNormal {
		t.Fatalf("unexpected normal section: %+v", normalModuleA[0])
	}

	normalModuleB, err := store.GetNormalSections("moduleB")
	if err != nil {
		t.Fatalf("GetNormalSections returned error for moduleB: %v", err)
	}
	if len(normalModuleB) != 1 {
		t.Fatalf("expected 1 normal section for moduleB, got %d", len(normalModuleB))
	}
	if normalModuleB[0].Code != "s3" {
		t.Fatalf("unexpected section for moduleB: %+v", normalModuleB[0])
	}
}

func TestSectionsStore_DeleteByCode(t *testing.T) {
	db := setupSectionTestDB(t)
	store := newSections(db)

	s := &model.Section{Code: "dels", Title: "ToDelete", ModuleCode: "moduleX", Sort: 1, Status: model.SectionStatusNormal}
	if err := store.Create(s); err != nil {
		t.Fatalf("failed to create section: %v", err)
	}

	if err := store.DeleteByCode("dels"); err != nil {
		t.Fatalf("DeleteByCode returned error: %v", err)
	}

	got, err := store.GetByCode("dels")
	if err != nil {
		t.Fatalf("GetByCode after delete returned error: %v", err)
	}
	if got != nil {
		t.Fatalf("expected section to be deleted, but found: %+v", got)
	}
}
