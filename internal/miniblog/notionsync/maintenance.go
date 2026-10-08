package notionsync

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"github.com/yshujie/miniblog/internal/miniblog/biz/article"
	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/source"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
)

func (s *Service) syncTopics(ctx context.Context, run model.NotionSyncRun, token store.LeaseToken, src scannedSource, counts *RunCounts) error {
	if src.SchemaError != nil {
		return nil
	}
	property, ok := schemaProperty(src.Schema, src.Config.TopicPropertyID)
	if !ok || property.Type != "select" {
		return nil
	}
	options := []article.TopicOption{{ID: "", Name: "未分类"}}
	for _, option := range property.Select.Options {
		options = append(options, article.TopicOption{ID: option.ID, Name: option.Name})
	}
	results, e := article.NewForSync(s.ds, s.opts.Author).EnsureSyncedTopics(ctx, token, src.Row.ID, src.Row.ConfigRevision, src.Config.TopicPropertyID, options)
	if e != nil {
		return e
	}
	for _, result := range results {
		if result.Outcome == "blocked" {
			counts.Blocked++
		}
		if e := s.recordCatalogItem(ctx, token, run.ID, src.Row.ID, result.OptionID, result.Outcome, result.Reason, result); e != nil {
			return e
		}
	}
	return nil
}
func (s *Service) previewTopics(ctx context.Context, run model.NotionSyncRun, token store.LeaseToken, src scannedSource) error {
	property, ok := schemaProperty(src.Schema, src.Config.TopicPropertyID)
	if !ok || property.Type != "select" {
		return nil
	}
	previewOptions := append([]source.NotionOption{{Name: "未分类"}}, property.Select.Options...)
	for _, option := range previewOptions {
		code, reason, e := s.catalogPlan(ctx, src, option.ID, option.Name)
		if e != nil {
			return e
		}
		if e := s.recordCatalogItem(ctx, token, run.ID, src.Row.ID, option.ID, "catalog_preview", reason, map[string]string{"option_name": option.Name, "section_code": code}); e != nil {
			return e
		}
	}

	return nil
}
func (s *Service) recordCatalogItem(ctx context.Context, token store.LeaseToken, runID, sourceID, optionID, outcome, reason string, after interface{}) error {
	return s.fenced(ctx, token, func(tx *gorm.DB, _ *model.NotionSyncControl) error {
		digest := sha256.Sum256([]byte(optionID))
		key := "catalog:" + sourceID + ":" + hex.EncodeToString(digest[:])
		item := model.NotionSyncRunItem{RunID: runID, ItemID: key, Outcome: outcome, Reason: reason, AfterJSON: jsonText(map[string]interface{}{"source_id": sourceID, "option_id": optionID, "details": after})}
		return tx.Clauses(clause.OnConflict{UpdateAll: true}).Create(&item).Error
	})
}

// Keep bootstrap journals and unresolved runs. Only fully resolved regular runs expire.
func (s *Service) pruneResolvedRuns(ctx context.Context, token store.LeaseToken) error {
	return s.fenced(ctx, token, func(tx *gorm.DB, _ *model.NotionSyncControl) error {
		var ids []string
		if e := tx.Model(&model.NotionSyncRun{}).Where("started_at < ? AND status = ? AND mode IN ?", time.Now().UTC().Add(-90*24*time.Hour), "completed", []string{"dry_run", "sync"}).Pluck("run_id", &ids).Error; e != nil {
			return e
		}
		if len(ids) == 0 {
			return nil
		}
		if e := tx.Where("run_id IN ?", ids).Delete(&model.NotionSyncRunItem{}).Error; e != nil {
			return e
		}
		return tx.Where("run_id IN ?", ids).Delete(&model.NotionSyncRun{}).Error
	})
}

