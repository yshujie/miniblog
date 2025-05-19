package store

import (
	"context"

	"github.com/yshujie/miniblog/internal/miniblog/model"
	"gorm.io/gorm"
)

type SectionStore interface {
	Create(ctx context.Context, section *model.Section) error
	GetByCode(ctx context.Context, code string) (*model.Section, error)
	GetListByModuleCode(ctx context.Context, moduleCode string) ([]*model.Section, error)
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

// GetByCode 获取章节
func (s *sections) GetByCode(ctx context.Context, code string) (*model.Section, error) {
	var section model.Section
	if err := s.db.Where("code = ?", code).First(&section).Error; err != nil {
		return nil, err
	}

	return &section, nil
}

// GetListByModuleCode 获取章节列表
func (s *sections) GetListByModuleCode(ctx context.Context, moduleCode string) ([]*model.Section, error) {
	var sections []*model.Section
	return sections, s.db.Where("module_code = ?", moduleCode).Find(&sections).Error
}
