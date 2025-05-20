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
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// TableName 指定表名
func (s *Section) TableName() string {
	return "section"
}

// BeforeCreate 在创建前设置信息
func (s *Section) BeforeCreate(tx *gorm.DB) (err error) {
	s.CreatedAt = time.Now()
	s.UpdatedAt = time.Now()
	return
}

// BeforeUpdate 在更新前设置信息
func (s *Section) BeforeUpdate(tx *gorm.DB) (err error) {
	s.UpdatedAt = time.Now()
	return
}
