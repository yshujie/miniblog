package store

import (
	"context"
	"errors"
	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/source"
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

func TestKnownURLsUseExactParsedIdentityAndRejectMultipleOwners(t *testing.T) {
	db := storeTestDB(t)
	if e := db.AutoMigrate(&model.Article{}, &model.NotionSyncControl{}, &model.NotionPageBinding{}); e != nil {
		t.Fatal(e)
	}
	canonical, _ := source.Parse("https://www.notion.so/aabbccddeeff00112233445566778899")
	a := model.Article{ID: 1, SourceKey: &canonical.SourceKey}
	if e := db.Create(&a).Error; e != nil {
		t.Fatal(e)
	}
	slug := "https://TEAM.notion.site:443/CaseSlug?filter=A#part"
	p := model.NotionPageBinding{PageID: "aabbccddeeff00112233445566778899", ArticleID: &a.ID, ManagementState: model.NotionManagementManaged, PublicURL: &slug}
	if e := db.Create(&p).Error; e != nil {
		t.Fatal(e)
	}
	ds := NewStore(db)
	exact, _ := source.Parse("https://team.notion.site/CaseSlug?filter=A#part")
	owner, e := FindSourceOwner(ds, exact.SourceKey)
	if e != nil || owner == nil || owner.ID != a.ID {
		t.Fatal(owner, e)
	}
	for _, raw := range []string{"https://team.notion.site/caseslug?filter=A#part", "https://team.notion.site/CaseSlug?filter=a#part", "https://team.notion.site/CaseSlug?filter=A#other", "https://team.notion.site/unknown-slug"} {
		other, _ := source.Parse(raw)
		owner, e = FindSourceOwner(ds, other.SourceKey)
		if e != nil || owner != nil {
			t.Fatal("guessed URL equivalence", raw, owner, e)
		}
	}
	// A direct generic owner alongside a known page's primary owner is a conflict.
	if e = db.Create(&model.Article{ID: 2, SourceKey: &exact.SourceKey}).Error; e != nil {
		t.Fatal(e)
	}
	if _, e = FindSourceOwner(ds, exact.SourceKey); !errors.Is(e, ErrSourceIdentityConflict) {
		t.Fatal(e)
	}
	if _, e = FindSourceOwnerForURLs(ds, "", canonical.CanonicalURL, slug); !errors.Is(e, ErrSourceIdentityConflict) {
		t.Fatal(e)
	}
	// Ignoring the current page must still find the manual generic owner.
	owner, e = FindSourceOwnerForURLs(ds, p.PageID, slug)
	if e != nil || owner == nil || owner.ID != 2 {
		t.Fatal(owner, e)
	}
}
