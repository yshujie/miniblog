package notionsync

import (
	"context"
	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/source"
	"reflect"
	"testing"
	"time"
)

type interruptedJournalWriter struct{ fixture *fixtureNotion }

func (w interruptedJournalWriter) UpdateBlogState(ctx context.Context, id, property, option string) error {
	e := w.fixture.UpdateBlogState(ctx, id, property, option)
	w.fixture.mu.Lock()
	w.fixture.getError[normalizePageID(id)] = &source.NotionAPIError{StatusCode: 503, Code: "unknown_write_result"}
	w.fixture.mu.Unlock()
	return e
}
func TestRolloutJournalScanCannotAuthorizeUnreviewedMetadata(t *testing.T) {
	for _, stage := range []string{"write_requested", "verified"} {
		for _, mode := range []string{"dry_run", "sync"} {
			for _, field := range []string{"title", "tags"} {
				t.Run(stage+"/"+mode+"/"+field, func(t *testing.T) {
					s, db, f := syncFixture(t)
					sourceID := AllowedSources()[0]
					if _, e := s.UpdateSource(context.Background(), sourceID, SourceInput{ModuleCode: "m1"}); e != nil {
						t.Fatal(e)
					}
					f.pages[firstPage] = fixturePage(firstPage, sourceID, "")
					preview, e := s.BootstrapPreview(context.Background())
					if e != nil || len(preview.Items) != 1 {
						t.Fatal(preview, e)
					}
					confirm := BootstrapConfirm{PageID: firstPage, NewPage: true, ExpectedState: "draft", ExpectedFingerprint: preview.Items[0].ExpectedFingerprint, ConfirmedBy: "fixture reviewer"}
					yes := true
					if _, e = s.UpdateControl(context.Background(), ControlInput{Paused: &yes, SourceWritesPaused: &yes}); e != nil {
						t.Fatal(e)
					}
					// PATCH changed remote state, but the read-back failed: ownership is still pending.
					interrupted, e := s.BootstrapApply(context.Background(), BootstrapInput{Confirmations: []BootstrapConfirm{confirm}}, interruptedJournalWriter{fixture: f})
					if e != nil || interrupted.Items[0].Outcome != "failed" || f.writes != 1 {
						t.Fatal(interrupted, e, f.writes)
					}
					delete(f.getError, firstPage)
					var before model.NotionPageBinding
					db.First(&before, "page_id = ?", firstPage)
					if before.BootstrapState != "write_requested" || before.ManagementState != model.NotionManagementBaselinePending {
						t.Fatal(before)
					}
					if stage == "verified" {
						// Represent the persisted read-back checkpoint before a failed final adoption.
						var src model.NotionSyncSource
						db.First(&src, "source_id = ?", sourceID)
						verified, parseErr := snapshotOf(f.pages[firstPage], sourceID, configOf(src))
						if parseErr != nil {
							t.Fatal(parseErr)
						}
						if e = db.Model(&before).Updates(map[string]interface{}{"bootstrap_state": "verified", "snapshot_json": jsonText(verified), "metadata_hash": metadataHash(verified), "desired_state": verified.DesiredState}).Error; e != nil {
							t.Fatal(e)
						}
						db.First(&before, "page_id = ?", firstPage)
					}
					page := f.pages[firstPage]
					if field == "title" {
						p := page.Properties["标题"]
						p.Title = []source.NotionRichText{{PlainText: "Unreviewed new title"}}
						page.Properties["标题"] = p
					} else {
						p := page.Properties["知识点"]
						p.MultiSelect = []source.NotionOption{{ID: "new", Name: "Unreviewed,tag"}}
						page.Properties["知识点"] = p
					}
					page.LastEditedTime = page.LastEditedTime.Add(time.Hour)
					f.pages[firstPage] = page
					if mode == "sync" {
						no := false
						if _, e = s.UpdateControl(context.Background(), ControlInput{Paused: &no, SourceWritesPaused: &no}); e != nil {
							t.Fatal(e)
						}
					}
					waitFixtureRun(t, s, mode)
					var after model.NotionPageBinding
					db.First(&after, "page_id = ?", firstPage)
					journalChanged := after.SnapshotJSON != before.SnapshotJSON || after.MetadataHash != before.MetadataHash || after.BootstrapExpectedFingerprint != before.BootstrapExpectedFingerprint || after.LocalBeforeJSON != before.LocalBeforeJSON || after.BootstrapState != before.BootstrapState || after.SourceID != before.SourceID || after.DesiredState != before.DesiredState || !reflect.DeepEqual(after.PublicURL, before.PublicURL) || !reflect.DeepEqual(after.NotionLastEditedAt, before.NotionLastEditedAt)
					if _, e = s.UpdateControl(context.Background(), ControlInput{Paused: &yes, SourceWritesPaused: &yes}); e != nil {
						t.Fatal(e)
					}
					retry, e := s.BootstrapApply(context.Background(), BootstrapInput{Confirmations: []BootstrapConfirm{confirm}}, f)
					if e != nil || retry.Items[0].Outcome != "failed" || f.writes != 1 {
						t.Fatalf("old review authorized changed metadata %+v %v writes=%d", retry, e, f.writes)
					}
					if journalChanged {
						t.Fatal("ordinary scan replaced reviewed journal", before, after)
					}
					db.First(&after, "page_id = ?", firstPage)
					if after.ManagementState != model.NotionManagementBaselinePending {
						t.Fatal("old confirmation adopted unreviewed page", after)
					}
				})
			}
		}
	}
}
func TestRolloutPendingWithoutJournalStillRefreshesOrdinaryPreview(t *testing.T) {
	s, db, f := syncFixture(t)
	sourceID := AllowedSources()[0]
	f.pages[firstPage] = fixturePage(firstPage, sourceID, "")
	waitFixtureRun(t, s, "dry_run")
	var before model.NotionPageBinding
	db.First(&before, "page_id = ?", firstPage)
	page := f.pages[firstPage]
	p := page.Properties["标题"]
	p.Title = []source.NotionRichText{{PlainText: "Updated before review"}}
	page.Properties["标题"] = p
	f.pages[firstPage] = page
	waitFixtureRun(t, s, "dry_run")
	var after model.NotionPageBinding
	db.First(&after, "page_id = ?", firstPage)
	if after.ManagementState != model.NotionManagementBaselinePending || after.BootstrapState != "pending" || after.SnapshotJSON == before.SnapshotJSON || after.MetadataHash == before.MetadataHash {
		t.Fatal("unstarted pending review failed to refresh", before, after)
	}
}

