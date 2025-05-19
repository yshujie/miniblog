package v1

// GetSectionListResponse 获取模块列表响应
type GetSectionListResponse struct {
	Sections []*SectionInfo `json:"sections"`
}

// GetSectionResponse 获取模块详情响应
type GetSectionResponse struct {
	Section *SectionInfo `json:"section"`
}

// SectionInfo 模块信息
type SectionInfo struct {
	Code       string `json:"code"`
	Title      string `json:"title"`
	ModuleCode string `json:"module_code"`
}
