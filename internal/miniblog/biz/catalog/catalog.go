// Package catalog owns directory relationships and mutations.
package catalog

import (
	"context"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"github.com/yshujie/miniblog/internal/pkg/errno"
	"gorm.io/gorm/clause"
)

type Placement struct {
	Module     *model.Module
	Section    *model.Section
	Subsection *model.Subsection
}
type Service struct{ ds store.IStore }

func New(ds store.IStore) *Service { return &Service{ds: ds} }
func invalid(message string) error {
	return &errno.Errno{HTTP: 400, Code: errno.ErrInvalidParameter.Code, Message: message}
}

func Resolve(ds store.IStore, sectionCode, subsectionCode string) (*Placement, error) {
	section, err := ds.Sections().GetByCode(sectionCode)
	if err != nil {
		return nil, err
	}
	if section == nil {
		return nil, errno.ErrSectionNotFound
	}
	module, err := ds.Modules().GetByCode(section.ModuleCode)
	if err != nil {
		return nil, err
	}
	if module == nil {
		return nil, errno.ErrModuleNotFound
	}
	p := &Placement{Module: module, Section: section}
	if subsectionCode != "" {
		sub, err := ds.Subsections().GetByCode(subsectionCode)
		if err != nil {
			return nil, err
		}
		if sub == nil {
			return nil, errno.ErrSubsectionNotFound
		}
		parent, err := ds.Sections().GetByCode(sub.SectionCode)
		if err != nil {
			return nil, err
		}
		if parent == nil || parent.ID != section.ID {
			return nil, invalid("子章节不属于当前章节")
		}
		p.Subsection = sub
	}
	return p, nil
}

func RequireVisible(p *Placement) error {
	if p.Module.Status != model.ModuleStatusNormal || p.Section.Status != model.SectionStatusNormal || (p.Subsection != nil && p.Subsection.Status != model.SubsectionStatusNormal) {
		return invalid("发布位置必须属于已上架目录")
	}
	return nil
}

// All writers lock module rows first, in ID order, then article rows.
func LockModules(ds store.IStore, codes ...string) error {
	if ds.DB() == nil {
		return nil
	}
	set := map[string]bool{}
	for _, code := range codes {
		if code != "" {
			set[code] = true
		}
	}
	ordered := make([]string, 0, len(set))
	for code := range set {
		ordered = append(ordered, code)
	}
	sort.Strings(ordered)
	if len(ordered) == 0 {
		return nil
	}
	var modules []*model.Module
	if err := ds.DB().Where("code IN ?", ordered).Order("id asc").Find(&modules).Error; err != nil {
		return err
	}
	if len(modules) != len(ordered) {
		return errno.ErrModuleNotFound
	}
	// An IN query can lock in index scan order before sorting. Acquire each
	// canonical primary-key row explicitly so opposite cross-module moves agree.
	for _, m := range modules {
		var locked model.Module
		if err := ds.DB().Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", m.ID).First(&locked).Error; err != nil {
			return err
		}
	}
	return nil
}

func validName(code, title string) error {
	if strings.TrimSpace(code) == "" || utf8.RuneCountInString(code) > 128 || strings.TrimSpace(title) == "" || utf8.RuneCountInString(title) > 255 {
		return invalid("目录编码须为1-128字，标题须为1-255字")
	}
	return nil
}

