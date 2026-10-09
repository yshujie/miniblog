package article

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/yshujie/miniblog/internal/miniblog/biz/catalog"
	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"reflect"
)

// AdoptSyncedSource is the only alias/rekey writer. It requires the frozen maintenance window.
func (b *articleBiz) AdoptSyncedSource(ctx context.Context, r AdoptInput) (result *SyncResult, err error) {
	identity, e := pageIdentity(r.PageID)
	if e != nil {
		return nil, e
	}
	r.PageID = identity.PageID
	if b.ds.DB() == nil || !store.HasNotionSyncSchema(b.ds.DB()) {
		return nil, store.ErrSyncNotReady
	}
	err = store.InTransaction(ctx, b.ds, func(ds store.IStore) error {
		c, e := store.LockLease(ds, r.Lease)
		if e != nil {
			return e
		}
		if !c.Paused || !c.SourceWritesPaused || !c.BaselineFrozen {
			return syncConflict("接管需要冻结的来源维护窗口")
		}
		var hint model.NotionPageBinding
		if e = ds.DB().Where("page_id = ?", r.PageID).First(&hint).Error; e != nil {
			return e
		}
		var src model.NotionSyncSource
		if e = ds.DB().Where("source_id = ?", r.SourceID).First(&src).Error; e != nil {
			return e
		}
		codes := []string{src.ModuleCode}
		var a *model.Article
		if hint.ArticleID != nil {
			if *hint.ArticleID != r.ExpectedArticleID {
				return syncConflict("审核文章ID已变化")
			}
			if e = ds.DB().First(&a, *hint.ArticleID).Error; e != nil {
				return e
			}
			placement, e := catalog.Resolve(ds, a.SectionCode, a.SubsectionCode)
			if e != nil {
				return e
			}
			codes = append(codes, placement.Module.Code)
		} else if r.ExpectedArticleID != 0 {
			return syncConflict("审核文章ID不匹配")
		}
		if e = catalog.LockModules(ds, codes...); e != nil {
			return e
		}
		if e = ds.DB().Clauses(clause.Locking{Strength: "UPDATE"}).Where("source_id = ?", r.SourceID).First(&src).Error; e != nil {
			return e
		}
		if src.ConfigRevision != r.ExpectedConfigRevision {
			return syncConflict("来源配置已变化")
		}
		if a != nil {
			if e = ds.DB().Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", a.ID).First(&a).Error; e != nil {
				return e
			}
		}
		var p model.NotionPageBinding
		if e = ds.DB().Clauses(clause.Locking{Strength: "UPDATE"}).Where("page_id = ?", r.PageID).First(&p).Error; e != nil {
			return e
		}
		if p.SourceID != r.SourceID || !reflect.DeepEqual(p.ArticleID, hint.ArticleID) {
			return syncConflict("审核绑定已变化")
		}
		if p.ManagementState == model.NotionManagementManaged && p.BootstrapState == "adopted" {
			id, state := uint64(0), 0
			if a != nil {
				id = a.ID
				state = a.Status
			}
			result = &SyncResult{ArticleID: id, PageID: p.PageID, Outcome: "unchanged", AppliedState: state, BindingRevision: p.Revision}
			return nil
		}
		if p.SourceID != r.SourceID || p.Revision != r.ExpectedBindingRevision || p.BootstrapState != "verified" || p.BootstrapExpectedFingerprint != r.ExpectedFingerprint {
			return syncConflict("审核快照未确认或已变化")
		}
		if p.DesiredState == model.ArticleStatusPublished || (p.BootstrapExpectedState != nil && *p.BootstrapExpectedState == model.ArticleStatusPublished) {
			if p.BootstrapExpectedState == nil || *p.BootstrapExpectedState != p.DesiredState {
				return syncConflict("已发布接管状态尚未回读确认")
			}
			var snapshot struct {
				ReviewedPublicationInput
				PageID       string `json:"page_id"`
				SourceID     string `json:"source_id"`
				DesiredState int    `json:"desired_state"`
			}
			if json.Unmarshal([]byte(p.SnapshotJSON), &snapshot) != nil || snapshot.PageID != p.PageID || snapshot.SourceID != p.SourceID || snapshot.DesiredState != p.DesiredState || !reflect.DeepEqual(snapshot.PublicURL, p.PublicURL) || snapshot.NativeArchived != p.NativeArchived || snapshot.InTrash != p.InTrash {
				return syncConflict("已发布接管缺少有效来源快照")
			}
			reviewed := snapshot.ReviewedPublicationInput
			var mapping struct {
				TopicPropertyID string `json:"topic_property_id"`
			}
			if json.Unmarshal([]byte(src.PropertyMappingJSON), &mapping) != nil || mapping.TopicPropertyID == "" {
				return syncConflict("已发布接管来源字段映射无效")
			}
			reviewed.TopicPropertyID = mapping.TopicPropertyID
			reviewed.PublicURL, reviewed.NativeArchived, reviewed.InTrash, reviewed.PublicationHeld = p.PublicURL, p.NativeArchived, p.InTrash, p.PublicationHeld
			if a != nil {
				reviewed.LocalSectionCode, reviewed.LocalSubsectionCode = a.SectionCode, a.SubsectionCode
			}
			if e = ValidateReviewedPublication(ds, &src, reviewed); e != nil {
				return e
			}
		}
		knownURLs := []string{identity.CanonicalURL, p.PageURL}
		if p.PublicURL != nil {
			knownURLs = append(knownURLs, *p.PublicURL)
		}
		knownOwner, knownErr := store.FindSourceOwnerForURLs(ds, r.PageID, knownURLs...)
		if knownErr != nil {
			if errors.Is(knownErr, store.ErrSourceIdentityConflict) || errors.Is(knownErr, store.ErrSourceManagedPending) {
				return syncConflict("审核页面地址存在归属冲突")
			}
			return knownErr
		}
		if knownOwner != nil && (a == nil || knownOwner.ID != a.ID) {
			return syncConflict("审核页面地址已属于其他文章")
		}
		if a != nil {
			before, parseErr := ParseArticleBeforeJSON(p.LocalBeforeJSON)
			if parseErr != nil {
				return syncConflict("本地审核快照无效")
			}
			if ArticleFingerprint(a) != ArticleFingerprint(before) {
				return syncConflict("本地文章在审核后已变化")
			}
			if p.BootstrapExpectedState == nil || *p.BootstrapExpectedState != a.Status || p.DesiredState != a.Status {
				return syncConflict("Notion状态尚未确认与历史状态一致")
			}
			owner, e := findSource(ds, identity.SourceKey)
			if e != nil {
				return e
			}
			if owner != nil && owner.ID != a.ID {
				return syncConflict("目标PageID已属于其他文章")
			}
			if a.SourceKey != nil && *a.SourceKey != identity.SourceKey {
				if p.LegacySourceKey != nil && *p.LegacySourceKey != *a.SourceKey {
					return syncConflict("历史来源别名不可改写")
				}
				key := *a.SourceKey
				p.LegacySourceKey = &key
				owner, e = findSource(ds, key)
				if e != nil {
					return e
				}
				if owner == nil || owner.ID != a.ID {
					return syncConflict("历史来源别名存在冲突")
				}
			}
			// UpdateColumns preserves the exact historical timestamps and all article fields.
			if e = ds.DB().Model(&model.Article{}).Where("id = ?", a.ID).UpdateColumns(map[string]interface{}{"provider": identity.Provider, "canonical_url": identity.CanonicalURL, "source_key": identity.SourceKey, "updated_at": a.UpdatedAt}).Error; e != nil {
				return e
			}
		} else {
			owner, e := findSource(ds, identity.SourceKey)
			if e != nil {
				return e
			}
			if owner != nil {
				return syncConflict("页面匹配到文章，须重新审核")
			}
		}
		p.ManagementState = model.NotionManagementManaged
		p.BootstrapState = "adopted"
		p.NeedsRevalidation = true
		p.PublishBlockReason = pagePublishReason(&p)
		p.LastError = ""
		if e = persistPage(ds, &p, false); e != nil {
			return e
		}
		if e = store.AuditSourceAliases(ctx, ds.DB()); e != nil {
			return e
		}
		id, state := uint64(0), 0
		if a != nil {
			id = a.ID
			state = a.Status
		}
		result = &SyncResult{ArticleID: id, PageID: p.PageID, Outcome: "adopted", AppliedState: state, BindingRevision: p.Revision}
		return nil
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, syncConflict("审核来源或文章已不存在")
	}
	return
}
