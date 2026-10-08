package notionsync

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/yshujie/miniblog/internal/miniblog/biz/article"
	"github.com/yshujie/miniblog/internal/miniblog/biz/catalog"
	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/source"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"sort"
	"strconv"
	"time"
)

func bootstrapFingerprint(a model.Article, snap Snapshot, revision uint64) string {
	sum := sha256.Sum256([]byte(jsonText(struct {
		ArticleFingerprint string   `json:"article_fingerprint"`
		Snapshot           Snapshot `json:"snapshot"`
		ConfigRevision     uint64   `json:"config_revision"`
	}{article.ArticleFingerprint(&a), snap, revision})))
	return hex.EncodeToString(sum[:])
}
func bootstrapMetadataFingerprint(snap Snapshot) string {
	snap.DesiredState = 0
	snap.StateOptionID = ""
	snap.Reason = ""
	snap.LastEditedAt = time.Time{}
	return metadataHash(snap)
}

// BootstrapPreview performs a fresh full scan; automatic matches use trusted IDs.
// Nonstandard aliases require a separate explicit operator review.
// It records review fingerprints but never modifies articles, catalog, or Notion.
func (s *Service) BootstrapPreview(ctx context.Context, review ...BootstrapReviewInput) (*BootstrapPreviewResult, error) {
	manual := map[string]string{}
	for _, r := range review {
		for _, m := range r.ManualMatches {
			key := normalizePageID(m.PageID)
			if manual[key] != "" {
				return nil, invalid("重复手工关联页面")
			}
			manual[key] = m.ArticleID
		}
	}
	result := &BootstrapPreviewResult{Items: []BootstrapCandidate{}}
	id, e := s.synchronousRun(ctx, "bootstrap_preview", func(active context.Context, run model.NotionSyncRun, token store.LeaseToken, counts *RunCounts) error {
		ctx = active
		if e := s.scanAndApply(ctx, run, token, counts); e != nil {
			return e
		}
		var pages []model.NotionPageBinding
		var articles []model.Article
		if e := s.ds.DB().WithContext(ctx).Where("management_state = ?", model.NotionManagementBaselinePending).Order("page_id").Find(&pages).Error; e != nil {
			return e
		}
		if e := s.ds.DB().WithContext(ctx).Order("id").Find(&articles).Error; e != nil {
			return e
		}
		matches := map[string][]model.Article{}
		byID := map[string]model.Article{}
		for _, a := range articles {
			byID[strconv.FormatUint(a.ID, 10)] = a
			pageID, e := source.MigrationNotionPageID(a.ExternalLink)
			if e == nil {
				matches[normalizePageID(pageID)] = append(matches[normalizePageID(pageID)], a)
			}
		}
		for _, p := range pages {
			var snap Snapshot
			if e := json.Unmarshal([]byte(p.SnapshotJSON), &snap); e != nil {
				return e
			}
			journal := p.BootstrapState == "write_requested" || p.BootstrapState == "verified"
			var freshPage source.NotionPage
			if journal {
				// Ordinary scans freeze an interrupted write's reviewed metadata.
				// Explicit review must independently read the page before replacing it.
				var e error
				freshPage, e = s.client.RetrievePage(ctx, p.PageID)
				if e != nil {
					return e
				}
				target := normalizeID(freshPage.Parent.DataSourceID)
				if freshPage.Parent.Type != "data_source_id" {
					return conflict("待审核页面已移出文章子库")
				}
				if _, ok := allowedSources[target]; !ok {
					return conflict("待审核页面已移出五库范围")
				}
				p.SourceID = target
			}
			var src model.NotionSyncSource
			if e := s.ds.DB().WithContext(ctx).Where("source_id = ?", p.SourceID).First(&src).Error; e != nil {
				return e
			}
			if journal {
				schema, e := s.client.RetrieveDataSource(ctx, p.SourceID)
				if e != nil {
					return e
				}
				cfg := configOf(src)
				if e = validateConfig(schema, cfg); e != nil {
					return e
				}
				snap, e = snapshotOf(freshPage, p.SourceID, cfg)
				if e != nil {
					return e
				}
				if e = validateBootstrapTopic(schema, cfg, snap); e != nil {
					return e
				}
			}
			c := BootstrapCandidate{PageID: p.PageID, SourceID: p.SourceID, Title: snap.Title, Topic: snap.TopicOptionName, NotionState: stateName(snap.DesiredState), CandidateArticleIDs: []string{}, TitleHintArticleIDs: []string{}, MatchMethod: "unmatched", NewPage: true, PublicCondition: "public_url_missing", PlacementChange: "目标主题章节直属"}
			if snap.PublicURL != nil && *snap.PublicURL != "" {
				c.PublicCondition = "public_url_available"
			}
			if snap.NativeArchived || snap.InTrash {
				c.PublicCondition = "native_archived_or_in_trash"
			}
			for _, a := range articles {
				if a.Title == snap.Title {
					c.TitleHintArticleIDs = append(c.TitleHintArticleIDs, strconv.FormatUint(a.ID, 10))
				}
			}
			trustedMatches := len(matches[p.PageID])
			urls := []string{"https://www.notion.so/" + p.PageID, snap.PageURL}
			if snap.PublicURL != nil {
				urls = append(urls, *snap.PublicURL)
			}
			owner, identityErr := store.FindSourceOwnerForURLs(store.NewStore(s.ds.DB().WithContext(ctx)), p.PageID, urls...)
			if identityErr != nil {
				if !errors.Is(identityErr, store.ErrSourceIdentityConflict) && !errors.Is(identityErr, store.ErrSourceManagedPending) {
					return identityErr
				}
				c.MatchMethod = "ambiguous"
				c.NewPage = false
				c.Reason = "historical_identity_conflict"
			} else if owner != nil {
				duplicate := false
				for _, a := range matches[p.PageID] {
					if a.ID == owner.ID {
						duplicate = true
					}
				}
				if !duplicate {
					matches[p.PageID] = append(matches[p.PageID], *owner)
				}
			}
			for _, a := range matches[p.PageID] {
				c.CandidateArticleIDs = append(c.CandidateArticleIDs, strconv.FormatUint(a.ID, 10))
			}
			var selected *model.Article
			if c.MatchMethod == "ambiguous" {
				// Explicit matching may not override conflicting observed source ownership.
			} else if explicit := manual[p.PageID]; explicit != "" {
				a, ok := byID[explicit]
				if !ok {
					return invalid("手工关联文章不存在")
				}
				if trusted, e := source.MigrationNotionPageID(a.ExternalLink); e == nil && normalizePageID(trusted) != p.PageID {
					return conflict("该文章已指向其他可信 Notion 页面，不能合并")
				}
				selected = &a
				c.MatchMethod = "explicit_alias_review"
				c.NewPage = false
			} else if len(matches[p.PageID]) == 1 {
				a := matches[p.PageID][0]
				selected = &a
				c.MatchMethod = "trusted_page_id"
				if trustedMatches == 0 {
					c.MatchMethod = "known_reading_url"
				}
				c.NewPage = false
			} else if len(matches[p.PageID]) > 1 {
				c.MatchMethod = "ambiguous"
				c.NewPage = false
				c.Reason = "historical_match_ambiguous"
			}
			var local model.Article
			if selected != nil {
				local = *selected
				_, trustedErr := source.MigrationNotionPageID(local.ExternalLink)
				c.RequiresLegacyAlias = trustedErr != nil
				if trusted, err := source.MigrationNotionPageID(local.ExternalLink); err == nil && normalizePageID(trusted) != p.PageID {
					c.MatchMethod = "ambiguous"
					c.Reason = "historical_identity_conflict"
				}
				c.ArticleID = strconv.FormatUint(local.ID, 10)
				c.LocalState = stateName(local.Status)
				c.LocalTitle = local.Title
				c.LocalSectionCode = local.SectionCode
				c.LocalSubsectionCode = local.SubsectionCode
				if c.LocalState == "" {
					c.Reason = "local_state_unknown"
				}
				if snap.DesiredState != 0 && snap.DesiredState != local.Status {
					c.Reason = "notion_state_conflict"
				}
			}

			code, catalogReason, e := s.catalogPlan(ctx, scannedSource{Row: src, Config: configOf(src)}, snap.TopicOptionID, snap.TopicOptionName)
			if e != nil {
				return e
			}
			c.ProposedSectionCode = code
			c.ProposedSectionTitle = snap.TopicOptionName
			switch catalogReason {
			case "bound", "would_create", "would_rename":
			default:
				c.PublishBlockReason = catalogReason
			}
			if code != "" && selected != nil && local.SectionCode == code {
				c.PlacementChange = "保留当前章节与子章节"
			}
			if p.PublicationHeld {
				c.PublishBlockReason = "publication_hold"
			}
			if snap.PublicURL == nil || *snap.PublicURL == "" {
				c.PublishBlockReason = "public_url_missing"
			}
			if snap.NativeArchived || snap.InTrash {
				c.PublishBlockReason = "native_archived_or_in_trash"
			}

			if src.ModuleCode == "" {
				c.Reason = "source_module_unconfigured"
			}
			if snap.NativeArchived || snap.InTrash {
				c.Reason = "native_archived_requires_review"
			}
			if snap.StateOptionID != "" && snap.DesiredState == 0 {
				c.Reason = "unknown_state"
			}
			if c.MatchMethod != "ambiguous" {
				c.ExpectedFingerprint = bootstrapFingerprint(local, snap, src.ConfigRevision)
				if e := s.fenced(ctx, token, func(tx *gorm.DB, _ *model.NotionSyncControl) error {
					var currentSource model.NotionSyncSource
					if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("source_id = ?", src.ID).First(&currentSource).Error; e != nil {
						return e
					}
					if currentSource.ConfigRevision != src.ConfigRevision {
						return conflict("审核中来源配置已变化，请重新预览")
					}
					var currentPage model.NotionPageBinding
					if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("page_id = ?", p.PageID).First(&currentPage).Error; e != nil {
						return e
					}
					if currentPage.ManagementState != model.NotionManagementBaselinePending || currentPage.Revision != p.Revision {
						return conflict("审核页面绑定已变化，请重新预览")
					}
					// The new fingerprint and reviewed snapshot are one fenced write.
					// A failed/interrupted preview can never pair old approval with new metadata.
					result := tx.Model(&model.NotionPageBinding{}).Where("page_id = ? AND management_state = ? AND revision = ?", p.PageID, model.NotionManagementBaselinePending, p.Revision).Updates(map[string]interface{}{
						"bootstrap_expected_fingerprint": c.ExpectedFingerprint, "snapshot_json": jsonText(snap), "source_id": snap.SourceID,
						"metadata_hash": metadataHash(snap), "desired_state": snap.DesiredState, "page_url": snap.PageURL, "public_url": snap.PublicURL,
						"native_archived": snap.NativeArchived, "in_trash": snap.InTrash, "notion_last_edited_at": snap.LastEditedAt,
					})
					if result.Error != nil {
						return result.Error
					}
					return nil
				}); e != nil {
					return e
				}
			}
			var matchedID *uint64
			if selected != nil {
				id := selected.ID
				matchedID = &id
			}
			if e := s.recordItem(ctx, token, run.ID, p.PageID, matchedID, "bootstrap_preview", c.Reason, "", nil, map[string]interface{}{"bootstrap_preview": c}); e != nil {
				return e
			}
			result.Items = append(result.Items, c)
		}
		return nil
	})
	result.RunID = id
	return result, e
}

