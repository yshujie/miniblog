package notionsync

import (
	"context"
	"time"

	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"gorm.io/gorm"
)

// runCoordinator shares durable run registration and execution while callers
// retain their distinct admission checks, busy-lease responses and time limits.
type runCoordinator struct{ service *Service }

type runCompletionPolicy struct {
	timeout          time.Duration
	blockedIsFailure bool
	scheduleNext     bool
	pruneResolved    bool
}

func (c runCoordinator) checkCooldown(db *gorm.DB, control *model.NotionSyncControl) error {
	now, err := store.DatabaseNow(db)
	if err != nil {
		return err
	}
	if control.CooldownUntil != nil && control.CooldownUntil.After(now) {
		return &Error{Code: "cooldown", Message: "Notion 限流冷却尚未结束", HTTPStatus: 429}
	}
	return nil
}

func (c runCoordinator) register(ctx context.Context, run *model.NotionSyncRun, token store.LeaseToken, abandonPrevious bool) error {
	return c.service.fenced(ctx, token, func(tx *gorm.DB, control *model.NotionSyncControl) error {
		previous := ""
		if abandonPrevious {
			previous = control.CurrentRunID
		}
		return store.NewNotionSyncRepository(tx).RegisterRunningRun(ctx, run, previous)
	})
}

func (c runCoordinator) execute(parent context.Context, run model.NotionSyncRun, token store.LeaseToken, policy runCompletionPolicy, fn func(context.Context, model.NotionSyncRun, store.LeaseToken, *RunCounts) error) error {
	var ctx context.Context
	var cancel context.CancelFunc
	if policy.timeout > 0 {
		ctx, cancel = context.WithTimeout(parent, policy.timeout)
	} else {
		ctx, cancel = context.WithCancel(parent)
	}
	ctx = context.WithValue(ctx, leaseContextKey{}, token)
	done := make(chan struct{})
	go c.service.heartbeat(ctx, token, cancel, done)
	counts := RunCounts{}
	runErr := fn(ctx, run, token, &counts)
	cancel()
	<-done

	finishctx, finishcancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer finishcancel()
	status := "completed"
	if runErr != nil {
		status = "failed"
	} else if counts.Failed > 0 || (policy.blockedIsFailure && counts.Blocked > 0) {
		status = "completed_with_errors"
	}
	updates := map[string]interface{}{"status": status, "phase": "finished", "finished_at": time.Now().UTC(), "counts_json": jsonText(counts)}
	if runErr != nil {
		updates["error"] = runErr.Error()
	}
	err := c.service.finishRun(finishctx, run, token, updates, policy.scheduleNext)
	if policy.pruneResolved {
		_ = c.service.pruneResolvedRuns(finishctx, token)
	}
	_ = store.NewNotionSyncRepository(c.service.ds.DB()).ReleaseLease(finishctx, token)
	if runErr != nil {
		return runErr
	}
	return err
}
