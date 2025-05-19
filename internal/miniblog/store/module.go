package store

import (
	"context"

	"github.com/yshujie/miniblog/internal/miniblog/model"
	"gorm.io/gorm"
)

type ModuleStore interface {
	Create(ctx context.Context, module *model.Module) error
	Get(ctx context.Context, id int) (*model.Module, error)
	Update(ctx context.Context, module *model.Module) error
	Delete(ctx context.Context, id int) error
	GetList(ctx context.Context, page, pageSize int) ([]*model.Module, error)
}

// ModuleStore 接口的实现
type modules struct {
	db *gorm.DB
}

var _ ModuleStore = &modules{}

// newModules 创建模块存储实例
func newModules(db *gorm.DB) *modules {
	return &modules{db}
}

// Create 创建模块
func (m *modules) Create(ctx context.Context, module *model.Module) error {
	return m.db.Create(module).Error
}

// Get 获取模块
func (m *modules) Get(ctx context.Context, id int) (*model.Module, error) {
	var module model.Module
	if err := m.db.Where("id = ?", id).First(&module).Error; err != nil {
		return nil, err
	}

	return &module, nil
}

// Update 更新模块
func (m *modules) Update(ctx context.Context, module *model.Module) error {
	return m.db.Model(&model.Module{}).Where("id = ?", module.ID).Updates(module).Error
}

// Delete 删除模块
func (m *modules) Delete(ctx context.Context, id int) error {
	return m.db.Where("id = ?", id).Delete(&model.Module{}).Error
}

// GetList 获取模块列表
func (m *modules) GetList(ctx context.Context, page, pageSize int) ([]*model.Module, error) {
	var modules []*model.Module
	return modules, m.db.Offset((page - 1) * pageSize).Limit(pageSize).Find(&modules).Error
}
