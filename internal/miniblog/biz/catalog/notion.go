package catalog

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"github.com/yshujie/miniblog/internal/pkg/errno"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"strings"
)

func catalogConflict(message string) error {
	return &errno.Errno{HTTP: 409, Code: "CatalogBindingConflict", Message: message}
}

// EnsureSyncedTheme borrows the page transaction. The caller already locked the module.
func EnsureSyncedTheme(ds store.IStore, src *model.NotionSyncSource, propertyID, optionID, optionName string) (*Placement, error) {
	module, e := ds.Modules().GetByCode(src.ModuleCode)
	if e != nil {
		return nil, e
	}
	if module == nil {
		return nil, errno.ErrModuleNotFound
	}
	if propertyID == "" {
		return nil, invalid("主题属性ID不能为空")
	}
	if optionID == "" {
		optionName = "未分类"
	}
	if strings.TrimSpace(optionName) == "" {
		return nil, invalid("主题名称不能为空")
	}
	var binding model.NotionCatalogBinding
	e = ds.DB().Clauses(clause.Locking{Strength: "UPDATE"}).Where("data_source_id = ? AND theme_property_id = ? AND option_id = ?", src.DataSourceID, propertyID, optionID).First(&binding).Error
	if e != nil && !errors.Is(e, gorm.ErrRecordNotFound) {
		return nil, e
	}
	if e == nil {
		if binding.SourceID != src.ID || binding.Status != model.NotionCatalogBound || binding.SectionCode == nil {
			return nil, catalogConflict("主题绑定无效，请先处理绑定")
		}
		p, e := Resolve(ds, *binding.SectionCode, "")
		if e != nil {
			return nil, catalogConflict("主题绑定目录已失效")
		}
		if p.Module.ID != module.ID {
			return nil, catalogConflict("主题绑定不属于当前模块")
		}
		if e = CheckSyncedThemeSectionAvailable(ds, p.Section.Code, binding.ID); e != nil {
			return nil, e
		}
		var collision int64
		if e = ds.DB().Model(&model.Section{}).Where("module_code = ? AND title = ? AND id <> ?", module.Code, optionName, p.Section.ID).Count(&collision).Error; e != nil {
			return nil, e
		}
		if collision > 0 {
			return nil, catalogConflict("存在同名章节，请先审核绑定")
		}
		if e = validName(p.Section.Code, optionName); e != nil {
			return nil, e
		}
		if p.Section.Title != optionName {
			if e = ds.DB().Model(&model.Section{}).Where("id = ?", p.Section.ID).Update("title", optionName).Error; e != nil {
				return nil, e
			}
			p.Section.Title = optionName
		}
		if binding.OptionName != optionName {
			if e = ds.DB().Model(&binding).Update("option_name", optionName).Error; e != nil {
				return nil, e
			}
		}
		return p, nil
	}
	var collision int64
	if e = ds.DB().Model(&model.Section{}).Where("module_code = ? AND title = ?", module.Code, optionName).Count(&collision).Error; e != nil {
		return nil, e
	}
	if collision > 0 {
		return nil, catalogConflict("存在同名章节，请先审核绑定")
	}
	tuple, _ := json.Marshal([]string{src.DataSourceID, propertyID, optionID})
	digest := sha256.Sum256(tuple)
	code := "ns_" + hex.EncodeToString(digest[:])
	if e = validName(code, optionName); e != nil {
		return nil, e
	}
	var max *int
	if e = ds.DB().Model(&model.Section{}).Where("module_code = ?", module.Code).Select("MAX(sort)").Scan(&max).Error; e != nil {
		return nil, e
	}
	order := 1
	if max != nil {
		order = *max + 1
	}
	section := &model.Section{Code: code, Title: optionName, ModuleCode: module.Code, Sort: order, Status: model.SectionStatusNormal}
	if e = ds.Sections().Create(section); e != nil {
		return nil, e
	}
	binding = model.NotionCatalogBinding{SourceID: src.ID, DataSourceID: src.DataSourceID, ThemePropertyID: propertyID, OptionID: optionID, OptionName: optionName, SectionCode: &section.Code, Status: model.NotionCatalogBound}
	if e = ds.DB().Create(&binding).Error; e != nil {
		return nil, e
	}
	return &Placement{Module: module, Section: section}, nil
}
func CheckModuleBindingDependency(ds store.IStore, code string) error {
	if ds.DB() == nil || !ds.DB().Migrator().HasTable(&model.NotionSyncSource{}) {
		return nil
	}
	var n int64
	if e := ds.DB().Model(&model.NotionSyncSource{}).Where("module_code = ?", code).Count(&n).Error; e != nil {
		return e
	}
	if n > 0 {
		return catalogConflict("模块仍有Notion来源绑定")
	}
	return nil
}
func CheckSectionBindingDependency(ds store.IStore, code string) error {
	if ds.DB() == nil || !ds.DB().Migrator().HasTable(&model.NotionCatalogBinding{}) {
		return nil
	}
	var n int64
	if e := ds.DB().Model(&model.NotionCatalogBinding{}).Where("section_code = ?", code).Count(&n).Error; e != nil {
		return e
	}
	if n > 0 {
		return catalogConflict("章节仍有Notion主题绑定")
	}
	return nil
}

// RecordBlockedSyncedTheme persists a conflict after the caller rolls back topic side effects.
func RecordBlockedSyncedTheme(ds store.IStore, src *model.NotionSyncSource, propertyID, optionID, optionName, reason string) error {
	if propertyID == "" {
		return nil
	}
	if optionID == "" {
		optionName = "未分类"
	}
	binding := model.NotionCatalogBinding{SourceID: src.ID, DataSourceID: src.DataSourceID, ThemePropertyID: propertyID, OptionID: optionID, OptionName: optionName, Status: model.NotionCatalogBlocked, Reason: reason}
	return ds.DB().Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "data_source_id"}, {Name: "theme_property_id"}, {Name: "option_id"}}, DoUpdates: clause.Assignments(map[string]interface{}{"status": model.NotionCatalogBlocked, "reason": reason, "option_name": optionName})}).Create(&binding).Error
}

// CheckSyncedThemeSectionAvailable requires the caller to hold the section's module lock.
// Compare through the SQL relationship so historical code collation remains canonical.
func CheckSyncedThemeSectionAvailable(ds store.IStore, sectionCode string, bindingID uint64) error {
	section, e := ds.Sections().GetByCode(sectionCode)
	if e != nil {
		return e
	}
	if section == nil {
		return errno.ErrSectionNotFound
	}
	var occupied int64
	e = ds.DB().Model(&model.NotionCatalogBinding{}).
		Joins("JOIN section AS bound_section ON bound_section.code = notion_catalog_bindings.section_code").
		Where("bound_section.id = ? AND notion_catalog_bindings.status = ? AND notion_catalog_bindings.id <> ?", section.ID, model.NotionCatalogBound, bindingID).
		Count(&occupied).Error
	if e != nil {
		return e
	}
	if occupied > 0 {
		return catalogConflict("章节已由其他主题占用，请先审核绑定")
	}
	return nil
}
