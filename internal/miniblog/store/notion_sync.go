package store

import (
	"context"
	"errors"
	"fmt"
	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/source"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
)

var (
	ErrLeaseLost              = errors.New("notion sync lease lost")
	ErrSyncNotReady           = errors.New("notion sync schema is unavailable")
	ErrSourceWritesPaused     = errors.New("external source writes are paused for maintenance")
	ErrSourceIdentityConflict = errors.New("source identity resolves to multiple articles")
	ErrSourceManagedPending   = errors.New("source is managed without a blog article")
)

type LeaseToken struct {
	Owner string
	Epoch uint64
}
type NotionSyncRepository struct{ db *gorm.DB }

func NewNotionSyncRepository(db *gorm.DB) *NotionSyncRepository { return &NotionSyncRepository{db: db} }
func (r *NotionSyncRepository) DB() *gorm.DB                    { return r.db }
func (s *datastore) NotionSync() *NotionSyncRepository          { return NewNotionSyncRepository(s.db) }
func HasNotionSyncSchema(db *gorm.DB) bool {
	return db != nil && db.Migrator().HasTable(&model.NotionPageBinding{}) && db.Migrator().HasTable(&model.NotionSyncControl{})
}

// DatabaseNow avoids client-clock skew in lease expiry decisions.
func DatabaseNow(db *gorm.DB) (time.Time, error) {
	var raw string
	var query string
	switch db.Dialector.Name() {
	case "mysql":
		query = "SELECT DATE_FORMAT(UTC_TIMESTAMP(6), '%Y-%m-%dT%H:%i:%s.%fZ')"
	case "sqlite":
		query = "SELECT STRFTIME('%Y-%m-%dT%H:%M:%fZ', 'now')"
	default:
		return time.Time{}, fmt.Errorf("unsupported sync dialect %s", db.Dialector.Name())
	}
	if e := db.Raw(query).Scan(&raw).Error; e != nil {
		return time.Time{}, e
	}
	return time.Parse(time.RFC3339Nano, raw)
}
func (r *NotionSyncRepository) Control(ctx context.Context) (*model.NotionSyncControl, error) {
	var c model.NotionSyncControl
	e := r.db.WithContext(ctx).First(&c, 1).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return nil, ErrSyncNotReady
	}
	return &c, e
}
func (r *NotionSyncRepository) AcquireLease(ctx context.Context, owner string, ttl time.Duration) (token LeaseToken, acquired bool, err error) {
	if owner == "" || len(owner) > 64 || ttl <= 0 {
		return token, false, fmt.Errorf("invalid lease owner or ttl")
	}
	err = InTransaction(ctx, NewStore(r.db), func(ds IStore) error {
		var c model.NotionSyncControl
		if e := ds.DB().Clauses(clause.Locking{Strength: "UPDATE"}).First(&c, 1).Error; e != nil {
			return e
		}
		now, e := DatabaseNow(ds.DB())
		if e != nil {
			return e
		}
		if c.LeaseOwner != "" && c.LeaseUntil != nil && c.LeaseUntil.After(now) {
			return nil
		}
		if c.LeaseEpoch == ^uint64(0) {
			return fmt.Errorf("lease epoch exhausted")
		}
		until := now.Add(ttl)
		token = LeaseToken{Owner: owner, Epoch: c.LeaseEpoch + 1}
		e = ds.DB().Model(&model.NotionSyncControl{}).Where("id = ?", 1).Updates(map[string]interface{}{"lease_owner": owner, "lease_epoch": token.Epoch, "lease_until": until, "updated_at": now}).Error
		acquired = e == nil
		return e
	})
	return
}

