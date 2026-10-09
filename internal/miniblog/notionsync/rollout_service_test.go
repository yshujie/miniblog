package notionsync

import (
	"context"
	"encoding/json"
	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/source"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"testing"
)

func TestRolloutSourceNoopAndLocalConfigurationKeepVisibility(t *testing.T) {
	s, db, f := syncFixture(t)
	id := AllowedSources()[0]
	schema, _ := f.RetrieveDataSource(context.Background(), id)
	cfg, err := discoverConfig(schema)
	if err != nil {
		t.Fatal(err)
	}
	src := model.NotionSyncSource{ID: id, DataSourceID: id, Label: "Source", ModuleCode: "m1", Enabled: true, ConfigRevision: 8, PropertyMappingJSON: jsonText(cfg), StatusMappingJSON: jsonText(cfg.StateOptionIDs)}
	if err = db.Create(&src).Error; err != nil {
		t.Fatal(err)
	}
	a := model.Article{ID: 9007199254741003, Title: "Local history", SectionCode: "s1", Status: 2, ExternalLink: "https://example.com/history"}
	if err = db.Create(&a).Error; err != nil {
		t.Fatal(err)
	}
	page := fixturePage(firstPage, id, "published")
	p := model.NotionPageBinding{PageID: firstPage, ArticleID: &a.ID, SourceID: id, ManagementState: "managed", Revision: 1, DesiredState: 2, PublicURL: page.PublicURL, PageURL: page.URL}
	if err = db.Create(&p).Error; err != nil {
		t.Fatal(err)
	}
	on := true
	expected := uint64(8)
	got, err := s.UpdateSource(context.Background(), id, SourceInput{Label: "Source", ModuleCode: "m1", Enabled: &on, Config: &cfg, ExpectedConfigRevision: &expected})
	if err != nil {
		t.Fatal(err)
	}
	if got.ConfigRevision != 8 {
		t.Fatalf("noop revision=%d", got.ConfigRevision)
	}
	db.First(&p, "page_id = ?", firstPage)
	if p.NeedsRevalidation || p.Revision != 1 {
		t.Fatalf("noop invalidated page: %+v", p)
	}
	got, err = s.UpdateSource(context.Background(), id, SourceInput{Label: "Renamed", ExpectedConfigRevision: &expected})
	if err != nil {
		t.Fatal(err)
	}
	if got.ConfigRevision != 9 {
		t.Fatal(got)
	}
	expected = 9
	off := false
	got, err = s.UpdateSource(context.Background(), id, SourceInput{Enabled: &off, Config: &cfg, ExpectedConfigRevision: &expected})
	if err != nil {
		t.Fatal(err)
	}
	if got.ConfigRevision != 10 {
		t.Fatal(got)
	}
	db.First(&p, "page_id = ?", firstPage)
	if p.NeedsRevalidation {
		t.Fatal("rename/disable with unchanged config invalidated page")
	}
	expected = 10
	cfg.StateOptionIDs["draft"] = "new-draft-option"
	_, err = s.UpdateSource(context.Background(), id, SourceInput{Config: &cfg, ExpectedConfigRevision: &expected})
	if err != nil {
		t.Fatal(err)
	}
	db.First(&p, "page_id = ?", firstPage)
	if !p.NeedsRevalidation {
		t.Fatal("actual mapping change must revalidate")
	}
}

