package notionsync

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/source"
	"gorm.io/gorm"
)

func preparePublishedFixtureCatalog(t *testing.T, s *Service, db *gorm.DB, id string) {
	t.Helper()
	if err := db.Model(&model.NotionSyncSource{}).Where("source_id = ?", id).Update("enabled", true).Error; err != nil {
		t.Fatal(err)
	}
	code := "s1"
	if err := db.Create(&model.NotionCatalogBinding{SourceID: id, DataSourceID: id, ThemePropertyID: "topic", OptionID: "opaque-option", OptionName: "主题一", SectionCode: &code, Status: model.NotionCatalogBound}).Error; err != nil {
		t.Fatal(err)
	}
}
func TestBootstrapPublishedRequiresReviewedPublicConditionsBeforeRemoteWrite(t *testing.T) {
	for _, gate := range []string{"source_disabled", "public_url_missing", "catalog_disabled", "publication_hold", "theme_binding_invalid", "theme_binding_conflict"} {
		t.Run(gate, func(t *testing.T) {
			s, db, f := syncFixture(t)
			id := AllowedSources()[0]
			if _, err := s.UpdateSource(context.Background(), id, SourceInput{ModuleCode: "m1"}); err != nil {
				t.Fatal(err)
			}
			preparePublishedFixtureCatalog(t, s, db, id)
			f.pages[firstPage] = fixturePage(firstPage, id, "")
			article := model.Article{Title: "History", SectionCode: "s1", ExternalLink: "https://notion.so/" + firstPage, Status: 2}
			if err := db.Create(&article).Error; err != nil {
				t.Fatal(err)
			}
			switch gate {
			case "source_disabled":
				db.Model(&model.NotionSyncSource{}).Where("source_id = ?", id).Update("enabled", false)
			case "public_url_missing":
				page := f.pages[firstPage]
				page.PublicURL = nil
				f.pages[firstPage] = page
			case "catalog_disabled":
				db.Model(&model.Section{}).Where("code = ?", "s1").Update("status", 2)
			case "theme_binding_invalid":
				db.Where("source_id = ?", id).Delete(&model.NotionCatalogBinding{})
			case "theme_binding_conflict":
				code := "s1"
				db.Create(&model.NotionCatalogBinding{SourceID: id, DataSourceID: id, ThemePropertyID: "topic", OptionID: "another", OptionName: "Another", SectionCode: &code, Status: model.NotionCatalogBound})
			}
			preview, err := s.BootstrapPreview(context.Background())
			if err != nil || len(preview.Items) != 1 {
				t.Fatal(preview, err)
			}
			if gate == "publication_hold" {
				db.Model(&model.NotionPageBinding{}).Where("page_id = ?", firstPage).Update("publication_held", true)
			}
			yes := true
			if _, err = s.UpdateControl(context.Background(), ControlInput{Paused: &yes, SourceWritesPaused: &yes}); err != nil {
				t.Fatal(err)
			}
			c := preview.Items[0]
			var src model.NotionSyncSource
			db.First(&src, "source_id = ?", id)
			result, err := s.BootstrapApply(context.Background(), BootstrapInput{SourceID: id, ExpectedConfigRevision: src.ConfigRevision, Confirmations: []BootstrapConfirm{{PageID: firstPage, ArticleID: strconv.FormatUint(article.ID, 10), ExpectedState: "published", ExpectedFingerprint: c.ExpectedFingerprint, ConfirmedBy: "fixture reviewer"}}}, f)
			if err != nil || len(result.Items) != 1 || result.Items[0].Outcome != "failed" || f.writes != 0 {
				t.Fatal(result, err, f.writes)
			}
			var binding model.NotionPageBinding
			db.First(&binding, "page_id = ?", firstPage)
			if binding.ManagementState != model.NotionManagementBaselinePending || binding.BootstrapState != "pending" {
				t.Fatal("blocked public backfill changed ownership/journal", binding)
			}
		})
	}
}
func TestScopedBootstrapAuditsAllConfirmationsBeforeAnyWrite(t *testing.T) {
	for _, gate := range []string{"other_local_source", "other_remote_source", "later_read_failed", "config_stale", "later_id_mismatch"} {
		t.Run(gate, func(t *testing.T) {
			s, db, f := syncFixture(t)
			id := AllowedSources()[0]
			other := AllowedSources()[1]
			if _, err := s.UpdateSource(context.Background(), id, SourceInput{ModuleCode: "m1"}); err != nil {
				t.Fatal(err)
			}
			f.pages[firstPage] = fixturePage(firstPage, id, "")
			f.pages[secondPage] = fixturePage(secondPage, id, "")
			preview, err := s.BootstrapPreview(context.Background())
			if err != nil || len(preview.Items) != 2 {
				t.Fatal(preview, err)
			}
			confirmations := []BootstrapConfirm{}
			for _, c := range preview.Items {
				confirmations = append(confirmations, BootstrapConfirm{PageID: c.PageID, NewPage: true, ExpectedState: "draft", ExpectedFingerprint: c.ExpectedFingerprint, ConfirmedBy: "fixture reviewer"})
			}
			yes := true
			if _, err = s.UpdateControl(context.Background(), ControlInput{Paused: &yes, SourceWritesPaused: &yes}); err != nil {
				t.Fatal(err)
			}
			var src model.NotionSyncSource
			db.First(&src, "source_id = ?", id)
			input := BootstrapInput{SourceID: id, ExpectedConfigRevision: src.ConfigRevision, Confirmations: confirmations}
			switch gate {
			case "other_local_source":
				db.Model(&model.NotionPageBinding{}).Where("page_id = ?", secondPage).Update("source_id", other)
			case "other_remote_source":
				page := f.pages[secondPage]
				page.Parent.DataSourceID = other
				f.pages[secondPage] = page
			case "later_read_failed":
				f.getError[secondPage] = errors.New("not readable")
			case "config_stale":
				input.ExpectedConfigRevision++
			case "later_id_mismatch":
				page := f.pages[secondPage]
				page.ID = firstPage
				f.pages[secondPage] = page
			}
			_, err = s.BootstrapApply(context.Background(), input, f)
			if err == nil || f.writes != 0 {
				t.Fatal("incomplete scope audit allowed earlier PATCH", gate, err, f.writes)
			}
			var count int64
			db.Model(&model.NotionPageBinding{}).Where("bootstrap_state <> ?", "pending").Count(&count)
			if count != 0 {
				t.Fatal("failed full scope audit changed journals", count)
			}
		})
	}
}

