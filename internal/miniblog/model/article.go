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
	Status      int       `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// 文章状态
const (
	ArticleStatusNormal  = iota + 1
	ArticleStatusDeleted = 2
)

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

// 发布文章
func (a *Article) Publish() {
	a.Status = ArticleStatusNormal
}

// 下架文章
func (a *Article) Unpublish() {
	a.Status = ArticleStatusDeleted
}

// 获取文章状态
func (a *Article) GetStatus() int {
	return a.Status
}
