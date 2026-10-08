package notionsync

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/source"
	"github.com/yshujie/miniblog/internal/miniblog/store"
)

type paginationQueryKey struct {
	sourceID string
	archived bool
	cursor   string
}

// Decode the actual list envelope so an omitted request_status cannot be
// confused with the legacy fixtures' explicit complete status.
type paginationWireClient struct {
	*fixtureNotion
	mu        sync.Mutex
	responses map[paginationQueryKey][]byte
	empty     []byte
	queries   []paginationQueryKey
	reads     []string
}

func newPaginationWireClient(t *testing.T, fixture *fixtureNotion) *paginationWireClient {
	t.Helper()
	return &paginationWireClient{
		fixtureNotion: fixture,
		responses:     map[paginationQueryKey][]byte{},
		empty:         paginationWireJSON(t, nil, false, nil, ""),
	}
}

func paginationWireJSON(t *testing.T, pages []source.NotionPage, more bool, cursor *string, status string) []byte {
	t.Helper()
	if pages == nil {
		pages = []source.NotionPage{}
	}
	envelope := map[string]interface{}{
		"object":              "list",
		"type":                "page_or_data_source",
		"page_or_data_source": map[string]interface{}{},
		"results":             pages,
		"has_more":            more,
		"next_cursor":         cursor,
	}
	if status != "" {
		envelope["request_status"] = map[string]string{"type": status}
	}
	raw, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func (f *paginationWireClient) QueryDataSource(_ context.Context, id string, archived bool, cursor string) (source.NotionQueryResult, error) {
	key := paginationQueryKey{id, archived, cursor}
	f.mu.Lock()
	f.queries = append(f.queries, key)
	raw, ok := f.responses[key]
	f.mu.Unlock()
	if !ok {
		if cursor != "" {
			return source.NotionQueryResult{}, fmt.Errorf("unexpected fixture cursor %q", cursor)
		}
		raw = f.empty
	}
	var result source.NotionQueryResult
	err := json.Unmarshal(raw, &result)
	return result, err
}

func (f *paginationWireClient) RetrievePage(ctx context.Context, id string) (source.NotionPage, error) {
	f.mu.Lock()
	f.reads = append(f.reads, normalizePageID(id))
	f.mu.Unlock()
	return f.fixtureNotion.RetrievePage(ctx, id)
}

func (f *paginationWireClient) partitionQueries(id string, archived bool) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	count := 0
	for _, query := range f.queries {
		if query.sourceID == id && query.archived == archived {
			count++
		}
	}
	return count
}

func (f *paginationWireClient) pageReads() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.reads...)
}

func scanPaginationSource(t *testing.T, s *Service, id string) ([]source.NotionPage, bool, error) {
	t.Helper()
	ctx := context.Background()
	repo := store.NewNotionSyncRepository(s.ds.DB())
	token, acquired, err := repo.AcquireLease(ctx, "pagination fixture", time.Minute)
	if err != nil || !acquired {
		t.Fatalf("acquire pagination lease: acquired=%v error=%v", acquired, err)
	}
	defer func() {
		if err := repo.ReleaseLease(ctx, token); err != nil {
			t.Errorf("release pagination lease: %v", err)
		}
	}()
	_, pages, complete, err := s.scanSource(ctx, token, id, false)
	return pages, complete, err
}

func TestQueryPaginationMissingStatusAcrossPartitions(t *testing.T) {
	s, _, fixture := syncFixture(t)
	wire := newPaginationWireClient(t, fixture)
	s.client = wire
	id := AllowedSources()[0]
	next := "shared-next-cursor"
	for _, archived := range []bool{false, true} {
		pageID := firstPage
		if archived {
			pageID = secondPage
		}
		page := fixturePage(pageID, id, "published")
		page.IsArchived = archived
		wire.responses[paginationQueryKey{id, archived, ""}] = paginationWireJSON(t, []source.NotionPage{page}, true, &next, "")
		wire.responses[paginationQueryKey{id, archived, next}] = paginationWireJSON(t, []source.NotionPage{page}, false, nil, "")
	}
	pages, complete, err := scanPaginationSource(t, s, id)
	if err != nil || !complete || len(pages) != 4 {
		t.Fatalf("missing status rejected valid multipage partitions: complete=%v pages=%d error=%v", complete, len(pages), err)
	}
	for _, archived := range []bool{false, true} {
		if calls := wire.partitionQueries(id, archived); calls != 2 {
			t.Fatalf("archived=%v queries=%d, want 2", archived, calls)
		}
	}
}