// LockLease borrows an existing transaction. Keep this row locked until the page commit.
func LockLease(ds IStore, token LeaseToken) (*model.NotionSyncControl, error) {
	if ds.DB() == nil || token.Owner == "" || token.Epoch == 0 {
		return nil, ErrLeaseLost
	}
	var c model.NotionSyncControl
	if e := ds.DB().Clauses(clause.Locking{Strength: "UPDATE"}).First(&c, 1).Error; e != nil {
		return nil, e
	}
	now, e := DatabaseNow(ds.DB())
	if e != nil {
		return nil, e
	}
	if c.LeaseOwner != token.Owner || c.LeaseEpoch != token.Epoch || c.LeaseUntil == nil || !c.LeaseUntil.After(now) {
		return nil, ErrLeaseLost
	}
	return &c, nil
}
func (r *NotionSyncRepository) RenewLease(ctx context.Context, token LeaseToken, ttl time.Duration) (renewed bool, err error) {
	if ttl <= 0 {
		return false, fmt.Errorf("invalid lease ttl")
	}
	err = InTransaction(ctx, NewStore(r.db), func(ds IStore) error {
		_, e := LockLease(ds, token)
		if errors.Is(e, ErrLeaseLost) {
			return nil
		}
		if e != nil {
			return e
		}
		now, e := DatabaseNow(ds.DB())
		if e != nil {
			return e
		}
		e = ds.DB().Model(&model.NotionSyncControl{}).Where("id = ?", 1).Updates(map[string]interface{}{"lease_until": now.Add(ttl), "updated_at": now}).Error
		renewed = e == nil
		return e
	})
	return
}
func (r *NotionSyncRepository) ReleaseLease(ctx context.Context, token LeaseToken) error {
	return r.db.WithContext(ctx).Model(&model.NotionSyncControl{}).Where("id = ? AND lease_owner = ? AND lease_epoch = ?", 1, token.Owner, token.Epoch).
		Updates(map[string]interface{}{"lease_owner": "", "lease_until": nil, "current_run_id": ""}).Error
}
func PageBindingByArticle(ds IStore, id uint64) (*model.NotionPageBinding, error) {
	if !HasNotionSyncSchema(ds.DB()) {
		return nil, nil
	}
	var p model.NotionPageBinding
	e := ds.DB().Where("article_id = ?", id).First(&p).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &p, e
}

// FindSourceOwner resolves canonical keys, immutable bootstrap aliases and known
// page URLs. URL identity comparisons use Parse in Go, never SQL URL collation.
func FindSourceOwner(ds IStore, key string) (*model.Article, error) {
	return findSourceOwnerByKeys(ds, "", []string{key})
}

