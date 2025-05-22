package v1

import "time"

// ArticleIdRequest 文章ID请求
type ArticleIdRequest struct {
	ID int `json:"id" valid:"required,numeric"`
}

// CreateArticleRequest 创建文章请求
type CreateArticleRequest struct {
	Title        string   `json:"title" valid:"required,stringlength(1|255)"`
	SectionCode  string   `json:"section_code" valid:"required,stringlength(1|255)"`
	Author       string   `json:"author" valid:"required,stringlength(1|255)"`
	Tags         []string `json:"tags" valid:"required"`
	ExternalLink string   `json:"external_link" valid:"required"`
}

// UpdateArticleRequest 更新文章请求
type UpdateArticleRequest struct {
	ID           int      `json:"id" valid:"required,numeric"`
	Title        string   `json:"title" valid:"required,stringlength(1|255)"`
	Content      string   `json:"content" valid:"required,stringlength(1|255)"`
	Author       string   `json:"author" valid:"required,stringlength(1|255)"`
	Tags         []string `json:"tags" valid:"required"`
	ExternalLink string   `json:"external_link" valid:"required"`
}

// CreateArticleResponse 创建文章响应
type CreateArticleResponse struct {
	Article *ArticleInfo `json:"article"`
}

// GetArticleListResponse 获取文章列表响应
type GetArticleListResponse struct {
	Articles []*ArticleInfo `json:"articles"`
}

// GetArticleResponse 获取文章响应
type GetArticleResponse struct {
	Article *ArticleInfo `json:"article"`
}

// ArticleInfo 文章信息
type ArticleInfo struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	SectionCode string    `json:"section_code"`
	Author      string    `json:"author"`
	Tags        []string  `json:"tags"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
