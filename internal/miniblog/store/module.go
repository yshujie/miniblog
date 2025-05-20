package store

import (
	"github.com/yshujie/miniblog/internal/miniblog/model"
	"gorm.io/gorm"
)

type ModuleStore interface {
	Create(module *model.Module) error
	GetByCode(code string) (*model.Module, error)
	GetAll() ([]*model.Module, error)
	Update(module *model.Module) error
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
func (m *modules) Create(module *model.Module) error {
	return m.db.Create(module).Error
}

// GetByCode 获取模块
func (m *modules) GetByCode(code string) (*model.Module, error) {
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
func (m *modules) GetAll() ([]*model.Module, error) {
	var modules []*model.Module
	if err := m.db.Find(&modules).Error; err != nil {
		return nil, err
	}
	return modules, nil
}

// Update 更新模块
func (m *modules) Update(module *model.Module) error {
	return m.db.Save(module).Error
}
