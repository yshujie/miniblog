package v1

import "time"

// GetModuleDetailRequest 获取模块详情请求
type GetModuleDetailRequest struct {
	ModuleCode string `form:"module_code" valid:"required"`
}

// GetModuleDetailResponse 获取模块详情响应
type GetModuleDetailResponse struct {
	ModuleDetail *ModuleDetail `json:"module_detail"`
}

// GetArticleDetailRequest 获取文章详情请求
type GetArticleDetailRequest struct {
	ArticleID int `form:"article_id" valid:"required"`
}

// GetArticleDetailResponse 获取文章详情响应
type GetArticleDetailResponse struct {
	ArticleDetail *ArticleDetail `json:"article_detail"`
}

// ModuleDetail 模块详情
type ModuleDetail struct {
	ID       int              `json:"id"`
	Code     string           `json:"code"`
	Title    string           `json:"title"`
	Sections []*SectionDetail `json:"sections"`
}

// SectionDetail 章节详情
type SectionDetail struct {
	ID         int              `json:"id"`
	Code       string           `json:"code"`
	Title      string           `json:"title"`
	Sort       int              `json:"sort"`
	ModuleCode string           `json:"module_code"`
	Articles   []*ArticleDetail `json:"articles"`
}

// ArticleDetail 文章详情
type ArticleDetail struct {
	ID           int       `json:"id"`
	Title        string    `json:"title"`
	Content      string    `json:"content"`
	ExternalLink string    `json:"external_link"`
	SectionCode  string    `json:"section_code"`
	Author       string    `json:"author"`
	Tags         []string  `json:"tags"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
