package article

import (
	"context"
	"strconv"
	"testing"

	"github.com/yshujie/miniblog/internal/miniblog/biz/catalog"
	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/store"
)

func TestReviewedCatalogActivationExactPublicDifferenceAndNoop(t *testing.T) {
	db, _, r := syncFixture(t)
	a := model.Article{ID: 9007199254740993, Title: "Approved public article", SectionCode: "s1", Status: 2, Pos: 42}
	if err := db.Create(&a).Error; err != nil {
		t.Fatal(err)
	}
	db.Model(&model.Section{}).Where("code = ?", "s1").Update("status", 2)
	db.Model(&model.NotionSyncControl{}).Where("id=1").Updates(map[string]interface{}{"paused": true, "source_writes_paused": true, "baseline_frozen": true})
	var chapter model.Section
	db.First(&chapter, "code = ?", "s1")
	input := catalog.CatalogActivationInput{Items: []catalog.CatalogActivationItem{{Kind: "section", Code: chapter.Code, ExpectedTitle: chapter.Title, ExpectedStatus: reuseInt(chapter.Status), ExpectedSort: reuseInt(chapter.Sort)}}, ExpectedNewlyPublicArticleIDs: []string{strconv.FormatUint(a.ID, 10)}}
	service := catalog.New(store.NewStore(db))
	result, err := service.ActivateReviewedCatalog(context.Background(), r.Lease, "a", 1, input)
	if err != nil || result.ConfigRevision != 2 || len(result.NewlyPublicArticleIDs) != 1 || result.NewlyPublicArticleIDs[0] != "9007199254740993" {
		t.Fatal(result, err)
	}
	var after model.Section
	db.First(&after, chapter.ID)
	if after.Status != 1 || after.Title != chapter.Title || after.Sort != chapter.Sort || after.Code != chapter.Code {
		t.Fatal(after)
	}
	input.Items[0].ExpectedStatus = reuseInt(1)
	input.ExpectedNewlyPublicArticleIDs = []string{}
	result, err = service.ActivateReviewedCatalog(context.Background(), r.Lease, "a", 2, input)
	if err != nil || result.ConfigRevision != 2 || len(result.NewlyPublicArticleIDs) != 0 {
		t.Fatal("no-op changed config", result, err)
	}
}
func TestReviewedCatalogActivationRejectsUnapprovedVisibilityAndDrift(t *testing.T) {
	for _, test := range []string{"extra_public", "missing_approval", "stale_title", "stale_status", "stale_sort", "foreign_module", "nil_approval", "duplicate_id", "invalid_bigint", "lease", "pause", "config"} {
		t.Run(test, func(t *testing.T) {
			db, _, r := syncFixture(t)
			a := model.Article{ID: 9007199254740993, Title: "Article", SectionCode: "s1", Status: 2}
			if err := db.Create(&a).Error; err != nil {
				t.Fatal(err)
			}
			db.Model(&model.Section{}).Where("code = ?", "s1").Update("status", 2)
			db.Model(&model.NotionSyncControl{}).Where("id=1").Updates(map[string]interface{}{"paused": true, "source_writes_paused": true, "baseline_frozen": true})
			var chapter model.Section
			db.First(&chapter, "code = ?", "s1")
			input := catalog.CatalogActivationInput{Items: []catalog.CatalogActivationItem{{Kind: "section", Code: chapter.Code, ExpectedTitle: chapter.Title, ExpectedStatus: reuseInt(chapter.Status), ExpectedSort: reuseInt(chapter.Sort)}}, ExpectedNewlyPublicArticleIDs: []string{"9007199254740993"}}
			revision := uint64(1)
			switch test {
			case "extra_public":
				extra := model.Article{ID: a.ID + 1, Title: "Unreviewed", SectionCode: "s1", Status: 2}
				db.Create(&extra)
			case "missing_approval":
				input.ExpectedNewlyPublicArticleIDs = []string{}
			case "stale_title":
				input.Items[0].ExpectedTitle = "Stale"
			case "stale_status":
				input.Items[0].ExpectedStatus = reuseInt(0)
			case "stale_sort":
				input.Items[0].ExpectedSort = reuseInt(999)
			case "foreign_module":
				input.Items[0].Code = "s2"
			case "nil_approval":
				input.ExpectedNewlyPublicArticleIDs = nil
			case "duplicate_id":
				input.ExpectedNewlyPublicArticleIDs = append(input.ExpectedNewlyPublicArticleIDs, input.ExpectedNewlyPublicArticleIDs[0])
			case "invalid_bigint":
				input.ExpectedNewlyPublicArticleIDs = []string{"9223372036854775808"}
			case "lease":
				r.Lease.Epoch++
			case "pause":
				db.Model(&model.NotionSyncControl{}).Where("id=1").Update("paused", false)
			case "config":
				revision++
			}
			_, err := catalog.New(store.NewStore(db)).ActivateReviewedCatalog(context.Background(), r.Lease, "a", revision, input)
			if err == nil {
				t.Fatal("guard accepted", test)
			}
			db.First(&chapter, chapter.ID)
			var src model.NotionSyncSource
			db.First(&src, "source_id = ?", "a")
			if chapter.Status != 2 || src.ConfigRevision != 1 {
				t.Fatal("rejected activation partially wrote", chapter, src)
			}
		})
	}
}