type movingReadClient struct {
	*fixtureNotion
	pageID string
	target string
	reads  int
}

func (m *movingReadClient) RetrievePage(ctx context.Context, id string) (source.NotionPage, error) {
	p, err := m.fixtureNotion.RetrievePage(ctx, id)
	if id == m.pageID {
		m.reads++
		if m.reads > 1 {
			p.Parent.DataSourceID = m.target
		}
	}
	return p, err
}
func TestScopedBootstrapRejectsSourceDriftAfterPreflight(t *testing.T) {
	s, db, f := syncFixture(t)
	id := AllowedSources()[0]
	if _, err := s.UpdateSource(context.Background(), id, SourceInput{ModuleCode: "m1"}); err != nil {
		t.Fatal(err)
	}
	f.pages[firstPage] = fixturePage(firstPage, id, "")
	preview, err := s.BootstrapPreview(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	yes := true
	s.UpdateControl(context.Background(), ControlInput{Paused: &yes, SourceWritesPaused: &yes})
	var src model.NotionSyncSource
	db.First(&src, "source_id = ?", id)
	s.client = &movingReadClient{fixtureNotion: f, pageID: firstPage, target: AllowedSources()[1]}
	c := preview.Items[0]
	result, err := s.BootstrapApply(context.Background(), BootstrapInput{SourceID: id, ExpectedConfigRevision: src.ConfigRevision, Confirmations: []BootstrapConfirm{{PageID: firstPage, NewPage: true, ExpectedState: "draft", ExpectedFingerprint: c.ExpectedFingerprint, ConfirmedBy: "reviewer"}}}, f)
	if err != nil || len(result.Items) != 1 || result.Items[0].Outcome != "failed" || f.writes != 0 {
		t.Fatal(result, err, f.writes)
	}
}

type catalogDriftWriter struct {
	fixture *fixtureNotion
	db      *gorm.DB
}

func (w catalogDriftWriter) UpdateBlogState(ctx context.Context, id, property, option string) error {
	if err := w.fixture.UpdateBlogState(ctx, id, property, option); err != nil {
		return err
	}
	return w.db.Model(&model.Section{}).Where("code = ?", "s1").Update("status", 2).Error
}
func TestBootstrapPublishedAdoptRechecksCatalogAfterSuccessfulRemoteWrite(t *testing.T) {
	s, db, f := syncFixture(t)
	id := AllowedSources()[0]
	if _, err := s.UpdateSource(context.Background(), id, SourceInput{ModuleCode: "m1"}); err != nil {
		t.Fatal(err)
	}
	preparePublishedFixtureCatalog(t, s, db, id)
	f.pages[firstPage] = fixturePage(firstPage, id, "")
	a := model.Article{Title: "History", SectionCode: "s1", ExternalLink: "https://notion.so/" + firstPage, Status: 2}
	if err := db.Create(&a).Error; err != nil {
		t.Fatal(err)
	}
	preview, err := s.BootstrapPreview(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	yes := true
	s.UpdateControl(context.Background(), ControlInput{Paused: &yes, SourceWritesPaused: &yes})
	c := preview.Items[0]
	result, err := s.BootstrapApply(context.Background(), BootstrapInput{Confirmations: []BootstrapConfirm{{PageID: firstPage, ArticleID: strconv.FormatUint(a.ID, 10), ExpectedState: "published", ExpectedFingerprint: c.ExpectedFingerprint, ConfirmedBy: "reviewer"}}}, catalogDriftWriter{fixture: f, db: db})
	if err != nil || len(result.Items) != 1 || result.Items[0].Outcome != "failed" || f.writes != 1 {
		t.Fatal(result, err, f.writes)
	}
	var p model.NotionPageBinding
	db.First(&p, "page_id = ?", firstPage)
	if p.BootstrapState != "verified" || p.ManagementState != model.NotionManagementBaselinePending {
		t.Fatal("catalog drift allowed adoption or lost recoverable journal", p)
	}
}

func TestBootstrapPreviewRetainsLatestSnapshotAndBindingAudit(t *testing.T) {
	s, _, f := syncFixture(t)
	ctx := context.Background()
	const pageID = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa1"
	sourceID := AllowedSources()[0]
	off := false
	for _, id := range AllowedSources() {
		if _, err := s.UpdateSource(ctx, id, SourceInput{ModuleCode: "m1", Enabled: &off}); err != nil {
			t.Fatal(err)
		}
	}
	f.pages[pageID] = fixturePage(pageID, sourceID, "")
	preview, err := s.BootstrapPreview(ctx)
	if err != nil || len(preview.Items) != 1 {
		t.Fatal(preview, err)
	}
	items, err := s.Items(ctx, preview.RunID, ListQuery{Page: 1, Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	var item ItemDTO
	pageRows := 0
	for _, row := range items.Items {
		if row.PageID == pageID {
			item = row
			pageRows++
		}
	}
	if pageRows != 1 {
		t.Fatalf("expected one retained page audit, got %d", pageRows)
	}
	before, ok := item.Before.(map[string]interface{})
	if !ok || before["management_state"] != model.NotionManagementBaselinePending {
		t.Fatal("missing audited binding state", item.Before)
	}
	after, ok := item.After.(map[string]interface{})
	if !ok {
		t.Fatal("missing audit object")
	}
	snap, ok := after["snapshot"].(map[string]interface{})
	if !ok || snap["page_id"] != pageID || snap["source_id"] != sourceID || snap["desired_state"] != float64(0) || snap["state_option_id"] != "" {
		t.Fatal("missing latest audited snapshot", snap)
	}
	candidate, ok := after["bootstrap_preview"].(map[string]interface{})
	if !ok || candidate["expected_fingerprint"] != preview.Items[0].ExpectedFingerprint || item.Outcome != "bootstrap_preview" {
		t.Fatal("candidate contract changed", candidate)
	}
	run, err := s.Run(ctx, preview.RunID)
	if err != nil || run.Status != "completed" || run.Counts.Seen != 1 {
		t.Fatal(run, err)
	}
}
