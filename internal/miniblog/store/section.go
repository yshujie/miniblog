package store

import (
	"github.com/yshujie/miniblog/internal/miniblog/model"
	"gorm.io/gorm"
)

type SectionStore interface {
	Create(section *model.Section) error
	GetByCode(code string) (*model.Section, error)
	GetSections(moduleCode string) ([]*model.Section, error)
	GetNormalSections(moduleCode string) ([]*model.Section, error)
	Update(section *model.Section) error
}

// SectionStore 接口的实现
type sections struct {
	db *gorm.DB
}

var _ SectionStore = &sections{}

// newSections 创建章节存储实例
func newSections(db *gorm.DB) *sections {
	return &sections{db}
}

// Create 创建章节
func (s *sections) Create(section *model.Section) error {
	return s.db.Create(section).Error
}

// GetByCode 获取章节
func (s *sections) GetByCode(code string) (*model.Section, error) {
	var section model.Section
	if err := s.db.Where("code = ?", code).First(&section).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &section, nil
}

// GetSections 获取章节列表
func (s *sections) GetSections(moduleCode string) ([]*model.Section, error) {
	var sections []*model.Section
	return sections, s.db.Where("module_code = ?", moduleCode).Order("sort asc").Find(&sections).Error
}

func (s *sections) GetNormalSections(moduleCode string) ([]*model.Section, error) {
	var sections []*model.Section
	return sections, s.db.Where("module_code = ?", moduleCode).Where("status = ?", model.SectionStatusNormal).Order("sort asc").Find(&sections).Error
}

// Update 更新章节
func (s *sections) Update(section *model.Section) error {
	return s.db.Save(section).Error
}
