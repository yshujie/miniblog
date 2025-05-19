package section

import (
	"context"

	"github.com/yshujie/miniblog/internal/miniblog/store"
	v1 "github.com/yshujie/miniblog/pkg/api/miniblog/v1"
)

// ISectionBiz 模块业务接口
type ISectionBiz interface {
	GetList(ctx context.Context, moduleCode string) ([]*v1.GetSectionListResponse, error)
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

// GetList 获取所有模块
func (b *sectionBiz) GetList(ctx context.Context, moduleCode string) ([]*v1.GetSectionListResponse, error) {
	sections, err := b.ds.Sections().GetListByModuleCode(ctx, moduleCode)
	if err != nil {
		return nil, err
	}

	response := make([]*v1.GetSectionListResponse, 0)
	for _, section := range sections {
		response = append(response, &v1.GetSectionListResponse{
			Sections: []*v1.SectionInfo{
				{
					Code:       section.Code,
					Title:      section.Title,
					ModuleCode: section.ModuleCode,
				},
			},
		})
	}

	return response, nil
}

// GetOne 获取模块详情
func (b *sectionBiz) GetOne(ctx context.Context, code string) (*v1.GetSectionResponse, error) {
	section, err := b.ds.Sections().GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}

	return &v1.GetSectionResponse{
		Section: &v1.SectionInfo{
			Code:       section.Code,
			Title:      section.Title,
			ModuleCode: section.ModuleCode,
		},
	}, nil
}
