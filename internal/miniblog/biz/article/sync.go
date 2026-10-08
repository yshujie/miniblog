package article

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/yshujie/miniblog/internal/miniblog/biz/catalog"
	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/source"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"github.com/yshujie/miniblog/internal/pkg/errno"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"reflect"
	"strings"
	"time"
)

func syncConflict(message string) error {
	return &errno.Errno{HTTP: 409, Code: "NotionSyncConflict", Message: message}
}
func pageIdentity(pageID string) (source.Identity, error) {
	id, e := source.Parse("https://www.notion.so/" + pageID)
	if e != nil || id.PageID == "" || id.PageID != strings.ToLower(strings.ReplaceAll(pageID, "-", "")) {
		return id, invalid("PageID无效")
	}
	return id, nil
}
func syncHash(r SyncInput) string {
	if r.MetadataHash != "" {
		return r.MetadataHash
	}
	raw, _ := json.Marshal(struct {
		Source, Page, Property, Option, Name, Title, URL string
		Tags                                             []string
		Public                                           *string
		State                                            int
		Archived, Trash                                  bool
	}{r.SourceID, r.PageID, r.ThemePropertyID, r.ThemeOptionID, r.ThemeOptionName, r.Title, r.PageURL, r.Tags, r.PublicURL, r.DesiredState, r.NativeArchived, r.InTrash})
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
func sameArticleValues(a, b *model.Article) bool {
	x, y := *a, *b
	x.CreatedAt = time.Time{}
	y.CreatedAt = time.Time{}
	x.UpdatedAt = time.Time{}
	y.UpdatedAt = time.Time{}
	return reflect.DeepEqual(x, y)
}
func pagePublishReason(p *model.NotionPageBinding) string {
	if p.NativeArchived {
		return "native_archived"
	}
	if p.InTrash {
		return "in_trash"
	}
	if p.PublicationHeld {
		return "publication_hold"
	}
	if p.NeedsRevalidation {
		return "needs_revalidation"
	}
	if p.DesiredState == model.ArticleStatusPublished && (p.PublicURL == nil || strings.TrimSpace(*p.PublicURL) == "") {
		return "public_url_missing"
	}
	return ""
}
func persistPage(ds store.IStore, p *model.NotionPageBinding, newPage bool) error {
	p.Revision++
	if newPage {
		return ds.DB().Create(p).Error
	}
	return ds.DB().Model(&model.NotionPageBinding{}).Where("page_id = ?", p.PageID).Select("*").Updates(p).Error
}

// ApplySyncedSource fetches no external data. All page/catalog/state writes commit together.
func (b *articleBiz) ApplySyncedSource(ctx context.Context, r SyncInput) (result *SyncResult, err error) {
	identity, e := pageIdentity(r.PageID)
	if e != nil {
		return nil, e
	}
	r.PageID = identity.PageID
	if r.SourceID == "" {
		return nil, invalid("source_id不能为空")
	}
	if r.MetadataComplete && (r.DesiredState < 1 || r.DesiredState > 4) {
		return nil, invalid("来源状态未知")
	}
	if r.PublicURL != nil && strings.TrimSpace(*r.PublicURL) != "" {
		if _, e = source.Parse(*r.PublicURL); e != nil {
			return nil, invalid("public_url无效")
		}
	}
	if b.ds.DB() == nil || !store.HasNotionSyncSchema(b.ds.DB()) {
		return nil, store.ErrSyncNotReady
	}
	err = store.InTransaction(ctx, b.ds, func(ds store.IStore) error {
		control, e := store.LockLease(ds, r.Lease)
		if e != nil {
			return e
		}
		if control.Paused || control.SourceWritesPaused {
			return syncConflict("同步未启用或正在维护")
		}
		ready, e := store.RegistrationReady(ctx, ds.DB())
		if e != nil {
			return e
		}
		if !ready {
			return syncConflict("来源唯一索引或历史回填尚未就绪")
		}
		var src model.NotionSyncSource
		if e = ds.DB().Where("source_id = ?", r.SourceID).First(&src).Error; e != nil {
			return e
		}
		var hint model.NotionPageBinding
		pageErr := ds.DB().Where("page_id = ?", r.PageID).First(&hint).Error
		if pageErr != nil && !errors.Is(pageErr, gorm.ErrRecordNotFound) {
			return pageErr
		}
		newPage := errors.Is(pageErr, gorm.ErrRecordNotFound)
		var a *model.Article
		if !newPage && hint.ArticleID != nil {
			if e = ds.DB().First(&a, *hint.ArticleID).Error; e != nil {
				return e
			}
		} else {
			a, e = store.FindSourceOwnerForURLs(ds, r.PageID, identity.CanonicalURL)
			if e != nil {
				if errors.Is(e, store.ErrSourceIdentityConflict) || errors.Is(e, store.ErrSourceManagedPending) {
					return syncConflict("PageID存在归属冲突，请先审核")
				}
				return e
			}
			if a != nil {
				return syncConflict("已有手工文章须先审核接管")
			}
		}
		var old *catalog.Placement
		codes := []string{src.ModuleCode}
		if a != nil {
			old, e = catalog.Resolve(ds, a.SectionCode, a.SubsectionCode)
			if e != nil {
				return e
			}
			codes = append(codes, old.Module.Code)
		}
		if e = catalog.LockModules(ds, codes...); e != nil {
			return e
		}
		var lockedSource model.NotionSyncSource
		if e = ds.DB().Clauses(clause.Locking{Strength: "UPDATE"}).Where("source_id = ?", r.SourceID).First(&lockedSource).Error; e != nil {
			return e
		}
		if lockedSource.ConfigRevision != r.ExpectedConfigRevision || lockedSource.ModuleCode != src.ModuleCode {
			return syncConflict("来源配置已变化，请重新扫描")
		}
		src = lockedSource
		if !src.Enabled {
			return syncConflict("来源已停用")
		}
		normalizeID := func(v string) string { return strings.ToLower(strings.ReplaceAll(v, "-", "")) }
		if r.MetadataError != "out_of_scope" && normalizeID(r.DataSourceID) != normalizeID(src.DataSourceID) {
			return syncConflict("页面不属于指定来源")
		}
		if a != nil {
			if e = ds.DB().Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", a.ID).First(&a).Error; e != nil {
				return e
			}
			current, e := catalog.Resolve(ds, a.SectionCode, a.SubsectionCode)
			if e != nil {
				return e
			}
			if current.Module.ID != old.Module.ID {
				return syncConflict("文章位置已变化，请重新扫描")
			}
			old = current
		}
		var p model.NotionPageBinding
		e = ds.DB().Clauses(clause.Locking{Strength: "UPDATE"}).Where("page_id = ?", r.PageID).First(&p).Error
		if e != nil && !errors.Is(e, gorm.ErrRecordNotFound) {
			return e
		}
		if newPage && e == nil {
			return syncConflict("页面绑定已变化")
		}
		if !newPage {
			if e != nil || !reflect.DeepEqual(p.ArticleID, hint.ArticleID) {
				return syncConflict("页面绑定已变化")
			}
			if p.ArticleID != nil && p.ManagementState != model.NotionManagementManaged {
				return syncConflict("历史文章尚未完成审核接管")
			}
			if p.ManagementState == model.NotionManagementDetached {
				return syncConflict("页面已解除同步接管")
			}
			if p.Revision != r.ExpectedBindingRevision {
				if a != nil && r.RunID != "" && p.LastApplyRunID == r.RunID && p.AppliedHash == syncHash(r) {
					result = &SyncResult{ArticleID: a.ID, PageID: p.PageID, Outcome: "unchanged", AppliedState: a.Status, BindingRevision: p.Revision}
					return nil
				}
				return syncConflict("页面已被更新，请重新读取")
			}
			if r.NotionLastEditedAt != nil && p.NotionLastEditedAt != nil && r.NotionLastEditedAt.Before(*p.NotionLastEditedAt) {
				return syncConflict("来源快照已过期")
			}
		} else {
			if r.ExpectedBindingRevision != 0 {
				return syncConflict("新页面版本不匹配")
			}
			p = model.NotionPageBinding{PageID: r.PageID, SourceID: src.ID, ManagementState: model.NotionManagementManaged}
		}
		primaryOwner, primaryErr := store.FindSourceOwnerForURLs(ds, r.PageID, identity.CanonicalURL)
		if primaryErr != nil {
			if errors.Is(primaryErr, store.ErrSourceIdentityConflict) || errors.Is(primaryErr, store.ErrSourceManagedPending) {
				return syncConflict("PageID存在归属冲突，请先审核")
			}
			return primaryErr
		}
		if primaryOwner != nil && (a == nil || primaryOwner.ID != a.ID) {
			return syncConflict("PageID已属于其他文章，须先审核")
		}
		// Successful observations establish known addresses. A changed reader URL
		// cannot overwrite another article. An already managed article still honors
		// withdrawal independently and blocks publication while ownership is reviewed.
		if r.MetadataComplete || r.PublicURLObserved {
			urls := []string{r.PageURL}
			if r.PublicURL != nil {
				urls = append(urls, *r.PublicURL)
			}
			owner, lookupErr := store.FindSourceOwnerForURLs(ds, r.PageID, urls...)
			addressConflict := errors.Is(lookupErr, store.ErrSourceIdentityConflict) || errors.Is(lookupErr, store.ErrSourceManagedPending) || (owner != nil && (a == nil || owner.ID != a.ID))
			if lookupErr != nil && !addressConflict {
				return lookupErr
			}
			if addressConflict {
				if a == nil {
					return syncConflict("来源地址已被其他文章收录，须先审核接管")
				}
				if r.DesiredState >= 1 && r.DesiredState <= 4 {
					p.DesiredState = r.DesiredState
					if r.DesiredState != model.ArticleStatusPublished && r.MetadataError != "out_of_scope" && a.Status != r.DesiredState {
						a.Status = r.DesiredState
						if e = save(ds, a); e != nil {
							return e
						}
					}
				}
				observePrivacy(&p, r)
				observeSuccessfulRevision(&p, r)
				p.NeedsRevalidation = true
				p.PublishBlockReason = "source_identity_conflict"
				p.LastError = "source_identity_conflict"
				p.LastSeenRunID = r.RunID
				if e = persistPage(ds, &p, newPage); e != nil {
					return e
				}
				result = &SyncResult{ArticleID: a.ID, PageID: p.PageID, Outcome: "state_only", Reason: p.PublishBlockReason, AppliedState: a.Status, BindingRevision: p.Revision}
				return nil
			}
		}
		// Failed reads preserve successful state and visibility. A successful observation
		// of URL withdrawal/native archival remains independent from an unknown state.
		unavailable := !r.MetadataComplete && (r.MetadataError == "source_unavailable" || r.MetadataError == "unknown_state" || r.MetadataError == "transport_error" || r.MetadataError == "rate_limited")
		if unavailable {
			p.LastError = r.MetadataError
			p.LastSeenRunID = r.RunID
			if r.MetadataError == "unknown_state" {
				observePrivacy(&p, r)
				observeSuccessfulRevision(&p, r)
			}
			if e = persistPage(ds, &p, newPage); e != nil {
				return e
			}
			id, state := uint64(0), 0
			if a != nil {
				id = a.ID
				state = a.Status
			}
			result = &SyncResult{ArticleID: id, PageID: p.PageID, Outcome: "retained", Reason: r.MetadataError, AppliedState: state, BindingRevision: p.Revision}
			return nil
		}
		if !r.MetadataComplete {
			observePrivacy(&p, r)
			observeSuccessfulRevision(&p, r)
			if r.DesiredState >= 1 && r.DesiredState <= 4 {
				p.DesiredState = r.DesiredState
			}
			if r.MetadataError == "invalid_field" || r.MetadataError == "invalid_metadata" {
				p.PublishBlockReason = "invalid_field"
				p.NeedsRevalidation = true
			}
			if a != nil && r.DesiredState >= 1 && r.DesiredState <= 4 && r.DesiredState != model.ArticleStatusPublished && r.MetadataError != "out_of_scope" {
				if a.Status != r.DesiredState {
					a.Status = r.DesiredState
					if e = save(ds, a); e != nil {
						return e
					}
				}
			}
			p.LastError = r.MetadataError
			p.LastSeenRunID = r.RunID
			if e = persistPage(ds, &p, newPage); e != nil {
				return e
			}
			id, state := uint64(0), 0
			if a != nil {
				id = a.ID
				state = a.Status
			}
			result = &SyncResult{ArticleID: id, PageID: p.PageID, Outcome: "state_only", Reason: p.PublishBlockReason, AppliedState: state, BindingRevision: p.Revision}
			return nil
		}
		// An unseen document is registered locally only after eligible publication.
		// Draft/unpublished/archived pages stay source records with a nullable article ID.
		if a == nil && (r.DesiredState != model.ArticleStatusPublished || r.PublicURL == nil || strings.TrimSpace(*r.PublicURL) == "" || r.NativeArchived || r.InTrash || p.PublicationHeld) {
			recordSnapshot(&p, r)
			p.DesiredState = r.DesiredState
			p.SourceID = src.ID
			p.ManagementState = model.NotionManagementManaged
			p.NeedsRevalidation = false
			p.PublishBlockReason = pagePublishReason(&p)
			if p.PublishBlockReason == "" {
				p.PublishBlockReason = "not_published"
			}
			now, e := store.DatabaseNow(ds.DB())
			if e != nil {
				return e
			}
			p.LastSuccessAt = &now
			if e = persistPage(ds, &p, newPage); e != nil {
				return e
			}
			result = &SyncResult{PageID: p.PageID, Outcome: "pending", Reason: p.PublishBlockReason, BindingRevision: p.Revision}
			return nil
		}
		var before *model.Article
		if a != nil {
			copy := *a
			before = &copy
		}
		if a == nil {
			a = &model.Article{Title: r.Title, ExternalLink: r.PageURL, Author: b.defaultAuthor, Status: model.ArticleStatusDraft}
			if a.ExternalLink == "" {
				a.ExternalLink = identity.CanonicalURL
			}
			bindIdentity(a, identity)
		}
		fieldErr := validateFields(r.Title, a.Author, r.Tags)
		var placement *catalog.Placement
		topicFailure := false
		archivedUnchanged := before != nil && before.Status == model.ArticleStatusDeleted && r.DesiredState == model.ArticleStatusDeleted
		if fieldErr == nil && !archivedUnchanged {
			if e = ds.DB().SavePoint("sync_catalog").Error; e != nil {
				return e
			}
			placement, fieldErr = catalog.EnsureSyncedTheme(ds, &src, r.ThemePropertyID, r.ThemeOptionID, r.ThemeOptionName)
			topicFailure = fieldErr != nil
			if fieldErr == nil && old != nil && old.Section.ID == placement.Section.ID {
				placement.Subsection = old.Subsection
			}
			if fieldErr == nil && r.DesiredState == model.ArticleStatusPublished {
				fieldErr = catalog.RequireVisible(placement)
			}
			if fieldErr != nil {
				if e = ds.DB().RollbackTo("sync_catalog").Error; e != nil {
					return e
				}
			}
		}
		if fieldErr != nil {
			if before == nil {
				a = nil
			} else if r.DesiredState != model.ArticleStatusPublished {
				a.Status = r.DesiredState
				if !sameArticleValues(a, before) {
					if e = save(ds, a); e != nil {
						return e
					}
				}
			}
			if topicFailure {
				if e = catalog.RecordBlockedSyncedTheme(ds, &src, r.ThemePropertyID, r.ThemeOptionID, r.ThemeOptionName, fieldErr.Error()); e != nil {
					return e
				}
			}
			p.DesiredState = r.DesiredState
			observeSuccessfulRevision(&p, r)
			p.LastError = fieldErr.Error()
			p.LastSeenRunID = r.RunID
			p.PublishBlockReason = "metadata_invalid"
			p.NeedsRevalidation = true
			if r.PublicURL == nil || strings.TrimSpace(*r.PublicURL) == "" {
				p.PublicURL = nil
			}
			if a == nil {
				recordSnapshot(&p, r)
				p.PublishBlockReason = "metadata_invalid"
			}
			if e = persistPage(ds, &p, newPage); e != nil {
				return e
			}
			id, state := uint64(0), 0
			if a != nil {
				id = a.ID
				state = a.Status
			}
			result = &SyncResult{ArticleID: id, PageID: p.PageID, Outcome: "state_only", Reason: p.LastError, AppliedState: state, BindingRevision: p.Revision}
			return nil
		}
		if !archivedUnchanged {
			a.Title = r.Title
			if e = model.SetArticleTags(a, r.Tags); e != nil {
				return e
			}
			moved := old == nil || old.Section.ID != placement.Section.ID
			canonicalPlacement(a, placement)
			if moved {
				a.Pos, e = nextPos(ds, a.SectionCode, a.SubsectionCode)
				if e != nil {
					return e
				}
			}
		}
		a.Status = r.DesiredState
		outcome := "unchanged"
		if before == nil {
			if e = ds.DB().Create(a).Error; e != nil {
				return e
			}
			outcome = "created"
		} else if !sameArticleValues(a, before) {
			if e = save(ds, a); e != nil {
				return e
			}
			outcome = "updated"
		}
		p.ArticleID = &a.ID
		p.SourceID = src.ID
		p.ManagementState = model.NotionManagementManaged
		p.DesiredState = r.DesiredState
		p.PageURL = r.PageURL
		p.PublicURL = r.PublicURL
		p.NativeArchived = r.NativeArchived
		p.InTrash = r.InTrash
		p.NeedsRevalidation = false
		p.PublishBlockReason = pagePublishReason(&p)
		p.MetadataHash = syncHash(r)
		p.SnapshotJSON = r.SnapshotJSON
		if !archivedUnchanged {
			p.AppliedHash = p.MetadataHash
		}
		p.NotionLastEditedAt = r.NotionLastEditedAt
		p.LastSeenRunID = r.RunID
		p.LastApplyRunID = r.RunID
		p.LastError = ""
		now, e := store.DatabaseNow(ds.DB())
		if e != nil {
			return e
		}
		p.LastSuccessAt = &now
		if e = persistPage(ds, &p, newPage); e != nil {
			return e
		}
		result = &SyncResult{ArticleID: a.ID, PageID: p.PageID, Outcome: outcome, Reason: p.PublishBlockReason, AppliedState: a.Status, BindingRevision: p.Revision}
		return nil
	})
	return
}

