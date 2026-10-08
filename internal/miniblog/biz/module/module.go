package module

import (
	"context"
	"github.com/yshujie/miniblog/internal/miniblog/biz/catalog"
	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"github.com/yshujie/miniblog/internal/pkg/errno"
	v1 "github.com/yshujie/miniblog/pkg/api/miniblog/v1"
)

type IModuleBiz interface {
	Create(context.Context, *v1.CreateModuleRequest) (*v1.CreateModuleResponse, error)
	Update(context.Context, string, *v1.UpdateModuleRequest) (*v1.UpdateModuleResponse, error)
	Publish(context.Context, string) (*v1.ModuleStatusResponse, error)
	Unpublish(context.Context, string) (*v1.ModuleStatusResponse, error)
	GetAll(context.Context) (*v1.GetModuleListResponse, error)
	GetOne(context.Context, string) (*v1.GetOneModuleResponse, error)
	Delete(context.Context, string) error
}
type moduleBiz struct{ ds store.IStore }

func New(ds store.IStore) *moduleBiz { return &moduleBiz{ds: ds} }
func (b *moduleBiz) contextStore(ctx context.Context) store.IStore {
	if b.ds.DB() == nil {
		return b.ds
	}
	return store.NewStore(b.ds.DB().WithContext(ctx))
}
func (b *moduleBiz) Create(ctx context.Context, r *v1.CreateModuleRequest) (*v1.CreateModuleResponse, error) {
	m, e := catalog.New(b.ds).CreateModule(ctx, catalog.ModuleInput{Code: r.Code, Title: r.Title, Sort: r.Sort})
	if e != nil {
		return nil, e
	}
	return &v1.CreateModuleResponse{Module: toModuleInfo(m)}, nil
}
func (b *moduleBiz) Update(ctx context.Context, code string, r *v1.UpdateModuleRequest) (*v1.UpdateModuleResponse, error) {
	m, e := catalog.New(b.ds).UpdateModule(ctx, code, catalog.UpdateInput{Title: r.Title, Sort: r.Sort})
	if e != nil {
		return nil, e
	}
	return &v1.UpdateModuleResponse{Module: toModuleInfo(m)}, nil
}
func (b *moduleBiz) Publish(ctx context.Context, code string) (*v1.ModuleStatusResponse, error) {
	m, e := catalog.New(b.ds).ModuleStatus(ctx, code, model.ModuleStatusNormal)
	if e != nil {
		return nil, e
	}
	return &v1.ModuleStatusResponse{Module: toModuleInfo(m)}, nil
}
func (b *moduleBiz) Unpublish(ctx context.Context, code string) (*v1.ModuleStatusResponse, error) {
	m, e := catalog.New(b.ds).ModuleStatus(ctx, code, model.ModuleStatusDeleted)
	if e != nil {
		return nil, e
	}
	return &v1.ModuleStatusResponse{Module: toModuleInfo(m)}, nil
}
func (b *moduleBiz) Delete(ctx context.Context, code string) error {
	return catalog.New(b.ds).DeleteModule(ctx, code)
}
func (b *moduleBiz) GetAll(ctx context.Context) (*v1.GetModuleListResponse, error) {
	rows, e := b.contextStore(ctx).Modules().GetAll()
	if e != nil {
		return nil, e
	}
	out := &v1.GetModuleListResponse{Modules: make([]*v1.ModuleInfo, 0, len(rows))}
	for _, m := range rows {
		out.Modules = append(out.Modules, toModuleInfo(m))
	}
	return out, nil
}
func (b *moduleBiz) GetOne(ctx context.Context, code string) (*v1.GetOneModuleResponse, error) {
	m, e := b.contextStore(ctx).Modules().GetByCode(code)
	if e != nil {
		return nil, e
	}
	if m == nil {
		return nil, errno.ErrModuleNotFound
	}
	return &v1.GetOneModuleResponse{Module: toModuleInfo(m)}, nil
}
func toModuleInfo(m *model.Module) *v1.ModuleInfo {
	if m == nil {
		return nil
	}
	return &v1.ModuleInfo{ID: int(m.ID), Code: m.Code, Title: m.Title, Status: m.Status, Sort: m.Sort}
}
