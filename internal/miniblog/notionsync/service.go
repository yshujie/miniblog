package notionsync

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/yshujie/miniblog/internal/miniblog/biz/catalog"
	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/source"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var allowedSources = map[string]string{
	"2bf330bd-ddf1-80a6-aa49-000bbd1e154b": "Go",
	"d579f619-4f1c-4e7e-9593-ab6529fc63d0": "DDD",
	"cfa962f3-12be-40aa-95eb-9122186bd4ba": "数据库",
	"03d24108-5172-4e8f-8cfd-e3bd52e437af": "数据结构&算法",
	"d13097e3-b78a-4b6e-8848-979749d4b2f3": "项目开发",
}

type Service struct {
	ds        store.IStore
	opts      Options
	client    source.SyncNotionClient
	mu        sync.Mutex
	stop      context.CancelFunc
	runCancel context.CancelFunc
	ctx       context.Context
	wg        sync.WaitGroup
	stopping  bool
	active    map[string]context.CancelFunc
}

func New(ds store.IStore, opts Options) *Service {
	if opts.Interval <= 0 {
		opts.Interval = 5 * time.Minute
	}
	if opts.LeaseDuration <= 0 {
		opts.LeaseDuration = 60 * time.Second
	}
	if opts.HeartbeatInterval <= 0 {
		opts.HeartbeatInterval = 15 * time.Second
	}
	if opts.OwnerID == "" {
		opts.OwnerID = newID()
	}
	if opts.Client == nil {
		opts.Client = source.NewNotionSyncClient(os.Getenv("MINIBLOG_NOTION_TOKEN"))
	}
	s := &Service{ds: ds, opts: opts, client: opts.Client, ctx: context.Background(), active: map[string]context.CancelFunc{}}
	if client, ok := s.client.(*source.HTTPNotionSyncClient); ok {
		client.SetCooldownObserver(s.observeCooldown)
	}
	return s
}
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
func (s *Service) db(ctx context.Context) (*gorm.DB, error) {
	if s.ds == nil || s.ds.DB() == nil {
		return nil, unavailable("同步数据库未配置")
	}
	db := s.ds.DB().WithContext(ctx)
	for _, m := range []interface{}{&model.NotionSyncControl{}, &model.NotionSyncSource{}, &model.NotionCatalogBinding{}, &model.NotionPageBinding{}, &model.NotionSyncRun{}, &model.NotionSyncRunItem{}} {
		if !db.Migrator().HasTable(m) {
			return nil, unavailable("请先完成 Notion 同步数据库迁移")
		}
	}
	return db, nil
}
func newID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 15) | 64
	b[8] = (b[8] & 63) | 128
	v := hex.EncodeToString(b[:])
	return v[:8] + "-" + v[8:12] + "-" + v[12:16] + "-" + v[16:20] + "-" + v[20:]
}
func normalizeID(id string) string {
	v := strings.ToLower(strings.ReplaceAll(id, "-", ""))
	if len(v) != 32 {
		return id
	}
	return v[:8] + "-" + v[8:12] + "-" + v[12:16] + "-" + v[16:20] + "-" + v[20:]
}
func normalizePageID(id string) string { return strings.ToLower(strings.ReplaceAll(id, "-", "")) }
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
func jsonText(v interface{}) string { b, _ := json.Marshal(v); return string(b) }
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
func stateName(v int) string {
	switch v {
	case 1:
		return "draft"
	case 2:
		return "published"
	case 3:
		return "unpublished"
	case 4:
		return "archived"
	}
	return ""
}
func stateValue(v string) int {
	switch v {
	case "draft":
		return 1
	case "published":
		return 2
	case "unpublished":
		return 3
	case "archived":
		return 4
	}
	return 0
}
func articleID(v *uint64) *string {
	if v == nil {
		return nil
	}
	result := strconv.FormatUint(*v, 10)
	return &result
}
func configOf(v model.NotionSyncSource) SourceConfig {
	var c SourceConfig
	_ = json.Unmarshal([]byte(v.PropertyMappingJSON), &c)
	_ = json.Unmarshal([]byte(v.StatusMappingJSON), &c.StateOptionIDs)
	return c
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
func (s *Service) UpdateControl(ctx context.Context, in ControlInput) (*StatusDTO, error) {
	db, e := s.db(ctx)
	if e != nil {
		return nil, e
	}
	if in.Enabled != nil && *in.Enabled != s.opts.Enabled {
		return nil, invalid("同步总开关由运行配置控制，请使用暂停")
	}
	revoke := (in.Paused != nil && *in.Paused) || (in.SourceWritesPaused != nil && *in.SourceWritesPaused)
	e = db.Transaction(func(tx *gorm.DB) error {
		if e := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&model.NotionSyncControl{ID: 1}).Error; e != nil {
			return e
		}
		var c model.NotionSyncControl
		if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&c, 1).Error; e != nil {
			return e
		}
		unpause := (in.Paused != nil && !*in.Paused && c.Paused) || (in.SourceWritesPaused != nil && !*in.SourceWritesPaused && c.SourceWritesPaused)
		if unpause && c.CurrentRunID != "" {
			var run model.NotionSyncRun
			e := tx.Where("run_id = ?", c.CurrentRunID).First(&run).Error
			if e != nil && !errors.Is(e, gorm.ErrRecordNotFound) {
				return e
			}
			now, e := store.DatabaseNow(tx)
			if e != nil {
				return e
			}
			if (run.Mode == "bootstrap_apply" || run.Mode == "catalog_prepare") && run.Status == "running" && c.LeaseUntil != nil && c.LeaseUntil.After(now) {
				return conflict("维护任务仍在进行，请等待完成或先暂停取消任务再解除维护")
			}
		}
		updates := map[string]interface{}{}
		if in.Paused != nil {
			updates["paused"] = *in.Paused
		}
		if in.SourceWritesPaused != nil {
			updates["source_writes_paused"] = *in.SourceWritesPaused
		}
		if revoke {
			if c.LeaseEpoch == ^uint64(0) {
				return conflict("lease epoch exhausted")
			}
			updates["lease_epoch"] = c.LeaseEpoch + 1
			updates["lease_owner"] = ""
			updates["lease_until"] = nil
			updates["current_run_id"] = ""
			if c.CurrentRunID != "" {
				now := time.Now().UTC()
				if e := tx.Model(&model.NotionSyncRun{}).Where("run_id = ? AND status = ?", c.CurrentRunID, "running").Updates(map[string]interface{}{"status": "cancelled", "phase": "finished", "finished_at": now, "error": "paused by operator"}).Error; e != nil {
					return e
				}
			}
		}
		if len(updates) == 0 {
			return nil
		}
		return tx.Model(&c).Updates(updates).Error
	})
	if e != nil {
		return nil, e
	}
	if revoke {
		s.mu.Lock()
		for _, cancel := range s.active {
			cancel()
		}
		s.mu.Unlock()
	}
	return s.Status(ctx)
}
func (s *Service) UpdateSource(ctx context.Context, id string, in SourceInput) (*SourceDTO, error) {
	id = normalizeID(id)
	label, ok := allowedSources[id]
	if !ok {
		return nil, invalid("数据源不在五库白名单")
	}
	db, e := s.db(ctx)
	if e != nil {
		return nil, e
	}
	if in.Config != nil && in.ExpectedConfigRevision == nil {
		return nil, invalid("修改字段映射需 expected_config_revision")
	}
	if in.Config != nil {
		if e = validateConfigShape(*in.Config); e != nil {
			return nil, e
		}
	}
	var row model.NotionSyncSource
	e = db.Transaction(func(tx *gorm.DB) error {
		if e := lockConfigurationControl(tx); e != nil {
			return e
		}
		var hint model.NotionSyncSource
		e := tx.Where("source_id = ?", id).First(&hint).Error
		if e != nil && !errors.Is(e, gorm.ErrRecordNotFound) {
			return e
		}
		if e := catalog.LockModules(store.NewStore(tx), hint.ModuleCode, in.ModuleCode); e != nil {
			return e
		}
		e = tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("source_id = ?", id).First(&row).Error
		created := errors.Is(e, gorm.ErrRecordNotFound)
		if created {
			row = model.NotionSyncSource{ID: id, DataSourceID: id, Label: label, Health: "not_configured"}
		} else if e != nil {
			return e
		}
		if in.ExpectedConfigRevision != nil && *in.ExpectedConfigRevision != row.ConfigRevision {
			return conflict("源配置已变化，请刷新")
		}
		before := row
		mappingChanged := in.Config != nil && !reflect.DeepEqual(configOf(row), *in.Config)
		moduleChanged := in.ModuleCode != "" && row.ModuleCode != "" && row.ModuleCode != in.ModuleCode
		if moduleChanged && in.ExpectedConfigRevision == nil {
			return invalid("模块重绑需 expected_config_revision")
		}
		if in.ModuleCode != "" {
			m, e := store.NewStore(tx).Modules().GetByCode(in.ModuleCode)
			if e != nil {
				return e
			}
			if m == nil {
				return invalid("模块不存在")
			}
			row.ModuleCode = in.ModuleCode
		}
		if strings.TrimSpace(in.Label) != "" {
			row.Label = strings.TrimSpace(in.Label)
		}
		if in.Enabled != nil {
			if *in.Enabled && (row.ModuleCode == "" || (!configComplete(configOf(row)) && in.Config == nil)) {
				return conflict("请先设置模块并完成字段映射预览")
			}
			row.Enabled = *in.Enabled
		}
		if mappingChanged {
			row.PropertyMappingJSON = jsonText(in.Config)
			row.StatusMappingJSON = jsonText(in.Config.StateOptionIDs)
		}
		if !created && before.Label == row.Label && before.ModuleCode == row.ModuleCode && before.Enabled == row.Enabled && !mappingChanged {
			return nil
		}
		if row.ConfigRevision == ^uint64(0) {
			return conflict("配置版本已耗尽")
		}
		row.ConfigRevision++
		if e := tx.Save(&row).Error; e != nil {
			return e
		}
		if moduleChanged {
			if e := tx.Model(&model.NotionCatalogBinding{}).Where("source_id = ?", row.ID).Updates(map[string]interface{}{"status": model.NotionCatalogBlocked, "reason": "module_binding_changed"}).Error; e != nil {
				return e
			}
		}
		if moduleChanged || mappingChanged {
			return tx.Model(&model.NotionPageBinding{}).Where("source_id = ? AND management_state = ?", row.ID, model.NotionManagementManaged).Update("needs_revalidation", true).Error
		}
		return nil
	})
	if e != nil {
		return nil, e
	}
	dto, e := s.sourceDTO(db, row)
	return &dto, e
}
func (s *Service) BindCatalog(ctx context.Context, in BindingInput) (*TopicBindingDTO, error) {
	db, e := s.db(ctx)
	if e != nil {
		return nil, e
	}
	if in.ExpectedConfigRevision == nil {
		return nil, invalid("目录绑定需 expected_config_revision")
	}
	var result model.NotionCatalogBinding
	e = db.Transaction(func(tx *gorm.DB) error {
		if e := lockConfigurationControl(tx); e != nil {
			return e
		}
		var hint model.NotionSyncSource
		if e := tx.Where("source_id = ?", normalizeID(in.SourceID)).First(&hint).Error; e != nil {
			return e
		}
		if e := catalog.LockModules(store.NewStore(tx), hint.ModuleCode); e != nil {
			return e
		}
		var src model.NotionSyncSource
		if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("source_id = ?", normalizeID(in.SourceID)).First(&src).Error; e != nil {
			return e
		}
		if in.ExpectedConfigRevision != nil && *in.ExpectedConfigRevision != src.ConfigRevision {
			return conflict("源配置已变化，请刷新")
		}
		query := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("source_id = ? AND option_id = ?", src.ID, in.OptionID)
		if in.BindingID != "" {
			query = query.Where("id = ?", in.BindingID)
		}
		if e := query.First(&result).Error; e != nil {
			return e
		}
		section, e := store.NewStore(tx).Sections().GetByCode(in.SectionCode)
		if e != nil {
			return e
		}
		if section == nil || section.ModuleCode != src.ModuleCode || section.Status != model.SectionStatusNormal {
			return invalid("请选择源模块下有效章节")
		}
		if e := catalog.CheckSyncedThemeSectionAvailable(store.NewStore(tx), section.Code, result.ID); e != nil {
			return e
		}
		if result.SectionCode != nil && *result.SectionCode == section.Code && result.Status == model.NotionCatalogBound && result.Reason == "" {
			return nil
		}
		result.SectionCode = &section.Code
		result.Status = model.NotionCatalogBound
		result.Reason = ""
		if e := tx.Save(&result).Error; e != nil {
			return e
		}
		if e := tx.Model(&src).Update("config_revision", src.ConfigRevision+1).Error; e != nil {
			return e
		}
		return tx.Model(&model.NotionPageBinding{}).Where("source_id = ? AND management_state = ?", src.ID, model.NotionManagementManaged).Update("needs_revalidation", true).Error
	})
	if e != nil {
		return nil, e
	}
	dto := bindingDTO(result)
	return &dto, nil
}