func observePrivacy(p *model.NotionPageBinding, r SyncInput) {
	if r.MetadataError == "out_of_scope" {
		p.PublishBlockReason = "out_of_scope"
		p.NeedsRevalidation = true
	}
	if r.NativeArchived {
		p.NativeArchived = true
		p.PublishBlockReason = "native_archived"
	}
	if r.InTrash {
		p.InTrash = true
		p.PublishBlockReason = "in_trash"
	}
	if (r.PublicURLObserved || r.MetadataComplete) && (r.PublicURL == nil || strings.TrimSpace(*r.PublicURL) == "") {
		p.PublicURL = nil
		p.PublishBlockReason = "public_url_missing"
	}
}

// A confirmed invalid snapshot still fences older source observations.
func observeSuccessfulRevision(p *model.NotionPageBinding, r SyncInput) {
	if (r.MetadataComplete || r.PublicURLObserved) && r.NotionLastEditedAt != nil {
		p.NotionLastEditedAt = r.NotionLastEditedAt
	}
}
func recordSnapshot(p *model.NotionPageBinding, r SyncInput) {
	p.DesiredState = r.DesiredState
	p.PageURL = r.PageURL
	p.PublicURL = r.PublicURL
	p.NativeArchived = r.NativeArchived
	p.InTrash = r.InTrash
	p.MetadataHash = syncHash(r)
	p.SnapshotJSON = r.SnapshotJSON
	p.NotionLastEditedAt = r.NotionLastEditedAt
	p.LastSeenRunID = r.RunID
	p.LastError = ""
}

// ArticleFingerprint protects the complete local before-image without timestamp precision drift.
type articleBeforeWire struct {
	model.Article
	SourceKey *string `json:"source_key"`
	TagsJSON  *string `json:"tags_json"`
}

func ArticleBeforeJSON(a *model.Article) (string, error) {
	if a == nil {
		return "null", nil
	}
	raw, e := json.Marshal(articleBeforeWire{Article: *a, SourceKey: a.SourceKey, TagsJSON: a.TagsJSON})
	return string(raw), e
}
func ParseArticleBeforeJSON(raw string) (*model.Article, error) {
	var w articleBeforeWire
	if e := json.Unmarshal([]byte(raw), &w); e != nil {
		return nil, e
	}
	w.Article.SourceKey = w.SourceKey
	w.Article.TagsJSON = w.TagsJSON
	return &w.Article, nil
}
func ArticleFingerprint(a *model.Article) string {
	if a == nil {
		return ""
	}
	copy := *a
	copy.CreatedAt = time.Time{}
	copy.UpdatedAt = time.Time{}
	raw, _ := ArticleBeforeJSON(&copy)
	hash := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(hash[:])
}
