package notionsync

import (
	"context"
	"errors"
	"github.com/yshujie/miniblog/internal/miniblog/biz/article"
	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/source"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"gorm.io/gorm"
	"reflect"
	"strconv"
	"testing"
	"time"
)

type rolloutSchema struct {
	*fixtureNotion
	target  string
	invalid bool
	reads   int
}

func (f *rolloutSchema) RetrieveDataSource(ctx context.Context, id string) (source.NotionDataSource, error) {
	schema, err := f.fixtureNotion.RetrieveDataSource(ctx, id)
	if id == f.target && f.invalid {
		prop := schema.Properties["博客状态"]
		prop.Select.Options = prop.Select.Options[1:] // Existing Published remains parseable; only Draft option vanished.
		schema.Properties["博客状态"] = prop
	}
	return schema, err
}
func (f *rolloutSchema) RetrievePage(ctx context.Context, id string) (source.NotionPage, error) {
	f.reads++
	return f.fixtureNotion.RetrievePage(ctx, id)
}

func TestRolloutCrossSourceIsolation(t *testing.T) {
	for _, reason := range []string{"target_disabled", "target_schema_changed"} {
		for _, state := range []string{"published", "draft", "unpublished", "archived", ""} {
			t.Run(reason+"/"+state, func(t *testing.T) {
				s, db, f, a, before := managedFixture(t)
				target := AllowedSources()[1]
				enableFixtureSource(t, s, target, "m2")
				client := &rolloutSchema{fixtureNotion: f, target: target, invalid: reason == "target_schema_changed"}
				s.client = client
				if reason == "target_disabled" {
					db.Model(&model.NotionSyncSource{}).Where("source_id = ?", target).Update("enabled", false)
				}
				page := fixturePage(firstPage, target, state)
				title := page.Properties["标题"]
				title.Title = []source.NotionRichText{{PlainText: "Must not project"}}
				page.Properties["标题"] = title
				page.LastEditedTime = page.LastEditedTime.Add(time.Hour)
				f.pages[firstPage] = page
				run := waitFixtureRun(t, s, "sync")
				var after model.NotionPageBinding
				db.Where("page_id = ?", firstPage).First(&after)
				db.First(&a, a.ID)
				if reason == "target_disabled" && after.LastError != "" {
					t.Fatal("disabled target reported an execution failure", after.LastError)
				}
				if reason == "target_schema_changed" && after.LastError == "" {
					t.Fatal("schema failure lost error audit")
				}
				want := 2
				if state == "draft" {
					want = 1
				}
				if state == "unpublished" {
					want = 3
				}
				if state == "archived" {
					want = 4
				}
				var visible int64
				store.PublishedArticles(context.Background(), db).Count(&visible)
				if a.Status != want || a.Title != "Last successful title" || a.Content != "Preserved body" || a.Author != "Local author" || a.Pos != 11 || a.SectionCode != "s1" || after.SourceID != before.SourceID || after.SnapshotJSON != before.SnapshotJSON || after.MetadataHash != before.MetadataHash || !after.NeedsRevalidation || after.PublishBlockReason != reason || visible != 0 || client.reads == 0 {
					t.Fatalf("isolation lost old fields or visibility: article=%+v binding=%+v visible=%d reads=%d run=%+v", a, after, visible, client.reads, run)
				}
				client.invalid = false
				db.Model(&model.NotionSyncSource{}).Where("source_id = ?", target).Update("enabled", true)
				f.pages[firstPage] = fixturePage(firstPage, target, "published")
				page = f.pages[firstPage]
				page.LastEditedTime = page.LastEditedTime.Add(2 * time.Hour)
				f.pages[firstPage] = page
				waitFixtureRun(t, s, "sync")
				db.Where("page_id = ?", firstPage).First(&after)
				db.First(&a, a.ID)
				store.PublishedArticles(context.Background(), db).Count(&visible)
				if after.SourceID != target || after.NeedsRevalidation || after.PublishBlockReason != "" || a.Status != 2 || a.Title != "Fixture title" || a.SectionCode == "s1" || visible != 1 {
					t.Fatalf("fresh recovery failed %+v %+v visible=%d", a, after, visible)
				}
			})
		}
	}
}

