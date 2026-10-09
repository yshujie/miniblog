package notionsync

import (
	"context"

	"github.com/yshujie/miniblog/internal/miniblog/biz/article"
	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"gorm.io/gorm"
)

func reviewedPublicationInput(snap Snapshot, cfg SourceConfig, binding model.NotionPageBinding, local *model.Article) article.ReviewedPublicationInput {
	r := article.ReviewedPublicationInput{Title: snap.Title, Tags: snap.Tags, TopicPropertyID: cfg.TopicPropertyID, TopicOptionID: snap.TopicOptionID, TopicOptionName: snap.TopicOptionName, PublicURL: snap.PublicURL, NativeArchived: snap.NativeArchived, InTrash: snap.InTrash, PublicationHeld: binding.PublicationHeld}
	if local != nil {
		r.LocalSectionCode, r.LocalSubsectionCode = local.SectionCode, local.SubsectionCode
	}
	return r
}

// validateBootstrapScope reads all scoped pages before any PATCH. An unreadable
// later confirmation cannot permit earlier confirmations to write outside a complete scope audit.
func (s *Service) validateBootstrapScope(ctx context.Context, token store.LeaseToken, in BootstrapInput) error {
	if in.SourceID == "" {
		return nil
	}
	check := func(tx *gorm.DB, c *model.NotionSyncControl) error {
		if !c.Paused || !c.SourceWritesPaused || !c.BaselineFrozen {
			return conflict("受限接管需要双暂停与冻结基线")
		}
		var src model.NotionSyncSource
		if err := tx.Where("source_id = ?", in.SourceID).First(&src).Error; err != nil {
			return err
		}
		if src.ConfigRevision != in.ExpectedConfigRevision {
			return conflict("接管来源配置已变化")
		}
		for _, confirm := range in.Confirmations {
			var p model.NotionPageBinding
			if err := tx.Where("page_id = ?", normalizePageID(confirm.PageID)).First(&p).Error; err != nil {
				return err
			}
			if p.SourceID != in.SourceID {
				return conflict("确认清单包含非当前来源页面")
			}
		}
		return nil
	}
	if err := s.fenced(ctx, token, check); err != nil {
		return err
	}
	for _, confirm := range in.Confirmations {
		page, err := s.client.RetrievePage(ctx, normalizePageID(confirm.PageID))
		if err != nil {
			return err
		}
		if normalizePageID(page.ID) != normalizePageID(confirm.PageID) || page.Parent.Type != "data_source_id" || normalizeID(page.Parent.DataSourceID) != in.SourceID {
			return conflict("确认清单远端页面身份或来源不一致")
		}
	}
	return s.fenced(ctx, token, check)
}
