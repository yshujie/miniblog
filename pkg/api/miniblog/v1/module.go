package v1

// CreateModuleRequest 创建模块请求
type CreateModuleRequest struct {
	Code  string `json:"code" valid:"required,stringlength(1|255)"`
	Title string `json:"title" valid:"required,stringlength(1|255)"`
}

// CreateModuleResponse 创建模块响应
type CreateModuleResponse struct {
	Module *ModuleInfo `json:"module"`
}

// UpdateModuleRequest 更新模块请求
type UpdateModuleRequest struct {
	Title string `json:"title" valid:"required,stringlength(1|255)"`
}

// UpdateModuleResponse 更新模块响应
type UpdateModuleResponse struct {
	Module *ModuleInfo `json:"module"`
}

// GetModuleListResponse 获取模块列表响应
type GetModuleListResponse struct {
	Modules []*ModuleInfo `json:"modules"`
}

// GetOneModuleResponse 获取模块详情响应
type GetOneModuleResponse struct {
	Module *ModuleInfo `json:"module"`
}

// ModuleStatusResponse 模块状态变更响应
type ModuleStatusResponse struct {
	Module *ModuleInfo `json:"module"`
}

// ModuleInfo 模块信息
type ModuleInfo struct {
	ID     int    `json:"id"`
	Code   string `json:"code"`
	Title  string `json:"title"`
	Status int    `json:"status"`
}
