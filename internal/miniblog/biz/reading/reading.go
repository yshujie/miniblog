package reading

import (
	"context"
	"errors"
	"strings"

	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"github.com/yshujie/miniblog/internal/pkg/errno"
	v1 "github.com/yshujie/miniblog/pkg/api/miniblog/v1"
	"gorm.io/gorm"
)

type Service struct{ ds store.IStore }

func New(ds store.IStore) *Service { return &Service{ds: ds} }

func (b *Service) Modules(ctx context.Context) (*v1.GetModuleListResponse, error) {
	rows := make([]*model.Module, 0)
	if err := b.ds.DB().WithContext(ctx).Where("status = ?", model.ModuleStatusNormal).Order("sort asc, id asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := &v1.GetModuleListResponse{Modules: make([]*v1.ModuleInfo, 0, len(rows))}
	for _, m := range rows {
		out.Modules = append(out.Modules, &v1.ModuleInfo{ID: int(m.ID), Code: m.Code, Title: m.Title, Sort: m.Sort, Status: m.Status})
	}
	return out, nil
}

// A module page uses four bounded queries, regardless of directory/article count.
func (b *Service) Module(ctx context.Context, code string) (*v1.GetModuleDetailResponse, error) {
	db := b.ds.DB().WithContext(ctx)
	var m model.Module
	err := db.Where("code = ? AND status = ?", code, model.ModuleStatusNormal).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errno.ErrModuleNotFound
	}
	if err != nil {
		return nil, err
	}
	out := &v1.ModuleDetail{ID: m.ID, Code: m.Code, Title: m.Title, Sections: make([]*v1.SectionDetail, 0)}
	sections := make([]*model.Section, 0)
	if err = db.Where("module_code = ? AND status = ?", m.Code, model.SectionStatusNormal).Order("sort asc, id asc").Find(&sections).Error; err != nil {
		return nil, err
	}
	sectionMap := map[string]*v1.SectionDetail{}
	codes := make([]string, 0, len(sections))
	for _, s := range sections {
		d := &v1.SectionDetail{ID: s.ID, Code: s.Code, Sort: s.Sort, ModuleCode: m.Code, Title: s.Title, Subsections: make([]*v1.SubsectionDetail, 0), Articles: make([]*v1.ArticleDetail, 0)}
		sectionMap[s.Code] = d
		codes = append(codes, s.Code)
		out.Sections = append(out.Sections, d)
	}
	if len(codes) == 0 {
		return &v1.GetModuleDetailResponse{ModuleDetail: out}, nil
	}
	type canonicalSubsection struct {
		model.Subsection     `gorm:"embedded"`
		CanonicalSectionCode string
	}
	subs := make([]*canonicalSubsection, 0)
	if err = db.Model(&model.Subsection{}).Joins("JOIN section ON section.code = subsection.section_code").
		Where("section.code IN ? AND subsection.status = ?", codes, model.SubsectionStatusNormal).
		Select("subsection.*, section.code AS canonical_section_code").Order("subsection.sort asc, subsection.id asc").Find(&subs).Error; err != nil {
		return nil, err
	}
	subMap := map[string]*v1.SubsectionDetail{}
	for _, s := range subs {
		if parent := sectionMap[s.CanonicalSectionCode]; parent != nil {
			d := &v1.SubsectionDetail{ID: s.ID, Code: s.Code, Sort: s.Sort, SectionCode: s.CanonicalSectionCode, Title: s.Title, Articles: make([]*v1.ArticleDetail, 0)}
			parent.Subsections = append(parent.Subsections, d)
			subMap[s.Code] = d
		}
	}
	articles := make([]*store.CanonicalArticle, 0)
	if err = store.PublishedArticles(ctx, db).Where("module.code = ?", m.Code).Select(store.CanonicalArticleSelect).Order("article.pos asc, article.id asc").Find(&articles).Error; err != nil {
		return nil, err
	}
	for _, a := range articles {
		a.SectionCode = a.CanonicalSectionCode
		a.SubsectionCode = a.CanonicalSubsectionCode
		d := toDetail(&a.Article, a.CanonicalModuleCode)
		if a.SubsectionCode == "" {
			if s := sectionMap[a.SectionCode]; s != nil {
				s.Articles = append(s.Articles, d)
			}
		} else if sub := subMap[a.SubsectionCode]; sub != nil && sub.SectionCode == a.SectionCode {
			sub.Articles = append(sub.Articles, d)
		}
	}
	return &v1.GetModuleDetailResponse{ModuleDetail: out}, nil
}
func (b *Service) Article(ctx context.Context, id uint64) (*v1.GetArticleDetailResponse, error) {
	if id == 0 {
		return nil, errno.ErrInvalidParameter
	}
	var a store.CanonicalArticle
	q := store.PublishedArticles(ctx, b.ds.DB()).Where("article.id = ?", id)
	err := q.Select(store.CanonicalArticleSelect).First(&a).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errno.ErrArticleNotFound
	}
	if err != nil {
		return nil, err
	}
	a.SectionCode = a.CanonicalSectionCode
	a.SubsectionCode = a.CanonicalSubsectionCode
	return &v1.GetArticleDetailResponse{ArticleDetail: toDetail(&a.Article, a.CanonicalModuleCode)}, nil
}
func toDetail(a *model.Article, module string) *v1.ArticleDetail {
	tags := []string{}
	if a.Tags != "" {
		tags = strings.Split(a.Tags, ",")
	}
	return &v1.ArticleDetail{ID: a.ID, Title: a.Title, Content: a.Content, ExternalLink: a.ExternalLink, ModuleCode: module, SectionCode: a.SectionCode, SubsectionCode: a.SubsectionCode, Author: a.Author, Tags: tags, Pos: a.Pos, Provider: a.Provider, CreatedAt: a.CreatedAt, UpdatedAt: a.UpdatedAt}
}
