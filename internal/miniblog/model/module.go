package model

import (
	"time"

	"github.com/yshujie/miniblog/pkg/util/idutil"
	"gorm.io/gorm"
)

// Module 模块
type Module struct {
	ID        uint64    `json:"id"`
	Code      string    `json:"code"`
	Title     string    `json:"title"`
	Status    int       `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// 模块状态
const (
	ModuleStatusNormal = iota + 1
	ModuleStatusDeleted
)

// TableName 指定表名
func (m *Module) TableName() string {
	return "module"
}

// BeforeCreate 在创建前设置信息
func (m *Module) BeforeCreate(tx *gorm.DB) (err error) {
	m.ID = idutil.GetIntID()
	m.CreatedAt = time.Now()
	m.UpdatedAt = time.Now()
	if m.Status == 0 {
		m.Status = ModuleStatusNormal
	}
	return
}

// BeforeUpdate 在更新前设置信息
func (m *Module) BeforeUpdate(tx *gorm.DB) (err error) {
	m.UpdatedAt = time.Now()
	return
}

// 发布模块
func (m *Module) Publish() {
	m.Status = ModuleStatusNormal
}

// 下架模块
func (m *Module) Unpublish() {
	m.Status = ModuleStatusDeleted
}

// 获取模块状态
func (m *Module) GetStatus() int {
	return m.Status
}