// FindSourceOwnerForURLs checks only observed addresses, without guessing slug aliases.
// Ignoring a page skips its binding projections, not a manual article's direct key.
func FindSourceOwnerForURLs(ds IStore, ignorePageID string, rawURLs ...string) (*model.Article, error) {
	keys := make([]string, 0, len(rawURLs))
	for _, raw := range rawURLs {
		if raw == "" {
			continue
		}
		identity, e := source.Parse(raw)
		if e != nil {
			return nil, e
		}
		keys = append(keys, identity.SourceKey)
	}
	return findSourceOwnerByKeys(ds, ignorePageID, keys)
}
func findSourceOwnerByKeys(ds IStore, ignorePageID string, keys []string) (*model.Article, error) {
	if len(keys) == 0 {
		return nil, nil
	}
	wanted := map[string]bool{}
	for _, key := range keys {
		wanted[key] = true
	}
	var rows []*model.Article
	if e := ds.DB().Where("source_key IN ?", keys).Find(&rows).Error; e != nil {
		return nil, e
	}
	owners := map[uint64]*model.Article{}
	for _, a := range rows {
		owners[a.ID] = a
	}
	pending := map[string]bool{}
	if HasNotionSyncSchema(ds.DB()) {
		// The six-table design keeps a page's current observed addresses in its
		// binding. A single projection read avoids per-page queries and new registries.
		var pages []model.NotionPageBinding
		if e := ds.DB().Select("page_id", "article_id", "legacy_source_key", "management_state", "page_url", "public_url").Find(&pages).Error; e != nil {
			return nil, e
		}
		ids := map[uint64]bool{}
		for _, p := range pages {
			if p.PageID == ignorePageID {
				continue
			}
			aliasMatch := p.LegacySourceKey != nil && wanted[*p.LegacySourceKey]
			match := aliasMatch
			canonical, e := source.Parse("https://www.notion.so/" + p.PageID)
			if e == nil && canonical.PageID != "" && wanted[canonical.SourceKey] {
				match = true
			}
			urls := []string{p.PageURL}
			if p.PublicURL != nil {
				urls = append(urls, *p.PublicURL)
			}
			for _, raw := range urls {
				identity, e := source.Parse(raw)
				if e == nil && wanted[identity.SourceKey] {
					match = true
				}
			}
			if !match {
				continue
			}
			if p.ArticleID == nil {
				if aliasMatch {
					return nil, ErrSourceIdentityConflict
				}
				if p.ManagementState == model.NotionManagementManaged {
					pending[p.PageID] = true
				}
				continue
			}
			ids[*p.ArticleID] = true
		}
		if len(ids) > 0 {
			wantedIDs := make([]uint64, 0, len(ids))
			for id := range ids {
				wantedIDs = append(wantedIDs, id)
			}
			var bound []*model.Article
			if e := ds.DB().Where("id IN ?", wantedIDs).Find(&bound).Error; e != nil {
				return nil, e
			}
			if len(bound) != len(ids) {
				return nil, ErrSourceIdentityConflict
			}
			for _, a := range bound {
				owners[a.ID] = a
			}
		}
	}
	if len(owners) > 1 || len(pending) > 1 || (len(pending) > 0 && len(owners) > 0) {
		return nil, ErrSourceIdentityConflict
	}
	if len(pending) > 0 {
		return nil, ErrSourceManagedPending
	}
	for _, a := range owners {
		return a, nil
	}
	return nil, nil
}
func CheckSourceWrites(ds IStore) error {
	if !HasNotionSyncSchema(ds.DB()) {
		return nil
	}
	var c model.NotionSyncControl
	e := ds.DB().First(&c, 1).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return ErrSyncNotReady
	}
	if e != nil {
		return e
	}
	if c.SourceWritesPaused {
		return ErrSourceWritesPaused
	}
	return nil
}

// AuditSourceAliases runs before the frozen bootstrap cutover; aliases remain immutable after it.
func AuditSourceAliases(ctx context.Context, db *gorm.DB) error {
	if !HasNotionSyncSchema(db) {
		return nil
	}
	var n int64
	e := db.WithContext(ctx).Model(&model.NotionPageBinding{}).
		Joins("LEFT JOIN article ON article.id = notion_page_bindings.article_id").
		Where("notion_page_bindings.legacy_source_key IS NOT NULL AND article.id IS NULL").Count(&n).Error
	if e != nil {
		return e
	}
	if n > 0 {
		return ErrSourceIdentityConflict
	}
	e = db.WithContext(ctx).Model(&model.NotionPageBinding{}).
		Joins("JOIN article ON article.source_key = notion_page_bindings.legacy_source_key").
		Where("article.id <> notion_page_bindings.article_id").Count(&n).Error
	if e != nil {
		return e
	}
	if n > 0 {
		return ErrSourceIdentityConflict
	}
	return nil
}

// LockSourceControl is the maintenance barrier for every article source writer.
// It is acquired before module/article locks; sorting never returns to this lock.
func LockSourceControl(ds IStore, checkPaused bool) error {
	if !HasNotionSyncSchema(ds.DB()) {
		return nil
	}
	var c model.NotionSyncControl
	if e := ds.DB().Clauses(clause.Locking{Strength: "UPDATE"}).First(&c, 1).Error; e != nil {
		return e
	}
	if checkPaused && c.SourceWritesPaused {
		return ErrSourceWritesPaused
	}
	return nil
}