func TestQueryPaginationEmptyMissingStatusFreezesBaseline(t *testing.T) {
	s, db, fixture := syncFixture(t)
	wire := newPaginationWireClient(t, fixture)
	s.client = wire
	run := waitFixtureRun(t, s, "dry_run")
	var control model.NotionSyncControl
	if err := db.First(&control, 1).Error; err != nil {
		t.Fatal(err)
	}
	var articles int64
	if err := db.Model(&model.Article{}).Count(&articles).Error; err != nil {
		t.Fatal(err)
	}
	queries := 0
	for _, id := range AllowedSources() {
		for _, archived := range []bool{false, true} {
			calls := wire.partitionQueries(id, archived)
			if calls != 1 {
				t.Fatalf("source=%s archived=%v queries=%d, want 1", id, archived, calls)
			}
			queries += calls
		}
	}
	if run.Status != "completed" || queries != 10 || !control.BaselineFrozen || control.BaselineAt == nil || control.LastCompleteScanAt == nil || articles != 0 || fixture.writes != 0 || len(wire.pageReads()) != 0 {
		t.Fatalf("valid empty wire union did not freeze baseline cleanly: run=%+v queries=%d control=%+v articles=%d writes=%d reads=%v", run, queries, control, articles, fixture.writes, wire.pageReads())
	}
}

func addPaginationPartition(t *testing.T, wire *paginationWireClient, id string, archived bool, count int) {
	t.Helper()
	pageID := firstPage
	if archived {
		pageID = secondPage
	}
	page := fixturePage(pageID, id, "published")
	page.IsArchived = archived
	cursor := ""
	for offset := 0; offset < count; {
		batchSize := min(100, count-offset)
		pages := make([]source.NotionPage, batchSize)
		for i := range pages {
			pages[i] = page // Repeated IDs still count toward the wire result limit.
		}
		offset += batchSize
		var next *string
		if offset < count {
			value := fmt.Sprintf("after-%d", offset)
			next = &value
		}
		wire.responses[paginationQueryKey{id, archived, cursor}] = paginationWireJSON(t, pages, next != nil, next, "")
		if next != nil {
			cursor = *next
		}
	}
}

func TestQueryPaginationCountsRawResultsPerPartition(t *testing.T) {
	for _, tc := range []struct {
		name     string
		active   int
		archived int
		complete bool
	}{
		{name: "9999 duplicate results", active: 9999, complete: true},
		{name: "10000 duplicate results", active: 10000, complete: false},
		{name: "6000 in each partition", active: 6000, archived: 6000, complete: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, _, fixture := syncFixture(t)
			wire := newPaginationWireClient(t, fixture)
			s.client = wire
			id := AllowedSources()[0]
			addPaginationPartition(t, wire, id, false, tc.active)
			addPaginationPartition(t, wire, id, true, tc.archived)
			pages, complete, err := scanPaginationSource(t, s, id)
			if complete != tc.complete || (err == nil) != tc.complete || len(pages) != tc.active+tc.archived {
				t.Fatalf("raw partition counts active=%d archived=%d: complete=%v pages=%d error=%v", tc.active, tc.archived, complete, len(pages), err)
			}
			for _, partition := range []struct {
				archived bool
				count    int
			}{{false, tc.active}, {true, tc.archived}} {
				want := max(1, (partition.count+99)/100)
				if calls := wire.partitionQueries(id, partition.archived); calls != want {
					t.Fatalf("archived=%v queries=%d, want %d", partition.archived, calls, want)
				}
			}
		})
	}
}

