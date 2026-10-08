package article

import (
	"context"
	"errors"
	"github.com/yshujie/miniblog/internal/miniblog/biz/catalog"
	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/source"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"github.com/yshujie/miniblog/internal/pkg/errno"
	v1 "github.com/yshujie/miniblog/pkg/api/miniblog/v1"
	"gorm.io/gorm/clause"
	"strings"
	"unicode/utf8"
)

func sourceManaged() error {
	return &errno.Errno{HTTP: 409, Code: "SourceManaged", Message: "文章由Notion管理，请在来源修改"}
}
func checkManualWritable(ds store.IStore, a *model.Article) error {
	if e := store.CheckSourceWrites(ds); e != nil {
		if errors.Is(e, store.ErrSourceWritesPaused) {
			return &errno.Errno{HTTP: 503, Code: "SourceWritesPaused", Message: "来源维护中，请稍后重试"}
		}
		return e
	}
	p, e := store.PageBindingByArticle(ds, a.ID)
	if e != nil {
		return e
	}
	if p != nil && p.ManagementState == model.NotionManagementManaged {
		return sourceManaged()
	}
	return nil
}

// A managed source page can be pending without a blog article. Manual writers
// must not create another ownership path for that document.
func checkPendingSourceOwner(ds store.IStore, identity source.Identity) error {
	_, e := findSource(ds, identity.SourceKey)
	return e
}
func bindParsedIdentity(ds store.IStore, a *model.Article, id source.Identity) error {
	p, e := store.PageBindingByArticle(ds, a.ID)
	if e != nil {
		return e
	}
	if p != nil {
		if (a.SourceKey != nil && *a.SourceKey == id.SourceKey) || (p.LegacySourceKey != nil && *p.LegacySourceKey == id.SourceKey) {
			return nil
		}
		return conflict("已核验PageID来源不能通过编辑更换")
	}
	bindIdentity(a, id)
	return nil
}
func lockedManagedBinding(ds store.IStore, a *model.Article) (*model.NotionPageBinding, error) {
	if !store.HasNotionSyncSchema(ds.DB()) {
		return nil, sourceManaged()
	}
	var p model.NotionPageBinding
	if e := ds.DB().Clauses(clause.Locking{Strength: "UPDATE"}).Where("article_id = ?", a.ID).First(&p).Error; e != nil {
		return nil, e
	}
	if p.ManagementState != model.NotionManagementManaged {
		return nil, conflict("操作仅适用于已接管的Notion文章")
	}
	return &p, nil
}
func (b *articleBiz) PatchLocal(ctx context.Context, id uint64, r LocalPatchInput) (*v1.ArticleInfoResponse, error) {
	if utf8.RuneCountInString(r.Author) > 128 {
		return nil, invalid("作者过长")
	}
	e := b.withLocked(ctx, id, "", "", func(ds store.IStore, a *model.Article, _ *catalog.Placement) error {
		if a.Status == model.ArticleStatusDeleted {
			return conflict("归档文章不能编辑本地信息")
		}
		if _, e := lockedManagedBinding(ds, a); e != nil {
			return e
		}
		if a.Author == r.Author {
			return nil
		}
		return ds.DB().Model(a).Update("author", r.Author).Error
	})
	if e != nil {
		return nil, e
	}
	out, e := b.GetOne(ctx, id)
	if e != nil {
		return nil, e
	}
	return &v1.ArticleInfoResponse{Article: out.Article}, nil
}
func (b *articleBiz) SetPublicationHold(ctx context.Context, id uint64, r HoldInput) (*v1.ArticleInfoResponse, error) {
	if utf8.RuneCountInString(r.Reason) > 255 {
		return nil, invalid("暂停原因过长")
	}
	e := b.withLocked(ctx, id, "", "", func(ds store.IStore, a *model.Article, _ *catalog.Placement) error {
		p, e := lockedManagedBinding(ds, a)
		if e != nil {
			return e
		}
		if p.PublicationHeld == r.Held && (!r.Held || p.PublicationHoldReason == strings.TrimSpace(r.Reason)) {
			return nil
		}
		values := map[string]interface{}{"publication_held": r.Held, "publication_hold_reason": strings.TrimSpace(r.Reason), "revision": p.Revision + 1}
		if r.Held {
			values["publish_block_reason"] = "publication_hold"
		} else {
			values["needs_revalidation"] = true
			values["publish_block_reason"] = "needs_revalidation"
			values["publication_hold_reason"] = ""
		}
		return ds.DB().Model(p).Updates(values).Error
	})
	if e != nil {
		return nil, e
	}
	out, e := b.GetOne(ctx, id)
	if e != nil {
		return nil, e
	}
	return &v1.ArticleInfoResponse{Article: out.Article}, nil
}

