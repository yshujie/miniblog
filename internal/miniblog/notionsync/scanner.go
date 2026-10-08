package notionsync

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/yshujie/miniblog/internal/miniblog/biz/article"
	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/source"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"sort"
	"time"
)

type leaseContextKey struct{}
type observation struct {
	Snapshot Snapshot
	Error    error
	Previous model.NotionPageBinding
	Seen     bool
	Fresh    bool
}
type scannedSource struct {
	Row         model.NotionSyncSource
	Config      SourceConfig
	Schema      source.NotionDataSource
	SchemaError error
}

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
	nowDB, e := store.DatabaseNow(db)
	if e != nil {
		return nil, e
	}
	if c.CooldownUntil != nil && c.CooldownUntil.After(nowDB) {
		return nil, &Error{Code: "cooldown", Message: "Notion 限流冷却尚未结束", HTTPStatus: 429}
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
	e = s.fenced(ctx, token, func(tx *gorm.DB, c *model.NotionSyncControl) error {
		if c.CurrentRunID != "" {
			if e := tx.Model(&model.NotionSyncRun{}).Where("run_id = ? AND status = ?", c.CurrentRunID, "running").Updates(map[string]interface{}{"status": "abandoned", "phase": "finished", "finished_at": now, "error": "previous worker lease expired"}).Error; e != nil {
				return e
			}
		}
		if e := tx.Create(&run).Error; e != nil {
			return e
		}
		return tx.Model(c).Update("current_run_id", id).Error
	})
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
	ctx, cancel := context.WithTimeout(parent, 4*time.Minute)
	ctx = context.WithValue(ctx, leaseContextKey{}, token)
	done := make(chan struct{})
	go s.heartbeat(ctx, token, cancel, done)
	counts := RunCounts{}
	runErr := s.scanAndApply(ctx, run, token, &counts)
	cancel()
	<-done
	finishctx, finishcancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer finishcancel()
	now := time.Now().UTC()
	status := "completed"
	if runErr != nil {
		status = "failed"
	} else if counts.Failed > 0 || counts.Blocked > 0 {
		status = "completed_with_errors"
	}

	updates := map[string]interface{}{"status": status, "phase": "finished", "finished_at": now, "counts_json": jsonText(counts)}
	if runErr != nil {
		updates["error"] = runErr.Error()
	}
	_ = s.finishRun(finishctx, run, token, updates, true)

	_ = s.pruneResolvedRuns(finishctx, token)
	_ = store.NewNotionSyncRepository(s.ds.DB()).ReleaseLease(finishctx, token)
}
func (s *Service) scanAndApply(ctx context.Context, run model.NotionSyncRun, token store.LeaseToken, counts *RunCounts) error {
	var previous []model.NotionPageBinding
	if e := s.ds.DB().WithContext(ctx).Find(&previous).Error; e != nil {
		return e
	}
	old := map[string]model.NotionPageBinding{}
	for _, p := range previous {
		old[p.PageID] = p
	}
	control, e := store.NewNotionSyncRepository(s.ds.DB()).Control(ctx)
	if e != nil {
		return e
	}
	firstBaseline := !control.BaselineFrozen
	sources := map[string]scannedSource{}
	observations := map[string]observation{}
	var scanErrors []error
	for _, id := range AllowedSources() {
		src, pages, complete, e := s.scanSource(ctx, token, id, run.Mode == "sync")
		if src.Row.ID != "" {
			sources[id] = src
		}
		if src.SchemaError != nil && run.Mode == "sync" && src.Row.Enabled {
			for pageID, p := range old {
				if p.SourceID == id && p.ManagementState == model.NotionManagementManaged {
					p.Revision++
					p.NeedsRevalidation = true
					p.PublishBlockReason = "source_schema_changed"
					old[pageID] = p
				}
			}
		}
		if e != nil {
			scanErrors = append(scanErrors, fmt.Errorf("source %s: %w", id, e))
			s.sourceFailure(ctx, token, id, e)
		}
		for _, page := range pages {
			snap, parseErr := snapshotOf(page, id, src.Config)
			pageID := normalizePageID(page.ID)
			obs := observation{Snapshot: snap, Error: parseErr, Previous: old[pageID], Seen: true}
			if existing, ok := observations[pageID]; ok && (existing.Snapshot.SourceID != id || metadataHash(existing.Snapshot) != metadataHash(snap)) {
				obs.Error = errors.New("page observed in conflicting source snapshots")
			}
			observations[pageID] = obs
		}
		if complete {
			now := time.Now().UTC()
			if e := s.fenced(ctx, token, func(tx *gorm.DB, _ *model.NotionSyncControl) error {
				return tx.Model(&model.NotionSyncSource{}).Where("source_id = ?", id).Updates(map[string]interface{}{"health": "complete", "last_complete_scan_at": now, "last_error": ""}).Error
			}); e != nil {
				return e
			}
		}
	}
	globalComplete := len(scanErrors) == 0
	// A complete union is required for absence checks. Disabled sources remain in the union.
	if globalComplete {
		for _, p := range previous {
			if p.ManagementState != model.NotionManagementManaged {
				continue
			}
			if _, seen := observations[p.PageID]; seen {
				continue
			}
			snap, readErr := s.readLatest(ctx, p.PageID, sources, p)
			observations[p.PageID] = observation{Snapshot: snap, Error: readErr, Previous: p, Fresh: readErr == nil}
		}
	}
	// Changes and revalidation use a fresh page read rather than applying a dry-run/query cache.
	for id, obs := range observations {
		src := sources[obs.Snapshot.SourceID]
		changed := obs.Previous.PageID == "" || obs.Previous.MetadataHash != metadataHash(obs.Snapshot) || obs.Previous.NeedsRevalidation
		conflict := obs.Error != nil
		if (conflict || (run.Mode == "sync" && src.Row.Enabled && changed)) && obs.Seen {
			snap, readErr := s.readLatest(ctx, id, sources, observationHint(obs))
			obs.Snapshot = snap
			obs.Error = readErr
			obs.Fresh = readErr == nil || snap.Reason == "invalid_field" || snap.Reason == "unknown_state"
			observations[id] = obs
		}
	}
	now := time.Now().UTC()
	if e := s.fenced(ctx, token, func(tx *gorm.DB, c *model.NotionSyncControl) error {
		if firstBaseline {
			for id, src := range sources {
				var current model.NotionSyncSource
				if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("source_id = ?", id).First(&current).Error; e != nil {
					return e
				}
				if current.ConfigRevision != src.Row.ConfigRevision {
					return conflict("基线扫描中源配置已变化，请重新预览")
				}
			}
			if run.Mode != "dry_run" && run.Mode != "bootstrap_preview" {
				return conflict("基线未冻结")
			}
			for _, obs := range observations {
				snap := obs.Snapshot
				if snap.PageID == "" {
					globalComplete = false
					continue
				}
				p := model.NotionPageBinding{PageID: snap.PageID, SourceID: snap.SourceID, ManagementState: model.NotionManagementBaselinePending, Revision: 1, BootstrapState: "pending", DesiredState: snap.DesiredState, SnapshotJSON: jsonText(snap), MetadataHash: metadataHash(snap), PageURL: snap.PageURL, PublicURL: snap.PublicURL, NotionLastEditedAt: &snap.LastEditedAt, LastSeenRunID: run.ID, NativeArchived: snap.NativeArchived, InTrash: snap.InTrash}
				if obs.Error != nil {
					p.LastError = obs.Error.Error()
				}
				if e := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&p).Error; e != nil {
					return e
				}
			}
			if globalComplete {
				if e := tx.Model(c).Updates(map[string]interface{}{"baseline_frozen": true, "baseline_at": now}).Error; e != nil {
					return e
				}
			}
		}
		if globalComplete {
			if e := tx.Model(c).Update("last_complete_scan_at", now).Error; e != nil {
				return e
			}
		}
		phase := "apply_partial"
		if globalComplete {
			phase = "apply_complete"
		}
		return tx.Model(&model.NotionSyncRun{}).Where("run_id = ?", run.ID).Update("phase", phase).Error
	}); e != nil {
		return e
	}
	if run.Mode != "sync" {
		for _, src := range sources {
			if e := s.previewTopics(ctx, run, token, src); e != nil {
				return e
			}
		}
	}
	if run.Mode == "sync" && !firstBaseline {
		for _, srcID := range AllowedSources() {
			src, ok := sources[srcID]
			if !ok || !src.Row.Enabled {
				continue
			}
			if e := s.syncTopics(ctx, run, token, src, counts); e != nil {
				scanErrors = append(scanErrors, e)
			}
		}
	}
	pageIDs := make([]string, 0, len(observations))
	for id := range observations {
		pageIDs = append(pageIDs, id)
	}
	sort.Strings(pageIDs)
	for _, id := range pageIDs {
		obs := observations[id]
		counts.Seen++
		src := sources[obs.Snapshot.SourceID]
		if e := s.processObservation(ctx, run, token, src, obs, firstBaseline, counts); e != nil {
			if errors.Is(e, store.ErrLeaseLost) || ctx.Err() != nil {
				return e
			}
			counts.Failed++
			if e2 := s.recordPageError(ctx, token, obs.Previous, e); e2 != nil {
				return e2
			}
			if e2 := s.recordItem(ctx, token, run.ID, id, obs.Previous.ArticleID, "failed", "", e.Error(), obs.Previous, obs.Snapshot); e2 != nil {
				return e2
			}
		}
		if e := s.fenced(ctx, token, func(tx *gorm.DB, _ *model.NotionSyncControl) error {
			return tx.Model(&model.NotionSyncRun{}).Where("run_id = ?", run.ID).Update("counts_json", jsonText(counts)).Error
		}); e != nil {
			return e
		}
	}
	if run.Mode == "sync" && globalComplete && counts.Failed == 0 && counts.Blocked == 0 {
		if e := s.fenced(ctx, token, func(tx *gorm.DB, c *model.NotionSyncControl) error {
			if e := tx.Model(c).Update("last_success_at", now).Error; e != nil {
				return e
			}
			return tx.Model(&model.NotionSyncSource{}).Where("source_id IN ?", AllowedSources()).Update("last_success_at", now).Error
		}); e != nil {
			return e
		}
	}
	return errors.Join(scanErrors...)
}
func (s *Service) scanSource(ctx context.Context, token store.LeaseToken, id string, isolate bool) (scannedSource, []source.NotionPage, bool, error) {
	var row model.NotionSyncSource
	e := s.ds.DB().WithContext(ctx).Where("source_id = ?", id).First(&row).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		row = model.NotionSyncSource{ID: id, DataSourceID: id, Label: allowedSources[id], Health: "not_configured"}
	} else if e != nil {
		return scannedSource{}, nil, false, e
	}
	now := time.Now().UTC()
	if e = s.fenced(ctx, token, func(tx *gorm.DB, _ *model.NotionSyncControl) error {
		if e := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error; e != nil {
			return e
		}
		return tx.Model(&row).Updates(map[string]interface{}{"last_attempt_at": now, "health": "scanning", "last_error": ""}).Error
	}); e != nil {
		return scannedSource{}, nil, false, e
	}
	schema, e := s.client.RetrieveDataSource(ctx, id)
	if e != nil {
		return scannedSource{Row: row}, nil, false, e
	}
	cfg := configOf(row)
	var schemaErr error
	if !configComplete(cfg) {
		cfg, schemaErr = discoverConfig(schema)
		if schemaErr != nil {
			if isolate && row.Enabled {
				if e := s.recordSchemaFailure(ctx, token, id, schema.InTrash, schemaErr); e != nil {
					return scannedSource{}, nil, false, e
				}
			}
			return scannedSource{Row: row, Schema: schema, SchemaError: schemaErr}, nil, false, schemaErr
		}
		prior := row.ConfigRevision
		row.PropertyMappingJSON = jsonText(cfg)
		row.StatusMappingJSON = jsonText(cfg.StateOptionIDs)
		row.ConfigRevision++
		e = s.fenced(ctx, token, func(tx *gorm.DB, _ *model.NotionSyncControl) error {
			result := tx.Model(&model.NotionSyncSource{}).Where("source_id = ? AND config_revision = ?", id, prior).Updates(map[string]interface{}{"property_mapping_json": row.PropertyMappingJSON, "status_mapping_json": row.StatusMappingJSON, "config_revision": row.ConfigRevision})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				return conflict("字段配置已变化，请重新扫描")
			}
			return nil
		})
		if e != nil {
			return scannedSource{}, nil, false, e
		}
	} else {
		schemaErr = validateConfig(schema, cfg)
	}
	src := scannedSource{Row: row, Config: cfg, Schema: schema, SchemaError: schemaErr}
	if schemaErr != nil && isolate && row.Enabled {
		if e := s.recordSchemaFailure(ctx, token, id, schema.InTrash, schemaErr); e != nil {
			return src, nil, false, e
		}
	}
	var pages []source.NotionPage
	var queryErrors []error
	if schemaErr != nil {
		queryErrors = append(queryErrors, schemaErr)
	}
	for _, archived := range []bool{false, true} {
		cursor := ""
		seenCursors := map[string]bool{}
		for {
			result, e := s.client.QueryDataSource(ctx, id, archived, cursor)
			// Complete pages already returned remain useful even when the union is incomplete.
			pages = append(pages, result.Results...)
			if e != nil {
				queryErrors = append(queryErrors, e)
				break
			}
			if result.RequestStatus.Type != "complete" {
				queryErrors = append(queryErrors, errors.New("incomplete data source query"))
				break
			}
			if !result.HasMore {
				break
			}
			if result.NextCursor == nil || *result.NextCursor == "" || seenCursors[*result.NextCursor] {
				queryErrors = append(queryErrors, errors.New("invalid or repeated query cursor"))
				break
			}
			cursor = *result.NextCursor
			seenCursors[cursor] = true
		}
	}
	return src, pages, len(queryErrors) == 0, errors.Join(queryErrors...)
}
func (s *Service) readLatest(ctx context.Context, pageID string, sources map[string]scannedSource, previous model.NotionPageBinding) (Snapshot, error) {
	page, e := s.client.RetrievePage(ctx, pageID)
	if e != nil {
		var snap Snapshot
		_ = json.Unmarshal([]byte(previous.SnapshotJSON), &snap)
		snap.PageID = pageID
		snap.SourceID = previous.SourceID
		snap.DesiredState = previous.DesiredState
		snap.Reason = "transport_error"
		if source.IsNotionUnavailable(e) {
			snap.Reason = "source_unavailable"
		}
		return snap, e
	}
	target := normalizeID(page.Parent.DataSourceID)
	src, inside := sources[target]
	if page.Parent.Type != "data_source_id" || !inside {
		if _, inScope := allowedSources[target]; inScope {
			return Snapshot{PageID: pageID, SourceID: previous.SourceID, Reason: "transport_error"}, errors.New("current parent schema is unavailable")
		}
		return Snapshot{PageID: pageID, SourceID: previous.SourceID, DesiredState: previous.DesiredState, PageURL: page.URL, PublicURL: page.PublicURL, NativeArchived: page.IsArchived, InTrash: page.InTrash, LastEditedAt: page.LastEditedTime, Reason: "out_of_scope"}, nil
	}
	return snapshotOf(page, src.Row.ID, src.Config)
}
func (s *Service) recordPageError(ctx context.Context, token store.LeaseToken, p model.NotionPageBinding, err error) error {
	if p.PageID == "" {
		return nil
	}
	return s.fenced(ctx, token, func(tx *gorm.DB, _ *model.NotionSyncControl) error {
		return tx.Model(&model.NotionPageBinding{}).Where("page_id = ?", p.PageID).Update("last_error", err.Error()).Error
	})
}

