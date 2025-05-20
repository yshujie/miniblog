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

// GetAllModulesResponse 获取所有模块响应
type GetAllModulesResponse struct {
	Modules []*ModuleInfo `json:"modules"`
}

// GetOneModuleResponse 获取模块详情响应
type GetOneModuleResponse struct {
	Module *ModuleInfo `json:"module"`
}

// ModuleInfo 模块信息
type ModuleInfo struct {
	Code  string `json:"code"`
	Title string `json:"title"`
}
