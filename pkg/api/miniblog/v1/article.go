package v1

// CreateArticleRequest 创建文章请求
type CreateArticleRequest struct {
	Title          string   `json:"title" valid:"required,stringlength(1|255)"`
	ModuleCode     string   `json:"module_code" valid:"required,stringlength(1|255)"`
	SectionCode    string   `json:"section_code" valid:"required,stringlength(1|255)"`
	SubsectionCode string   `json:"subsection_code" valid:"optional,stringlength(1|128)"`
	Author         string   `json:"author" valid:"optional,stringlength(0|128)"`
	Tags           []string `json:"tags"`
	ExternalLink   string   `json:"external_link" valid:"required"`
	Content        *string  `json:"content,omitempty"`
}

// UpdateArticleRequest 更新文章请求
type UpdateArticleRequest struct {
	ID             string   `json:"id" valid:"required"`
	Title          string   `json:"title" valid:"required,stringlength(1|255)"`
	Author         string   `json:"author" valid:"optional,stringlength(0|128)"`
	Tags           []string `json:"tags"`
	ModuleCode     string   `json:"module_code" valid:"required"`
	SectionCode    string   `json:"section_code" valid:"required"`
	Content        *string  `json:"content,omitempty"`
	ExternalLink   string   `json:"external_link"`
	SubsectionCode string   `json:"subsection_code" valid:"optional"`
}

// ArticleListRequest 文章列表请求
type ArticleListRequest struct {
	ModuleCode     string `form:"module_code" valid:"required,stringlength(1|255)"`
	SectionCode    string `form:"section_code" valid:"required,stringlength(1|255)"`
	SubsectionCode string `form:"subsection_code" valid:"optional,stringlength(1|128)"`
	Page           int    `form:"page" valid:"required,numeric"`
	Limit          int    `form:"limit" valid:"required,numeric"`
	Title          string `form:"title"`
	Status         string `form:"status"`
	DirectOnly     bool   `form:"direct_only"`
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
	ID           uint64          `json:"id"`
	IDText       string          `json:"id_text"`
	Title        string          `json:"title"`
	Content      string          `json:"content"`
	ExternalLink string          `json:"external_link"`
	Module       ModuleInfo      `json:"module"`
	Section      SectionInfo     `json:"section"`
	Subsection   *SubsectionInfo `json:"subsection,omitempty"`
	Author       string          `json:"author"`
	Tags         []string        `json:"tags"`
	Pos          int             `json:"pos"`
	Status       string          `json:"status"`
	CreatedAt    string          `json:"created_at"`
	UpdatedAt    string          `json:"updated_at"`
	Provider     *string         `json:"provider,omitempty"`
	CanonicalURL *string         `json:"canonical_url,omitempty"`
}

// ArticleDTO is used by new commands. Legacy ArticleInfo keeps numeric id.
type ArticleDTO struct {
	*ArticleInfo
	ID string `json:"id"`
}

type PreviewSourceRequest struct {
	ExternalLink string `json:"external_link"`
}
type PreviewSourceResponse struct {
	Provider        string      `json:"provider"`
	CanonicalURL    string      `json:"canonical_url"`
	Title           string      `json:"title"`
	MetadataStatus  string      `json:"metadata_status"`
	Reason          string      `json:"reason,omitempty"`
	ExistingArticle *ArticleDTO `json:"existing_article,omitempty"`
}
type RegisterArticleRequest struct {
	ExternalLink   string   `json:"external_link"`
	Title          string   `json:"title"`
	SectionCode    string   `json:"section_code"`
	SubsectionCode string   `json:"subsection_code,omitempty"`
	Author         string   `json:"author,omitempty"`
	Tags           []string `json:"tags,omitempty"`
	Publish        bool     `json:"publish"`
}
type RegisterArticleResponse struct {
	Outcome string      `json:"outcome"`
	Article *ArticleDTO `json:"article"`
}
type MoveArticleRequest struct {
	SectionCode    string `json:"section_code"`
	SubsectionCode string `json:"subsection_code,omitempty"`
}
type ArticleCommandResponse struct {
	Article *ArticleDTO `json:"article"`
}
type ReorderArticlesRequest struct {
	SectionCode    string   `json:"section_code"`
	SubsectionCode string   `json:"subsection_code,omitempty"`
	ArticleIDs     []string `json:"article_ids"`
}