func managedEligible(p *model.NotionPageBinding) bool {
	return p == nil || p.ManagementState != model.NotionManagementManaged || (!p.PublicationHeld && !p.NeedsRevalidation && !p.NativeArchived && !p.InTrash && p.PublishBlockReason == "" && p.PublicURL != nil && strings.TrimSpace(*p.PublicURL) != "")
}
func decorateManagement(out *v1.ArticleInfo, a *model.Article, associations *store.ArticleAssociations) {
	out.ReadingURL = a.ExternalLink
	out.Management = v1.ArticleManagement{Mode: "manual", ManagedFields: []string{}}
	p := associations.PageBindings[a.ID]
	if p != nil && p.ManagementState == model.NotionManagementManaged {
		if p.PageURL != "" {
			out.ReadingURL = p.PageURL
		}
		if p.PublicURL != nil && *p.PublicURL != "" {
			out.ReadingURL = *p.PublicURL
		}
	}
	if p != nil && p.ManagementState == model.NotionManagementManaged {
		out.Management = v1.ArticleManagement{Mode: "notion_sync", SourceID: p.SourceID, ManagedFields: []string{"title", "tags", "module_code", "section_code", "subsection_code", "status", "reading_url", "external_link"}}
		out.PublicationHold = v1.PublicationHold{Held: p.PublicationHeld, Reason: p.PublicationHoldReason}
		out.AllowedActions = []string{"view_source", "reorder"}
		if a.Status != model.ArticleStatusDeleted {
			out.AllowedActions = append(out.AllowedActions, "edit_local_fields")
		}
		if p.PublicationHeld {
			out.AllowedActions = append(out.AllowedActions, "release_hold")
		} else {
			out.AllowedActions = append(out.AllowedActions, "hold")
		}
	} else {
		if a.Status == model.ArticleStatusDeleted {
			out.AllowedActions = []string{"restore"}
		} else {
			out.AllowedActions = []string{"edit", "move", "archive", "reorder"}
			if a.Status == model.ArticleStatusPublished {
				out.AllowedActions = append(out.AllowedActions, "unpublish")
			} else {
				out.AllowedActions = append(out.AllowedActions, "publish")
			}
		}
	}
	out.EffectiveVisibility = a.Status == model.ArticleStatusPublished && out.Module.Status == model.ModuleStatusNormal && out.Section.Status == model.SectionStatusNormal && (out.Subsection == nil || out.Subsection.Status == model.SubsectionStatusNormal) && managedEligible(p)
}

func editIsNoop(ds store.IStore, a *model.Article, target *catalog.Placement, r UpdateInput) (bool, error) {
	tagsSame, e := model.ArticleTagsEqual(a, r.Tags)
	if e != nil {
		return false, e
	}
	if a.Title != r.Title || a.Author != r.Author || a.ExternalLink != r.ExternalLink || !tagsSame || (r.Content != nil && *r.Content != a.Content) {
		return false, nil
	}
	old, e := catalog.Resolve(ds, a.SectionCode, a.SubsectionCode)
	if e != nil {
		return false, e
	}
	if old.Section.ID != target.Section.ID || (old.Subsection == nil) != (target.Subsection == nil) {
		return false, nil
	}
	if old.Subsection != nil && old.Subsection.ID != target.Subsection.ID {
		return false, nil
	}
	if r.ModuleCode != "" {
		m, e := ds.Modules().GetByCode(r.ModuleCode)
		if e != nil {
			return false, e
		}
		if m == nil || m.ID != target.Module.ID {
			return false, invalid("章节不属于当前模块")
		}
	}
	return true, nil
}
