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

func (b *blogBiz) GetModuleDetail(req *v1.GetModuleDetailRequest) (*v1.GetModuleDetailResponse, error) {
	log.Infow("start to get module detail in biz layer", "moduleCode", req.ModuleCode)

	module, _ := b.ds.Modules().GetByCode(req.ModuleCode)
	moduleDetail := &v1.ModuleDetail{
		ID:    module.ID,
		Code:  module.Code,
		Title: module.Title,
	}

	sections, _ := b.ds.Sections().GetNormalSections(req.ModuleCode)
	for _, section := range sections {
		sectionDetail := &v1.SectionDetail{
			ID:         section.ID,
			Code:       section.Code,
			Sort:       section.Sort,
			ModuleCode: section.ModuleCode,
			Title:      section.Title,
		}

		subsections, _ := b.ds.Subsections().GetNormalSubsections(section.Code)
		for _, subsection := range subsections {
			subsectionDetail := &v1.SubsectionDetail{
				ID:          subsection.ID,
				Code:        subsection.Code,
				Sort:        subsection.Sort,
				SectionCode: subsection.SectionCode,
				Title:       subsection.Title,
			}

			filter := map[string]interface{}{
				"section_code":    section.Code,
				"subsection_code": subsection.Code,
				"status":          model.ArticleStatusPublished,
			}
			articles, _ := b.ds.Articles().GetList(filter, 1, 100)
			for _, article := range articles {
				subsectionDetail.Articles = append(subsectionDetail.Articles, toBlogArticleDetail(article))
			}

			sectionDetail.Subsections = append(sectionDetail.Subsections, subsectionDetail)
		}

		sectionFilter := map[string]interface{}{
			"section_code": section.Code,
			"status":       model.ArticleStatusPublished,
		}
		sectionArticles, _ := b.ds.Articles().GetList(sectionFilter, 1, 100)
		for _, article := range sectionArticles {
			if article.SubsectionCode != "" {
				continue
			}
			sectionDetail.Articles = append(sectionDetail.Articles, toBlogArticleDetail(article))
		}

		moduleDetail.Sections = append(moduleDetail.Sections, sectionDetail)
	}

	return &v1.GetModuleDetailResponse{
		ModuleDetail: moduleDetail,
	}, nil
}

func (b *blogBiz) GetArticleDetail(req *v1.GetArticleDetailRequest) (*v1.GetArticleDetailResponse, error) {
	log.Infow("start to get article detail in biz layer", "articleID", req.ArticleID)

	article, _ := b.ds.Articles().GetOne(uint64(req.ArticleID))
	articleDetail := &v1.ArticleDetail{
		ID:             article.ID,
		Title:          article.Title,
		Content:        article.Content,
		ExternalLink:   article.ExternalLink,
		SectionCode:    article.SectionCode,
		SubsectionCode: article.SubsectionCode,
		Author:         article.Author,
		Tags:           strings.Split(article.Tags, ","),
		Pos:            article.Pos,
		CreatedAt:      article.CreatedAt,
		UpdatedAt:      article.UpdatedAt,
	}

	return &v1.GetArticleDetailResponse{
		ArticleDetail: articleDetail,
	}, nil
}

func toBlogArticleDetail(article *model.Article) *v1.ArticleDetail {
	return &v1.ArticleDetail{
		ID:             article.ID,
		Title:          article.Title,
		SectionCode:    article.SectionCode,
		SubsectionCode: article.SubsectionCode,
		Author:         article.Author,
	}
}
