package model

import (
	"time"

	"gorm.io/gorm"
)

// Section 章节
type Section struct {
	ID         int       `json:"id"`
	Code       string    `json:"code"`
	Title      string    `json:"title"`
	ModuleCode string    `json:"module_code"`
	CreateTime time.Time `json:"create_time"`
	UpdateTime time.Time `json:"update_time"`
}

// TableName 指定表名
func (s *Section) TableName() string {
	return "section"
}

// BeforeCreate 在创建前设置信息
func (s *Section) BeforeCreate(tx *gorm.DB) (err error) {
	s.CreateTime = time.Now()
	s.UpdateTime = time.Now()
	return
}

// BeforeUpdate 在更新前设置信息
func (s *Section) BeforeUpdate(tx *gorm.DB) (err error) {
	s.UpdateTime = time.Now()
	return
}
