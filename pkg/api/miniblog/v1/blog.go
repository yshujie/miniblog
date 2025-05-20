package v1

// GetModuleDetailRequest 获取模块详情请求
type GetModuleDetailRequest struct {
	ModuleCode string `json:"module_code" valid:"required"`
}

// GetModuleDetailResponse 获取模块详情响应
type GetModuleDetailResponse struct {
	ModuleDetail *ModuleDetail `json:"module_detail"`
}

// ModuleDetail 模块详情
type ModuleDetail struct {
	Code     string           `json:"code"`
	Title    string           `json:"title"`
	Sections []*SectionDetail `json:"sections"`
}

// SectionDetail 章节详情
type SectionDetail struct {
	Code     string           `json:"code"`
	Title    string           `json:"title"`
	Articles []*ArticleDetail `json:"articles"`
}

// ArticleDetail 文章详情
type ArticleDetail struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
}
