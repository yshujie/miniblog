package article

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/source"
	"gorm.io/gorm"
)

func requireRawSubsection(t *testing.T, db *gorm.DB, id uint64, want sql.NullString) {
	t.Helper()
	var got sql.NullString
	if err := db.Raw("SELECT subsection_code FROM article WHERE id = ?", id).Row().Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("physical subsection changed: got=%+v want=%+v", got, want)
	}
}

func TestSyncProjectionPreservesPhysicalPlacement(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value any
		want  sql.NullString
	}{
		{"null", nil, sql.NullString{}}, {"empty", "", sql.NullString{Valid: true}}, {"subsection", "sub1", sql.NullString{String: "sub1", Valid: true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, b, r := syncFixture(t)
			first := syncApply(t, db, b, r)
			if err := db.Exec("UPDATE article SET subsection_code = ?, content = NULL, pos = 29 WHERE id = ?", tc.value, first.ArticleID).Error; err != nil {
				t.Fatal(err)
			}
			before := syncArticle(t, db, first.ArticleID)
			requireRawSubsection(t, db, first.ArticleID, tc.want)
			for i, state := range []int{2, 1, 4, 1, 2, 3, 2} {
				r.DesiredState = state
				r.Title = fmt.Sprintf("Changed title %d", i)
				r.Tags = []string{fmt.Sprintf("tag %d", i), "comma,tag"}
				out := syncApply(t, db, b, r)
				requireRawSubsection(t, db, first.ArticleID, tc.want)
				after := syncArticle(t, db, first.ArticleID)
				if out.ArticleID != before.ID || after.SectionCode != before.SectionCode || after.Pos != before.Pos || after.Status != state || after.Author != before.Author || after.ExternalLink != before.ExternalLink || !after.CreatedAt.Equal(before.CreatedAt) {
					t.Fatal("sync changed local fields or placement")
				}
				var content sql.NullString
				if err := db.Raw("SELECT content FROM article WHERE id = ?", first.ArticleID).Row().Scan(&content); err != nil {
					t.Fatal(err)
				}
				if content.Valid {
					t.Fatal("sync normalized local NULL content")
				}
			}
			beforeNoop := syncArticle(t, db, first.ArticleID)
			out := syncApply(t, db, b, r)
			afterNoop := syncArticle(t, db, first.ArticleID)
			if out.Outcome != "unchanged" || !afterNoop.UpdatedAt.Equal(beforeNoop.UpdatedAt) {
				t.Fatal("no-op sync changed update time", out)
			}
			requireRawSubsection(t, db, first.ArticleID, tc.want)
			r.SourceID = "b"
			r.DataSourceID = syncDataSourceB
			r.ThemeOptionID = "o2"
			r.ThemeOptionName = "S2"
			moved := syncApply(t, db, b, r)
			requireRawSubsection(t, db, moved.ArticleID, sql.NullString{Valid: true})
			afterMove := syncArticle(t, db, moved.ArticleID)
			if afterMove.ID != before.ID || afterMove.SectionCode != "s2" || afterMove.Pos == before.Pos {
				t.Fatal("real move did not update placement", afterMove)
			}
		})
	}
}

func TestSyncStateOnlyPreservesPhysicalNull(t *testing.T) {
	for _, reason := range []string{"invalid_field", "field_validation", "address_conflict"} {
		t.Run(reason, func(t *testing.T) {
			db, b, r := syncFixture(t)
			first := syncApply(t, db, b, r)
			if err := db.Exec("UPDATE article SET subsection_code = NULL, content = NULL WHERE id = ?", first.ArticleID).Error; err != nil {
				t.Fatal(err)
			}
			before := syncArticle(t, db, first.ArticleID)
			r.DesiredState = model.ArticleStatusDraft
			switch reason {
			case "invalid_field":
				r.MetadataComplete = false
				r.MetadataError = "invalid_field"
			case "field_validation":
				r.Title = ""
			case "address_conflict":
				other := model.Article{Title: "Other owner", Author: "Other author", ExternalLink: "https://notion.site/owned-slug", SectionCode: "s2", Status: 1}
				parsed, err := source.Parse(other.ExternalLink)
				if err != nil {
					t.Fatal(err)
				}
				bindIdentity(&other, parsed)
				if err := db.Create(&other).Error; err != nil {
					t.Fatal(err)
				}
				r.PublicURL = syncString(other.ExternalLink)
				r.PublicURLObserved = true
			}
			out := syncApply(t, db, b, r)
			if out.Outcome != "state_only" {
				t.Fatal(out)
			}
			requireRawSubsection(t, db, first.ArticleID, sql.NullString{})
			after := syncArticle(t, db, first.ArticleID)
			if after.Status != model.ArticleStatusDraft || after.Title != before.Title || after.Pos != before.Pos {
				t.Fatal("state-only sync changed unrelated fields")
			}
			var content sql.NullString
			if err := db.Raw("SELECT content FROM article WHERE id = ?", first.ArticleID).Row().Scan(&content); err != nil {
				t.Fatal(err)
			}
			if content.Valid {
				t.Fatal("state-only sync normalized local content")
			}
		})
	}
}

func TestSyncIsolationPreservesPhysicalNull(t *testing.T) {
	db, b, r, id := isolationFixture(t)
	if err := db.Exec("UPDATE article SET subsection_code = NULL, content = NULL WHERE id = ?", id).Error; err != nil {
		t.Fatal(err)
	}
	r.DesiredState = model.ArticleStatusDraft
	out, err := b.IsolateSyncedTarget(context.Background(), r)
	if err != nil || out.AppliedState != model.ArticleStatusDraft {
		t.Fatal(out, err)
	}
	requireRawSubsection(t, db, id, sql.NullString{})
	var content sql.NullString
	if err := db.Raw("SELECT content FROM article WHERE id = ?", id).Row().Scan(&content); err != nil {
		t.Fatal(err)
	}
	if content.Valid {
		t.Fatal("isolation normalized local content")
	}
}

func TestSyncProjectionPhysicalNullRollsBackWithBinding(t *testing.T) {
	db, b, r := syncFixture(t)
	first := syncApply(t, db, b, r)
	if err := db.Exec("UPDATE article SET subsection_code = NULL WHERE id = ?", first.ArticleID).Error; err != nil {
		t.Fatal(err)
	}
	before := syncArticle(t, db, first.ArticleID)
	if err := db.Exec("CREATE TRIGGER fail_projection_binding BEFORE UPDATE ON notion_page_bindings BEGIN SELECT RAISE(FAIL, 'fixture binding failure'); END").Error; err != nil {
		t.Fatal(err)
	}
	r.Title = "Must roll back"
	r.DesiredState = model.ArticleStatusDraft
	r.ExpectedBindingRevision = syncBinding(t, db).Revision
	if _, err := b.ApplySyncedSource(context.Background(), r); err == nil {
		t.Fatal("binding failure committed projection")
	}
	after := syncArticle(t, db, first.ArticleID)
	if !sameArticleValues(&after, &before) || !after.UpdatedAt.Equal(before.UpdatedAt) {
		t.Fatal("partial projection escaped transaction rollback")
	}
	requireRawSubsection(t, db, first.ArticleID, sql.NullString{})
}