func TestRolloutPostBaselinePreviewDoesNotClaimNewPage(t *testing.T) {
	s, db, f := syncFixture(t)
	waitFixtureRun(t, s, "dry_run")
	enableFixtureSource(t, s, AllowedSources()[0], "m1")
	f.pages[firstPage] = fixturePage(firstPage, AllowedSources()[0], "published")
	waitFixtureRun(t, s, "dry_run")
	var n int64
	db.Model(&model.NotionPageBinding{}).Where("page_id = ?", firstPage).Count(&n)
	if n != 0 {
		t.Fatal("post-baseline preview claimed page", n)
	}
	run := waitFixtureRun(t, s, "sync")
	if run.Counts.Created != 1 {
		t.Fatal(run)
	}
}

func TestRolloutDisabledTargetReadFailureRetainsSuccessfulProjection(t *testing.T) {
	for _, status := range []int{403, 404, 503} {
		t.Run(strconv.Itoa(status), func(t *testing.T) {
			s, db, f, a, before := managedFixture(t)
			db.First(&a, a.ID)
			target := AllowedSources()[1]
			enableFixtureSource(t, s, target, "m2")
			db.Model(&model.NotionSyncSource{}).Where("source_id = ?", target).Update("enabled", false)
			page := fixturePage(firstPage, target, "draft")
			page.PublicURL = nil
			f.pages[firstPage] = page
			f.getError[firstPage] = &source.NotionAPIError{StatusCode: status, Code: "fixture_read_failed"}
			waitFixtureRun(t, s, "sync")
			var after model.NotionPageBinding
			db.Where("page_id = ?", firstPage).First(&after)
			var got model.Article
			db.First(&got, a.ID)
			var visible int64
			store.PublishedArticles(context.Background(), db).Count(&visible)
			if !reflect.DeepEqual(a, got) || after.SourceID != before.SourceID || after.PublicURL == nil || *after.PublicURL != *before.PublicURL || after.SnapshotJSON != before.SnapshotJSON || after.NeedsRevalidation || after.PublishBlockReason != "" || visible != 1 {
				t.Fatalf("failed read caused withdrawal %+v %+v visible=%d", got, after, visible)
			}
		})
	}
}
func TestRolloutDisabledTargetClearsConfirmedMissingPublicURL(t *testing.T) {
	s, db, f, a, before := managedFixture(t)
	target := AllowedSources()[1]
	enableFixtureSource(t, s, target, "m2")
	db.Model(&model.NotionSyncSource{}).Where("source_id = ?", target).Update("enabled", false)
	page := fixturePage(firstPage, target, "")
	page.PublicURL = nil
	f.pages[firstPage] = page
	waitFixtureRun(t, s, "sync")
	var after model.NotionPageBinding
	db.Where("page_id = ?", firstPage).First(&after)
	db.First(&a, a.ID)
	if a.Status != 2 || after.PublicURL != nil || after.SourceID != before.SourceID || after.PageURL != before.PageURL || !after.NeedsRevalidation || after.PublishBlockReason != "target_disabled" {
		t.Fatal(a, after)
	}
}
func TestRolloutDisabledTargetNewPageDoesNotCreateOrClaim(t *testing.T) {
	s, db, f := syncFixture(t)
	waitFixtureRun(t, s, "dry_run")
	enableFixtureSource(t, s, AllowedSources()[0], "m1")
	f.pages[firstPage] = fixturePage(firstPage, AllowedSources()[1], "published")
	waitFixtureRun(t, s, "sync")
	var articles, pages int64
	db.Model(&model.Article{}).Count(&articles)
	db.Model(&model.NotionPageBinding{}).Count(&pages)
	if articles != 0 || pages != 0 {
		t.Fatal("disabled source claimed new page", articles, pages)
	}
}
func TestRolloutPausedPreviewPreservesManagedCrossSourceProjection(t *testing.T) {
	s, db, f, a, before := managedFixture(t)
	db.First(&a, a.ID)
	db.Where("page_id = ?", firstPage).First(&before)
	db.Model(&model.NotionSyncControl{}).Where("id = 1").Update("paused", true)
	f.pages[firstPage] = fixturePage(firstPage, AllowedSources()[1], "draft")
	if _, e := s.Trigger(context.Background(), TriggerInput{Mode: "sync"}); e == nil {
		t.Fatal("global pause accepted sync")
	}
	waitFixtureRun(t, s, "dry_run")
	var after model.NotionPageBinding
	db.Where("page_id = ?", firstPage).First(&after)
	var got model.Article
	db.First(&got, a.ID)
	if !reflect.DeepEqual(a, got) || !reflect.DeepEqual(before, after) {
		t.Fatal("paused preview changed projection", got, after)
	}
}
func TestRolloutReleasedHoldAlwaysFreshReadsUnchangedQuery(t *testing.T) {
	for _, kind := range []string{"withdrawn", "404", "valid"} {
		t.Run(kind, func(t *testing.T) {
			s, db, f, a, _ := managedFixture(t)
			b := article.NewForSync(s.ds, "")
			if _, e := b.SetPublicationHold(context.Background(), a.ID, article.HoldInput{Held: true}); e != nil {
				t.Fatal(e)
			}
			if _, e := b.SetPublicationHold(context.Background(), a.ID, article.HoldInput{Held: false}); e != nil {
				t.Fatal(e)
			}
			staleQuery := f.pages[firstPage]
			fresh := staleQuery
			if kind == "withdrawn" {
				fresh.PublicURL = nil
			}
			client := &rolloutFreshPage{fixtureNotion: f, page: fresh}
			if kind == "404" {
				client.fail = &source.NotionAPIError{StatusCode: 404, Code: "object_not_found"}
			}
			s.client = client
			run := waitFixtureRun(t, s, "sync")
			var after model.NotionPageBinding
			db.Where("page_id = ?", firstPage).First(&after)
			var visible int64
			store.PublishedArticles(context.Background(), db).Count(&visible)
			if client.reads == 0 {
				t.Fatal("released hold reused unchanged query", run)
			}
			switch kind {
			case "withdrawn":
				if after.PublicURL != nil || visible != 0 {
					t.Fatal(after, visible)
				}
			case "404":
				if !after.NeedsRevalidation || visible != 0 {
					t.Fatal(after, visible)
				}
			case "valid":
				if after.NeedsRevalidation || after.PublishBlockReason != "" || visible != 1 {
					t.Fatal(after, visible)
				}
			}
		})
	}
}

