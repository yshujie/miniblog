package notionsync

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/yshujie/miniblog/internal/miniblog/biz/catalog"
	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"reflect"
	"strings"
	"time"
)

func configOf(v model.NotionSyncSource) SourceConfig {
	var c SourceConfig
	_ = json.Unmarshal([]byte(v.PropertyMappingJSON), &c)
	_ = json.Unmarshal([]byte(v.StatusMappingJSON), &c.StateOptionIDs)
	return c
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
