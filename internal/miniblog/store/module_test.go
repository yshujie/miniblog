package store

import (
	"testing"

	"github.com/yshujie/miniblog/internal/miniblog/model"
	gormsqlite "gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupModuleTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(gormsqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite database: %v", err)
	}
	if err := db.AutoMigrate(&model.Module{}); err != nil {
		t.Fatalf("failed to migrate module model: %v", err)
	}
	return db
}

func TestModuleStore_GettersAndUpdate(t *testing.T) {
	db := setupModuleTestDB(t)
	store := newModules(db)

	modules := []*model.Module{
		{Code: "m1", Title: "Module 1", Status: model.ModuleStatusNormal},
		{Code: "m2", Title: "Module 2", Status: model.ModuleStatusDeleted},
	}
	for _, m := range modules {
		if err := store.Create(m); err != nil {
			t.Fatalf("failed to create module %s: %v", m.Code, err)
		}
	}

	all, err := store.GetAll()
	if err != nil {
		t.Fatalf("GetAll returned error: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 modules, got %d", len(all))
	}

	normal, err := store.GetNormalModules()
	if err != nil {
		t.Fatalf("GetNormalModules returned error: %v", err)
	}
	if len(normal) != 1 || normal[0].Code != "m1" {
		t.Fatalf("expected only module m1 as normal, got %+v", normal)
	}

	// Update title and ensure persistence
	m1, err := store.GetByCode("m1")
	if err != nil {
		t.Fatalf("GetByCode returned error: %v", err)
	}
	m1.Title = "Module 1 Updated"
	if err := store.Update(m1); err != nil {
		t.Fatalf("Update returned error: %v", err)
	}

	updated, err := store.GetByCode("m1")
	if err != nil {
		t.Fatalf("GetByCode after update returned error: %v", err)
	}
	if updated.Title != "Module 1 Updated" {
		t.Fatalf("expected updated title, got %s", updated.Title)
	}
}

func TestModuleStore_DeleteByCode(t *testing.T) {
	db := setupModuleTestDB(t)
	store := newModules(db)

	m := &model.Module{Code: "delm", Title: "ToDelete", Status: model.ModuleStatusNormal}
	if err := store.Create(m); err != nil {
		t.Fatalf("failed to create module: %v", err)
	}

	if err := store.DeleteByCode("delm"); err != nil {
		t.Fatalf("DeleteByCode returned error: %v", err)
	}

	got, err := store.GetByCode("delm")
	if err != nil {
		t.Fatalf("GetByCode after delete returned error: %v", err)
	}
	if got != nil {
		t.Fatalf("expected module to be deleted, but found: %+v", got)
	}
}