type rolloutFreshPage struct {
	*fixtureNotion
	page  source.NotionPage
	fail  error
	reads int
}

func (f *rolloutFreshPage) RetrievePage(_ context.Context, id string) (source.NotionPage, error) {
	f.reads++
	return f.page, f.fail
}

type rolloutUnavailableTarget struct {
	*rolloutFreshPage
	target string
}

func (f *rolloutUnavailableTarget) RetrieveDataSource(ctx context.Context, id string) (source.NotionDataSource, error) {
	if id == f.target {
		return source.NotionDataSource{}, &source.NotionAPIError{StatusCode: 403, Code: "restricted_resource"}
	}
	return f.fixtureNotion.RetrieveDataSource(ctx, id)
}
func TestRolloutFreshMoveToUnreadableSchemaDoesNotInferWithdrawal(t *testing.T) {
	s, db, f, a, before := managedFixture(t)
	db.First(&a, a.ID)
	target := AllowedSources()[1]
	enableFixtureSource(t, s, target, "m2")
	query := f.pages[firstPage]
	queryTitle := query.Properties["标题"]
	queryTitle.Title = []source.NotionRichText{{PlainText: "Query changed title"}}
	query.Properties["标题"] = queryTitle
	query.LastEditedTime = query.LastEditedTime.Add(time.Hour)
	f.pages[firstPage] = query
	fresh := fixturePage(firstPage, target, "draft")
	fresh.PublicURL = nil
	client := &rolloutUnavailableTarget{rolloutFreshPage: &rolloutFreshPage{fixtureNotion: f, page: fresh}, target: target}
	s.client = client
	waitFixtureRun(t, s, "sync")
	var got model.Article
	db.First(&got, a.ID)
	var p model.NotionPageBinding
	db.First(&p, "page_id = ?", firstPage)
	var visible int64
	store.PublishedArticles(context.Background(), db).Count(&visible)
	if client.reads == 0 || !reflect.DeepEqual(a, got) || p.SourceID != before.SourceID || p.PublicURL == nil || p.NeedsRevalidation || p.PublishBlockReason != "" || visible != 1 {
		t.Fatalf("schema403 projected withdrawal %+v %+v reads=%d visible=%d", got, p, client.reads, visible)
	}
}