// BootstrapApply requires an explicit writer and reviewed fingerprints. Runtime sync
// cannot invoke it. Each write is journaled and always followed by a fresh read.
func (s *Service) BootstrapApply(ctx context.Context, in BootstrapInput, writer source.BootstrapNotionWriter) (*BootstrapApplyResult, error) {
	if writer == nil {
		return nil, unavailable("接管写客户端未显式配置")
	}
	if client, ok := writer.(*source.HTTPNotionSyncClient); ok {
		client.SetCooldownObserver(s.observeCooldown)
	}
	if len(in.Confirmations) == 0 {
		return nil, invalid("请提供已核对的接管确认清单")
	}
	seen := map[string]bool{}
	for _, c := range in.Confirmations {
		if c.PageID == "" || (!c.NewPage && c.ArticleID == "") || (c.NewPage && c.ArticleID != "") || c.ExpectedFingerprint == "" || c.ConfirmedBy == "" || stateValue(c.ExpectedState) == 0 {
			return nil, invalid("接管项缺少ID、指纹、状态或确认人")
		}
		if seen[normalizePageID(c.PageID)] {
			return nil, invalid("接管清单包含重复页面")
		}
		seen[normalizePageID(c.PageID)] = true
	}
	result := &BootstrapApplyResult{Items: []ItemDTO{}}
	id, e := s.synchronousRun(ctx, "bootstrap_apply", func(active context.Context, run model.NotionSyncRun, token store.LeaseToken, counts *RunCounts) error {
		ctx = active
		control, e := store.NewNotionSyncRepository(s.ds.DB()).Control(ctx)
		if e != nil {
			return e
		}
		if !control.BaselineFrozen || !control.Paused || !control.SourceWritesPaused {
			return conflict("接管前需冻结基线并暂停同步和外部来源写入")
		}
		for _, confirm := range in.Confirmations {
			counts.Seen++
			item, e := s.bootstrapOne(ctx, run, token, confirm, writer)
			if e != nil {
				counts.Failed++
				item = ItemDTO{PageID: normalizePageID(confirm.PageID), Outcome: "failed", Error: e.Error()}
				if pageError := s.recordPageError(ctx, token, model.NotionPageBinding{PageID: item.PageID}, e); pageError != nil {
					return pageError
				}
			} else {
				counts.Updated++
			}
			result.Items = append(result.Items, item)
			if e := s.recordItem(ctx, token, run.ID, item.PageID, nil, item.Outcome, item.Reason, item.Error, item.Before, item.After); e != nil {
				return e
			}
			if errors.Is(e, store.ErrLeaseLost) {
				return e
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}
		}
		return nil
	})
	result.RunID = id
	return result, e
}
func (s *Service) bootstrapOne(ctx context.Context, run model.NotionSyncRun, token store.LeaseToken, confirm BootstrapConfirm, writer source.BootstrapNotionWriter) (ItemDTO, error) {
	pageID := normalizePageID(confirm.PageID)
	articleID := uint64(0)
	var e error
	if !confirm.NewPage {
		articleID, e = strconv.ParseUint(confirm.ArticleID, 10, 63)
		if e != nil || articleID == 0 {
			return ItemDTO{}, invalid("历史文章ID无效")
		}
	}
	db := s.ds.DB().WithContext(ctx)
	var p model.NotionPageBinding
	var src model.NotionSyncSource
	var local model.Article
	if e = db.Where("page_id = ?", pageID).First(&p).Error; e != nil {
		return ItemDTO{}, e
	}
	if p.ManagementState == model.NotionManagementManaged && ((confirm.NewPage && p.ArticleID == nil) || (!confirm.NewPage && p.ArticleID != nil && *p.ArticleID == articleID)) {
		return ItemDTO{PageID: pageID, ArticleID: articleIDString(articleID), Outcome: "already_adopted"}, nil
	}
	if p.ManagementState != model.NotionManagementBaselinePending {
		return ItemDTO{}, conflict("页面不是待核对历史项")
	}
	if e = db.Where("source_id = ?", p.SourceID).First(&src).Error; e != nil {
		return ItemDTO{}, e
	}
	if src.ModuleCode == "" {
		return ItemDTO{}, conflict("请先配置源模块")
	}
	module, e := s.ds.Modules().GetByCode(src.ModuleCode)
	if e != nil {
		return ItemDTO{}, e
	}
	if module == nil {
		return ItemDTO{}, conflict("源模块已失效")
	}
	var localPointer *model.Article
	if !confirm.NewPage {
		if e = db.First(&local, articleID).Error; e != nil {
			return ItemDTO{}, e
		}
		localPointer = &local
		trusted, matchErr := source.MigrationNotionPageID(local.ExternalLink)
		if matchErr == nil && normalizePageID(trusted) != pageID {
			return ItemDTO{}, conflict("该文章指向其他可信页面，禁止合并")
		}
		if matchErr != nil && !confirm.AllowLegacyAlias {
			return ItemDTO{}, conflict("非标准历史别名需显式 allow_legacy_alias 审核")
		}
	}
	identity, e := source.NotionIdentity(pageID)
	if e != nil {
		return ItemDTO{}, e
	}
	owner, e := store.FindSourceOwner(store.NewStore(db), identity.SourceKey)
	if e != nil {
		return ItemDTO{}, conflict("来源身份或历史别名冲突，请重新审核")
	}
	if owner != nil && (confirm.NewPage || owner.ID != articleID) {
		return ItemDTO{}, conflict("目标页面已属于文章，请重新审核")
	}
	if localPointer != nil {
		var other model.NotionPageBinding
		e = db.Where("article_id = ? AND page_id <> ?", articleID, pageID).First(&other).Error
		if e == nil {
			return ItemDTO{}, conflict("历史文章已绑定其他页面")
		}
		if !errors.Is(e, gorm.ErrRecordNotFound) {
			return ItemDTO{}, e
		}
		if local.SourceKey != nil {
			aliasOwner, e := store.FindSourceOwner(store.NewStore(db), *local.SourceKey)
			if e != nil || aliasOwner == nil || aliasOwner.ID != articleID {
				return ItemDTO{}, conflict("历史来源别名存在冲突")
			}
		}
	}
	desired := stateValue(confirm.ExpectedState)
	if !confirm.NewPage && local.Status != desired {
		return ItemDTO{}, conflict("历史状态已变化，请重新预览")
	}
	cfg := configOf(src)
	schema, e := s.client.RetrieveDataSource(ctx, src.ID)
	if e != nil {
		return ItemDTO{}, e
	}
	if e = validateConfig(schema, cfg); e != nil {
		return ItemDTO{}, e
	}
	page, e := s.client.RetrievePage(ctx, pageID)
	if e != nil {
		return ItemDTO{}, e
	}
	fresh, e := snapshotOf(page, p.SourceID, cfg)
	if e != nil {
		return ItemDTO{}, e
	}
	if e := validateBootstrapTopic(schema, cfg, fresh); e != nil {
		return ItemDTO{}, e
	}
	freshURLs := []string{identity.CanonicalURL, fresh.PageURL}
	if fresh.PublicURL != nil {
		freshURLs = append(freshURLs, *fresh.PublicURL)
	}
	urlOwner, e := store.FindSourceOwnerForURLs(store.NewStore(db), pageID, freshURLs...)
	if e != nil {
		return ItemDTO{}, conflict("已知页面地址的来源身份冲突，请重新审核")
	}
	if urlOwner != nil && (confirm.NewPage || urlOwner.ID != articleID) {
		return ItemDTO{}, conflict("已知页面地址已登记，不能作为新页面或关联另一文章")
	}
	if fresh.NativeArchived || fresh.InTrash {
		return ItemDTO{}, conflict("Notion 原生归档或垃圾箱页面需先人工处理")
	}
	var beforeSnapshot Snapshot
	if e = json.Unmarshal([]byte(p.SnapshotJSON), &beforeSnapshot); e != nil {
		return ItemDTO{}, e
	}
	// A new preview and explicit confirmation may restart a stale journal only
	// when every freshly read input exactly matches that newly reviewed fingerprint.
	reviewedFresh := bootstrapFingerprint(local, fresh, src.ConfigRevision) == confirm.ExpectedFingerprint
	resuming := (p.BootstrapState == "write_requested" || p.BootstrapState == "verified") && !reviewedFresh
	if p.BootstrapExpectedFingerprint != confirm.ExpectedFingerprint {
		return ItemDTO{}, conflict("审核指纹已变化，请重新预览")
	}
	if !resuming && !reviewedFresh {
		return ItemDTO{}, conflict("核对后元数据或配置已变化，请重新预览")
	}
	if resuming {
		var journal struct {
			Revision uint64 `json:"_bootstrap_config_revision"`
			NewPage  bool   `json:"_bootstrap_new_page"`
		}
		if json.Unmarshal([]byte(p.LocalBeforeJSON), &journal) != nil || journal.Revision != src.ConfigRevision || journal.NewPage != confirm.NewPage {
			return ItemDTO{}, conflict("接管配置或确认类型已变化")
		}
		if !confirm.NewPage {
			old, parseErr := article.ParseArticleBeforeJSON(p.LocalBeforeJSON)
			if parseErr != nil || article.ArticleFingerprint(old) != article.ArticleFingerprint(&local) || p.ArticleID == nil || *p.ArticleID != articleID {
				return ItemDTO{}, conflict("接管历史文章核验不一致")
			}
		}
		if p.BootstrapExpectedState == nil || *p.BootstrapExpectedState != desired || bootstrapMetadataFingerprint(beforeSnapshot) != bootstrapMetadataFingerprint(fresh) {
			return ItemDTO{}, conflict("接管恢复状态或元数据核验不一致")
		}
	}
	if fresh.StateOptionID != "" && fresh.DesiredState == 0 {
		return ItemDTO{}, conflict("Notion 状态为未映射的非空值，请先人工核对")
	}
	if fresh.DesiredState != 0 && fresh.DesiredState != desired {
		return ItemDTO{}, conflict("Notion 状态已有其他值，禁止覆盖")
	}
	if !resuming {
		beforeJSON, e := bootstrapBeforeJSON(localPointer, src.ConfigRevision, confirm.NewPage)
		if e != nil {
			return ItemDTO{}, e
		}
		now := time.Now().UTC()
		if e = s.fenced(ctx, token, func(tx *gorm.DB, c *model.NotionSyncControl) error {
			if !c.Paused || !c.SourceWritesPaused {
				return conflict("维护暂停已解除")
			}
			codes := []string{src.ModuleCode}
			if localPointer != nil {
				placement, e := catalog.Resolve(store.NewStore(tx), local.SectionCode, local.SubsectionCode)
				if e != nil {
					return e
				}
				codes = append(codes, placement.Module.Code)
			}
			if e := catalog.LockModules(store.NewStore(tx), codes...); e != nil {
				return e
			}
			var currentSource model.NotionSyncSource
			if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("source_id = ?", src.ID).First(&currentSource).Error; e != nil {
				return e
			}
			if currentSource.ConfigRevision != src.ConfigRevision {
				return conflict("审核后源配置已变化")
			}
			if localPointer != nil {
				var current model.Article
				if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&current, articleID).Error; e != nil {
					return e
				}
				if article.ArticleFingerprint(&current) != article.ArticleFingerprint(localPointer) {
					return conflict("文章在核对后发生变化")
				}
			}
			var journalArticleID interface{}
			if localPointer != nil {
				journalArticleID = articleID
			}
			result := tx.Model(&model.NotionPageBinding{}).Where("page_id = ? AND management_state = ? AND revision = ?", pageID, model.NotionManagementBaselinePending, p.Revision).Updates(map[string]interface{}{"article_id": journalArticleID, "bootstrap_state": "write_requested", "bootstrap_expected_state": desired, "confirmed_by": confirm.ConfirmedBy, "confirmed_at": now, "local_before_json": beforeJSON, "snapshot_json": jsonText(fresh), "last_error": ""})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				return conflict("接管绑定已变化")
			}
			return nil
		}); e != nil {
			return ItemDTO{}, e
		}
	}
	if e := s.fenced(ctx, token, func(tx *gorm.DB, c *model.NotionSyncControl) error {
		if !c.Paused || !c.SourceWritesPaused {
			return conflict("维护暂停已解除")
		}
		var current model.NotionSyncSource
		if e := tx.Where("source_id = ?", src.ID).First(&current).Error; e != nil {
			return e
		}
		if current.ConfigRevision != src.ConfigRevision {
			return conflict("审核后源配置已变化")
		}
		return nil
	}); e != nil {
		return ItemDTO{}, e
	}
	var writeErr error
	if fresh.DesiredState != desired {
		writeErr = writer.UpdateBlogState(ctx, pageID, cfg.StatePropertyID, cfg.StateOptionIDs[confirm.ExpectedState])
	}
	verifiedPage, readErr := s.client.RetrievePage(ctx, pageID)
	if readErr != nil {
		return ItemDTO{}, fmt.Errorf("接管写入结果待核验: %w", readErr)
	}
	verified, e := snapshotOf(verifiedPage, p.SourceID, cfg)
	if e != nil {
		return ItemDTO{}, e
	}
	verifiedSchema, e := s.client.RetrieveDataSource(ctx, src.ID)
	if e != nil {
		return ItemDTO{}, e
	}
	if e := validateConfig(verifiedSchema, cfg); e != nil {
		return ItemDTO{}, e
	}
	if e := validateBootstrapTopic(verifiedSchema, cfg, verified); e != nil {
		return ItemDTO{}, e
	}
	if verified.DesiredState != desired || bootstrapMetadataFingerprint(verified) != bootstrapMetadataFingerprint(fresh) {
		if writeErr != nil {
			return ItemDTO{}, fmt.Errorf("接管未确认写入: %w", writeErr)
		}
		return ItemDTO{}, conflict("Notion 回读状态或元数据不一致")
	}
	if e = s.fenced(ctx, token, func(tx *gorm.DB, c *model.NotionSyncControl) error {
		if !c.Paused || !c.SourceWritesPaused {
			return conflict("维护暂停已解除")
		}
		result := tx.Model(&model.NotionPageBinding{}).Where("page_id = ? AND revision = ?", pageID, p.Revision).Updates(map[string]interface{}{"bootstrap_state": "verified", "snapshot_json": jsonText(verified), "desired_state": desired, "page_url": verified.PageURL, "public_url": verified.PublicURL, "notion_last_edited_at": verified.LastEditedAt, "metadata_hash": metadataHash(verified)})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return conflict("接管绑定已变化")
		}
		return nil
	}); e != nil {
		return ItemDTO{}, e
	}
	_, e = article.NewForSync(s.ds, s.opts.Author).AdoptSyncedSource(ctx, article.AdoptInput{Lease: token, PageID: pageID, SourceID: src.ID, ExpectedFingerprint: confirm.ExpectedFingerprint, ExpectedArticleID: articleID, ExpectedBindingRevision: p.Revision, ExpectedConfigRevision: src.ConfigRevision})
	if e != nil {
		return ItemDTO{}, e
	}
	return ItemDTO{PageID: pageID, ArticleID: articleIDString(articleID), Outcome: "adopted", Before: map[string]interface{}{"state": stateName(local.Status), "article_id": confirm.ArticleID, "section_code": local.SectionCode}, After: map[string]interface{}{"state": stateName(desired), "management_state": "managed", "new_page": confirm.NewPage}}, nil
}
func articleIDString(id uint64) *string {
	if id == 0 {
		return nil
	}
	v := strconv.FormatUint(id, 10)
	return &v
}
func bootstrapBeforeJSON(a *model.Article, revision uint64, newPage bool) (string, error) {
	raw, e := article.ArticleBeforeJSON(a)
	if e != nil {
		return "", e
	}
	var document map[string]json.RawMessage
	if e = json.Unmarshal([]byte(raw), &document); e != nil {
		return "", e
	}
	if document == nil {
		document = map[string]json.RawMessage{}
	}
	document["_bootstrap_config_revision"] = json.RawMessage(strconv.FormatUint(revision, 10))
	if newPage {
		document["_bootstrap_new_page"] = json.RawMessage("true")
	} else {
		document["_bootstrap_new_page"] = json.RawMessage("false")
	}
	result, e := json.Marshal(document)
	return string(result), e
}

