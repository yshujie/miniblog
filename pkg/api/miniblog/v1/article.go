package v1

import "time"

// CreateArticleRequest 创建文章请求
type CreateArticleRequest struct {
	Title       string   `json:"title" valid:"required,stringlength(1|255)"`
	Content     string   `json:"content" valid:"required,stringlength(1|100000)"`
	SectionCode string   `json:"section_code" valid:"required,stringlength(1|255)"`
	Author      string   `json:"author" valid:"required,stringlength(1|255)"`
	Tags        []string `json:"tags" valid:"required,dive,stringlength(1|255)"`
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
	CreateTime  time.Time `json:"create_time"`
	UpdateTime  time.Time `json:"update_time"`
}
