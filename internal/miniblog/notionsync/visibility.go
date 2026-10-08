package notionsync

import (
	"context"
	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"gorm.io/gorm"
	"strings"
)

// Public conditions use the reader's authoritative query rather than a second
// approximation of its directory and source visibility policy.
func publicationBlockedPages(ctx context.Context, db *gorm.DB) *gorm.DB {
	visible := store.PublishedArticles(ctx, db).Select("article.id")
	return db.WithContext(ctx).Model(&model.NotionPageBinding{}).
		Joins("LEFT JOIN article ON article.id = notion_page_bindings.article_id").
		Where("notion_page_bindings.management_state = ?", model.NotionManagementManaged).
		Where("notion_page_bindings.desired_state = ? OR article.status = ? OR notion_page_bindings.publication_held = ? OR notion_page_bindings.needs_revalidation = ?", model.ArticleStatusPublished, model.ArticleStatusPublished, true, true).
		Where("notion_page_bindings.article_id IS NULL OR notion_page_bindings.article_id NOT IN (?)", visible)
}

func decoratePageVisibility(ctx context.Context, db *gorm.DB, rows []model.NotionPageBinding, items []PageDTO) error {
	ids := []uint64{}
	for _, p := range rows {
		if p.ArticleID != nil {
			ids = append(ids, *p.ArticleID)
		}
	}
	articles := []*model.Article{}
	visibleIDs := []uint64{}
	if len(ids) > 0 {
		if e := db.WithContext(ctx).Where("id IN ?", ids).Find(&articles).Error; e != nil {
			return e
		}
		if e := store.PublishedArticles(ctx, db).Where("article.id IN ?", ids).Pluck("article.id", &visibleIDs).Error; e != nil {
			return e
		}
	}
	associations, e := store.LoadArticleAssociations(ctx, db, articles)
	if e != nil {
		return e
	}
	byID := map[uint64]*model.Article{}
	visible := map[uint64]bool{}
	for _, a := range articles {
		byID[a.ID] = a
	}
	for _, id := range visibleIDs {
		visible[id] = true
	}
	for i, p := range rows {
		items[i].NeedsRevalidation = p.NeedsRevalidation
		var a *model.Article
		if p.ArticleID != nil {
			a = byID[*p.ArticleID]
			items[i].EffectiveVisibility = visible[*p.ArticleID]
		}
		if a != nil && stateName(a.Status) != "" {
			state := stateName(a.Status)
			items[i].LocalState = &state
		}
		items[i].VisibilityReason = pageVisibilityReason(p, a, associations, items[i].EffectiveVisibility)
	}
	return nil
}

func pageVisibilityReason(p model.NotionPageBinding, a *model.Article, relations *store.ArticleAssociations, visible bool) string {
	if visible {
		return ""
	}
	if p.ManagementState == model.NotionManagementBaselinePending {
		return "historical_match_requires_confirmation"
	}
	if p.ArticleID != nil && a == nil {
		return "article_missing"
	}
	if p.ManagementState == model.NotionManagementManaged {
		if p.NativeArchived || p.InTrash {
			return "native_archived_or_in_trash"
		}
		if p.PublicationHeld {
			return "publication_hold"
		}
		// An explicit isolation reason explains why revalidation is required.
		if p.PublishBlockReason != "" {
			return p.PublishBlockReason
		}
		if p.NeedsRevalidation {
			return "needs_revalidation"
		}
		if (a != nil && a.Status != model.ArticleStatusPublished) || (a == nil && p.DesiredState != model.ArticleStatusPublished) {
			return "state_not_published"
		}
		if p.PublicURL == nil || strings.TrimSpace(*p.PublicURL) == "" {
			return "public_url_missing"
		}
	}
	if a == nil {
		return "article_not_registered"
	}
	if a.Status != model.ArticleStatusPublished {
		return "state_not_published"
	}
	section := relations.Sections[a.SectionCode]
	if section == nil {
		return "catalog_invalid"
	}
	module := relations.Modules[section.ModuleCode]
	if module == nil {
		return "catalog_invalid"
	}
	var sub *model.Subsection
	if a.SubsectionCode != "" {
		sub = relations.Subsections[a.SubsectionCode]
		if sub == nil {
			return "catalog_invalid"
		}
	}
	if section.Status != model.SectionStatusNormal || module.Status != model.ModuleStatusNormal || (sub != nil && sub.Status != model.SubsectionStatusNormal) {
		return "catalog_disabled"
	}
	return "needs_revalidation"
}