func TestRolloutBootstrapUnknownNonemptyStateCannotBeOverwritten(t *testing.T) {
	s, db, f := syncFixture(t)
	sourceID := AllowedSources()[0]
	if _, err := s.UpdateSource(context.Background(), sourceID, SourceInput{ModuleCode: "m1"}); err != nil {
		t.Fatal(err)
	}
	const unknownOption = "unknown-nonempty-option"
	f.pages[firstPage] = fixturePage(firstPage, sourceID, unknownOption)
	preview, err := s.BootstrapPreview(context.Background())
	if err != nil || len(preview.Items) != 1 {
		t.Fatal(preview, err)
	}
	candidate := preview.Items[0]
	if candidate.Reason != "unknown_state" {
		t.Errorf("nonempty unknown state lacks the review warning: %q", candidate.Reason)
	}
	yes := true
	if _, err = s.UpdateControl(context.Background(), ControlInput{Paused: &yes, SourceWritesPaused: &yes}); err != nil {
		t.Fatal(err)
	}
	confirm := BootstrapConfirm{PageID: firstPage, NewPage: true, ExpectedState: "draft", ExpectedFingerprint: candidate.ExpectedFingerprint, ConfirmedBy: "fixture reviewer"}
	result, err := s.BootstrapApply(context.Background(), BootstrapInput{Confirmations: []BootstrapConfirm{confirm}}, f)
	if err != nil || len(result.Items) != 1 || result.Items[0].Outcome != "failed" || f.writes != 0 {
		t.Fatalf("unknown nonempty state was overwritten: %+v err=%v writes=%d", result, err, f.writes)
	}
	var binding model.NotionPageBinding
	if err = db.First(&binding, "page_id = ?", firstPage).Error; err != nil {
		t.Fatal(err)
	}
	state := f.pages[firstPage].Properties["博客状态"].Select
	if state == nil || state.ID != unknownOption || binding.ManagementState != model.NotionManagementBaselinePending || binding.BootstrapState != "pending" {
		t.Fatal("rejected unknown state changed remote state or ownership", state, binding)
	}
}
