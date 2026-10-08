package article

import (
	"context"
	"errors"
	"github.com/yshujie/miniblog/internal/miniblog/model"
	"gorm.io/gorm"
	"reflect"
	"testing"
	"time"
)

func isolationFixture(t *testing.T) (*gorm.DB, *articleBiz, TargetIsolationInput, uint64) {
	t.Helper()
	db, b, r := syncFixture(t)
	stamp := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	r.NotionLastEditedAt = &stamp
	out := syncApply(t, db, b, r)
	p := syncBinding(t, db)
	db.Model(&model.NotionSyncSource{}).Where("source_id = ?", "b").Update("enabled", false)
	fresh := stamp.Add(time.Hour)
	return db, b, TargetIsolationInput{Lease: r.Lease, RunID: "isolation-run", PageID: r.PageID, TargetSourceID: "b", DataSourceID: syncDataSourceB, Reason: TargetDisabled, DesiredState: 2, NotionLastEditedAt: &fresh, ExpectedConfigRevision: 1, ExpectedBindingRevision: p.Revision}, out.ArticleID
}
func TestTargetIsolationFencesEveryWriter(t *testing.T) {
	tests := map[string]func(*gorm.DB, *TargetIsolationInput){
		"paused": func(db *gorm.DB, r *TargetIsolationInput) {
			db.Model(&model.NotionSyncControl{}).Where("id = 1").Update("paused", true)
		},
		"maintenance": func(db *gorm.DB, r *TargetIsolationInput) {
			db.Model(&model.NotionSyncControl{}).Where("id = 1").Update("source_writes_paused", true)
		},
		"lease":   func(db *gorm.DB, r *TargetIsolationInput) { r.Lease.Epoch++ },
		"config":  func(db *gorm.DB, r *TargetIsolationInput) { r.ExpectedConfigRevision++ },
		"binding": func(db *gorm.DB, r *TargetIsolationInput) { r.ExpectedBindingRevision++ },
		"old_snapshot": func(db *gorm.DB, r *TargetIsolationInput) {
			stamp := r.NotionLastEditedAt.Add(-2 * time.Hour)
			r.NotionLastEditedAt = &stamp
		},
		"target_enabled": func(db *gorm.DB, r *TargetIsolationInput) {
			db.Model(&model.NotionSyncSource{}).Where("source_id = ?", "b").Update("enabled", true)
		},
		"wrong_parent": func(db *gorm.DB, r *TargetIsolationInput) { r.DataSourceID = syncDataSourceA },
		"unmanaged": func(db *gorm.DB, r *TargetIsolationInput) {
			db.Model(&model.NotionPageBinding{}).Where("page_id = ?", r.PageID).Update("management_state", model.NotionManagementDetached)
		},
		"pending_no_article": func(db *gorm.DB, r *TargetIsolationInput) {
			db.Model(&model.NotionPageBinding{}).Where("page_id = ?", r.PageID).Update("article_id", nil)
		},
		"unknown_page": func(db *gorm.DB, r *TargetIsolationInput) { r.PageID = "ffeeddccbbaa00998877665544332211" },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			db, b, r, id := isolationFixture(t)
			mutate(db, &r)
			before := syncArticle(t, db, id)
			beforePage := syncBinding(t, db)
			_, e := b.IsolateSyncedTarget(context.Background(), r)
			if e == nil {
				t.Fatal("guard accepted", name)
			}
			after := syncArticle(t, db, id)
			afterPage := syncBinding(t, db)
			if !reflect.DeepEqual(before, after) || !reflect.DeepEqual(beforePage, afterPage) {
				t.Fatalf("rejected isolation wrote values: %v %v", after, afterPage)
			}
		})
	}
}
func TestTargetIsolationReceiptAndPrivacyWithdrawal(t *testing.T) {
	db, b, r, id := isolationFixture(t)
	before := syncArticle(t, db, id)
	beforePage := syncBinding(t, db)
	r.DesiredState = 1
	r.PublicURLWithdrawn = true
	result, e := b.IsolateSyncedTarget(context.Background(), r)
	if e != nil {
		t.Fatal(e)
	}
	after := syncArticle(t, db, id)
	p := syncBinding(t, db)
	want := before
	want.Status = 1
	if result.Outcome != "blocked" || after.Status != 1 || !sameArticleValues(&after, &want) || p.PublicURL != nil || p.PageURL != beforePage.PageURL || p.SourceID != beforePage.SourceID || p.MetadataHash != beforePage.MetadataHash || p.SnapshotJSON != beforePage.SnapshotJSON {
		t.Fatalf("lost withdrawal/history %+v %+v", after, p)
	}
	result, e = b.IsolateSyncedTarget(context.Background(), r) // Same receipt may retry its original revision.
	if e != nil || result.BindingRevision != p.Revision {
		t.Fatal(result, e)
	}
	changed := r
	changed.DesiredState = 3
	if _, e = b.IsolateSyncedTarget(context.Background(), changed); e == nil {
		t.Fatal("receipt accepted a different observation")
	}
	requirePublic(t, db, id, false)
}
func TestTargetIsolationArticleAndBindingRollbackTogether(t *testing.T) {
	db, b, r, id := isolationFixture(t)
	before := syncArticle(t, db, id)
	beforePage := syncBinding(t, db)
	fail := errors.New("binding persistence failed")
	db.Callback().Update().Before("gorm:update").Register("isolation_test_fail", func(tx *gorm.DB) {
		if tx.Statement.Table == "notion_page_bindings" {
			tx.AddError(fail)
		}
	})
	t.Cleanup(func() { db.Callback().Update().Remove("isolation_test_fail") })
	r.DesiredState = model.ArticleStatusDeleted
	if _, e := b.IsolateSyncedTarget(context.Background(), r); !errors.Is(e, fail) {
		t.Fatal(e)
	}
	if after := syncArticle(t, db, id); !reflect.DeepEqual(before, after) {
		t.Fatal("article state escaped rollback", after)
	}
	if after := syncBinding(t, db); !reflect.DeepEqual(beforePage, after) {
		t.Fatal("binding escaped rollback", after)
	}
}
func TestTargetIsolationDoesNotWeakenNormalApplyEnabledGate(t *testing.T) {
	db, b, r, _ := isolationFixture(t)
	normal := SyncInput{Lease: r.Lease, SourceID: "b", DataSourceID: syncDataSourceB, PageID: r.PageID, ExpectedConfigRevision: 1, ExpectedBindingRevision: syncBinding(t, db).Revision}
	if _, e := b.ApplySyncedSource(context.Background(), normal); e == nil {
		t.Fatal("disabled source accepted normal projection")
	}
}
