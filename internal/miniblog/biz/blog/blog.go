package blog

import (
	"strings"

	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"github.com/yshujie/miniblog/internal/pkg/log"
	v1 "github.com/yshujie/miniblog/pkg/api/miniblog/v1"
)

// BlogBiz 博客业务接口
type IBlogBiz interface {
	GetModuleList() (*v1.GetModuleListResponse, error)
	GetModuleDetail(req *v1.GetModuleDetailRequest) (*v1.GetModuleDetailResponse, error)
	GetArticleDetail(req *v1.GetArticleDetailRequest) (*v1.GetArticleDetailResponse, error)
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

func (b *blogBiz) GetModuleList() (*v1.GetModuleListResponse, error) {
	log.Infow("start to get all modules in biz layer")

	modules, _ := b.ds.Modules().GetNormalModules()
	response := &v1.GetModuleListResponse{
		Modules: make([]*v1.ModuleInfo, 0),
	}
	for _, module := range modules {
		response.Modules = append(response.Modules, &v1.ModuleInfo{
			Code:  module.Code,
			Title: module.Title,
		})
	}

	return response, nil
}

// Create 创建用户
func (b *blogBiz) GetModuleDetail(req *v1.GetModuleDetailRequest) (*v1.GetModuleDetailResponse, error) {
	log.Infow("start to get module detail in biz layer", "moduleCode", req.ModuleCode)

	// 获取模块
	module, _ := b.ds.Modules().GetByCode(req.ModuleCode)
	moduleDetail := &v1.ModuleDetail{
		ID:    module.ID,
		Code:  module.Code,
		Title: module.Title,
	}

	// 获取章节列表
	sections, _ := b.ds.Sections().GetListByModuleCode(req.ModuleCode)
	for _, section := range sections {
		sectionDetail := &v1.SectionDetail{
			ID:         section.ID,
			Code:       section.Code,
			Sort:       section.Sort,
			ModuleCode: section.ModuleCode,
			Title:      section.Title,
		}

		// 获取文章列表
		filter := map[string]interface{}{
			"section_code": section.Code,
			"status":       model.ArticleStatusPublished,
		}
		articles, _ := b.ds.Articles().GetList(filter, 1, 100)
		for _, article := range articles {
			articleDetail := &v1.ArticleDetail{
				ID:          article.ID,
				Title:       article.Title,
				SectionCode: article.SectionCode,
				Author:      article.Author,
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

func (b *blogBiz) GetArticleDetail(req *v1.GetArticleDetailRequest) (*v1.GetArticleDetailResponse, error) {
	log.Infow("start to get article detail in biz layer", "articleID", req.ArticleID)

	// 获取文章
	article, _ := b.ds.Articles().GetOne(req.ArticleID)
	articleDetail := &v1.ArticleDetail{
		ID:          article.ID,
		Title:       article.Title,
		Content:     article.Content,
		ExternalUrl: article.ExternalUrl,
		SectionCode: article.SectionCode,
		Author:      article.Author,
		Tags:        strings.Split(article.Tags, ","),
		CreatedAt:   article.CreatedAt,
		UpdatedAt:   article.UpdatedAt,
	}

	return &v1.GetArticleDetailResponse{
		ArticleDetail: articleDetail,
	}, nil
}
