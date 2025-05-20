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
	Status     int       `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// 章节状态
const (
	SectionStatusNormal = iota + 1
	SectionStatusDeleted
)

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

// 发布章节
func (s *Section) Publish() {
	s.Status = SectionStatusNormal
}

// 下架章节
func (s *Section) Unpublish() {
	s.Status = SectionStatusDeleted
}

// 获取章节状态
func (s *Section) GetStatus() int {
	return s.Status
}
