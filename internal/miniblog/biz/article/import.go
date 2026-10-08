package article

import (
	"context"
	"errors"
	"github.com/yshujie/miniblog/internal/miniblog/biz/catalog"
	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/source"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"math"
	"strings"
)

type ImportRequest struct {
	ID                                                       uint64
	Title, ExternalLink, SectionCode, SubsectionCode, Author string
	Content                                                  *string
	Tags                                                     []string
	Pos                                                      *int
	Status                                                   *int
}
type ImportResult struct {
	Outcome string
	ID      uint64
}

func (b *articleBiz) Import(ctx context.Context, r ImportRequest, dryRun bool) (*ImportResult, error) {
	if r.ID > math.MaxInt64 {
		return nil, invalid("显式ID超出有符号BIGINT范围")
	}
	var identity *source.Identity
	if strings.TrimSpace(r.ExternalLink) != "" {
		parsed, e := source.Parse(r.ExternalLink)
		if e != nil {
			return nil, invalid(e.Error())
		}
		identity = &parsed
	}
	// Without an explicit ID this is registration, never an update command.
	if r.ID == 0 && identity != nil {
		existing, err := findSource(b.contextStore(ctx), identity.SourceKey)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			return &ImportResult{Outcome: "already_registered", ID: existing.ID}, nil
		}
	}
	if e := validateFields(r.Title, r.Author, r.Tags); e != nil {
		return nil, e
	}
	if r.Status != nil && (*r.Status < 1 || *r.Status > 4) {
		return nil, invalid("文章状态无效")
	}
	result := &ImportResult{}
	e := store.InTransaction(ctx, b.ds, func(ds store.IStore) error {
		p, e := catalog.Resolve(ds, r.SectionCode, r.SubsectionCode)
		if e != nil {
			return e
		}
		var existing *model.Article
		if r.ID != 0 {
			existing, e = ds.Articles().GetOne(r.ID)
			if errors.Is(e, gorm.ErrRecordNotFound) {
				existing = nil
				e = nil
			}
			if e != nil {
				return e
			}
		} else if identity != nil {
			existing, e = findSource(ds, identity.SourceKey)
			if e != nil {
				return e
			}
			if existing != nil {
				result.Outcome = "already_registered"
				result.ID = existing.ID
				return nil
			}
		}
		codes := []string{p.Module.Code}
		var oldModuleID uint64
		if existing != nil {
			old, e := catalog.Resolve(ds, existing.SectionCode, existing.SubsectionCode)
			if e != nil {
				return e
			}
			codes = append(codes, old.Module.Code)
			oldModuleID = old.Module.ID
		}
		if e = catalog.LockModules(ds, codes...); e != nil {
			return e
		}
		p, e = catalog.Resolve(ds, r.SectionCode, r.SubsectionCode)
		if e != nil {
			return e
		}
		if identity != nil {
			duplicate, e := findSource(ds, identity.SourceKey)
			if e != nil {
				return e
			}
			if r.ID == 0 && duplicate != nil {
				result.Outcome = "already_registered"
				result.ID = duplicate.ID
				return nil
			}
			if duplicate != nil && (existing == nil || existing.ID != duplicate.ID) {
				return conflict("该文档已登记为另一ID")
			}
		}
		result.Outcome = "created"
		result.ID = r.ID
		if existing != nil {
			result.Outcome = "updated"
			result.ID = existing.ID
			if ds.DB() != nil {
				if e = ds.DB().Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", existing.ID).First(&existing).Error; e != nil {
					return e
				}
				current, err := catalog.Resolve(ds, existing.SectionCode, existing.SubsectionCode)
				if err != nil {
					return err
				}
				if current.Module.ID != oldModuleID {
					return conflict("文章位置已变更，请刷新后重试")
				}
			}
		}
		status := model.ArticleStatusDraft
		if existing != nil {
			status = existing.Status
		}
		if r.Status != nil {
			status = *r.Status
		}
		if existing != nil && existing.Status == model.ArticleStatusDeleted {
			same := status == existing.Status && r.Title == existing.Title && r.ExternalLink == existing.ExternalLink && r.Author == existing.Author && strings.Join(r.Tags, ",") == existing.Tags
			same = same && existing.SectionCode == p.Section.Code && ((p.Subsection == nil && existing.SubsectionCode == "") || (p.Subsection != nil && existing.SubsectionCode == p.Subsection.Code))
			if r.Content != nil {
				same = same && *r.Content == existing.Content
			}
			if r.Pos != nil {
				same = same && *r.Pos == existing.Pos
			}
			if !same {
				return conflict("归档文章须先独立恢复，再编辑或移动")
			}
			result.Outcome = "already_registered"
			return nil
		}
		candidate := &model.Article{}
		if existing != nil {
			copy := *existing
			candidate = &copy
		}
		candidate.Title = r.Title
		candidate.Author = r.Author
		candidate.Tags = strings.Join(r.Tags, ",")
		candidate.ExternalLink = r.ExternalLink
		if e = validatePublication(candidate, p, status); e != nil {
			return e
		}
		if dryRun {
			return nil
		}
		a := existing
		if a == nil {
			a = &model.Article{ID: r.ID}
		}
		changed := a.SectionCode != p.Section.Code || (p.Subsection == nil && a.SubsectionCode != "") || (p.Subsection != nil && a.SubsectionCode != p.Subsection.Code)
		a.Title = r.Title
		a.ExternalLink = r.ExternalLink
		if r.Content != nil {
			a.Content = *r.Content
		}
		a.Author = r.Author
		a.Tags = strings.Join(r.Tags, ",")
		a.Status = status
		canonicalPlacement(a, p)
		if identity != nil {
			bindIdentity(a, *identity)
		} else {
			a.Provider = nil
			a.CanonicalURL = nil
			a.SourceKey = nil
		}
		if r.Pos != nil {
			a.Pos = *r.Pos
		} else if existing == nil || changed {
			a.Pos, e = nextPos(ds, a.SectionCode, a.SubsectionCode)
			if e != nil {
				return e
			}
		}
		if existing == nil {
			e = ds.Articles().Create(a)
		} else {
			e = save(ds, a)
		}
		result.ID = a.ID
		return e
	})
	if e != nil && sourceUniqueError(e) && r.ID == 0 && identity != nil {
		existing, lookup := findSource(b.contextStore(ctx), identity.SourceKey)
		if lookup == nil && existing != nil {
			return &ImportResult{Outcome: "already_registered", ID: existing.ID}, nil
		}
	}
	if sourceUniqueError(e) {
		return nil, conflict("该文档已登记为另一ID")
	}
	return result, e
}
