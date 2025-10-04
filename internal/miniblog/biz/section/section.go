package section

import (
	"context"

	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"github.com/yshujie/miniblog/internal/pkg/errno"
	v1 "github.com/yshujie/miniblog/pkg/api/miniblog/v1"
)

// ISectionBiz 模块业务接口
type ISectionBiz interface {
	Create(ctx context.Context, r *v1.CreateSectionRequest) (*v1.CreateSectionResponse, error)
	GetList(ctx context.Context, moduleCode string) (*v1.GetSectionListResponse, error)
	GetOne(ctx context.Context, code string) (*v1.GetSectionResponse, error)
}

// sectionBiz 模块业务实现
type sectionBiz struct {
	ds store.IStore
}

// 确保 sectionBiz 实现了 ISectionBiz 接口
var _ ISectionBiz = (*sectionBiz)(nil)

// New 简单工程函数，创建 sectionBiz 实例
func New(ds store.IStore) *sectionBiz {
	return &sectionBiz{ds}
}

// Create 创建 section 记录
func (b *sectionBiz) Create(ctx context.Context, r *v1.CreateSectionRequest) (*v1.CreateSectionResponse, error) {
	// 检查 code 是否已存在
	existingSection, err := b.ds.Sections().GetByCode(r.Code)
	if err != nil {
		return nil, err
	}
	if existingSection != nil {
		return nil, errno.ErrSectionAlreadyExists
	}

	// 检查 module_code 是否已存在
	existingModule, err := b.ds.Modules().GetByCode(r.ModuleCode)
	if err != nil {
		return nil, err
	}
	if existingModule == nil {
		return nil, errno.ErrModuleNotFound
	}

	// 创建 section 记录
	section := &model.Section{
		Code:       r.Code,
		Title:      r.Title,
		ModuleCode: r.ModuleCode,
	}
	if err = b.ds.Sections().Create(section); err != nil {
		return nil, err
	}

	return &v1.CreateSectionResponse{
		Section: &v1.SectionInfo{
			Code:       section.Code,
			Title:      section.Title,
			ModuleCode: section.ModuleCode,
		},
	}, nil
}

// GetList 获取所有模块
func (b *sectionBiz) GetList(ctx context.Context, moduleCode string) (*v1.GetSectionListResponse, error) {
	sections, err := b.ds.Sections().GetListByModuleCode(moduleCode)
	if err != nil {
		return nil, err
	}

	response := &v1.GetSectionListResponse{
		Sections: make([]*v1.SectionInfo, 0, len(sections)),
	}
	for _, section := range sections {
		response.Sections = append(response.Sections, &v1.SectionInfo{
			Code:       section.Code,
			Title:      section.Title,
			ModuleCode: section.ModuleCode,
		})
	}

	return response, nil
}

// GetOne 获取模块详情
func (b *sectionBiz) GetOne(ctx context.Context, code string) (*v1.GetSectionResponse, error) {
	section, err := b.ds.Sections().GetByCode(code)
	if err != nil {
		return nil, err
	}
	if section == nil {
		return nil, errno.ErrSectionNotFound
	}

	return &v1.GetSectionResponse{
		Section: &v1.SectionInfo{
			Code:       section.Code,
			Title:      section.Title,
			ModuleCode: section.ModuleCode,
		},
	}, nil
}
