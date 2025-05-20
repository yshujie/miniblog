package model

import (
	"time"

	"gorm.io/gorm"
)

// Module 模块
type Module struct {
	ID        int       `json:"id"`
	Code      string    `json:"code"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定表名
func (m *Module) TableName() string {
	return "module"
}

// BeforeCreate 在创建前设置信息
func (m *Module) BeforeCreate(tx *gorm.DB) (err error) {
	m.CreatedAt = time.Now()
	m.UpdatedAt = time.Now()
	return
}

// BeforeUpdate 在更新前设置信息
func (m *Module) BeforeUpdate(tx *gorm.DB) (err error) {
	m.UpdatedAt = time.Now()
	return
}
