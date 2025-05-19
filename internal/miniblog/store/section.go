package store

import (
	"context"

	"github.com/yshujie/miniblog/internal/miniblog/model"
	"gorm.io/gorm"
)

type SectionStore interface {
	Create(ctx context.Context, section *model.Section) error
	Get(ctx context.Context, id int) (*model.Section, error)
	Update(ctx context.Context, section *model.Section) error
	Delete(ctx context.Context, id int) error
	GetListByModuleCode(ctx context.Context, moduleCode string, page, pageSize int) ([]*model.Section, error)
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
func (s *sections) Create(ctx context.Context, section *model.Section) error {
	return s.db.Create(section).Error
}

// Get 获取章节
func (s *sections) Get(ctx context.Context, id int) (*model.Section, error) {
	var section model.Section
	if err := s.db.Where("id = ?", id).First(&section).Error; err != nil {
		return nil, err
	}

	return &section, nil
}

// Update 更新章节
func (s *sections) Update(ctx context.Context, section *model.Section) error {
	return s.db.Model(&model.Section{}).Where("id = ?", section.ID).Updates(section).Error
}

// Delete 删除章节
func (s *sections) Delete(ctx context.Context, id int) error {
	return s.db.Where("id = ?", id).Delete(&model.Section{}).Error
}

// GetListByModuleCode 获取章节列表
func (s *sections) GetListByModuleCode(ctx context.Context, moduleCode string, page, pageSize int) ([]*model.Section, error) {
	var sections []*model.Section
	return sections, s.db.Where("module_code = ?", moduleCode).Offset((page - 1) * pageSize).Limit(pageSize).Find(&sections).Error
}
