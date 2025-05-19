package model

import (
	"time"

	"gorm.io/gorm"
)

// Module 模块
type Module struct {
	ID         int       `json:"id"`
	Code       string    `json:"code"`
	Title      string    `json:"title"`
	CreateTime time.Time `json:"create_time"`
	UpdateTime time.Time `json:"update_time"`
}

// TableName 指定表名
func (m *Module) TableName() string {
	return "module"
}

// BeforeCreate 在创建前设置信息
func (m *Module) BeforeCreate(tx *gorm.DB) (err error) {
	m.CreateTime = time.Now()
	m.UpdateTime = time.Now()
	return
}

// BeforeUpdate 在更新前设置信息
func (m *Module) BeforeUpdate(tx *gorm.DB) (err error) {
	m.UpdateTime = time.Now()
	return
}
