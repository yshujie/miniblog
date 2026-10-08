package notionsync

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/store"
)

func TestMaintenanceRunPreservesCountsAndErrorContract(t *testing.T) {
	failure := errors.New("fixture run failed")
	for _, tc := range []struct {
		name   string
		counts RunCounts
		err    error
		status string
	}{
		{"blocked is a review count", RunCounts{Blocked: 1}, nil, "completed"},
		{"pending and frozen are normal", RunCounts{Pending: 2, Frozen: 3}, nil, "completed"},
		{"failed item", RunCounts{Failed: 1}, nil, "completed_with_errors"},
		{"execution error", RunCounts{}, failure, "failed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, db, _ := syncFixture(t)
			id, err := s.synchronousRun(context.Background(), "bootstrap_preview", func(ctx context.Context, run model.NotionSyncRun, token store.LeaseToken, counts *RunCounts) error {
				if run.Phase != "bootstrap" || token.Owner == "" || token.Epoch == 0 || ctx.Value(leaseContextKey{}) != token {
					t.Fatalf("run coordination context missing: %+v %+v", run, token)
				}
				*counts = tc.counts
				return tc.err
			})
			if err != tc.err || id == "" {
				t.Fatalf("id=%q error=%v want=%v", id, err, tc.err)
			}
			run, err := s.Run(context.Background(), id)
			if err != nil || run.Status != tc.status || run.Phase != "finished" || run.Counts != tc.counts || run.FinishedAt == nil {
				t.Fatalf("run=%+v error=%v", run, err)
			}
			control, err := store.NewNotionSyncRepository(db).Control(context.Background())
			if err != nil || control.CurrentRunID != "" || control.LeaseOwner != "" || control.LeaseUntil != nil || control.NextRunAt != nil {
				t.Fatalf("maintenance cleanup changed: %+v error=%v", control, err)
			}
		})
	}
}

func TestMaintenanceRunKeepsCallerDeadline(t *testing.T) {
	s, _, _ := syncFixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	deadline, _ := ctx.Deadline()
	_, err := s.synchronousRun(ctx, "catalog_prepare", func(active context.Context, _ model.NotionSyncRun, _ store.LeaseToken, _ *RunCounts) error {
		got, ok := active.Deadline()
		if !ok || got != deadline {
			t.Fatalf("deadline=%v want=%v", got, deadline)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestBusyLeaseKeepsAsyncAndMaintenanceReturnContracts(t *testing.T) {
	s, db, _ := syncFixture(t)
	repo := store.NewNotionSyncRepository(db)
	token, acquired, err := repo.AcquireLease(context.Background(), "other worker", time.Minute)
	if err != nil || !acquired {
		t.Fatal(token, acquired, err)
	}
	defer repo.ReleaseLease(context.Background(), token)
	if err = db.Model(&model.NotionSyncControl{}).Where("id = 1").Update("current_run_id", "already-running").Error; err != nil {
		t.Fatal(err)
	}
	result, err := s.Trigger(context.Background(), TriggerInput{Mode: "dry_run"})
	if err != nil || result == nil || result.RunID != "already-running" {
		t.Fatal(result, err)
	}
	called := false
	id, err := s.synchronousRun(context.Background(), "bootstrap_preview", func(context.Context, model.NotionSyncRun, store.LeaseToken, *RunCounts) error {
		called = true
		return nil
	})
	if id != "" || fixtureStatus(err) != 409 || err.Error() != "已有同步或接管任务运行" || called {
		t.Fatal(id, err, called)
	}
	var count int64
	if err = db.Model(&model.NotionSyncRun{}).Count(&count).Error; err != nil || count != 0 {
		t.Fatal(count, err)
	}
}

func TestStopWaitsForMaintenanceCancellationAndLeaseCleanup(t *testing.T) {
	s, db, _ := syncFixture(t)
	started := make(chan struct{})
	finished := make(chan error, 1)
	go func() {
		_, err := s.synchronousRun(context.Background(), "bootstrap_preview", func(ctx context.Context, _ model.NotionSyncRun, _ store.LeaseToken, _ *RunCounts) error {
			close(started)
			<-ctx.Done()
			return ctx.Err()
		})
		finished <- err
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("maintenance did not start")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := s.Stop(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-finished:
		if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	case <-ctx.Done():
		t.Fatal("maintenance did not stop")
	}
	control, err := store.NewNotionSyncRepository(db).Control(context.Background())
	if err != nil || control.LeaseOwner != "" || control.CurrentRunID != "" {
		t.Fatal(control, err)
	}
	if _, err = s.Trigger(context.Background(), TriggerInput{Mode: "dry_run"}); fixtureStatus(err) != 503 {
		t.Fatal(err)
	}
}

func TestCoordinatedScanKeepsTimeoutBlockedFailureAndNextRun(t *testing.T) {
	s, db, _ := syncFixture(t)
	repo := store.NewNotionSyncRepository(db)
	token, acquired, err := repo.AcquireLease(context.Background(), "scan worker", time.Minute)
	if err != nil || !acquired {
		t.Fatal(token, acquired, err)
	}
	run := model.NotionSyncRun{ID: "scan-coordinator", Mode: "sync", Status: "running", Phase: "schema", StartedAt: time.Now().UTC(), LeaseEpoch: token.Epoch}
	coordinator := runCoordinator{service: s}
	if err = coordinator.register(context.Background(), &run, token, true); err != nil {
		t.Fatal(err)
	}
	before := time.Now()
	err = coordinator.execute(context.Background(), run, token, runCompletionPolicy{
		timeout: 4 * time.Minute, blockedIsFailure: true, scheduleNext: true, pruneResolved: true,
	}, func(ctx context.Context, _ model.NotionSyncRun, _ store.LeaseToken, counts *RunCounts) error {
		deadline, ok := ctx.Deadline()
		if !ok || deadline.Before(before.Add(4*time.Minute)) || deadline.After(time.Now().Add(4*time.Minute)) {
			t.Fatalf("scan deadline changed: %v", deadline)
		}
		*counts = RunCounts{Blocked: 1, Pending: 2, Frozen: 3}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	finished, err := s.Run(context.Background(), run.ID)
	if err != nil || finished.Status != "completed_with_errors" || finished.Counts.Blocked != 1 || finished.Counts.Pending != 2 || finished.Counts.Frozen != 3 {
		t.Fatal(finished, err)
	}
	control, err := repo.Control(context.Background())
	if err != nil || control.NextRunAt == nil || control.NextRunAt.Before(before.Add(s.opts.Interval)) || control.LeaseOwner != "" || control.CurrentRunID != "" {
		t.Fatal(control, err)
	}
}
