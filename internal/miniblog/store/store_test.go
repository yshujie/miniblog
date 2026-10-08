package store

import (
	"context"
	"errors"
	"testing"

	"github.com/yshujie/miniblog/internal/miniblog/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func storeTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Module{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestNewStoreDoesNotReplaceGlobalOrReuseOtherConnection(t *testing.T) {
	global := S
	a, b := NewStore(storeTestDB(t)), NewStore(storeTestDB(t))
	if a == b || a.DB() == b.DB() {
		t.Fatal("constructors reused a connection")
	}
	if S != global {
		t.Fatal("constructor changed the global store")
	}
	if err := a.Modules().Create(&model.Module{Code: "only-a"}); err != nil {
		t.Fatal(err)
	}
	got, err := b.Modules().GetByCode("only-a")
	if err != nil || got != nil {
		t.Fatalf("stores are not isolated: got=%v err=%v", got, err)
	}
}

func TestInTransactionBindsRepositoriesAndRollsBack(t *testing.T) {
	ds := NewStore(storeTestDB(t))
	stop := errors.New("rollback")
	err := InTransaction(context.Background(), ds, func(tx IStore) error {
		if tx.DB() == ds.DB() {
			t.Fatal("repository still uses the root DB")
		}
		if err := tx.Modules().Create(&model.Module{Code: "rolled-back"}); err != nil {
			return err
		}
		inside, err := tx.Modules().GetByCode("rolled-back")
		if err != nil || inside == nil {
			t.Fatalf("transaction cannot read its write: %v", err)
		}
		return stop
	})
	if !errors.Is(err, stop) {
		t.Fatalf("got %v", err)
	}
	got, err := ds.Modules().GetByCode("rolled-back")
	if err != nil || got != nil {
		t.Fatalf("transaction write escaped: got=%v err=%v", got, err)
	}
}