func (s *Service) CreateModule(ctx context.Context, r ModuleInput) (result *model.Module, err error) {
	if err = validName(r.Code, r.Title); err != nil {
		return
	}
	err = store.InTransaction(ctx, s.ds, func(ds store.IStore) error {
		existing, e := ds.Modules().GetByCode(r.Code)
		if e != nil {
			return e
		}
		if existing != nil {
			return errno.ErrModuleAlreadyExists
		}
		result = &model.Module{Code: r.Code, Title: r.Title, Status: model.ModuleStatusNormal}
		if r.Sort != nil {
			result.Sort = *r.Sort
		}
		return ds.Modules().Create(result)
	})
	return
}
func (s *Service) UpdateModule(ctx context.Context, code string, r UpdateInput) (result *model.Module, err error) {
	err = store.InTransaction(ctx, s.ds, func(ds store.IStore) error {
		if e := LockModules(ds, code); e != nil {
			return e
		}
		m, e := ds.Modules().GetByCode(code)
		if e != nil {
			return e
		}
		if m == nil {
			return errno.ErrModuleNotFound
		}
		if e := validName(m.Code, r.Title); e != nil {
			return e
		}
		m.Title = r.Title
		if r.Sort != nil {
			m.Sort = *r.Sort
		}
		result = m
		return ds.Modules().Update(m)
	})
	return
}
func (s *Service) ModuleStatus(ctx context.Context, code string, status int) (result *model.Module, err error) {
	err = store.InTransaction(ctx, s.ds, func(ds store.IStore) error {
		if e := LockModules(ds, code); e != nil {
			return e
		}
		m, e := ds.Modules().GetByCode(code)
		if e != nil {
			return e
		}
		if m == nil {
			return errno.ErrModuleNotFound
		}
		m.Status = status
		result = m
		return ds.Modules().Update(m)
	})
	return
}
func (s *Service) DeleteModule(ctx context.Context, code string) error {
	return store.InTransaction(ctx, s.ds, func(ds store.IStore) error {
		if e := LockModules(ds, code); e != nil {
			return e
		}
		m, e := ds.Modules().GetByCode(code)
		if e != nil {
			return e
		}
		if m == nil {
			return errno.ErrModuleNotFound
		}
		if e = CheckModuleBindingDependency(ds, m.Code); e != nil {
			return e
		}
		children, e := ds.Sections().GetSections(m.Code)
		if e != nil {
			return e
		}
		if len(children) > 0 {
			return errno.ErrModuleHasDependents
		}
		return ds.Modules().DeleteByCode(m.Code)
	})
}
func (s *Service) CreateSection(ctx context.Context, r SectionInput) (result *model.Section, err error) {
	if err = validName(r.Code, r.Title); err != nil {
		return
	}
	err = store.InTransaction(ctx, s.ds, func(ds store.IStore) error {
		if e := LockModules(ds, r.ModuleCode); e != nil {
			return e
		}
		parent, e := ds.Modules().GetByCode(r.ModuleCode)
		if e != nil {
			return e
		}
		if parent == nil {
			return errno.ErrModuleNotFound
		}
		existing, e := ds.Sections().GetByCode(r.Code)
		if e != nil {
			return e
		}
		if existing != nil {
			return errno.ErrSectionAlreadyExists
		}
		result = &model.Section{Code: r.Code, Title: r.Title, ModuleCode: parent.Code, Status: model.SectionStatusNormal}
		if r.Sort != nil {
			result.Sort = *r.Sort
		}
		return ds.Sections().Create(result)
	})
	return
}
func (s *Service) mutateSection(ctx context.Context, code string, fn func(store.IStore, *model.Section) error) error {
	return store.InTransaction(ctx, s.ds, func(ds store.IStore) error {
		section, e := ds.Sections().GetByCode(code)
		if e != nil {
			return e
		}
		if section == nil {
			return errno.ErrSectionNotFound
		}
		if e = LockModules(ds, section.ModuleCode); e != nil {
			return e
		}
		section, e = ds.Sections().GetByCode(code)
		if e != nil {
			return e
		}
		if section == nil {
			return errno.ErrSectionNotFound
		}
		return fn(ds, section)
	})
}
func (s *Service) UpdateSection(ctx context.Context, code string, r UpdateInput) (result *model.Section, err error) {
	err = s.mutateSection(ctx, code, func(ds store.IStore, m *model.Section) error {
		if e := validName(m.Code, r.Title); e != nil {
			return e
		}
		if m.Title != r.Title {
			if e := CheckSectionBindingDependency(ds, m.Code); e != nil {
				return e
			}
		}
		m.Title = r.Title
		if r.Sort != nil {
			m.Sort = *r.Sort
		}
		result = m
		return ds.Sections().Update(m)
	})
	return
}
func (s *Service) SectionStatus(ctx context.Context, code string, status int) (result *model.Section, err error) {
	err = s.mutateSection(ctx, code, func(ds store.IStore, m *model.Section) error {
		m.Status = status
		result = m
		return ds.Sections().Update(m)
	})
	return
}
func (s *Service) DeleteSection(ctx context.Context, code string) error {
	return s.mutateSection(ctx, code, func(ds store.IStore, m *model.Section) error {
		if e := CheckSectionBindingDependency(ds, m.Code); e != nil {
			return e
		}
		children, e := ds.Subsections().GetSubsections(m.Code)
		if e != nil {
			return e
		}
		if len(children) > 0 {
			return errno.ErrSectionHasSubsections
		}
		articles, e := ds.Articles().GetList(map[string]interface{}{"section_code": m.Code}, 1, 1)
		if e != nil {
			return e
		}
		if len(articles) > 0 {
			return errno.ErrSectionHasArticles
		}
		return ds.Sections().DeleteByCode(m.Code)
	})
}
func (s *Service) CreateSubsection(ctx context.Context, r SubsectionInput) (result *model.Subsection, err error) {
	if err = validName(r.Code, r.Title); err != nil {
		return
	}
	err = store.InTransaction(ctx, s.ds, func(ds store.IStore) error {
		p, e := Resolve(ds, r.SectionCode, "")
		if e != nil {
			return e
		}
		if e = LockModules(ds, p.Module.Code); e != nil {
			return e
		}
		p, e = Resolve(ds, r.SectionCode, "")
		if e != nil {
			return e
		}
		existing, e := ds.Subsections().GetByCode(r.Code)
		if e != nil {
			return e
		}
		if existing != nil {
			return errno.ErrSubsectionAlreadyExists
		}
		result = &model.Subsection{Code: r.Code, Title: r.Title, SectionCode: p.Section.Code, Status: model.SubsectionStatusNormal}
		if r.Sort != nil {
			result.Sort = *r.Sort
		}
		return ds.Subsections().Create(result)
	})
	return
}
func (s *Service) mutateSubsection(ctx context.Context, code string, fn func(store.IStore, *model.Subsection) error) error {
	return store.InTransaction(ctx, s.ds, func(ds store.IStore) error {
		sub, e := ds.Subsections().GetByCode(code)
		if e != nil {
			return e
		}
		if sub == nil {
			return errno.ErrSubsectionNotFound
		}
		p, e := Resolve(ds, sub.SectionCode, sub.Code)
		if e != nil {
			return e
		}
		if e = LockModules(ds, p.Module.Code); e != nil {
			return e
		}
		sub, e = ds.Subsections().GetByCode(code)
		if e != nil {
			return e
		}
		if sub == nil {
			return errno.ErrSubsectionNotFound
		}
		return fn(ds, sub)
	})
}
func (s *Service) UpdateSubsection(ctx context.Context, code string, r UpdateInput) (result *model.Subsection, err error) {
	err = s.mutateSubsection(ctx, code, func(ds store.IStore, m *model.Subsection) error {
		if e := validName(m.Code, r.Title); e != nil {
			return e
		}
		m.Title = r.Title
		if r.Sort != nil {
			m.Sort = *r.Sort
		}
		result = m
		return ds.Subsections().Update(m)
	})
	return
}
func (s *Service) SubsectionStatus(ctx context.Context, code string, status int) (result *model.Subsection, err error) {
	err = s.mutateSubsection(ctx, code, func(ds store.IStore, m *model.Subsection) error {
		m.Status = status
		result = m
		return ds.Subsections().Update(m)
	})
	return
}
func (s *Service) DeleteSubsection(ctx context.Context, code string) error {
	return s.mutateSubsection(ctx, code, func(ds store.IStore, m *model.Subsection) error {
		articles, e := ds.Articles().GetList(map[string]interface{}{"subsection_code": m.Code}, 1, 1)
		if e != nil {
			return e
		}
		if len(articles) > 0 {
			return errno.ErrSubsectionHasArticles
		}
		return ds.Subsections().DeleteByCode(m.Code)
	})
}

