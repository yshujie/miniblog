package catalog

import (
	"context"
	"errors"
	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"github.com/yshujie/miniblog/internal/pkg/errno"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"reflect"
)

type TopicOption struct{ ID, Name string }
type TopicResult struct {
	BindingID                              uint64
	OptionID, SectionCode, Outcome, Reason string
}

func syncTopicConflict(message string) error {
	return &errno.Errno{HTTP: 409, Code: "NotionSyncConflict", Message: message}
}

// EnsureSyncedTopics includes options without pages. Ordering remains locally owned.
func (s *Service) EnsureSyncedTopics(ctx context.Context, token store.LeaseToken, sourceID string, expectedRevision uint64, propertyID string, options []TopicOption) ([]TopicResult, error) {
	results, _, err := s.syncedTopics(ctx, token, sourceID, expectedRevision, propertyID, options, false)
	return results, err
}

// PrepareSyncedTopics prepares only directory bindings in a frozen maintenance window.
// A disabled source is allowed; no page binding or article is created or adopted.
func (s *Service) PrepareSyncedTopics(ctx context.Context, token store.LeaseToken, sourceID string, expectedRevision uint64, propertyID string, options []TopicOption) ([]TopicResult, uint64, error) {
	return s.syncedTopics(ctx, token, sourceID, expectedRevision, propertyID, options, true)
}

func (s *Service) syncedTopics(ctx context.Context, token store.LeaseToken, sourceID string, expectedRevision uint64, propertyID string, options []TopicOption, prepare bool) (results []TopicResult, revision uint64, err error) {
	if s.ds.DB() == nil {
		return nil, 0, store.ErrSyncNotReady
	}
	err = store.InTransaction(ctx, s.ds, func(ds store.IStore) error {
		c, e := store.LockLease(ds, token)
		if e != nil {
			return e
		}
		if prepare {
			if !c.Paused || !c.SourceWritesPaused || !c.BaselineFrozen {
				return syncTopicConflict("目录准备要求暂停同步、冻结来源写入且历史基线已冻结")
			}
		} else if c.Paused || c.SourceWritesPaused {
			return syncTopicConflict("同步暂停或维护中")
		}
		var hint model.NotionSyncSource
		if e = ds.DB().Where("source_id = ?", sourceID).First(&hint).Error; e != nil {
			return e
		}
		if e = LockModules(ds, hint.ModuleCode); e != nil {
			return e
		}
		var src model.NotionSyncSource
		if e = ds.DB().Clauses(clause.Locking{Strength: "UPDATE"}).Where("source_id = ?", sourceID).First(&src).Error; e != nil {
			return e
		}
		if src.ConfigRevision != expectedRevision || src.ModuleCode != hint.ModuleCode || (!prepare && !src.Enabled) {
			return syncTopicConflict("来源配置已变化")
		}
		revision = src.ConfigRevision
		if src.ModuleCode == "" {
			return syncTopicConflict("来源尚未绑定模块")
		}
		var before topicDirectoryState
		if prepare {
			before, e = topicCatalogState(ds, &src, propertyID)
			if e != nil {
				return e
			}
		}
		results, e = ensureTopics(ds, &src, propertyID, options)
		if e != nil {
			return e
		}
		if !prepare {
			return nil
		}
		after, e := topicCatalogState(ds, &src, propertyID)
		if e != nil {
			return e
		}
		if reflect.DeepEqual(before, after) {
			return nil
		}
		if src.ConfigRevision == ^uint64(0) {
			return syncTopicConflict("来源配置版本已耗尽")
		}
		revision = src.ConfigRevision + 1
		return ds.DB().Model(&src).Update("config_revision", revision).Error
	})
	if err != nil {
		return nil, 0, err
	}
	return
}

type topicDirectoryState struct {
	Sections []model.Section
	Bindings []model.NotionCatalogBinding
}

func topicCatalogState(ds store.IStore, src *model.NotionSyncSource, propertyID string) (state topicDirectoryState, err error) {
	// Timestamps do not constitute a catalog change; only observable directory/binding values do.
	err = ds.DB().Select("id", "code", "title", "module_code", "sort", "status").Where("module_code = ?", src.ModuleCode).Order("id").Find(&state.Sections).Error
	if err != nil {
		return
	}
	err = ds.DB().Select("id", "source_id", "data_source_id", "theme_property_id", "option_id", "option_name", "section_code", "status", "reason").Where("data_source_id = ? AND theme_property_id = ?", src.DataSourceID, propertyID).Order("id").Find(&state.Bindings).Error
	return
}
func ensureTopics(ds store.IStore, src *model.NotionSyncSource, propertyID string, options []TopicOption) ([]TopicResult, error) {
	seen := map[string]bool{}
	for _, option := range options {
		if seen[option.ID] {
			return nil, invalid("重复主题option_id")
		}
		seen[option.ID] = true
	}
	if !seen[""] {
		options = append(append([]TopicOption(nil), options...), TopicOption{ID: "", Name: "未分类"})
	}
	results := make([]TopicResult, 0, len(options))
	for _, option := range options {
		var before model.NotionCatalogBinding
		e := ds.DB().Where("data_source_id = ? AND theme_property_id = ? AND option_id = ?", src.DataSourceID, propertyID, option.ID).First(&before).Error
		if e != nil && !errors.Is(e, gorm.ErrRecordNotFound) {
			return nil, e
		}
		newBinding := errors.Is(e, gorm.ErrRecordNotFound)
		if e = ds.DB().SavePoint("sync_topic").Error; e != nil {
			return nil, e
		}
		p, topicErr := EnsureSyncedTheme(ds, src, propertyID, option.ID, option.Name)
		result := TopicResult{OptionID: option.ID, Outcome: "bound"}
		if topicErr != nil {
			if e = ds.DB().RollbackTo("sync_topic").Error; e != nil {
				return nil, e
			}
			var businessErr *errno.Errno
			if !errors.As(topicErr, &businessErr) {
				return nil, topicErr
			}
			if e = RecordBlockedSyncedTheme(ds, src, propertyID, option.ID, option.Name, topicErr.Error()); e != nil {
				return nil, e
			}
			result.Outcome = "blocked"
			result.Reason = topicErr.Error()
		} else {
			result.SectionCode = p.Section.Code
			if newBinding {
				result.Outcome = "created"
			}
		}
		var binding model.NotionCatalogBinding
		if e = ds.DB().Where("data_source_id = ? AND theme_property_id = ? AND option_id = ?", src.DataSourceID, propertyID, option.ID).First(&binding).Error; e != nil {
			return nil, e
		}
		result.BindingID = binding.ID
		results = append(results, result)
	}
	return results, nil
}