func (s *Service) synchronousRun(ctx context.Context, mode string, fn func(context.Context, model.NotionSyncRun, store.LeaseToken, *RunCounts) error) (string, error) {
	taskctx, taskcancel, taskID, e := s.beginTask(ctx)
	if e != nil {
		return "", e
	}
	defer s.endTask(taskID, taskcancel)
	ctx = taskctx
	db, e := s.db(ctx)
	if e != nil {
		return "", e
	}
	repo := store.NewNotionSyncRepository(db)
	control, e := repo.Control(ctx)
	if e != nil {
		return "", e
	}
	nowDB, e := store.DatabaseNow(db)
	if e != nil {
		return "", e
	}
	if control.CooldownUntil != nil && control.CooldownUntil.After(nowDB) {
		return "", &Error{Code: "cooldown", Message: "Notion 限流冷却尚未结束", HTTPStatus: 429}
	}
	token, ok, e := repo.AcquireLease(ctx, s.opts.OwnerID, s.opts.LeaseDuration)
	if e != nil {
		return "", e
	}
	if !ok {
		return "", conflict("已有同步或接管任务运行")
	}
	id := newID()
	run := model.NotionSyncRun{ID: id, Mode: mode, Status: "running", Phase: "bootstrap", StartedAt: time.Now().UTC(), LeaseEpoch: token.Epoch, CountsJSON: jsonText(RunCounts{})}
	if e = s.fenced(ctx, token, func(tx *gorm.DB, c *model.NotionSyncControl) error {
		if e := tx.Create(&run).Error; e != nil {
			return e
		}
		return tx.Model(c).Update("current_run_id", id).Error
	}); e != nil {
		_ = repo.ReleaseLease(context.Background(), token)
		return id, e
	}
	runctx, cancel := context.WithCancel(ctx)
	runctx = context.WithValue(runctx, leaseContextKey{}, token)
	done := make(chan struct{})
	go s.heartbeat(runctx, token, cancel, done)
	counts := RunCounts{}
	runErr := fn(runctx, run, token, &counts)
	cancel()
	<-done
	finishctx, finishcancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer finishcancel()
	now := time.Now().UTC()
	status := "completed"
	if runErr != nil {
		status = "failed"
	} else if counts.Failed > 0 {
		status = "completed_with_errors"
	}

	updates := map[string]interface{}{"status": status, "phase": "finished", "finished_at": now, "counts_json": jsonText(counts)}
	if runErr != nil {
		updates["error"] = runErr.Error()
	}
	e = s.finishRun(finishctx, run, token, updates, false)

	_ = repo.ReleaseLease(finishctx, token)
	if runErr != nil {
		return id, runErr
	}
	return id, e
}

// AllowedSources returns the fixed data-source inventory, without credentials.
func AllowedSources() []string {
	result := make([]string, 0, len(allowedSources))
	for id := range allowedSources {
		result = append(result, id)
	}
	sort.Strings(result)
	return result
}

func validateBootstrapTopic(schema source.NotionDataSource, cfg SourceConfig, snap Snapshot) error {
	if snap.TopicOptionID == "" {
		return nil
	}
	property, ok := schemaProperty(schema, cfg.TopicPropertyID)
	if !ok || property.Type != "select" {
		return conflict("主题字段已变化，请重新审核")
	}
	for _, option := range property.Select.Options {
		if option.ID == snap.TopicOptionID {
			if option.Name != snap.TopicOptionName {
				return conflict("主题选项名称已变化，请重新审核")
			}
			return nil
		}
	}
	return conflict("主题选项身份已变化，请重新审核")
}