func (s *Service) Reorder(ctx context.Context, r ReorderInput) error {
	return store.InTransaction(ctx, s.ds, func(ds store.IStore) error {
		if ds.DB() == nil {
			return invalid("目录排序需要数据库")
		}
		seen := map[string]bool{}
		for _, code := range r.Codes {
			if seen[code] {
				return invalid("排序编码不能重复")
			}
			seen[code] = true
		}
		var actual []string
		switch r.Kind {
		case "module":
			var rows []*model.Module
			if e := ds.DB().Clauses(clause.Locking{Strength: "UPDATE"}).Order("id asc").Find(&rows).Error; e != nil {
				return e
			}
			for _, row := range rows {
				actual = append(actual, row.Code)
			}
		case "section":
			if r.ParentCode == "" {
				return invalid("必须指定父目录")
			}
			if e := LockModules(ds, r.ParentCode); e != nil {
				return e
			}
			rows, e := ds.Sections().GetSections(r.ParentCode)
			if e != nil {
				return e
			}
			for _, row := range rows {
				actual = append(actual, row.Code)
			}
		case "subsection":
			p, e := Resolve(ds, r.ParentCode, "")
			if e != nil {
				return e
			}
			if e = LockModules(ds, p.Module.Code); e != nil {
				return e
			}
			rows, e := ds.Subsections().GetSubsections(p.Section.Code)
			if e != nil {
				return e
			}
			for _, row := range rows {
				actual = append(actual, row.Code)
			}
		default:
			return invalid("目录类型无效")
		}
		if len(actual) != len(r.Codes) {
			return invalid("请提交当前目录完整排序集合")
		}
		for _, code := range actual {
			if !seen[code] {
				return invalid("排序集合与当前目录不一致")
			}
		}
		for i, code := range r.Codes {
			var e error
			switch r.Kind {
			case "module":
				m, err := ds.Modules().GetByCode(code)
				if err != nil {
					return err
				}
				if m == nil {
					return errno.ErrModuleNotFound
				}
				m.Sort = i + 1
				e = ds.Modules().Update(m)
			case "section":
				m, err := ds.Sections().GetByCode(code)
				if err != nil {
					return err
				}
				if m == nil {
					return errno.ErrSectionNotFound
				}
				m.Sort = i + 1
				e = ds.Sections().Update(m)
			case "subsection":
				m, err := ds.Subsections().GetByCode(code)
				if err != nil {
					return err
				}
				if m == nil {
					return errno.ErrSubsectionNotFound
				}
				m.Sort = i + 1
				e = ds.Subsections().Update(m)
			}
			if e != nil {
				return e
			}
		}
		return nil
	})
}