func TestRolloutSameSourceDisabledChangedPageStaysFrozen(t *testing.T) {
	s, db, f, a, before := managedFixture(t)
	db.First(&a, a.ID)
	db.First(&before, "page_id = ?", firstPage)
	db.Model(&model.NotionSyncSource{}).Where("source_id = ?", before.SourceID).Update("enabled", false)
	page := fixturePage(firstPage, before.SourceID, "draft")
	page.PublicURL = nil
	f.pages[firstPage] = page
	waitFixtureRun(t, s, "sync")
	var got model.Article
	db.First(&got, a.ID)
	var after model.NotionPageBinding
	db.First(&after, "page_id = ?", firstPage)
	var visible int64
	store.PublishedArticles(context.Background(), db).Count(&visible)
	if !reflect.DeepEqual(a, got) || !reflect.DeepEqual(before, after) || visible != 1 {
		t.Fatalf("same-source disabled projected changes %+v %+v visible=%d", got, after, visible)
	}
}

func TestRolloutReliableUnpublishedNewPagesArePendingNotErrors(t *testing.T) {
	for _, state := range []string{"draft", "unpublished", "archived"} {
		t.Run(state, func(t *testing.T) {
			s, db, f := syncFixture(t)
			waitFixtureRun(t, s, "dry_run")
			enableFixtureSource(t, s, AllowedSources()[0], "m1")
			page := fixturePage(firstPage, AllowedSources()[0], state)
			page.PublicURL = nil
			f.pages[firstPage] = page
			run := waitFixtureRun(t, s, "sync")
			var n int64
			db.Model(&model.Article{}).Count(&n)
			if run.Status != "completed" || run.Counts.Pending != 1 || run.Counts.Blocked != 0 || run.Counts.Failed != 0 || n != 0 {
				t.Fatalf("normal nonpublication reported a failure %+v articles=%d", run, n)
			}
		})
	}
}
func TestRolloutRealPublicationAndReadFailuresRemainNonzeroRuns(t *testing.T) {
	t.Run("public_url_missing", func(t *testing.T) {
		s, _, f := syncFixture(t)
		waitFixtureRun(t, s, "dry_run")
		enableFixtureSource(t, s, AllowedSources()[0], "m1")
		page := fixturePage(firstPage, AllowedSources()[0], "published")
		page.PublicURL = nil
		f.pages[firstPage] = page
		run := waitFixtureRun(t, s, "sync")
		if run.Status != "completed_with_errors" || run.Counts.Blocked != 1 || run.Counts.Pending != 0 {
			t.Fatal(run)
		}
	})
	t.Run("read_404", func(t *testing.T) {
		s, _, f, _, _ := managedFixture(t)
		delete(f.pages, firstPage)
		run := waitFixtureRun(t, s, "sync")
		if run.Status != "completed_with_errors" || run.Counts.Failed != 1 {
			t.Fatal(run)
		}
	})
}
func TestRolloutFrozenSourceIsNotRunFailure(t *testing.T) {
	s, db, f, a, before := managedFixture(t)
	db.Model(&model.NotionSyncSource{}).Where("source_id = ?", before.SourceID).Update("enabled", false)
	page := fixturePage(firstPage, before.SourceID, "draft")
	page.PublicURL = nil
	f.pages[firstPage] = page
	run := waitFixtureRun(t, s, "sync")
	if run.Status != "completed" || run.Counts.Frozen != 1 || run.Counts.Blocked != 0 || run.Counts.Failed != 0 {
		t.Fatal(run)
	}
	db.First(&a, a.ID)
	var after model.NotionPageBinding
	db.First(&after, "page_id = ?", firstPage)
	if a.Status != 2 || after.SourceID != before.SourceID || after.PublicURL == nil || after.NeedsRevalidation {
		t.Fatal(a, after)
	}
}
func TestRolloutRetentionPreservesCompletedPendingUntilReviewResolved(t *testing.T) {
	s, db, _ := syncFixture(t)
	stamp := time.Now().Add(-91 * 24 * time.Hour)
	for _, id := range []string{"pending-run", "resolved-run"} {
		if e := db.Create(&model.NotionSyncRun{ID: id, Mode: "dry_run", Status: "completed", StartedAt: stamp}).Error; e != nil {
			t.Fatal(e)
		}
	}
	for _, p := range []model.NotionPageBinding{{PageID: firstPage, ManagementState: model.NotionManagementBaselinePending}, {PageID: secondPage, ManagementState: model.NotionManagementManaged}} {
		if e := db.Create(&p).Error; e != nil {
			t.Fatal(e)
		}
	}
	for _, item := range []model.NotionSyncRunItem{{RunID: "pending-run", ItemID: firstPage, PageID: firstPage, Outcome: "baseline_pending"}, {RunID: "resolved-run", ItemID: secondPage, PageID: secondPage, Outcome: "baseline_pending"}} {
		if e := db.Create(&item).Error; e != nil {
			t.Fatal(e)
		}
	}
	token, ok, e := store.NewNotionSyncRepository(db).AcquireLease(context.Background(), "retention", time.Minute)
	if e != nil || !ok {
		t.Fatal(token, ok, e)
	}
	if e = s.pruneResolvedRuns(context.Background(), token); e != nil {
		t.Fatal(e)
	}
	var run model.NotionSyncRun
	if e = db.First(&run, "run_id = ?", "pending-run").Error; e != nil {
		t.Fatal("unresolved pending audit removed", e)
	}
	var count int64
	db.Model(&model.NotionSyncRunItem{}).Where("run_id = ?", "pending-run").Count(&count)
	if count != 1 {
		t.Fatal("pending item removed", count)
	}
	if e = db.First(&model.NotionSyncRun{}, "run_id = ?", "resolved-run").Error; !errors.Is(e, gorm.ErrRecordNotFound) {
		t.Fatal("resolved run not expired", e)
	}
	db.Model(&model.NotionPageBinding{}).Where("page_id = ?", firstPage).Update("management_state", model.NotionManagementManaged)
	if e = s.pruneResolvedRuns(context.Background(), token); e != nil {
		t.Fatal(e)
	}
	if e = db.First(&model.NotionSyncRun{}, "run_id = ?", "pending-run").Error; !errors.Is(e, gorm.ErrRecordNotFound) {
		t.Fatal("review resolved but audit could not expire", e)
	}
}
