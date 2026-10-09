package notionsync

import (
	"context"
	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"gorm.io/gorm"
	"time"
)

type leaseContextKey struct{}

func (s *Service) fenced(ctx context.Context, token store.LeaseToken, fn func(*gorm.DB, *model.NotionSyncControl) error) error {
	return store.InTransaction(ctx, s.ds, func(ds store.IStore) error {
		c, e := store.LockLease(ds, token)
		if e != nil {
			return e
		}
		return fn(ds.DB(), c)
	})
}
func (s *Service) Trigger(ctx context.Context, in TriggerInput) (*TriggerResult, error) {
	if in.Mode != "dry_run" && in.Mode != "sync" {
		return nil, invalid("mode 只能为 dry_run 或 sync")
	}
	s.mu.Lock()
	parent := s.ctx
	s.mu.Unlock()
	runctx, cancel, id, e := s.beginTask(parent)
	if e != nil {
		return nil, e
	}
	handedOff := false
	defer func() {
		if !handedOff {
			s.endTask(id, cancel)
		}
	}()
	requestctx, requestcancel := context.WithCancel(ctx)
	unlink := context.AfterFunc(runctx, requestcancel)
	defer requestcancel()
	defer unlink()
	ctx = requestctx
	db, e := s.db(ctx)
	if e != nil {
		return nil, e
	}
	repo := store.NewNotionSyncRepository(db)
	c, e := repo.Control(ctx)
	if e != nil {
		return nil, unavailable("请先完成同步迁移并初始化控制行")
	}
	coordinator := runCoordinator{service: s}
	if e := coordinator.checkCooldown(db, c); e != nil {
		return nil, e
	}
	if in.Mode == "sync" {
		if !s.opts.Enabled {
			return nil, unavailable("同步尚未启用，请先开启运行配置")
		}
		if c.Paused || c.SourceWritesPaused {
			return nil, conflict("同步或外部来源写入已暂停")
		}
		if !c.BaselineFrozen {
			return nil, conflict("请先完成一次完整基线预览")
		}
	}
	token, ok, e := repo.AcquireLease(ctx, s.opts.OwnerID, s.opts.LeaseDuration)
	if e != nil {
		return nil, e
	}
	if !ok {
		active, e := repo.Control(ctx)
		if e != nil {
			return nil, e
		}
		if active.CurrentRunID != "" {
			return &TriggerResult{RunID: active.CurrentRunID}, nil
		}
		return nil, conflict("同步租约正在初始化，请稍后刷新")
	}
	now := time.Now().UTC()
	run := model.NotionSyncRun{ID: id, Mode: in.Mode, Status: "running", Phase: "schema", StartedAt: now, LeaseEpoch: token.Epoch, CountsJSON: jsonText(RunCounts{})}
	e = coordinator.register(ctx, &run, token, true)
	if e != nil {
		_ = repo.ReleaseLease(context.Background(), token)
		return nil, e
	}
	handedOff = true
	go func() { defer s.endTask(id, cancel); s.execute(runctx, run, token) }()
	return &TriggerResult{RunID: id}, nil
}
func (s *Service) heartbeat(ctx context.Context, token store.LeaseToken, cancel context.CancelFunc, done chan<- struct{}) {
	defer close(done)
	ticker := time.NewTicker(s.opts.HeartbeatInterval)
	defer ticker.Stop()
	repo := store.NewNotionSyncRepository(s.ds.DB())
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ok, e := repo.RenewLease(ctx, token, s.opts.LeaseDuration)
			if e != nil || !ok {
				cancel()
				return
			}
		}
	}
}
func (s *Service) execute(parent context.Context, run model.NotionSyncRun, token store.LeaseToken) {
	_ = (runCoordinator{service: s}).execute(parent, run, token, runCompletionPolicy{
		timeout: 4 * time.Minute, blockedIsFailure: true, scheduleNext: true, pruneResolved: true,
	}, s.scanAndApply)
}
func (s *Service) synchronousRun(ctx context.Context, mode string, fn func(context.Context, model.NotionSyncRun, store.LeaseToken, *RunCounts) error) (string, error) {
	taskctx, taskcancel, taskID, e := s.beginTask(ctx)
	if e != nil {
		return "", e
	}
	defer s.endTask(taskID, taskcancel)
	ctx = taskctx
	db, e := s.db(ctx)
	if e != nil {
		return "", e
	}
	repo := store.NewNotionSyncRepository(db)
	control, e := repo.Control(ctx)
	if e != nil {
		return "", e
	}
	coordinator := runCoordinator{service: s}
	if e := coordinator.checkCooldown(db, control); e != nil {
		return "", e
	}
	token, ok, e := repo.AcquireLease(ctx, s.opts.OwnerID, s.opts.LeaseDuration)
	if e != nil {
		return "", e
	}
	if !ok {
		return "", conflict("已有同步或接管任务运行")
	}
	id := newID()
	run := model.NotionSyncRun{ID: id, Mode: mode, Status: "running", Phase: "bootstrap", StartedAt: time.Now().UTC(), LeaseEpoch: token.Epoch, CountsJSON: jsonText(RunCounts{})}
	if e = coordinator.register(ctx, &run, token, false); e != nil {
		_ = repo.ReleaseLease(context.Background(), token)
		return id, e
	}
	return id, coordinator.execute(ctx, run, token, runCompletionPolicy{}, fn)
}

func (s *Service) observeCooldown(ctx context.Context, until time.Time) error {
	token, ok := ctx.Value(leaseContextKey{}).(store.LeaseToken)
	if !ok {
		return nil
	}
	return s.fenced(ctx, token, func(tx *gorm.DB, c *model.NotionSyncControl) error {
		return tx.Model(c).Update("cooldown_until", until).Error
	})
}
