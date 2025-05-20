package blog

import (
	"context"

	"github.com/yshujie/miniblog/internal/miniblog/store"
	"github.com/yshujie/miniblog/internal/pkg/log"
	v1 "github.com/yshujie/miniblog/pkg/api/miniblog/v1"
)

// BlogBiz 博客业务接口
type IBlogBiz interface {
	GetModuleDetail(ctx context.Context, req *v1.GetModuleDetailRequest) (*v1.GetModuleDetailResponse, error)
}

// blogBiz 博客业务实现
type blogBiz struct {
	ds store.IStore
}

// 确保 blogBiz 实现了 BlogBiz 接口
var _ IBlogBiz = (*blogBiz)(nil)

// New 简单工程函数，创建 blogBiz 实例
func New(ds store.IStore) *blogBiz {
	return &blogBiz{ds}
}

// Create 创建用户
func (b *blogBiz) GetModuleDetail(ctx context.Context, req *v1.GetModuleDetailRequest) (*v1.GetModuleDetailResponse, error) {
	log.C(ctx).Infow("start to get module detail in biz layer", "moduleCode", req.ModuleCode)

	// 获取模块
	module, _ := b.ds.Modules().GetByCode(req.ModuleCode)
	moduleDetail := &v1.ModuleDetail{
		Code:  module.Code,
		Title: module.Title,
	}

	// 获取章节列表
	sections, _ := b.ds.Sections().GetListByModuleCode(req.ModuleCode)
	for _, section := range sections {
		sectionDetail := &v1.SectionDetail{
			Code:  section.Code,
			Title: section.Title,
		}

		// 获取文章列表
		articles, _ := b.ds.Articles().GetListBySectionCode(section.Code)
		for _, article := range articles {
			articleDetail := &v1.ArticleDetail{
				ID:    article.ID,
				Title: article.Title,
			}
			sectionDetail.Articles = append(sectionDetail.Articles, articleDetail)
		}

		moduleDetail.Sections = append(moduleDetail.Sections, sectionDetail)
	}

	// 返回模块详情
	return &v1.GetModuleDetailResponse{
		ModuleDetail: moduleDetail,
	}, nil
}
