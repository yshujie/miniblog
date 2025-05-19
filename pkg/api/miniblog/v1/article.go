package v1

import "time"

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
