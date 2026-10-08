package notionsync

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"gorm.io/gorm"
	"sort"
	"strconv"
	"strings"
	"time"
)

func normalizedQuery(q ListQuery) ListQuery {
	if q.Page > 1000000 {
		q.Page = 1000000
	}
	if q.Page < 1 {
		q.Page = 1
	}
	if q.Limit < 1 {
		q.Limit = 20
	}
	if q.Limit > 100 {
		q.Limit = 100
	}
	return q
}
func decodeJSON(v string) interface{} {
	if v == "" {
		return nil
	}
	var result interface{}
	if json.Unmarshal([]byte(v), &result) != nil {
		return nil
	}
	return result
}
func articleID(v *uint64) *string {
	if v == nil {
		return nil
	}
	result := strconv.FormatUint(*v, 10)
	return &result
}
func bindingDTO(v model.NotionCatalogBinding) TopicBindingDTO {
	code := ""
	if v.SectionCode != nil {
		code = *v.SectionCode
	}
	return TopicBindingDTO{ID: strconv.FormatUint(v.ID, 10), SourceID: v.SourceID, OptionID: v.OptionID, OptionName: v.OptionName, SectionCode: code, Status: v.Status, Reason: v.Reason}
}
func pageDTO(v model.NotionPageBinding) PageDTO {
	var snap Snapshot
	_ = json.Unmarshal([]byte(v.SnapshotJSON), &snap)
	return PageDTO{PageID: v.PageID, ArticleID: articleID(v.ArticleID), SourceID: v.SourceID, Title: snap.Title, DesiredState: stateName(v.DesiredState), Status: v.ManagementState, ManagementState: v.ManagementState, PublishBlockReason: v.PublishBlockReason, LastError: v.LastError, PageURL: v.PageURL, PublicURL: v.PublicURL, PublicationHold: v.PublicationHeld, PublicationHoldReason: v.PublicationHoldReason, BootstrapState: v.BootstrapState, Revision: v.Revision}
}
func runDTO(v model.NotionSyncRun) RunDTO {
	var counts RunCounts
	_ = json.Unmarshal([]byte(v.CountsJSON), &counts)
	return RunDTO{RunID: v.ID, Mode: v.Mode, Status: v.Status, Phase: v.Phase, StartedAt: &v.StartedAt, FinishedAt: v.FinishedAt, Counts: counts, Error: v.Error}
}
func (s *Service) sourceDTO(db *gorm.DB, v model.NotionSyncSource) (SourceDTO, error) {
	var bindings []model.NotionCatalogBinding
	if e := db.Where("source_id = ?", v.ID).Order("id").Find(&bindings).Error; e != nil {
		return SourceDTO{}, e
	}
	dto := SourceDTO{SourceID: v.ID, Label: v.Label, ModuleCode: v.ModuleCode, Enabled: v.Enabled, ConfigRevision: v.ConfigRevision, Config: configOf(v), Health: v.Health, LastAttemptAt: v.LastAttemptAt, LastCompleteScanAt: v.LastCompleteScanAt, LastSuccessAt: v.LastSuccessAt, LastError: v.LastError, CatalogBindings: []TopicBindingDTO{}}
	for _, b := range bindings {
		dto.CatalogBindings = append(dto.CatalogBindings, bindingDTO(b))
	}
	return dto, nil
}
func (s *Service) Status(ctx context.Context) (*StatusDTO, error) {
	db, e := s.db(ctx)
	if e != nil {
		return nil, e
	}
	var c model.NotionSyncControl
	e = db.First(&c, 1).Error
	if e != nil && !errors.Is(e, gorm.ErrRecordNotFound) {
		return nil, e
	}
	list, e := s.Sources(ctx, ListQuery{Page: 1, Limit: 100})
	if e != nil {
		return nil, e
	}
	dto := &StatusDTO{Enabled: s.opts.Enabled, Paused: c.Paused, SourceWritesPaused: c.SourceWritesPaused, BaselineFrozen: c.BaselineFrozen, Health: "not_configured", Sources: list.Items}
	if e = db.Model(&model.NotionPageBinding{}).Where("management_state = ?", model.NotionManagementBaselinePending).Count(&dto.PendingCount).Error; e != nil {
		return nil, e
	}
	if e = db.Model(&model.NotionPageBinding{}).Where("last_error <> ''").Count(&dto.ErrorCount).Error; e != nil {
		return nil, e
	}
	if e = publicationBlockedPages(ctx, db).Count(&dto.BlockedCount).Error; e != nil {
		return nil, e
	}
	for _, src := range list.Items {
		if src.LastAttemptAt != nil && (dto.LastAttemptAt == nil || src.LastAttemptAt.After(*dto.LastAttemptAt)) {
			dto.LastAttemptAt = src.LastAttemptAt
		}
	}
	dto.LastCompleteScanAt = c.LastCompleteScanAt
	dto.LastSuccessAt = c.LastSuccessAt
	if dto.LastCompleteScanAt != nil {
		dto.Health = "healthy"
		if dto.ErrorCount > 0 {
			dto.Health = "degraded"
		}
		for _, src := range list.Items {
			if src.Health == "error" {
				dto.Health = "degraded"
			}
		}
		if time.Since(*dto.LastCompleteScanAt) > 15*time.Minute {
			dto.Health = "stale"
		}
	}
	now, e := store.DatabaseNow(db)
	if e != nil {
		return nil, e
	}
	if c.CurrentRunID != "" && c.LeaseUntil != nil && c.LeaseUntil.After(now) {
		dto.CurrentRunID = &c.CurrentRunID
		dto.Health = "running"
	}
	if c.Paused {
		dto.Health = "paused"
	}
	return dto, nil
}
func (s *Service) Sources(ctx context.Context, q ListQuery) (*PageResult[SourceDTO], error) {
	db, e := s.db(ctx)
	if e != nil {
		return nil, e
	}
	q = normalizedQuery(q)
	var rows []model.NotionSyncSource
	if e = db.Order("source_id").Find(&rows).Error; e != nil {
		return nil, e
	}
	seen := map[string]bool{}
	for _, r := range rows {
		seen[r.ID] = true
	}
	for id, label := range allowedSources {
		if !seen[id] {
			rows = append(rows, model.NotionSyncSource{ID: id, DataSourceID: id, Label: label, Health: "not_configured"})
		}
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })
	// The fixed inventory is small; paginate after adding unconfigured sources.
	total := len(rows)
	start := (q.Page - 1) * q.Limit
	end := start + q.Limit
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}
	result := &PageResult[SourceDTO]{Items: []SourceDTO{}, Total: int64(total), Page: q.Page, Limit: q.Limit}
	for _, r := range rows[start:end] {
		dto, e := s.sourceDTO(db, r)
		if e != nil {
			return nil, e
		}
		result.Items = append(result.Items, dto)
	}
	return result, nil
}
func (s *Service) Pages(ctx context.Context, q PageQuery) (*PageResult[PageDTO], error) {
	db, e := s.db(ctx)
	if e != nil {
		return nil, e
	}
	q.ListQuery = normalizedQuery(q.ListQuery)
	base := db.Model(&model.NotionPageBinding{})
	if q.SourceID != "" {
		base = base.Where("source_id = ?", normalizeID(q.SourceID))
	}
	if q.ManagementState != "" {
		base = base.Where("management_state = ?", q.ManagementState)
	}
	if q.Status != "" {
		base = base.Where("management_state = ?", q.Status)
	}
	var rows []model.NotionPageBinding
	var total int64
	if q.Title != "" {
		var candidates []model.NotionPageBinding
		if e = base.Order("page_id").Find(&candidates).Error; e != nil {
			return nil, e
		}
		for _, r := range candidates {
			if strings.Contains(strings.ToLower(pageDTO(r).Title), strings.ToLower(q.Title)) {
				rows = append(rows, r)
			}
		}
		total = int64(len(rows))
		start := (q.Page - 1) * q.Limit
		if start > len(rows) {
			start = len(rows)
		}
		end := start + q.Limit
		if end > len(rows) {
			end = len(rows)
		}
		rows = rows[start:end]
	} else {
		if e = base.Count(&total).Error; e != nil {
			return nil, e
		}
		if e = base.Order("page_id").Offset((q.Page - 1) * q.Limit).Limit(q.Limit).Find(&rows).Error; e != nil {
			return nil, e
		}
	}
	result := &PageResult[PageDTO]{Items: []PageDTO{}, Total: total, Page: q.Page, Limit: q.Limit}
	for _, r := range rows {
		result.Items = append(result.Items, pageDTO(r))
	}
	if e := decoratePageVisibility(ctx, db, rows, result.Items); e != nil {
		return nil, e
	}
	return result, nil
}
func (s *Service) Runs(ctx context.Context, q ListQuery) (*PageResult[RunDTO], error) {
	db, e := s.db(ctx)
	if e != nil {
		return nil, e
	}
	q = normalizedQuery(q)
	var rows []model.NotionSyncRun
	var total int64
	base := db.Model(&model.NotionSyncRun{})
	if e = base.Count(&total).Error; e != nil {
		return nil, e
	}
	if e = base.Order("started_at DESC, run_id DESC").Offset((q.Page - 1) * q.Limit).Limit(q.Limit).Find(&rows).Error; e != nil {
		return nil, e
	}
	result := &PageResult[RunDTO]{Items: []RunDTO{}, Total: total, Page: q.Page, Limit: q.Limit}
	for _, r := range rows {
		result.Items = append(result.Items, runDTO(r))
	}
	return result, nil
}
func (s *Service) Run(ctx context.Context, id string) (*RunDTO, error) {
	db, e := s.db(ctx)
	if e != nil {
		return nil, e
	}
	var r model.NotionSyncRun
	if e = db.Where("run_id = ?", id).First(&r).Error; e != nil {
		if errors.Is(e, gorm.ErrRecordNotFound) {
			return nil, &Error{Code: "not_found", Message: "运行记录不存在", HTTPStatus: 404}
		}
		return nil, e
	}
	dto := runDTO(r)
	return &dto, nil
}
func (s *Service) Items(ctx context.Context, id string, q ListQuery) (*PageResult[ItemDTO], error) {
	db, e := s.db(ctx)
	if e != nil {
		return nil, e
	}
	q = normalizedQuery(q)
	var rows []model.NotionSyncRunItem
	var total int64
	base := db.Model(&model.NotionSyncRunItem{}).Where("run_id = ?", id)
	if e = base.Count(&total).Error; e != nil {
		return nil, e
	}
	if e = base.Order("item_id").Offset((q.Page - 1) * q.Limit).Limit(q.Limit).Find(&rows).Error; e != nil {
		return nil, e
	}
	result := &PageResult[ItemDTO]{Items: []ItemDTO{}, Total: total, Page: q.Page, Limit: q.Limit}
	for _, r := range rows {
		result.Items = append(result.Items, ItemDTO{ItemID: r.ItemID, PageID: r.PageID, ArticleID: articleID(r.ArticleID), Outcome: r.Outcome, Reason: r.Reason, Before: decodeJSON(r.BeforeJSON), After: decodeJSON(r.AfterJSON), Error: r.Error})
	}
	return result, nil
}
