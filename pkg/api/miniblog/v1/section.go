package v1

// CreateSectionRequest 创建 section 请求
type CreateSectionRequest struct {
	Code       string `json:"code" valid:"required,stringlength(1|255)"`
	Title      string `json:"title" valid:"required,stringlength(1|255)"`
	ModuleCode string `json:"module_code" valid:"required,stringlength(1|255)"`
	Sort       *int   `json:"sort,omitempty" valid:"optional,int"`
}

// CreateSectionResponse 创建 section 响应
type CreateSectionResponse struct {
	Section *SectionInfo `json:"section"`
}

// UpdateSectionRequest 更新 section 请求
type UpdateSectionRequest struct {
	Title string `json:"title" valid:"required,stringlength(1|255)"`
	Sort  *int   `json:"sort,omitempty" valid:"optional,int"`
}

// UpdateSectionResponse 更新 section 响应
type UpdateSectionResponse struct {
	Section *SectionInfo `json:"section"`
}

// GetSectionListResponse 获取模块列表响应
type GetSectionListResponse struct {
	Sections []*SectionInfo `json:"sections"`
}

// GetSectionResponse 获取模块详情响应
type GetSectionResponse struct {
	Section *SectionInfo `json:"section"`
}

// SectionStatusResponse section 状态变更响应
type SectionStatusResponse struct {
	Section *SectionInfo `json:"section"`
}

// SectionInfo 模块信息
type SectionInfo struct {
	Code       string `json:"code"`
	Title      string `json:"title"`
	ModuleCode string `json:"module_code"`
	Sort       int    `json:"sort"`
	Status     int    `json:"status"`
}