func TestRolloutBootstrapPreviewRecordedForReadOnlyReview(t *testing.T) {
	s, db, f := syncFixture(t)
	id := AllowedSources()[0]
	f.pages[firstPage] = fixturePage(firstPage, id, "")
	a := model.Article{ID: 9007199254741005, Title: "Historical title", SectionCode: "s1", Status: 2, ExternalLink: "https://www.notion.so/" + firstPage}
	if err := db.Create(&a).Error; err != nil {
		t.Fatal(err)
	}
	_, err := s.UpdateSource(context.Background(), id, SourceInput{ModuleCode: "m1"})
	if err != nil {
		t.Fatal(err)
	}
	preview, err := s.BootstrapPreview(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	items, err := s.Items(context.Background(), preview.RunID, ListQuery{Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, item := range items.Items {
		if item.PageID != firstPage {
			continue
		}
		after, ok := item.After.(map[string]interface{})
		if !ok {
			continue
		}
		candidate, ok := after["bootstrap_preview"].(map[string]interface{})
		if !ok {
			continue
		}
		found = true
		if candidate["article_id"] != "9007199254741005" || candidate["local_state"] != "published" || candidate["expected_fingerprint"] == "" {
			t.Fatal(candidate)
		}
	}
	if !found {
		t.Fatal("bootstrap preview missing from read-only run items")
	}
	var current model.Article
	db.First(&current, a.ID)
	if current.Title != a.Title || current.Status != a.Status || current.ExternalLink != a.ExternalLink {
		t.Fatal("preview changed article")
	}
	var count int64
	if err = store.PublishedArticles(context.Background(), db).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("visibility changed: count=%d err=%v", count, err)
	}
}

func TestRolloutVisibilityAndHealthSeparateReviewBlockersAndFailures(t *testing.T) {
	s, db, _ := syncFixture(t)
	public := "https://fixture.notion.site/11111111111141118111111111111111"
	cases := []struct {
		page, reason     string
		article          uint64
		state            int
		management       string
		hold, revalidate bool
		public           *string
		lastError        string
		visible          bool
	}{
		{"visible", "", 9007199254741011, 2, "managed", false, false, &public, "", true},
		{"hold", "publication_hold", 9007199254741012, 2, "managed", true, false, &public, "", false},
		{"revalidate", "needs_revalidation", 9007199254741013, 2, "managed", false, true, &public, "", false},
		{"draft", "state_not_published", 9007199254741014, 1, "managed", false, false, &public, "", false},
		{"missing_public", "public_url_missing", 9007199254741015, 2, "managed", false, false, nil, "", false},
		{"pending", "historical_match_requires_confirmation", 0, 0, "baseline_pending", false, false, nil, "", false},
		{"read_error", "", 9007199254741016, 2, "managed", false, false, &public, "permission denied", true},
	}
	for _, c := range cases {
		var id *uint64
		if c.article != 0 {
			a := model.Article{ID: c.article, Title: c.page, SectionCode: "s1", Status: c.state, ExternalLink: "https://example.com/" + c.page}
			if err := db.Create(&a).Error; err != nil {
				t.Fatal(err)
			}
			id = &a.ID
		}
		p := model.NotionPageBinding{PageID: c.page, ArticleID: id, SourceID: AllowedSources()[0], ManagementState: c.management, Revision: 1, DesiredState: c.state, PublicURL: c.public, PublicationHeld: c.hold, NeedsRevalidation: c.revalidate, LastError: c.lastError}
		if c.page == "missing_public" {
			p.PublishBlockReason = "public_url_missing"
		}
		if err := db.Create(&p).Error; err != nil {
			t.Fatal(err)
		}
	}
	pages, err := s.Pages(context.Background(), PageQuery{ListQuery: ListQuery{Limit: 100}})
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(pages.Items)
	if err != nil {
		t.Fatal(err)
	}
	var items []map[string]interface{}
	if err = json.Unmarshal(encoded, &items); err != nil {
		t.Fatal(err)
	}
	byPage := map[string]map[string]interface{}{}
	for _, item := range items {
		byPage[item["page_id"].(string)] = item
	}
	for _, c := range cases {
		item := byPage[c.page]
		if item["effective_visibility"] != c.visible || item["visibility_reason"] != c.reason || item["needs_revalidation"] != c.revalidate {
			t.Fatalf("page %s: %+v", c.page, item)
		}
		if c.article == 0 && item["local_state"] != nil {
			t.Fatal(item)
		}
		if c.article != 0 && item["local_state"] != stateName(c.state) {
			t.Fatal(item)
		}
	}
	status, err := s.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	encoded, _ = json.Marshal(status)
	var health map[string]interface{}
	json.Unmarshal(encoded, &health)
	if health["pending_count"] != float64(1) || health["blocked_count"] != float64(3) || health["error_count"] != float64(1) {
		t.Fatal(health)
	}
}

func TestRolloutSuccessfulPilotDoesNotClaimDisabledLibrariesApplied(t *testing.T) {
	s, db, f := syncFixture(t)
	pilot, disabled := AllowedSources()[0], AllowedSources()[1]
	f.pages[firstPage] = fixturePage(firstPage, disabled, "")
	waitFixtureRun(t, s, "dry_run")
	enableFixtureSource(t, s, pilot, "m1")
	f.pages[secondPage] = fixturePage(secondPage, pilot, "published")
	run := waitFixtureRun(t, s, "sync")
	if run.Status != "completed" || run.Counts.Pending != 1 || run.Counts.Failed != 0 || run.Counts.Blocked != 0 {
		t.Fatalf("normal historical review misclassified as execution failure: %+v", run)
	}
	var control model.NotionSyncControl
	if err := db.First(&control, 1).Error; err != nil {
		t.Fatal(err)
	}
	if control.LastSuccessAt == nil {
		t.Fatal("pending historical pages in a disabled source prevented successful pilot projection")
	}
	var sources []model.NotionSyncSource
	if err := db.Find(&sources).Error; err != nil {
		t.Fatal(err)
	}
	for _, source := range sources {
		if (source.LastSuccessAt != nil) != (source.ID == pilot) {
			t.Fatalf("misleading applied timestamp for source %s enabled=%v success=%v", source.ID, source.Enabled, source.LastSuccessAt)
		}
	}
}

func TestRolloutJournalRepreviewRequiresFreshReadAndAtomicReview(t *testing.T) {
	s, db, f := syncFixture(t)
	id, target := AllowedSources()[0], AllowedSources()[1]
	for _, sourceID := range []string{id, target} {
		if _, err := s.UpdateSource(context.Background(), sourceID, SourceInput{ModuleCode: "m1"}); err != nil {
			t.Fatal(err)
		}
	}
	f.pages[firstPage] = fixturePage(firstPage, id, "")
	preview, err := s.BootstrapPreview(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	confirm := BootstrapConfirm{PageID: firstPage, NewPage: true, ExpectedState: "draft", ExpectedFingerprint: preview.Items[0].ExpectedFingerprint, ConfirmedBy: "reviewer"}
	yes := true
	if _, err = s.UpdateControl(context.Background(), ControlInput{Paused: &yes, SourceWritesPaused: &yes}); err != nil {
		t.Fatal(err)
	}
	f.writeErr = &source.NotionAPIError{StatusCode: 503, Code: "service_unavailable"}
	result, err := s.BootstrapApply(context.Background(), BootstrapInput{Confirmations: []BootstrapConfirm{confirm}}, f)
	if err != nil || result.Items[0].Outcome != "failed" {
		t.Fatalf("journal %+v %v", result, err)
	}
	var before model.NotionPageBinding
	db.First(&before, "page_id = ?", firstPage)
	page := f.pages[firstPage]
	page.Parent.DataSourceID = target
	title := page.Properties["标题"]
	title.Title[0].PlainText = "Newly reviewed title"
	page.Properties["标题"] = title
	f.pages[firstPage] = page
	f.getError[firstPage] = &source.NotionAPIError{StatusCode: 403, Code: "restricted_resource"}
	if _, err = s.BootstrapPreview(context.Background()); err == nil {
		t.Fatal("journal preview used query cache despite fresh read failure")
	}
	var after model.NotionPageBinding
	db.First(&after, "page_id = ?", firstPage)
	if after.SnapshotJSON != before.SnapshotJSON || after.BootstrapExpectedFingerprint != before.BootstrapExpectedFingerprint {
		t.Fatal("failed review replaced journal snapshot or fingerprint")
	}
	delete(f.getError, firstPage)
	reviewed, err := s.BootstrapPreview(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if reviewed.Items[0].Title != "Newly reviewed title" || reviewed.Items[0].ExpectedFingerprint == confirm.ExpectedFingerprint {
		t.Fatal("new review did not expose fresh metadata")
	}
	items, err := s.Items(context.Background(), reviewed.RunID, ListQuery{Page: 1, Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	pageRows := 0
	for _, item := range items.Items {
		if item.PageID != firstPage {
			continue
		}
		pageRows++
		beforeAudit, ok := item.Before.(map[string]interface{})
		if !ok || beforeAudit["source_id"] != id {
			t.Fatal("cross-library review lost original binding source", item.Before)
		}
		afterAudit, ok := item.After.(map[string]interface{})
		if !ok {
			t.Fatal("missing cross-library review audit")
		}
		snapshotAudit, ok := afterAudit["snapshot"].(map[string]interface{})
		if !ok || snapshotAudit["source_id"] != target {
			t.Fatal("review snapshot lost current source", afterAudit["snapshot"])
		}
		candidateAudit, ok := afterAudit["bootstrap_preview"].(map[string]interface{})
		if !ok || candidateAudit["source_id"] != target {
			t.Fatal("review candidate lost current source", afterAudit["bootstrap_preview"])
		}
	}
	if pageRows != 1 {
		t.Fatalf("expected one cross-library page audit, got %d", pageRows)
	}
	db.First(&after, "page_id = ?", firstPage)
	var snap Snapshot
	if err = json.Unmarshal([]byte(after.SnapshotJSON), &snap); err != nil {
		t.Fatal(err)
	}
	if snap.Title != reviewed.Items[0].Title || snap.SourceID != target || after.SourceID != target || after.BootstrapExpectedFingerprint != reviewed.Items[0].ExpectedFingerprint {
		t.Fatal("review snapshot and fingerprint not saved together")
	}
	f.writeErr = nil
	stale, err := s.BootstrapApply(context.Background(), BootstrapInput{Confirmations: []BootstrapConfirm{confirm}}, f)
	if err != nil || stale.Items[0].Outcome != "failed" || f.writes != 1 {
		t.Fatalf("old review accepted %+v %v", stale, err)
	}
	confirm.ExpectedFingerprint = reviewed.Items[0].ExpectedFingerprint
	final, err := s.BootstrapApply(context.Background(), BootstrapInput{Confirmations: []BootstrapConfirm{confirm}}, f)
	if err != nil || final.Items[0].Outcome != "adopted" || f.writes != 2 {
		t.Fatalf("new review cannot recover %+v %v", final, err)
	}
}
