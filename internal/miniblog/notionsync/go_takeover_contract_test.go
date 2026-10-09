package notionsync

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/yshujie/miniblog/internal/miniblog/biz/catalog"
	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/source"
	"github.com/yshujie/miniblog/internal/miniblog/store"
)

type goTrialReadbackFailure struct {
	fixture     *fixtureNotion
	pageID      string
	interrupted bool
}

func (w *goTrialReadbackFailure) UpdateBlogState(ctx context.Context, id, property, option string) error {
	err := w.fixture.UpdateBlogState(ctx, id, property, option)
	if err == nil && id == w.pageID && !w.interrupted {
		w.interrupted = true
		w.fixture.mu.Lock()
		w.fixture.getError[id] = &source.NotionAPIError{StatusCode: 503, Code: "fixture_readback_unknown"}
		w.fixture.mu.Unlock()
	}
	return err
}

func TestGoFourteenHistoricalAndTwentyFourDraftsTakeover(t *testing.T) {
	for _, interrupt := range []bool{false, true} {
		t.Run(fmt.Sprintf("readback_interrupted_%t", interrupt), func(t *testing.T) {
			ctx := context.Background()
			s, db, f := syncFixture(t)
			const goSource = "2bf330bd-ddf1-80a6-aa49-000bbd1e154b"
			off, on := false, true
			for _, id := range AllowedSources() {
				module := "m2"
				if id == goSource {
					module = "m1"
				}
				if _, err := s.UpdateSource(ctx, id, SourceInput{ModuleCode: module, Enabled: &off}); err != nil {
					t.Fatal(err)
				}
			}
			originals := map[uint64]model.Article{}
			var legacyNullID uint64
			historyIDs := map[string]uint64{}
			managedIDs := map[uint64]bool{}
			for i := 0; i < 38; i++ {
				id := fmt.Sprintf("%032x", i+1)
				page := fixturePage(id, goSource, "")
				property := page.Properties["标题"]
				property.Title = []source.NotionRichText{{PlainText: fmt.Sprintf("Notion article %02d", i)}}
				page.Properties["标题"] = property
				f.pages[id] = page
				if i < 14 {
					a := model.Article{ID: 9007199254740993 + uint64(i), Title: fmt.Sprintf("Historical article %02d", i), Content: fmt.Sprintf("Preserved body %02d", i), Author: "Original author", ExternalLink: "https://notion.so/" + id + "?historical=keep#original", SectionCode: "s1", Pos: i + 11, Status: model.ArticleStatusPublished, CreatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), UpdatedAt: time.Date(2020, 2, 1, 0, 0, 0, 0, time.UTC)}
					identity, err := source.Parse(a.ExternalLink)
					if err != nil {
						t.Fatal(err)
					}
					a.Provider, a.CanonicalURL, a.SourceKey = &identity.Provider, &identity.CanonicalURL, &identity.SourceKey
					if err := db.Create(&a).Error; err != nil {
						t.Fatal(err)
					}
					if i == 0 {
						legacyNullID = a.ID
						if err := db.Model(&model.Article{}).Where("id = ?", a.ID).UpdateColumn("subsection_code", nil).Error; err != nil {
							t.Fatal(err)
						}
					}
					if err := db.First(&a, a.ID).Error; err != nil {
						t.Fatal(err)
					}
					originals[a.ID] = a
					historyIDs[id] = a.ID
					managedIDs[a.ID] = true
				}
			}
			for i := 0; i < 6; i++ {
				status := model.ArticleStatusPublished
				if i == 0 {
					status = model.ArticleStatusDraft
				}
				a := model.Article{ID: 9007199254742000 + uint64(i), Title: fmt.Sprintf("Manual Go article %d", i), Content: "Manual body", Author: "Manual author", ExternalLink: fmt.Sprintf("https://example.com/manual/%d", i), SectionCode: "s1", Pos: 100 + i, Status: status}
				identity, err := source.Parse(a.ExternalLink)
				if err != nil {
					t.Fatal(err)
				}
				a.Provider, a.CanonicalURL, a.SourceKey = &identity.Provider, &identity.CanonicalURL, &identity.SourceKey
				if err := db.Create(&a).Error; err != nil {
					t.Fatal(err)
				}
				if err := db.First(&a, a.ID).Error; err != nil {
					t.Fatal(err)
				}
				originals[a.ID] = a
			}
			var otherSource string
			for _, id := range AllowedSources() {
				if id != goSource {
					otherSource = id
					break
				}
			}
			const otherPage = "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"
			f.pages[otherPage] = fixturePage(otherPage, otherSource, "")
			first, err := s.BootstrapPreview(ctx)
			if err != nil || len(first.Items) != 39 {
				t.Fatal(first, err)
			}
			if _, err = s.UpdateControl(ctx, ControlInput{Paused: &on, SourceWritesPaused: &on}); err != nil {
				t.Fatal(err)
			}
			var row model.NotionSyncSource
			if err = db.First(&row, "source_id = ?", goSource).Error; err != nil {
				t.Fatal(err)
			}
			zero, normal := 0, 1
			prepared, err := s.PrepareCatalog(ctx, CatalogPrepareInput{SourceID: goSource, ExpectedConfigRevision: row.ConfigRevision, ReuseMap: []catalog.TopicReuseInput{{OptionID: "opaque-option", ExpectedOptionName: "主题一", SectionCode: "s1", ExpectedSectionTitle: "Historical section", ExpectedSectionStatus: &normal, ExpectedSectionSort: &zero}}})
			if err != nil {
				t.Fatal(err)
			}
			src, err := s.UpdateSource(ctx, goSource, SourceInput{Enabled: &on, ExpectedConfigRevision: &prepared.ConfigRevision})
			if err != nil {
				t.Fatal(err)
			}
			final, err := s.BootstrapPreview(ctx)
			if err != nil {
				t.Fatal(err)
			}
			drafts, published := []BootstrapConfirm{}, []BootstrapConfirm{}
			for _, candidate := range final.Items {
				if candidate.SourceID != goSource {
					continue
				}
				c := BootstrapConfirm{PageID: candidate.PageID, ExpectedFingerprint: candidate.ExpectedFingerprint, ConfirmedBy: "fixture approved reviewer"}
				if articleID, ok := historyIDs[candidate.PageID]; ok {
					if candidate.ArticleID != strconv.FormatUint(articleID, 10) || candidate.PublishBlockReason != "" {
						t.Fatal(candidate)
					}
					c.ArticleID = candidate.ArticleID
					c.ExpectedState = "published"
					published = append(published, c)
				} else {
					c.NewPage = true
					c.ExpectedState = "draft"
					drafts = append(drafts, c)
				}
			}
			if len(drafts) != 24 || len(published) != 14 {
				t.Fatal(len(drafts), len(published))
			}
			draftResult, err := s.BootstrapApply(ctx, BootstrapInput{SourceID: goSource, ExpectedConfigRevision: src.ConfigRevision, Confirmations: drafts}, f)
			if err != nil || len(draftResult.Items) != 24 {
				t.Fatal(draftResult, err)
			}
			for _, item := range draftResult.Items {
				if item.Outcome != "adopted" {
					t.Fatal(item)
				}
			}
			writer := &goTrialReadbackFailure{fixture: f}
			if interrupt {
				writer.pageID = published[0].PageID
			}
			publishResult, err := s.BootstrapApply(ctx, BootstrapInput{SourceID: goSource, ExpectedConfigRevision: src.ConfigRevision, Confirmations: published}, writer)
			if err != nil || len(publishResult.Items) != 14 {
				t.Fatal(publishResult, err)
			}
			failures := 0
			for _, item := range publishResult.Items {
				if item.Outcome == "failed" {
					failures++
				} else if item.Outcome != "adopted" {
					t.Fatal(item)
				}
			}
			if (interrupt && failures != 1) || (!interrupt && failures != 0) {
				t.Fatal(failures)
			}
			if interrupt {
				f.mu.Lock()
				delete(f.getError, writer.pageID)
				f.mu.Unlock()
				resumed, err := s.BootstrapApply(ctx, BootstrapInput{SourceID: goSource, ExpectedConfigRevision: src.ConfigRevision, Confirmations: published}, f)
				if err != nil {
					t.Fatal(resumed, err)
				}
				for _, item := range resumed.Items {
					if item.Outcome != "adopted" && item.Outcome != "already_adopted" {
						t.Fatal(item)
					}
				}
			}
			if f.writes != 38 {
				t.Fatalf("expected exactly one state write per page, got %d", f.writes)
			}
			var articleCount int64
			db.Model(&model.Article{}).Count(&articleCount)
			if articleCount != 20 {
				t.Fatal(articleCount)
			}
			for id, before := range originals {
				var after model.Article
				if err = db.First(&after, id).Error; err != nil {
					t.Fatal(err)
				}
				if after.Content != before.Content || after.Author != before.Author || after.ExternalLink != before.ExternalLink || after.Pos != before.Pos || after.Status != before.Status || !after.CreatedAt.Equal(before.CreatedAt) || !after.UpdatedAt.Equal(before.UpdatedAt) {
					t.Fatalf("takeover changed preserved historical fields for %s", strconv.FormatUint(id, 10))
				}
			}
			requireLegacyNull := func() {
				t.Helper()
				var raw sql.NullString
				if err := db.Raw("SELECT subsection_code FROM article WHERE id = ?", legacyNullID).Row().Scan(&raw); err != nil {
					t.Fatal(err)
				}
				if raw.Valid {
					t.Fatal("takeover or sync normalized legacy NULL subsection")
				}
			}
			requireLegacyNull()
			if _, err = s.UpdateControl(ctx, ControlInput{Paused: &off, SourceWritesPaused: &off}); err != nil {
				t.Fatal(err)
			}
			synced := waitFixtureRun(t, s, "sync")
			if synced.Status != "completed" || synced.Counts.Created != 0 {
				items, _ := s.Items(ctx, synced.RunID, ListQuery{Page: 1, Limit: 100})
				if items != nil {
					for _, item := range items.Items {
						if item.Error != "" {
							t.Logf("first sync failure: %s", item.Error)
							break
						}
					}
				}
				t.Fatalf("sync=%+v", synced)
			}
			db.Model(&model.Article{}).Count(&articleCount)
			if articleCount != 20 {
				t.Fatal(articleCount)
			}
			requireLegacyNull()
			var visible int64
			store.PublishedArticles(ctx, db).Where("module.code = ?", "m1").Count(&visible)
			if visible != 19 {
				t.Fatalf("Go visible articles=%d", visible)
			}
			var draftBindings int64
			db.Model(&model.NotionPageBinding{}).Where("source_id = ? AND management_state = ? AND article_id IS NULL AND desired_state = ?", goSource, model.NotionManagementManaged, model.ArticleStatusDraft).Count(&draftBindings)
			if draftBindings != 24 {
				t.Fatal(draftBindings)
			}
			for id, before := range originals {
				var after model.Article
				db.First(&after, id)
				if after.Content != before.Content || after.Author != before.Author || after.ExternalLink != before.ExternalLink || after.Pos != before.Pos || after.Status != before.Status || !after.CreatedAt.Equal(before.CreatedAt) {
					t.Fatal("fresh sync lost historical fields", id)
				}
				if !managedIDs[id] && !reflect.DeepEqual(before, after) {
					t.Fatal("manual record changed", id)
				}
			}
			var other model.NotionPageBinding
			db.First(&other, "page_id = ?", otherPage)
			if other.ManagementState != model.NotionManagementBaselinePending {
				t.Fatal(other)
			}
			const newPage = "ffffffffffffffffffffffffffffffff"
			f.pages[newPage] = fixturePage(newPage, goSource, "published")
			preview, err := s.BootstrapPreview(ctx)
			if err != nil {
				t.Fatal(err)
			}
			for _, candidate := range preview.Items {
				if candidate.PageID == newPage {
					t.Fatal("new page was incorrectly frozen as historical")
				}
			}
			created := waitFixtureRun(t, s, "sync")
			if created.Counts.Created != 1 {
				t.Fatal(created)
			}
			var newBinding model.NotionPageBinding
			db.First(&newBinding, "page_id = ?", newPage)
			if newBinding.ArticleID == nil {
				t.Fatal(newBinding)
			}
			var newArticle model.Article
			db.First(&newArticle, *newBinding.ArticleID)
			if newArticle.Author != "Default author" {
				t.Fatal(newArticle.Author)
			}
			beforeRepeat := newArticle
			repeated := waitFixtureRun(t, s, "sync")
			db.First(&newArticle, *newBinding.ArticleID)
			if repeated.Counts.Created != 0 || !reflect.DeepEqual(beforeRepeat, newArticle) {
				t.Fatal("repeat changed article", repeated)
			}
			latest, err := s.Status(ctx)
			if err != nil {
				t.Fatal(err)
			}
			for _, source := range latest.Sources {
				if source.SourceID != goSource && (source.Enabled || source.LastSuccessAt != nil) {
					t.Fatal("non-Go source was applied", source)
				}
			}
			if src.ConfigRevision == 0 {
				t.Fatal("missing source revision")
			}
		})
	}
}