func TestQueryPaginationRejectsIncompleteChains(t *testing.T) {
	type step struct {
		cursor string
		more   bool
		next   *string
		status string
	}
	a, b, empty := "cursor-a", "cursor-b", ""
	for _, tc := range []struct {
		name      string
		steps     []step
		keptPages int
	}{
		{"middle page incomplete", []step{{more: true, next: &a}, {cursor: a, more: true, next: &b, status: "incomplete"}}, 2},
		{"repeated cursor", []step{{more: true, next: &a}, {cursor: a, more: true, next: &a}}, 1},
		{"cursor cycle", []step{{more: true, next: &a}, {cursor: a, more: true, next: &b}, {cursor: b, more: true, next: &a}}, 2},
		{"terminal cursor", []step{{more: false, next: &a}}, 0},
		{"missing next cursor", []step{{more: true}}, 0},
		{"empty next cursor", []step{{more: true, next: &empty}}, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, _, fixture := syncFixture(t)
			wire := newPaginationWireClient(t, fixture)
			s.client = wire
			id := AllowedSources()[0]
			page := fixturePage(firstPage, id, "published")
			for _, step := range tc.steps {
				wire.responses[paginationQueryKey{id, false, step.cursor}] = paginationWireJSON(t, []source.NotionPage{page}, step.more, step.next, step.status)
			}
			pages, complete, err := scanPaginationSource(t, s, id)
			if complete || err == nil || len(pages) != tc.keptPages {
				t.Fatalf("invalid chain accepted or lost safe prior pages: complete=%v pages=%d want=%d error=%v", complete, len(pages), tc.keptPages, err)
			}
			if calls := wire.partitionQueries(id, false); calls != len(tc.steps) {
				t.Fatalf("queries=%d, want stop after %d", calls, len(tc.steps))
			}
			if calls := wire.partitionQueries(id, true); calls != 1 {
				t.Fatalf("other partition queries=%d, want 1", calls)
			}
		})
	}
}

func TestQueryPaginationGlobalFailureDoesNotFreezeBaseline(t *testing.T) {
	s, db, fixture := syncFixture(t)
	wire := newPaginationWireClient(t, fixture)
	s.client = wire
	id := AllowedSources()[0]
	next := "middle-incomplete"
	wire.responses[paginationQueryKey{id, false, ""}] = paginationWireJSON(t, []source.NotionPage{fixturePage(firstPage, id, "published")}, true, &next, "")
	wire.responses[paginationQueryKey{id, false, next}] = paginationWireJSON(t, []source.NotionPage{fixturePage(secondPage, id, "published")}, false, nil, "incomplete")
	run := waitFixtureRun(t, s, "dry_run")
	var control model.NotionSyncControl
	if err := db.First(&control, 1).Error; err != nil {
		t.Fatal(err)
	}
	var pending, articles int64
	if err := db.Model(&model.NotionPageBinding{}).Where("management_state = ?", model.NotionManagementBaselinePending).Count(&pending).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.Article{}).Count(&articles).Error; err != nil {
		t.Fatal(err)
	}
	if run.Status != "failed" || control.BaselineFrozen || control.BaselineAt != nil || control.LastCompleteScanAt != nil || pending != 2 || articles != 0 {
		t.Fatalf("incomplete union froze baseline or discarded partial pages: run=%+v control=%+v pending=%d articles=%d", run, control, pending, articles)
	}
}

