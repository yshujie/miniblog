package model

import (
	"time"

	"gorm.io/gorm"
)

// Article 文章
type Article struct {
	ID           int       `json:"id"`
	Title        string    `json:"title"`
	Content      string    `json:"content"`
	ExternalLink string    `json:"external_link"`
	SectionCode  string    `json:"section_code"`
	Author       string    `json:"author"`
	Tags         string    `json:"tags"`
	Status       int       `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// 文章状态
const (
	ArticleStatusDraft       = 1 // 草稿
	ArticleStatusPublished   = 2 // 已发布
	ArticleStatusUnpublished = 3 // 已下架
	ArticleStatusDeleted     = 4 // 已删除
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

// SaveDraft 存草稿
func (a *Article) SaveDraft() {
	a.Status = ArticleStatusDraft
}

// Publish 发布文章
func (a *Article) Publish() {
	a.Status = ArticleStatusPublished
}

// Unpublish 下架文章
func (a *Article) Unpublish() {
	a.Status = ArticleStatusUnpublished
}

// Delete 删除文章
func (a *Article) Delete() {
	a.Status = ArticleStatusDeleted
}

// GetStatus 获取文章状态
func (a *Article) GetStatus() int {
	return a.Status
}

// GetStatusString 获取文章状态字符串
func (a *Article) GetStatusString() string {
	switch a.Status {
	case ArticleStatusDraft:
		return "Draft"
	case ArticleStatusPublished:
		return "Published"
	case ArticleStatusUnpublished:
		return "Unpublished"
	case ArticleStatusDeleted:
		return "Deleted"
	default:
		return "Unknown"
	}
}
