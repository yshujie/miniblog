package store

import (
	"context"

	"github.com/yshujie/miniblog/internal/miniblog/model"
	"gorm.io/gorm"
)

type ModuleStore interface {
	Create(ctx context.Context, module *model.Module) error
	GetByCode(ctx context.Context, code string) (*model.Module, error)
	GetAll(ctx context.Context) ([]*model.Module, error)
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

// GetByCode 获取模块
func (m *modules) GetByCode(ctx context.Context, code string) (*model.Module, error) {
	var module model.Module
	if err := m.db.Where("code = ?", code).First(&module).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &module, nil
}

// GetAll 获取所有模块
func (m *modules) GetAll(ctx context.Context) ([]*model.Module, error) {
	var modules []*model.Module
	if err := m.db.Find(&modules).Error; err != nil {
		return nil, err
	}
	return modules, nil
}
