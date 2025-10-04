package module

import (
	"context"

	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"github.com/yshujie/miniblog/internal/pkg/errno"
	v1 "github.com/yshujie/miniblog/pkg/api/miniblog/v1"
)

// ModuleBiz 模块业务接口
type IModuleBiz interface {
	Create(ctx context.Context, r *v1.CreateModuleRequest) (*v1.CreateModuleResponse, error)
	Update(ctx context.Context, code string, r *v1.UpdateModuleRequest) (*v1.UpdateModuleResponse, error)
	Publish(ctx context.Context, code string) (*v1.ModuleStatusResponse, error)
	Unpublish(ctx context.Context, code string) (*v1.ModuleStatusResponse, error)
	GetAll(ctx context.Context) (*v1.GetModuleListResponse, error)
	GetOne(ctx context.Context, code string) (*v1.GetOneModuleResponse, error)
	// Delete physically deletes a module by code
	Delete(ctx context.Context, code string) error
}

// moduleBiz 模块业务实现
type moduleBiz struct {
	ds store.IStore
}

// 确保 moduleBiz 实现了 ModuleBiz 接口
var _ IModuleBiz = (*moduleBiz)(nil)

// New 简单工程函数，创建 moduleBiz 实例
func New(ds store.IStore) *moduleBiz {
	return &moduleBiz{ds}
}

// Create 创建模块
func (b *moduleBiz) Create(ctx context.Context, r *v1.CreateModuleRequest) (*v1.CreateModuleResponse, error) {
	// 检查 code 是否已存在
	module, err := b.ds.Modules().GetByCode(r.Code)
	if err != nil {
		return nil, err
	}
	if module != nil {
		return nil, errno.ErrModuleAlreadyExists
	}
	// 创建 module 记录
	module = &model.Module{
		Code:  r.Code,
		Title: r.Title,
	}
	if err = b.ds.Modules().Create(module); err != nil {
		return nil, err
	}

	// 返回响应 CreateModuleResponse
	response := &v1.CreateModuleResponse{
		Module: toModuleInfo(module),
	}

	return response, nil
}

// Delete 物理删除模块，删除前检查是否存在关联的 section 或 article
func (b *moduleBiz) Delete(ctx context.Context, code string) error {
	module, err := b.ds.Modules().GetByCode(code)
	if err != nil {
		return err
	}
	if module == nil {
		return errno.ErrModuleNotFound
	}

	// 检查是否存在关联的 sections
	sections, err := b.ds.Sections().GetSections(code)
	if err != nil {
		return err
	}
	if len(sections) > 0 {
		return errno.ErrModuleHasDependents
	}

	// 检查是否存在关联的 articles（article 可能直接引用 module_code）
	filter := map[string]interface{}{"module_code": code}
	articles, err := b.ds.Articles().GetList(filter, 1, 1)
	if err != nil {
		return err
	}
	if len(articles) > 0 {
		return errno.ErrModuleHasDependents
	}

	return b.ds.Modules().DeleteByCode(code)
}

// Update 更新模块
func (b *moduleBiz) Update(ctx context.Context, code string, r *v1.UpdateModuleRequest) (*v1.UpdateModuleResponse, error) {
	module, err := b.ds.Modules().GetByCode(code)
	if err != nil {
		return nil, err
	}
	if module == nil {
		return nil, errno.ErrModuleNotFound
	}

	module.Title = r.Title

	if err = b.ds.Modules().Update(module); err != nil {
		return nil, err
	}

	return &v1.UpdateModuleResponse{Module: toModuleInfo(module)}, nil
}

// Publish 上架模块
func (b *moduleBiz) Publish(ctx context.Context, code string) (*v1.ModuleStatusResponse, error) {
	module, err := b.ds.Modules().GetByCode(code)
	if err != nil {
		return nil, err
	}
	if module == nil {
		return nil, errno.ErrModuleNotFound
	}

	module.Publish()

	if err = b.ds.Modules().Update(module); err != nil {
		return nil, err
	}

	return &v1.ModuleStatusResponse{Module: toModuleInfo(module)}, nil
}

// Unpublish 下架模块
func (b *moduleBiz) Unpublish(ctx context.Context, code string) (*v1.ModuleStatusResponse, error) {
	module, err := b.ds.Modules().GetByCode(code)
	if err != nil {
		return nil, err
	}
	if module == nil {
		return nil, errno.ErrModuleNotFound
	}

	module.Unpublish()

	if err = b.ds.Modules().Update(module); err != nil {
		return nil, err
	}

	return &v1.ModuleStatusResponse{Module: toModuleInfo(module)}, nil
}

// GetAll 获取所有模块
func (b *moduleBiz) GetAll(ctx context.Context) (*v1.GetModuleListResponse, error) {
	modules, err := b.ds.Modules().GetAll()
	if err != nil {
		return nil, err
	}

	// 将 modules 追加到 GetAllModulesResponse.Modules 中
	response := &v1.GetModuleListResponse{
		Modules: make([]*v1.ModuleInfo, 0, len(modules)),
	}
	for _, module := range modules {
		response.Modules = append(response.Modules, toModuleInfo(module))
	}

	return response, nil
}

// GetOne 获取模块详情
func (b *moduleBiz) GetOne(ctx context.Context, code string) (*v1.GetOneModuleResponse, error) {
	module, err := b.ds.Modules().GetByCode(code)
	if err != nil {
		return nil, err
	}
	if module == nil {
		return nil, errno.ErrModuleNotFound
	}

	return &v1.GetOneModuleResponse{
		Module: toModuleInfo(module),
	}, nil
}

func toModuleInfo(module *model.Module) *v1.ModuleInfo {
	if module == nil {
		return nil
	}

	return &v1.ModuleInfo{
		ID:     int(module.ID),
		Code:   module.Code,
		Title:  module.Title,
		Status: module.Status,
	}
}
