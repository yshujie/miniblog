package store

import (
	"context"

	"github.com/yshujie/miniblog/internal/miniblog/model"
)

// RegisterRunningRun borrows the fenced transaction that already owns the
// control row. Only scan runs pass a previous run to mark as abandoned.
func (r *NotionSyncRepository) RegisterRunningRun(ctx context.Context, run *model.NotionSyncRun, previousRunID string) error {
	db := r.db.WithContext(ctx)
	if previousRunID != "" {
		if err := db.Model(&model.NotionSyncRun{}).Where("run_id = ? AND status = ?", previousRunID, "running").Updates(map[string]interface{}{
			"status": "abandoned", "phase": "finished", "finished_at": run.StartedAt, "error": "previous worker lease expired",
		}).Error; err != nil {
			return err
		}
	}
	if err := db.Create(run).Error; err != nil {
		return err
	}
	return db.Model(&model.NotionSyncControl{ID: 1}).Update("current_run_id", run.ID).Error
}