func (s *Service) sourceFailure(ctx context.Context, token store.LeaseToken, id string, err error) {
	_ = s.fenced(ctx, token, func(tx *gorm.DB, _ *model.NotionSyncControl) error {
		return tx.Model(&model.NotionSyncSource{}).Where("source_id = ?", id).Updates(map[string]interface{}{"health": "error", "last_error": err.Error()}).Error
	})
}
func (s *Service) recordItem(ctx context.Context, token store.LeaseToken, runID, pageID string, id *uint64, outcome, reason, err string, before, after interface{}) error {
	return s.fenced(ctx, token, func(tx *gorm.DB, _ *model.NotionSyncControl) error {
		item := model.NotionSyncRunItem{RunID: runID, ItemID: pageID, PageID: pageID, ArticleID: id, Outcome: outcome, Reason: reason, Error: err, BeforeJSON: jsonText(before), AfterJSON: jsonText(after)}
		return tx.Clauses(clause.OnConflict{UpdateAll: true}).Create(&item).Error
	})
}
func (s *Service) processObservation(ctx context.Context, run model.NotionSyncRun, token store.LeaseToken, src scannedSource, obs observation, firstBaseline bool, counts *RunCounts) error {
	snap := obs.Snapshot
	if snap.PageID == "" {
		return errors.New("invalid page identity in scan")
	}
	if run.Mode != "sync" || firstBaseline || obs.Previous.ManagementState == model.NotionManagementBaselinePending || obs.Previous.ManagementState == model.NotionManagementDetached {
		outcome := "preview"
		reason := snap.Reason
		if firstBaseline || obs.Previous.ManagementState == model.NotionManagementBaselinePending {
			outcome = "baseline_pending"
			reason = "historical_match_requires_confirmation"
		}
		if obs.Error != nil {
			reason = obs.Error.Error()
			counts.Failed++
		} else if reason != "" {
			counts.Blocked++
		} else {
			counts.Unchanged++
		}
		e := s.fenced(ctx, token, func(tx *gorm.DB, _ *model.NotionSyncControl) error {
			var p model.NotionPageBinding
			e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("page_id = ?", snap.PageID).First(&p).Error
			if errors.Is(e, gorm.ErrRecordNotFound) {
				if !firstBaseline {
					return nil
				}
				p = model.NotionPageBinding{PageID: snap.PageID, SourceID: snap.SourceID, ManagementState: model.NotionManagementBaselinePending, Revision: 1, BootstrapState: "pending"}
			} else if e != nil {
				return e
			}
			if p.ManagementState == model.NotionManagementManaged {
				if obs.Error != nil {
					return tx.Model(&p).Update("last_error", obs.Error.Error()).Error
				}
				return nil
			}
			// A failed read never replaces the last successful metadata/visibility snapshot.
			if !obs.Seen && obs.Error != nil {
				return tx.Model(&p).Updates(map[string]interface{}{"last_error": obs.Error.Error(), "last_seen_run_id": run.ID}).Error
			}
			// Preview never changes ownership, local status, or publication holds.
			p.SourceID = snap.SourceID
			p.SnapshotJSON = jsonText(snap)
			p.MetadataHash = metadataHash(snap)
			p.DesiredState = snap.DesiredState
			p.PageURL = snap.PageURL
			p.PublicURL = snap.PublicURL
			p.NotionLastEditedAt = &snap.LastEditedAt
			p.LastSeenRunID = run.ID
			p.NativeArchived = snap.NativeArchived
			p.InTrash = snap.InTrash
			p.LastError = ""
			if obs.Error != nil {
				p.LastError = obs.Error.Error()
			}
			return tx.Save(&p).Error
		})
		if e != nil {
			return e
		}
		return s.recordItem(ctx, token, run.ID, snap.PageID, obs.Previous.ArticleID, outcome, reason, "", obs.Previous, snap)
	}
	if !src.Row.Enabled {
		counts.Blocked++
		return s.recordItem(ctx, token, run.ID, snap.PageID, obs.Previous.ArticleID, "blocked", "source_disabled", "", obs.Previous, snap)
	}

	metadataComplete := obs.Error == nil && snap.Reason == "" && src.SchemaError == nil
	metadataError := snap.Reason
	if src.SchemaError != nil {
		metadataError = "source_schema_changed"
	}
	if snap.DesiredState == 0 && src.SchemaError == nil {
		metadataError = "unknown_state"
	}
	result, e := article.NewForSync(s.ds, s.opts.Author).ApplySyncedSource(ctx, article.SyncInput{Lease: token, RunID: run.ID, SourceID: src.Row.ID, DataSourceID: src.Row.DataSourceID, PageID: snap.PageID, ThemePropertyID: src.Config.TopicPropertyID, ThemeOptionID: snap.TopicOptionID, ThemeOptionName: snap.TopicOptionName, Title: snap.Title, Tags: snap.Tags, PageURL: snap.PageURL, PublicURL: snap.PublicURL, PublicURLObserved: obs.Fresh, DesiredState: snap.DesiredState, NativeArchived: snap.NativeArchived, InTrash: snap.InTrash, MetadataComplete: metadataComplete, MetadataError: metadataError, NotionLastEditedAt: &snap.LastEditedAt, MetadataHash: metadataHash(snap), SnapshotJSON: jsonText(snap), ExpectedConfigRevision: src.Row.ConfigRevision, ExpectedBindingRevision: obs.Previous.Revision})
	if e != nil {
		return e
	}
	var id *uint64
	if result.ArticleID != 0 {
		id = &result.ArticleID
	}
	switch result.Outcome {
	case "created":
		counts.Created++
	case "updated", "state_only":
		counts.Updated++
	case "unchanged":
		counts.Unchanged++
	case "retained":
		counts.Failed++
	default:
		counts.Blocked++
	}
	if obs.Error != nil {
		if e := s.recordPageError(ctx, token, model.NotionPageBinding{PageID: snap.PageID}, obs.Error); e != nil {
			return e
		}
	}
	return s.recordItem(ctx, token, run.ID, snap.PageID, id, result.Outcome, result.Reason, "", obs.Previous, snap)
}

func observationHint(obs observation) model.NotionPageBinding {
	if obs.Previous.PageID != "" {
		return obs.Previous
	}
	return model.NotionPageBinding{PageID: obs.Snapshot.PageID, SourceID: obs.Snapshot.SourceID, DesiredState: obs.Snapshot.DesiredState, SnapshotJSON: jsonText(obs.Snapshot)}
}
