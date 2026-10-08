package article

import (
	"context"
	"github.com/yshujie/miniblog/internal/miniblog/biz/catalog"
	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"gorm.io/gorm/clause"
)

// EnsureSyncedTopics includes options without pages. Ordering remains locally owned.
func (b *articleBiz) EnsureSyncedTopics(ctx context.Context, token store.LeaseToken, sourceID string, expectedRevision uint64, propertyID string, options []TopicOption) (results []TopicResult, err error) {
	if b.ds.DB() == nil {
		return nil, store.ErrSyncNotReady
	}
	err = store.InTransaction(ctx, b.ds, func(ds store.IStore) error {
		c, e := store.LockLease(ds, token)
		if e != nil {
			return e
		}
		if c.Paused || c.SourceWritesPaused {
			return syncConflict("同步暂停或维护中")
		}
		var hint model.NotionSyncSource
		if e = ds.DB().Where("source_id = ?", sourceID).First(&hint).Error; e != nil {
			return e
		}
		if e = catalog.LockModules(ds, hint.ModuleCode); e != nil {
			return e
		}
		var src model.NotionSyncSource
		if e = ds.DB().Clauses(clause.Locking{Strength: "UPDATE"}).Where("source_id = ?", sourceID).First(&src).Error; e != nil {
			return e
		}
		if src.ConfigRevision != expectedRevision || src.ModuleCode != hint.ModuleCode || !src.Enabled {
			return syncConflict("来源配置已变化")
		}
		seen := map[string]bool{}
		for _, option := range options {
			if seen[option.ID] {
				return invalid("重复主题option_id")
			}
			seen[option.ID] = true
		}
		if !seen[""] {
			options = append(append([]TopicOption(nil), options...), TopicOption{ID: "", Name: "未分类"})
		}
		for _, option := range options {
			if e = ds.DB().SavePoint("sync_topic").Error; e != nil {
				return e
			}
			p, topicErr := catalog.EnsureSyncedTheme(ds, &src, propertyID, option.ID, option.Name)
			if topicErr != nil {
				if e = ds.DB().RollbackTo("sync_topic").Error; e != nil {
					return e
				}
				if e = catalog.RecordBlockedSyncedTheme(ds, &src, propertyID, option.ID, option.Name, topicErr.Error()); e != nil {
					return e
				}
				results = append(results, TopicResult{OptionID: option.ID, Outcome: "blocked", Reason: topicErr.Error()})
				continue
			}
			results = append(results, TopicResult{OptionID: option.ID, SectionCode: p.Section.Code, Outcome: "bound"})
		}
		return nil
	})
	return
}