func (s *Service) recordSchemaFailure(ctx context.Context, token store.LeaseToken, sourceID string, inTrash bool, err error) error {
	return s.fenced(ctx, token, func(tx *gorm.DB, _ *model.NotionSyncControl) error {
		reason := "source_schema_changed"
		if inTrash {
			reason = "source_in_trash"
		}
		return tx.Model(&model.NotionPageBinding{}).Where("source_id = ? AND management_state = ?", sourceID, model.NotionManagementManaged).Updates(map[string]interface{}{"needs_revalidation": true, "publish_block_reason": reason, "last_error": err.Error(), "revision": gorm.Expr("revision + 1")}).Error
	})
}

// A lost lease may finish its own audit record, never business data or a newer run.
func (s *Service) finishRun(ctx context.Context, run model.NotionSyncRun, token store.LeaseToken, updates map[string]interface{}, nextRun bool) error {
	e := s.fenced(ctx, token, func(tx *gorm.DB, c *model.NotionSyncControl) error {
		if e := tx.Model(&model.NotionSyncRun{}).Where("run_id = ? AND lease_epoch = ? AND status = ?", run.ID, token.Epoch, "running").Updates(updates).Error; e != nil {
			return e
		}
		if nextRun {
			return tx.Model(c).Update("next_run_at", time.Now().UTC().Add(s.opts.Interval)).Error
		}
		return nil
	})
	if e != nil {
		// Operator cancellation or an abandoned run already marked by a new worker wins.
		audit := map[string]interface{}{"status": "abandoned", "phase": "finished", "finished_at": time.Now().UTC(), "error": "worker lease ended before run completion", "counts_json": updates["counts_json"]}
		if logErr := s.ds.DB().WithContext(ctx).Model(&model.NotionSyncRun{}).Where("run_id = ? AND lease_epoch = ? AND status = ?", run.ID, token.Epoch, "running").Updates(audit).Error; logErr != nil {
			return logErr
		}
	}
	return e
}

func (s *Service) catalogPlan(ctx context.Context, src scannedSource, optionID, optionName string) (string, string, error) {
	if src.SchemaError != nil {
		return "", "source_schema_changed", nil
	}
	if src.Row.ModuleCode == "" {
		return "", "source_module_unconfigured", nil
	}
	db := s.ds.DB().WithContext(ctx)
	module, e := store.NewStore(db).Modules().GetByCode(src.Row.ModuleCode)
	if e != nil {
		return "", "", e
	}
	if module == nil {
		return "", "catalog_binding_invalid", nil
	}
	if optionID == "" {
		optionName = "未分类"
	}
	var binding model.NotionCatalogBinding
	e = db.Where("data_source_id = ? AND theme_property_id = ? AND option_id = ?", src.Row.DataSourceID, src.Config.TopicPropertyID, optionID).First(&binding).Error
	if e != nil && !errors.Is(e, gorm.ErrRecordNotFound) {
		return "", "", e
	}
	code := ""
	reason := "would_create"
	var sectionID uint64
	if e == nil {
		if binding.SourceID != src.Row.ID || binding.Status != model.NotionCatalogBound || binding.SectionCode == nil {
			return "", "catalog_binding_invalid", nil
		}
		section, e := store.NewStore(db).Sections().GetByCode(*binding.SectionCode)
		if e != nil {
			return "", "", e
		}
		if section == nil || section.ModuleCode != module.Code {
			return "", "catalog_binding_invalid", nil
		}
		code = section.Code
		sectionID = section.ID
		reason = "bound"
		if section.Title != optionName {
			reason = "would_rename"
		}
		if section.Status != model.SectionStatusNormal || module.Status != model.ModuleStatusNormal {
			return code, "catalog_directory_hidden", nil
		}
	} else if module.Status != model.ModuleStatusNormal {
		return "", "catalog_directory_hidden", nil
	}
	var collisions int64
	query := db.Model(&model.Section{}).Where("module_code = ? AND title = ?", module.Code, optionName)
	if sectionID != 0 {
		query = query.Where("id <> ?", sectionID)
	}
	if e := query.Count(&collisions).Error; e != nil {
		return "", "", e
	}
	if collisions > 0 {
		return code, "catalog_same_name_conflict", nil
	}
	return code, reason, nil
}
