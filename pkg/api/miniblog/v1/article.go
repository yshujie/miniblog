package v1

// CreateArticleRequest 创建文章请求
type CreateArticleRequest struct {
	Title        string   `json:"title" valid:"required,stringlength(1|255)"`
	ModuleCode   string   `json:"module_code" valid:"required,stringlength(1|255)"`
	SectionCode  string   `json:"section_code" valid:"required,stringlength(1|255)"`
	Author       string   `json:"author" valid:"required,stringlength(1|255)"`
	Tags         []string `json:"tags" valid:"required"`
	ExternalLink string   `json:"external_link" valid:"required"`
}

// UpdateArticleRequest 更新文章请求
type UpdateArticleRequest struct {
	ID           string   `json:"id" valid:"required"`
	Title        string   `json:"title" valid:"required,stringlength(1|255)"`
	Author       string   `json:"author" valid:"required,stringlength(1|255)"`
	Tags         []string `json:"tags" valid:"required"`
	ModuleCode   string   `json:"module_code" valid:"required"`
	SectionCode  string   `json:"section_code" valid:"required"`
	Content      string   `json:"content" valid:"required"`
	ExternalLink string   `json:"external_link" valid:"required"`
}

// ArticleListRequest 文章列表请求
type ArticleListRequest struct {
	ModuleCode  string `form:"module_code" valid:"required,stringlength(1|255)"`
	SectionCode string `form:"section_code" valid:"required,stringlength(1|255)"`
	Page        int    `form:"page" valid:"required,numeric"`
	Limit       int    `form:"limit" valid:"required,numeric"`
}

// ArticleInfoResponse 文章信息响应
type ArticleInfoResponse struct {
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
	ID           uint64      `json:"id"`
	Title        string      `json:"title"`
	Content      string      `json:"content"`
	ExternalLink string      `json:"external_link"`
	Module       ModuleInfo  `json:"module"`
	Section      SectionInfo `json:"section"`
	Author       string      `json:"author"`
	Tags         []string    `json:"tags"`
	Status       string      `json:"status"`
	CreatedAt    string      `json:"created_at"`
	UpdatedAt    string      `json:"updated_at"`
}
