package catalog

import (
	"context"
	"reflect"
	"sort"
	"strconv"

	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"gorm.io/gorm/clause"
)

type CatalogActivationItem struct {
	Kind           string `json:"kind"`
	Code           string `json:"code"`
	ExpectedTitle  string `json:"expected_title"`
	ExpectedStatus *int   `json:"expected_status"`
	ExpectedSort   *int   `json:"expected_sort"`
}
type CatalogActivationInput struct {
	Items                         []CatalogActivationItem `json:"items"`
	ExpectedNewlyPublicArticleIDs []string                `json:"expected_newly_public_article_ids"`
}
type CatalogActivationResult struct {
	ConfigRevision        uint64                  `json:"config_revision"`
	Items                 []CatalogActivationItem `json:"items"`
	NewlyPublicArticleIDs []string                `json:"newly_public_article_ids"`
}

// ActivateReviewedCatalog is a maintenance-only status command. It validates
// every reviewed tuple and the exact newly visible article set in one transaction.
func (s *Service) ActivateReviewedCatalog(ctx context.Context, token store.LeaseToken, sourceID string, expectedRevision uint64, input CatalogActivationInput) (*CatalogActivationResult, error) {
	if s.ds.DB() == nil {
		return nil, store.ErrSyncNotReady
	}
	if expectedRevision == 0 || len(input.Items) == 0 || input.ExpectedNewlyPublicArticleIDs == nil {
		return nil, catalogConflict("目录启用须提供版本、目录清单与明确的新增公开文章清单")
	}
	expected := map[uint64]bool{}
	for _, raw := range input.ExpectedNewlyPublicArticleIDs {
		id, err := strconv.ParseUint(raw, 10, 63)
		if err != nil || id == 0 || strconv.FormatUint(id, 10) != raw || expected[id] {
			return nil, catalogConflict("新增公开文章ID清单无效或重复")
		}
		expected[id] = true
	}
	result := &CatalogActivationResult{Items: input.Items, NewlyPublicArticleIDs: []string{}}
	err := store.InTransaction(ctx, s.ds, func(ds store.IStore) error {
		control, err := store.LockLease(ds, token)
		if err != nil {
			return err
		}
		if !control.Paused || !control.SourceWritesPaused || !control.BaselineFrozen {
			return syncTopicConflict("目录启用要求双暂停且历史基线已冻结")
		}
		var hint, src model.NotionSyncSource
		if err = ds.DB().Where("source_id = ?", sourceID).First(&hint).Error; err != nil {
			return err
		}
		if hint.ModuleCode == "" {
			return catalogConflict("来源尚未绑定模块")
		}
		if err = LockModules(ds, hint.ModuleCode); err != nil {
			return err
		}
		if err = ds.DB().Clauses(clause.Locking{Strength: "UPDATE"}).Where("source_id = ?", sourceID).First(&src).Error; err != nil {
			return err
		}
		if src.ConfigRevision != expectedRevision || src.ModuleCode != hint.ModuleCode {
			return syncTopicConflict("来源配置已变化")
		}
		result.ConfigRevision = src.ConfigRevision
		modules := map[string]*model.Module{}
		sections := map[string]*model.Section{}
		seen := map[string]bool{}
		changed := false
		for _, item := range input.Items {
			if item.Code == "" || item.ExpectedTitle == "" || item.ExpectedStatus == nil || item.ExpectedSort == nil || seen[item.Kind+":"+item.Code] {
				return catalogConflict("审核目录清单缺少前值或重复")
			}
			seen[item.Kind+":"+item.Code] = true
			switch item.Kind {
			case "module":
				row, e := ds.Modules().GetByCode(item.Code)
				if e != nil {
					return e
				}
				if row == nil || row.Code != src.ModuleCode || row.Code != item.Code || row.Title != item.ExpectedTitle || row.Status != *item.ExpectedStatus || row.Sort != *item.ExpectedSort {
					return catalogConflict("审核模块前值已变化或不属于来源")
				}
				modules[item.Code] = row
				changed = changed || row.Status != model.ModuleStatusNormal
			case "section":
				row, e := ds.Sections().GetByCode(item.Code)
				if e != nil {
					return e
				}
				if row == nil || row.ModuleCode != src.ModuleCode || row.Code != item.Code || row.Title != item.ExpectedTitle || row.Status != *item.ExpectedStatus || row.Sort != *item.ExpectedSort {
					return catalogConflict("审核章节前值已变化或不属于来源")
				}
				sections[item.Code] = row
				changed = changed || row.Status != model.SectionStatusNormal
			default:
				return catalogConflict("目录启用仅支持module或section")
			}
		}
		before, e := publicIDs(ctx, ds, src.ModuleCode)
		if e != nil {
			return e
		}
		// Reuse the normal state commands on the borrowed transaction store.
		// Their nested savepoints cannot commit the enclosing maintenance transaction.
		service := New(ds)
		for _, item := range input.Items {
			if row := modules[item.Code]; item.Kind == "module" && row.Status != model.ModuleStatusNormal {
				if _, e = service.ModuleStatus(ctx, row.Code, model.ModuleStatusNormal); e != nil {
					return e
				}
			}
			if row := sections[item.Code]; item.Kind == "section" && row.Status != model.SectionStatusNormal {
				if _, e = service.SectionStatus(ctx, row.Code, model.SectionStatusNormal); e != nil {
					return e
				}
			}
		}
		after, e := publicIDs(ctx, ds, src.ModuleCode)
		if e != nil {
			return e
		}
		actual := map[uint64]bool{}
		for id := range after {
			if !before[id] {
				actual[id] = true
			}
		}
		if !reflect.DeepEqual(expected, actual) {
			return catalogConflict("新增公开文章与审核清单不一致，请重新核对")
		}
		ids := make([]uint64, 0, len(actual))
		for id := range actual {
			ids = append(ids, id)
		}
		sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
		for _, id := range ids {
			result.NewlyPublicArticleIDs = append(result.NewlyPublicArticleIDs, strconv.FormatUint(id, 10))
		}
		if !changed {
			return nil
		}
		if src.ConfigRevision == ^uint64(0) {
			return syncTopicConflict("来源配置版本已耗尽")
		}
		result.ConfigRevision = src.ConfigRevision + 1
		return ds.DB().Model(&src).Update("config_revision", result.ConfigRevision).Error
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}
func publicIDs(ctx context.Context, ds store.IStore, moduleCode string) (map[uint64]bool, error) {
	var ids []uint64
	if err := store.PublishedArticles(ctx, ds.DB()).Where("module.code = ?", moduleCode).Pluck("article.id", &ids).Error; err != nil {
		return nil, err
	}
	result := map[uint64]bool{}
	for _, id := range ids {
		result[id] = true
	}
	return result, nil
}
