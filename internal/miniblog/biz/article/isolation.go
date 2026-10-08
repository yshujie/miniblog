package article

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/yshujie/miniblog/internal/miniblog/biz/catalog"
	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"gorm.io/gorm/clause"
	"reflect"
	"strings"
	"time"
)

type TargetIsolationReason string

const (
	TargetDisabled      TargetIsolationReason = "target_disabled"
	TargetSchemaChanged TargetIsolationReason = "target_schema_changed"
)

// TargetIsolationInput is a successful fresh location observation, not a failed read.
// It cannot project metadata or create an article, binding, or catalog entry.
type TargetIsolationInput struct {
	Lease                                           store.LeaseToken
	RunID, PageID, TargetSourceID, DataSourceID     string
	Reason                                          TargetIsolationReason
	DesiredState                                    int // Zero means the state could not be reliably parsed.
	PublicURLWithdrawn, NativeArchived, InTrash     bool
	NotionLastEditedAt                              *time.Time
	ExpectedConfigRevision, ExpectedBindingRevision uint64
}

func targetIsolationHash(r TargetIsolationInput) string {
	r.Lease = store.LeaseToken{}
	r.RunID = ""
	r.ExpectedBindingRevision = 0
	raw, _ := json.Marshal(r)
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:])
}

// IsolateSyncedTarget is separate from Apply's enabled-source gate. Only an
// already managed article can withdraw while its confirmed target is unusable.
func (b *articleBiz) IsolateSyncedTarget(ctx context.Context, r TargetIsolationInput) (result *SyncResult, err error) {
	identity, e := pageIdentity(r.PageID)
	if e != nil {
		return nil, e
	}
	r.PageID = identity.PageID
	if r.TargetSourceID == "" || (r.Reason != TargetDisabled && r.Reason != TargetSchemaChanged) || r.DesiredState < 0 || r.DesiredState > 4 || r.NotionLastEditedAt == nil || r.NotionLastEditedAt.IsZero() {
		return nil, invalid("目标隔离必须提供已确认位置、快照时间和有效原因")
	}
	if !store.HasNotionSyncSchema(b.ds.DB()) {
		return nil, store.ErrSyncNotReady
	}
	err = store.InTransaction(ctx, b.ds, func(ds store.IStore) error {
		c, e := store.LockLease(ds, r.Lease)
		if e != nil {
			return e
		}
		if c.Paused || c.SourceWritesPaused {
			return syncConflict("同步暂停或维护中")
		}
		var hint model.NotionPageBinding
		if e = ds.DB().Where("page_id = ?", r.PageID).First(&hint).Error; e != nil {
			return e
		}
		if hint.ManagementState != model.NotionManagementManaged || hint.ArticleID == nil {
			return syncConflict("目标隔离仅允许已托管且已登记文章")
		}
		if r.Reason == TargetDisabled && r.TargetSourceID == hint.SourceID {
			return syncConflict("同库停用仅冻结同步，不能隔离已有文章")
		}
		var a model.Article
		if e = ds.DB().First(&a, *hint.ArticleID).Error; e != nil {
			return e
		}
		old, e := catalog.Resolve(ds, a.SectionCode, a.SubsectionCode)
		if e != nil {
			return e
		}
		var srcHint model.NotionSyncSource
		if e = ds.DB().Where("source_id = ?", r.TargetSourceID).First(&srcHint).Error; e != nil {
			return e
		}
		if e = catalog.LockModules(ds, old.Module.Code, srcHint.ModuleCode); e != nil {
			return e
		}
		var src model.NotionSyncSource
		if e = ds.DB().Clauses(clause.Locking{Strength: "UPDATE"}).Where("source_id = ?", r.TargetSourceID).First(&src).Error; e != nil {
			return e
		}
		if src.ConfigRevision != r.ExpectedConfigRevision || src.ModuleCode != srcHint.ModuleCode {
			return syncConflict("目标来源配置已变化，请重新扫描")
		}
		if r.Reason == TargetDisabled && src.Enabled {
			return syncConflict("目标来源已启用，请重新校验")
		}
		normalize := func(v string) string { return strings.ToLower(strings.ReplaceAll(v, "-", "")) }
		if normalize(r.DataSourceID) != normalize(src.DataSourceID) {
			return syncConflict("页面不属于已确认目标来源")
		}
		if e = ds.DB().Clauses(clause.Locking{Strength: "UPDATE"}).First(&a, *hint.ArticleID).Error; e != nil {
			return e
		}
		current, e := catalog.Resolve(ds, a.SectionCode, a.SubsectionCode)
		if e != nil {
			return e
		}
		if current.Module.ID != old.Module.ID {
			return syncConflict("文章位置已变化，请重新扫描")
		}
		var p model.NotionPageBinding
		if e = ds.DB().Clauses(clause.Locking{Strength: "UPDATE"}).Where("page_id = ?", r.PageID).First(&p).Error; e != nil {
			return e
		}
		if p.ManagementState != model.NotionManagementManaged || !reflect.DeepEqual(p.ArticleID, hint.ArticleID) {
			return syncConflict("页面管理权已变化")
		}
		if p.Revision != r.ExpectedBindingRevision {
			if r.RunID != "" && p.LastApplyRunID == r.RunID && p.AppliedHash == targetIsolationHash(r) {
				result = &SyncResult{ArticleID: a.ID, PageID: p.PageID, Outcome: "blocked", Reason: p.PublishBlockReason, AppliedState: a.Status, BindingRevision: p.Revision}
				return nil
			}
			return syncConflict("页面版本已变化，请重新读取")
		}
		if p.NotionLastEditedAt != nil && r.NotionLastEditedAt.Before(*p.NotionLastEditedAt) {
			return syncConflict("来源快照已过期")
		}
		if r.DesiredState != 0 {
			p.DesiredState = r.DesiredState
			if r.DesiredState != model.ArticleStatusPublished && a.Status != r.DesiredState {
				a.Status = r.DesiredState
				if e = save(ds, &a); e != nil {
					return e
				}
			}
		}
		// Keep the last successfully projected source and all content/placement/URLs.
		// A confirmed privacy withdrawal is the sole exception to URL retention.
		if r.PublicURLWithdrawn {
			p.PublicURL = nil
		}
		if r.NativeArchived {
			p.NativeArchived = true
		}
		if r.InTrash {
			p.InTrash = true
		}
		p.NeedsRevalidation = true
		p.PublishBlockReason = string(r.Reason)
		p.LastError = ""
		if r.Reason == TargetSchemaChanged {
			p.LastError = string(r.Reason)
		}
		p.NotionLastEditedAt = r.NotionLastEditedAt
		p.LastSeenRunID = r.RunID
		p.LastApplyRunID = r.RunID
		p.AppliedHash = targetIsolationHash(r)
		if e = persistPage(ds, &p, false); e != nil {
			return e
		}
		result = &SyncResult{ArticleID: a.ID, PageID: p.PageID, Outcome: "blocked", Reason: string(r.Reason), AppliedState: a.Status, BindingRevision: p.Revision}
		return nil
	})
	return
}
