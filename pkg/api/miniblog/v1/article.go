package v1

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

// ArticleListRequest 文章列表请求
type ArticleListRequest struct {
	ModuleCode  string `form:"module_code" valid:"required,stringlength(1|255)"`
	SectionCode string `form:"section_code" valid:"required,stringlength(1|255)"`
	Page        int    `form:"page" valid:"required,numeric"`
	Limit       int    `form:"limit" valid:"required,numeric"`
}

// CreateArticleResponse 创建文章响应
type CreateArticleResponse struct {
	Article *ArticleInfo `json:"article"`
}

// GetArticleListResponse 获取文章列表响应
type GetArticleListResponse struct {
	Articles []*ArticleInfo `json:"articles"`
	Total    int            `json:"total"`
}

// GetArticleResponse 获取文章响应
type GetArticleResponse struct {
	Article *ArticleInfo `json:"article"`
}

// ArticleInfo 文章信息
type ArticleInfo struct {
	ID          int      `json:"id"`
	Title       string   `json:"title"`
	Content     string   `json:"content"`
	SectionCode string   `json:"section_code"`
	Author      string   `json:"author"`
	Tags        []string `json:"tags"`
	Status      string   `json:"status"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
}
