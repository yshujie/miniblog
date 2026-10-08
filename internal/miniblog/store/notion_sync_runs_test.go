package store

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/yshujie/miniblog/internal/miniblog/model"
)

func TestRunningRunRegistrationBorrowsAtomicTransaction(t *testing.T) {
	for _, failure := range []string{"caller rollback", "insert failure"} {
		t.Run(failure, func(t *testing.T) {
			db := storeTestDB(t)
			if err := db.AutoMigrate(&model.NotionSyncControl{}, &model.NotionSyncRun{}); err != nil {
				t.Fatal(err)
			}
			old := model.NotionSyncRun{ID: "old-run", Status: "running", Phase: "schema", StartedAt: time.Now().UTC()}
			if err := db.Create(&old).Error; err != nil {
				t.Fatal(err)
			}
			if err := db.Create(&model.NotionSyncControl{ID: 1, CurrentRunID: old.ID}).Error; err != nil {
				t.Fatal(err)
			}
			if failure == "insert failure" {
				if err := db.Exec("CREATE TRIGGER reject_new_run BEFORE INSERT ON notion_sync_runs WHEN NEW.run_id = 'new-run' BEGIN SELECT RAISE(ABORT, 'fixture insert failed'); END").Error; err != nil {
					t.Fatal(err)
				}
			}
			rollback := errors.New("fixture caller rollback")
			err := InTransaction(context.Background(), NewStore(db), func(ds IStore) error {
				run := model.NotionSyncRun{ID: "new-run", Status: "running", StartedAt: time.Now().UTC()}
				if err := NewNotionSyncRepository(ds.DB()).RegisterRunningRun(context.Background(), &run, old.ID); err != nil {
					return err
				}
				return rollback
			})
			if err == nil {
				t.Fatal("failure did not propagate")
			}
			if failure == "caller rollback" && !errors.Is(err, rollback) {
				t.Fatal(err)
			}
			var after model.NotionSyncRun
			if err := db.First(&after, "run_id = ?", old.ID).Error; err != nil {
				t.Fatal(err)
			}
			control, err := NewNotionSyncRepository(db).Control(context.Background())
			var count int64
			if err := db.Model(&model.NotionSyncRun{}).Where("run_id = ?", "new-run").Count(&count).Error; err != nil {
				t.Fatal(err)
			}
			if err != nil || after.Status != "running" || after.Phase != "schema" || after.FinishedAt != nil || control.CurrentRunID != old.ID || count != 0 {
				t.Fatal(after, control, count, err)
			}
		})
	}
}
