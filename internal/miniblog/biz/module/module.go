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
	GetAll(ctx context.Context) ([]*v1.GetAllModulesResponse, error)
	GetOne(ctx context.Context, code string) (*v1.GetOneModuleResponse, error)
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
	module, err := b.ds.Modules().GetByCode(ctx, r.Code)
	if err != nil {
		return nil, err
	}
	if module != nil {
		return nil, errno.ErrModuleAlreadyExists
	}

	// 创建 module 记录
	err = b.ds.Modules().Create(ctx, &model.Module{
		Code:  r.Code,
		Title: r.Title,
	})
	if err != nil {
		return nil, err
	}

	// 返回响应 CreateModuleResponse
	response := &v1.CreateModuleResponse{
		Module: &v1.ModuleInfo{
			Code:  r.Code,
			Title: r.Title,
		},
	}

	return response, nil
}

// GetAll 获取所有模块
func (b *moduleBiz) GetAll(ctx context.Context) ([]*v1.GetAllModulesResponse, error) {
	modules, err := b.ds.Modules().GetAll(ctx)
	if err != nil {
		return nil, err
	}

	response := make([]*v1.GetAllModulesResponse, 0)
	for _, module := range modules {
		response = append(response, &v1.GetAllModulesResponse{
			Modules: []*v1.ModuleInfo{
				{
					Code:  module.Code,
					Title: module.Title,
				},
			},
		})
	}

	return response, nil
}

// GetOne 获取模块详情
func (b *moduleBiz) GetOne(ctx context.Context, code string) (*v1.GetOneModuleResponse, error) {
	module, err := b.ds.Modules().GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}

	return &v1.GetOneModuleResponse{
		Module: &v1.ModuleInfo{
			Code:  module.Code,
			Title: module.Title,
		},
	}, nil
}
