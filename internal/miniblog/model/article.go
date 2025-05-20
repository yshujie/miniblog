package model

import (
	"time"

	"gorm.io/gorm"
)

// Article 文章
type Article struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	SectionCode string    `json:"section_code"`
	Author      string    `json:"author"`
	Tags        string    `json:"tags"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TableName 指定表名
func (a *Article) TableName() string {
	return "article"
}

// BeforeCreate 在创建前设置信息
func (a *Article) BeforeCreate(tx *gorm.DB) (err error) {
	a.CreatedAt = time.Now()
	a.UpdatedAt = time.Now()
	return
}

// BeforeUpdate 在更新前设置信息
func (a *Article) BeforeUpdate(tx *gorm.DB) (err error) {
	a.UpdatedAt = time.Now()
	return
}