// Configuration and directory writers share control -> module -> source order.
// A remote bootstrap write cannot race an operator changing its reviewed mapping.
func lockConfigurationControl(tx *gorm.DB) error {
	var control model.NotionSyncControl
	if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&control, 1).Error; e != nil {
		return e
	}
	if control.CurrentRunID != "" {
		var run model.NotionSyncRun
		e := tx.Where("run_id = ?", control.CurrentRunID).First(&run).Error
		if e != nil && !errors.Is(e, gorm.ErrRecordNotFound) {
			return e
		}
		now, e := store.DatabaseNow(tx)
		if e != nil {
			return e
		}
		if (run.Mode == "bootstrap_apply" || run.Mode == "catalog_prepare") && run.Status == "running" && control.LeaseUntil != nil && control.LeaseUntil.After(now) {
			return conflict("维护任务正在进行，请等待完成后修改配置")
		}
	}
	return nil
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

func (s *Service) observeCooldown(ctx context.Context, until time.Time) error {
	token, ok := ctx.Value(leaseContextKey{}).(store.LeaseToken)
	if !ok {
		return nil
	}
	return s.fenced(ctx, token, func(tx *gorm.DB, c *model.NotionSyncControl) error {
		return tx.Model(c).Update("cooldown_until", until).Error
	})
}