func TestQueryPaginationGlobalFailureDoesNotRetrieveAbsentManagedPage(t *testing.T) {
	s, db, fixture, article, before := managedFixture(t)
	wire := newPaginationWireClient(t, fixture)
	s.client = wire
	failedID := AllowedSources()[1]
	wire.responses[paginationQueryKey{failedID, false, ""}] = paginationWireJSON(t, nil, false, nil, "incomplete")
	var initialControl model.NotionSyncControl
	if err := db.First(&initialControl, 1).Error; err != nil {
		t.Fatal(err)
	}
	run := waitFixtureRun(t, s, "sync")
	var after model.NotionPageBinding
	if err := db.Where("page_id = ?", firstPage).First(&after).Error; err != nil {
		t.Fatal(err)
	}
	var current model.Article
	if err := db.First(&current, "id = ?", article.ID).Error; err != nil {
		t.Fatal(err)
	}
	var control model.NotionSyncControl
	if err := db.First(&control, 1).Error; err != nil {
		t.Fatal(err)
	}
	if run.Status != "failed" || len(wire.pageReads()) != 0 || after.Revision != before.Revision || after.SnapshotJSON != before.SnapshotJSON || after.AppliedHash != before.AppliedHash || current.Title != article.Title || current.Status != article.Status {
		t.Fatalf("incomplete union treated absence as a managed-page read/update: run=%+v reads=%v before=%+v after=%+v article=%+v", run, wire.pageReads(), before, after, current)
	}
	if !control.BaselineFrozen || control.LastCompleteScanAt == nil || initialControl.LastCompleteScanAt == nil || !control.LastCompleteScanAt.Equal(*initialControl.LastCompleteScanAt) {
		t.Fatalf("incomplete union advanced complete-scan evidence: before=%+v after=%+v", initialControl, control)
	}
}

func TestQueryPaginationPartialValidPageStillUsesFreshApply(t *testing.T) {
	s, db, fixture, article, before := managedFixture(t)
	wire := newPaginationWireClient(t, fixture)
	s.client = wire
	id := AllowedSources()[0]
	queryPage := fixturePage(firstPage, id, "published")
	queryTitle := queryPage.Properties["标题"]
	queryTitle.Title = []source.NotionRichText{{PlainText: "Cached query title"}}
	queryPage.Properties["标题"] = queryTitle
	queryPage.LastEditedTime = queryPage.LastEditedTime.Add(time.Hour)
	freshPage := fixturePage(firstPage, id, "published")
	freshTitle := freshPage.Properties["标题"]
	freshTitle.Title = []source.NotionRichText{{PlainText: "Fresh authoritative title"}}
	freshPage.Properties["标题"] = freshTitle
	freshPage.LastEditedTime = queryPage.LastEditedTime.Add(time.Hour)
	fixture.pages[firstPage] = freshPage
	next := "incomplete-after-valid-page"
	wire.responses[paginationQueryKey{id, false, ""}] = paginationWireJSON(t, []source.NotionPage{queryPage}, true, &next, "")
	wire.responses[paginationQueryKey{id, false, next}] = paginationWireJSON(t, nil, false, nil, "incomplete")
	run := waitFixtureRun(t, s, "sync")
	var current model.Article
	if err := db.First(&current, "id = ?", article.ID).Error; err != nil {
		t.Fatal(err)
	}
	var after model.NotionPageBinding
	if err := db.Where("page_id = ?", firstPage).First(&after).Error; err != nil {
		t.Fatal(err)
	}
	var snap Snapshot
	if err := json.Unmarshal([]byte(after.SnapshotJSON), &snap); err != nil {
		t.Fatal(err)
	}
	reads := wire.pageReads()
	if run.Status != "failed" || run.Counts.Updated != 1 || len(reads) != 1 || reads[0] != firstPage || current.Title != "Fresh authoritative title" || snap.Title != current.Title || after.Revision <= before.Revision || after.AppliedHash != metadataHash(snap) {
		t.Fatalf("partial valid page did not apply a fresh read: run=%+v reads=%v article=%+v binding=%+v snapshot=%+v", run, reads, current, after, snap)
	}
	if current.Content != article.Content || current.Author != article.Author || current.Pos != article.Pos || current.Status != article.Status {
		t.Fatalf("partial fresh apply changed locally owned fields: before=%+v after=%+v", article, current)
	}
}
