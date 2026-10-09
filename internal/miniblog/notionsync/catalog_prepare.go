package notionsync

import (
	"context"
	"errors"
	"time"

	"github.com/yshujie/miniblog/internal/miniblog/biz/catalog"
	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"gorm.io/gorm"
)

type CatalogPrepareInput struct {
	SourceID               string                    `json:"source_id"`
	ExpectedConfigRevision uint64                    `json:"expected_config_revision"`
	ReuseMap               []catalog.TopicReuseInput `json:"reuse_map,omitempty"`
}

type CatalogReuseInput = catalog.TopicReuseInput

type CatalogPrepareResult struct {
	RunID          string            `json:"run_id"`
	SourceID       string            `json:"source_id"`
	ConfigRevision uint64            `json:"config_revision"`
	Items          []TopicBindingDTO `json:"items"`
}

// PrepareCatalog is an operator-only maintenance command. It fetches a single
// schema and prepares its catalog, never querying or projecting Notion pages.
// Normal REST Trigger deliberately still only accepts dry_run or sync.
func (s *Service) PrepareCatalog(ctx context.Context, input CatalogPrepareInput) (*CatalogPrepareResult, error) {
	id := normalizeID(input.SourceID)
	if _, ok := allowedSources[id]; !ok {
		return nil, invalid("数据源不在五库白名单")
	}
	if input.ExpectedConfigRevision == 0 {
		return nil, invalid("目录准备需已配置来源版本 expected_config_revision")
	}
	db, err := s.db(ctx)
	if err != nil {
		return nil, err
	}
	var row model.NotionSyncSource
	if err := db.Where("source_id = ?", id).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, invalid("请先配置来源并核对 schema 映射")
		}
		return nil, err
	}
	if row.ConfigRevision != input.ExpectedConfigRevision {
		return nil, conflict("来源配置已变化，请重新核对")
	}
	if row.ModuleCode == "" || !configComplete(configOf(row)) {
		return nil, invalid("请先配置源模块并核对字段映射")
	}
	result := &CatalogPrepareResult{SourceID: id, ConfigRevision: row.ConfigRevision, Items: []TopicBindingDTO{}}
	ctx, cancel := context.WithTimeout(ctx, 4*time.Minute)
	defer cancel()
	runID, err := s.synchronousRun(ctx, "catalog_prepare", func(active context.Context, run model.NotionSyncRun, token store.LeaseToken, counts *RunCounts) error {
		control, err := store.NewNotionSyncRepository(s.ds.DB()).Control(active)
		if err != nil {
			return err
		}
		if !control.Paused || !control.SourceWritesPaused || !control.BaselineFrozen {
			return conflict("目录准备需冻结基线并暂停同步和来源写入")
		}
		if err := s.fenced(active, token, func(tx *gorm.DB, _ *model.NotionSyncControl) error {
			return tx.Model(&model.NotionSyncRun{}).Where("run_id = ?", run.ID).Update("phase", "catalog_prepare").Error
		}); err != nil {
			return err
		}
		schema, err := s.client.RetrieveDataSource(active, id)
		if err != nil {
			return err
		}
		if err := validateSchemaIdentity(schema, id); err != nil {
			return err
		}
		cfg := configOf(row)
		if err := validateConfig(schema, cfg); err != nil {
			return err
		}
		topic, _ := schemaProperty(schema, cfg.TopicPropertyID)
		options := []catalog.TopicOption{{ID: "", Name: "未分类"}}
		optionIDs := []string{""}
		for _, option := range topic.Select.Options {
			options = append(options, catalog.TopicOption{ID: option.ID, Name: option.Name})
			optionIDs = append(optionIDs, option.ID)
		}
		prepared, revision, err := catalog.New(s.ds).PrepareSyncedTopicsWithReuse(active, token, id, input.ExpectedConfigRevision, cfg.TopicPropertyID, options, input.ReuseMap)
		if err != nil {
			return err
		}
		result.ConfigRevision = revision
		for _, item := range prepared {
			counts.Seen++
			if item.Outcome == "blocked" {
				counts.Blocked++
				counts.Failed++
			} else if item.Outcome == "created" {
				counts.Created++
			} else {
				counts.Unchanged++
			}
			if err := s.recordCatalogItem(active, token, run.ID, id, item.OptionID, "catalog_prepare", item.Reason, item); err != nil {
				return err
			}
		}
		var bindings []model.NotionCatalogBinding
		if err := s.ds.DB().WithContext(active).Where("source_id = ? AND theme_property_id = ? AND option_id IN ?", id, cfg.TopicPropertyID, optionIDs).Order("id").Find(&bindings).Error; err != nil {
			return err
		}
		for _, binding := range bindings {
			result.Items = append(result.Items, bindingDTO(binding))
		}
		return nil
	})
	result.RunID = runID
	return result, err
}
