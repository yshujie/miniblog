package store

import (
	"context"
	"errors"
	"github.com/yshujie/miniblog/internal/miniblog/model"
	"testing"
	"time"
)

func TestLeaseEpochFencesExpiredOwner(t *testing.T) {
	db := storeTestDB(t)
	if e := db.AutoMigrate(&model.NotionSyncControl{}); e != nil {
		t.Fatal(e)
	}
	if e := db.Create(&model.NotionSyncControl{ID: 1, Paused: true}).Error; e != nil {
		t.Fatal(e)
	}
	repo := NewNotionSyncRepository(db)
	ctx := context.Background()
	a, ok, e := repo.AcquireLease(ctx, "a", time.Minute)
	if e != nil || !ok || a.Epoch != 1 {
		t.Fatal(a, ok, e)
	}
	_, ok, e = repo.AcquireLease(ctx, "b", time.Minute)
	if e != nil || ok {
		t.Fatal(ok, e)
	}
	expired := time.Now().UTC().Add(-time.Hour)
	db.Model(&model.NotionSyncControl{}).Where("id=1").Update("lease_until", expired)
	ok, e = repo.RenewLease(ctx, a, time.Minute)
	if e != nil || ok {
		t.Fatal("expired owner renewed", ok, e)
	}
	b, ok, e := repo.AcquireLease(ctx, "b", time.Minute)
	if e != nil || !ok || b.Epoch != 2 {
		t.Fatal(b, ok, e)
	}
	e = InTransaction(ctx, NewStore(db), func(ds IStore) error { _, e := LockLease(ds, a); return e })
	if !errors.Is(e, ErrLeaseLost) {
		t.Fatal(e)
	}
	if e = repo.ReleaseLease(ctx, a); e != nil {
		t.Fatal(e)
	}
	c, e := repo.Control(ctx)
	if e != nil || c.LeaseOwner != "b" || c.LeaseEpoch != 2 {
		t.Fatal(c, e)
	}
	ok, e = repo.RenewLease(ctx, b, time.Minute)
	if e != nil || !ok {
		t.Fatal(ok, e)
	}
}
func TestAliasLookupRejectsCrossTableCollision(t *testing.T) {
	db := storeTestDB(t)
	if e := db.AutoMigrate(&model.Article{}, &model.NotionSyncControl{}, &model.NotionPageBinding{}); e != nil {
		t.Fatal(e)
	}
	canonical, legacy := "canonical", "legacy"
	a := &model.Article{ID: 1, SourceKey: &canonical}
	db.Create(a)
	db.Create(&model.NotionPageBinding{PageID: "page", ArticleID: &a.ID, LegacySourceKey: &legacy})
	owner, e := FindSourceOwner(NewStore(db), legacy)
	if e != nil || owner.ID != 1 {
		t.Fatal(owner, e)
	}
	db.Create(&model.Article{ID: 2, SourceKey: &legacy})
	if _, e = FindSourceOwner(NewStore(db), legacy); !errors.Is(e, ErrSourceIdentityConflict) {
		t.Fatal(e)
	}
	if e = AuditSourceAliases(context.Background(), db); !errors.Is(e, ErrSourceIdentityConflict) {
		t.Fatal(e)
	}
}
