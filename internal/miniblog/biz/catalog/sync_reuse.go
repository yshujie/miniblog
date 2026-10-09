package catalog

import (
	"errors"
	"strings"

	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type reviewedTopic struct {
	input   TopicReuseInput
	section *model.Section
	binding model.NotionCatalogBinding
	exists  bool
}

// applyReviewedTopicReuse borrows the maintenance transaction. Validate the whole
// review before binding anything; the caller already holds lease, module and source locks.
func applyReviewedTopicReuse(ds store.IStore, src *model.NotionSyncSource, propertyID string, options []TopicOption, reuse []TopicReuseInput) (map[string]bool, error) {
	strict := map[string]bool{}
	if len(reuse) == 0 {
		return strict, nil
	}
	if propertyID == "" {
		return nil, catalogConflict("审核主题属性ID不能为空")
	}
	names := map[string]string{"": "未分类"}
	seenOptions := map[string]bool{}
	for _, option := range options {
		if seenOptions[option.ID] {
			return nil, catalogConflict("主题schema包含重复option_id")
		}
		seenOptions[option.ID] = true
		if option.ID == "" && option.Name != "未分类" {
			return nil, catalogConflict("空主题名称必须为未分类")
		}
		names[option.ID] = option.Name
	}
	targets := map[uint64]bool{}
	reviewed := make([]reviewedTopic, 0, len(reuse))
	for _, item := range reuse {
		if strict[item.OptionID] {
			return nil, catalogConflict("审核清单包含重复主题")
		}
		name, ok := names[item.OptionID]
		if !ok || name != item.ExpectedOptionName {
			return nil, catalogConflict("审核主题身份或名称已变化，请重新核对schema")
		}
		if strings.TrimSpace(item.ExpectedOptionName) == "" || strings.TrimSpace(item.SectionCode) == "" || strings.TrimSpace(item.ExpectedSectionTitle) == "" || item.ExpectedSectionStatus == nil || item.ExpectedSectionSort == nil {
			return nil, catalogConflict("审核目录清单缺少明确前值")
		}
		section, err := ds.Sections().GetByCode(item.SectionCode)
		if err != nil {
			return nil, err
		}
		if section == nil || section.ModuleCode != src.ModuleCode {
			return nil, catalogConflict("审核章节不存在或不属于来源模块")
		}
		if section.Code != item.SectionCode || section.Title != item.ExpectedSectionTitle || section.Status != *item.ExpectedSectionStatus || section.Sort != *item.ExpectedSectionSort {
			return nil, catalogConflict("审核章节标题、状态或排序已变化")
		}
		if targets[section.ID] {
			return nil, catalogConflict("审核清单中多个主题占用同一章节")
		}
		var binding model.NotionCatalogBinding
		err = ds.DB().Clauses(clause.Locking{Strength: "UPDATE"}).Where("data_source_id = ? AND theme_property_id = ? AND option_id = ?", src.DataSourceID, propertyID, item.OptionID).First(&binding).Error
		exists := err == nil
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		if exists {
			if binding.SourceID != src.ID {
				return nil, catalogConflict("主题已有其他生效绑定，请重新审核")
			}
			if binding.Status == model.NotionCatalogBound {
				if binding.SectionCode == nil {
					return nil, catalogConflict("主题已有无效生效绑定，请重新审核")
				}
				bound, err := ds.Sections().GetByCode(*binding.SectionCode)
				if err != nil {
					return nil, err
				}
				if bound == nil || bound.ID != section.ID {
					return nil, catalogConflict("主题已有其他生效绑定，请重新审核")
				}
			}
		}
		if err = CheckSyncedThemeSectionAvailable(ds, section.Code, binding.ID); err != nil {
			return nil, err
		}
		var collision int64
		if err = ds.DB().Model(&model.Section{}).Where("module_code = ? AND title = ? AND id <> ?", src.ModuleCode, name, section.ID).Count(&collision).Error; err != nil {
			return nil, err
		}
		if collision > 0 {
			return nil, catalogConflict("存在同名章节，请先审核绑定")
		}
		if err = validName(section.Code, name); err != nil {
			return nil, catalogConflict(err.Error())
		}
		strict[item.OptionID] = true
		targets[section.ID] = true
		reviewed = append(reviewed, reviewedTopic{input: item, section: section, binding: binding, exists: exists})
	}
	for _, item := range reviewed {
		if !item.exists {
			binding := model.NotionCatalogBinding{SourceID: src.ID, DataSourceID: src.DataSourceID, ThemePropertyID: propertyID, OptionID: item.input.OptionID, OptionName: item.input.ExpectedOptionName, SectionCode: &item.section.Code, Status: model.NotionCatalogBound}
			if err := ds.DB().Create(&binding).Error; err != nil {
				return nil, err
			}
		} else if item.binding.Status != model.NotionCatalogBound || item.binding.SectionCode == nil || *item.binding.SectionCode != item.section.Code || item.binding.OptionName != item.input.ExpectedOptionName || item.binding.Reason != "" {
			if err := ds.DB().Model(&item.binding).Updates(map[string]interface{}{"section_code": item.section.Code, "status": model.NotionCatalogBound, "option_name": item.input.ExpectedOptionName, "reason": ""}).Error; err != nil {
				return nil, err
			}
		}
	}
	return strict, nil
}
