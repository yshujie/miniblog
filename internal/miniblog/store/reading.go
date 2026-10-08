package store

import (
	"context"
	"github.com/yshujie/miniblog/internal/miniblog/model"
	"gorm.io/gorm"
	"strings"
)

type ArticleFilter struct {
	ModuleCode, SectionCode, SubsectionCode, Title string
	Status                                         int
	DirectOnly                                     bool
	Page, Limit                                    int
}

func FilterArticles(db *gorm.DB, f ArticleFilter) *gorm.DB {
	q := db.Model(&model.Article{})
	if f.ModuleCode != "" {
		q = q.Joins("JOIN section ON section.code = article.section_code").Where("section.module_code = ?", f.ModuleCode)
	}
	if f.SectionCode != "" {
		q = q.Where("article.section_code = ?", f.SectionCode)
	}
	if f.SubsectionCode != "" {
		q = q.Where("article.subsection_code = ?", f.SubsectionCode)
	} else if f.DirectOnly {
		q = q.Where("article.subsection_code = '' OR article.subsection_code IS NULL")
	}
	if f.Title != "" {
		title := strings.NewReplacer("#", "##", "%", "#%", "_", "#_").Replace(f.Title)
		q = q.Where("article.title LIKE ? ESCAPE '#'", "%"+title+"%")
	}
	if f.Status != 0 {
		q = q.Where("article.status = ?", f.Status)
	}
	return q
}

func ListArticles(ctx context.Context, db *gorm.DB, f ArticleFilter) ([]*model.Article, int, error) {
	var total int64
	if err := FilterArticles(db.WithContext(ctx), f).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	rows := make([]*model.Article, 0)
	err := FilterArticles(db.WithContext(ctx), f).Select("article.*").Order("article.pos asc, article.id asc").Offset((f.Page - 1) * f.Limit).Limit(f.Limit).Find(&rows).Error
	return rows, int(total), err
}

// PublishedArticles applies the same full-path visibility policy to list and detail.
func PublishedArticles(ctx context.Context, db *gorm.DB) *gorm.DB {
	return db.WithContext(ctx).Model(&model.Article{}).
		Joins("JOIN section ON section.code = article.section_code").
		Joins("JOIN module ON module.code = section.module_code").
		Joins("LEFT JOIN subsection ON subsection.code = article.subsection_code AND subsection.section_code = section.code").
		Where("article.status = ? AND section.status = ? AND module.status = ?", model.ArticleStatusPublished, model.SectionStatusNormal, model.ModuleStatusNormal).
		Where("article.subsection_code IS NULL OR article.subsection_code = '' OR subsection.status = ?", model.SubsectionStatusNormal)
}

type ArticleAssociations struct {
	Modules     map[string]*model.Module
	Sections    map[string]*model.Section
	Subsections map[string]*model.Subsection
}

// CanonicalArticle projects the actual directory codes selected by the database.
// MySQL collation equivalence is broader than Go string/lowercase equivalence.
type CanonicalArticle struct {
	model.Article           `gorm:"embedded"`
	CanonicalModuleCode     string
	CanonicalSectionCode    string
	CanonicalSubsectionCode string
}

const CanonicalArticleSelect = "article.*, module.code AS canonical_module_code, section.code AS canonical_section_code, subsection.code AS canonical_subsection_code"

func LoadArticleAssociations(ctx context.Context, db *gorm.DB, articles []*model.Article) (*ArticleAssociations, error) {
	result := &ArticleAssociations{Modules: map[string]*model.Module{}, Sections: map[string]*model.Section{}, Subsections: map[string]*model.Subsection{}}
	if len(articles) == 0 {
		return result, nil
	}
	type relation struct{ RawSectionCode, RawSubsectionCode, SectionCode, ModuleCode, SubsectionCode string }
	ids := make([]uint64, 0, len(articles))
	for _, a := range articles {
		ids = append(ids, a.ID)
	}
	relations := []relation{}
	err := db.WithContext(ctx).Model(&model.Article{}).
		Joins("LEFT JOIN section ON section.code = article.section_code").
		Joins("LEFT JOIN module ON module.code = section.module_code").
		Joins("LEFT JOIN subsection ON subsection.code = article.subsection_code AND subsection.section_code = section.code").
		Where("article.id IN ?", ids).
		Select("article.section_code AS raw_section_code, article.subsection_code AS raw_subsection_code, section.code AS section_code, module.code AS module_code, subsection.code AS subsection_code").
		Scan(&relations).Error
	if err != nil {
		return nil, err
	}
	sections, modules, subs := []string{}, []string{}, []string{}
	for _, r := range relations {
		if r.SectionCode != "" {
			sections = append(sections, r.SectionCode)
		}
		if r.ModuleCode != "" {
			modules = append(modules, r.ModuleCode)
		}
		if r.SubsectionCode != "" {
			subs = append(subs, r.SubsectionCode)
		}
	}
	sectionMap := map[string]*model.Section{}
	moduleMap := map[string]*model.Module{}
	subMap := map[string]*model.Subsection{}
	if len(sections) > 0 {
		var rows []*model.Section
		if err = db.WithContext(ctx).Where("code IN ?", sections).Find(&rows).Error; err != nil {
			return nil, err
		}
		for _, s := range rows {
			sectionMap[s.Code] = s
		}
	}
	if len(modules) > 0 {
		var rows []*model.Module
		if err = db.WithContext(ctx).Where("code IN ?", modules).Find(&rows).Error; err != nil {
			return nil, err
		}
		for _, m := range rows {
			moduleMap[m.Code] = m
		}
	}
	if len(subs) > 0 {
		var rows []*model.Subsection
		if err = db.WithContext(ctx).Where("code IN ?", subs).Find(&rows).Error; err != nil {
			return nil, err
		}
		for _, s := range rows {
			subMap[s.Code] = s
		}
	}
	for _, r := range relations {
		if s := sectionMap[r.SectionCode]; s != nil {
			result.Sections[r.RawSectionCode] = s
			result.Modules[s.ModuleCode] = moduleMap[r.ModuleCode]
		}
		if s := subMap[r.SubsectionCode]; s != nil {
			result.Subsections[r.RawSubsectionCode] = s
		}
	}
	return result, nil
}
