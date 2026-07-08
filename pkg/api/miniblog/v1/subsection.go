package v1

// CreateSubsectionRequest 创建子章节请求
type CreateSubsectionRequest struct {
	Code        string `json:"code" valid:"required,stringlength(1|128)"`
	Title       string `json:"title" valid:"required,stringlength(1|255)"`
	SectionCode string `json:"section_code" valid:"required,stringlength(1|128)"`
	Sort        *int   `json:"sort,omitempty" valid:"optional,int"`
}

// CreateSubsectionResponse 创建子章节响应
type CreateSubsectionResponse struct {
	Subsection *SubsectionInfo `json:"subsection"`
}

// UpdateSubsectionRequest 更新子章节请求
type UpdateSubsectionRequest struct {
	Title string `json:"title" valid:"required,stringlength(1|255)"`
	Sort  *int   `json:"sort,omitempty" valid:"optional,int"`
}

// UpdateSubsectionResponse 更新子章节响应
type UpdateSubsectionResponse struct {
	Subsection *SubsectionInfo `json:"subsection"`
}

// GetSubsectionListResponse 获取子章节列表响应
type GetSubsectionListResponse struct {
	Subsections []*SubsectionInfo `json:"subsections"`
}

// GetSubsectionResponse 获取子章节详情响应
type GetSubsectionResponse struct {
	Subsection *SubsectionInfo `json:"subsection"`
}

// SubsectionStatusResponse 子章节状态变更响应
type SubsectionStatusResponse struct {
	Subsection *SubsectionInfo `json:"subsection"`
}

// SubsectionInfo 子章节信息
type SubsectionInfo struct {
	Code        string `json:"code"`
	Title       string `json:"title"`
	SectionCode string `json:"section_code"`
	Sort        int    `json:"sort"`
	Status      int    `json:"status"`
}
