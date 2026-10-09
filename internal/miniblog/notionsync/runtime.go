package notionsync

import (
	"context"
	"time"
)

func (s *Service) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopping {
		return unavailable("同步服务已关闭")
	}
	if s.stop != nil {
		return nil
	}
	s.ctx, s.stop = context.WithCancel(ctx)
	if !s.opts.Enabled {
		return nil
	}
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(s.opts.Interval)
		defer ticker.Stop()
		for {
			select {
			case <-s.ctx.Done():
				return
			case <-ticker.C:
				_, _ = s.Trigger(s.ctx, TriggerInput{Mode: "sync"})
			}
		}
	}()
	return nil
}
func (s *Service) Stop(ctx context.Context) error {
	s.mu.Lock()
	s.stopping = true
	for _, cancel := range s.active {
		cancel()
	}
	if s.stop != nil {
		s.stop()
	}
	if s.runCancel != nil {
		s.runCancel()
	}
	s.mu.Unlock()
	done := make(chan struct{})
	go func() { s.wg.Wait(); close(done) }()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (s *Service) beginTask(parent context.Context) (context.Context, context.CancelFunc, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopping {
		return nil, nil, "", unavailable("同步服务已关闭")
	}
	ctx, cancel := context.WithCancel(parent)
	id := newID()
	s.active[id] = cancel
	s.wg.Add(1)
	return ctx, cancel, id, nil
}
func (s *Service) endTask(id string, cancel context.CancelFunc) {
	cancel()
	s.mu.Lock()
	delete(s.active, id)
	s.mu.Unlock()
	s.wg.Done()
}
