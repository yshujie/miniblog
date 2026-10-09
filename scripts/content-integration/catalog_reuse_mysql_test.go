package contentintegration

import (
	"context"
	"reflect"
	"strconv"
	"testing"

	"github.com/yshujie/miniblog/internal/miniblog/biz/catalog"
	"github.com/yshujie/miniblog/internal/miniblog/model"
)

func reviewedInt(v int) *int { return &v }
func TestMySQLReviewedReuseLegacyDirectoryAndAtomicRejection(t *testing.T) {
	db, ds, r, id := rolloutMySQLFixture(t)
	beforeArticle, beforePage := rolloutLoadArticle(t, db, id), rolloutLoadPage(t, db)
	db.Model(&model.NotionSyncControl{}).Where("id=1").Updates(map[string]interface{}{"paused": true, "source_writes_paused": true})
	chapter := model.Section{Code: "concurrency&sync", Title: "并发&同步", ModuleCode: "m1", Sort: 73, Status: 2}
	if err := db.Create(&chapter).Error; err != nil {
		t.Fatal(err)
	}
	// Legacy status zero is preserved instead of being silently enabled by a model default.
	if err := db.Model(&chapter).Update("status", 0).Error; err != nil {
		t.Fatal(err)
	}
	chapter.Status = 0
	plan := []catalog.TopicReuseInput{{OptionID: "opaque%2Ftheme", ExpectedOptionName: "并发编程", SectionCode: chapter.Code, ExpectedSectionTitle: chapter.Title, ExpectedSectionStatus: reviewedInt(0), ExpectedSectionSort: reviewedInt(73)}}
	options := []catalog.TopicOption{{ID: "one", Name: "One"}, {ID: "opaque%2Ftheme", Name: "并发编程"}}
	service := catalog.New(ds)
	results, revision, err := service.PrepareSyncedTopicsWithReuse(context.Background(), r.Lease, rolloutSourceA, 1, "topic", options, plan)
	if err != nil || revision != 2 || results[1].SectionCode != chapter.Code || results[1].Outcome != "bound" {
		t.Fatal(results, revision, err)
	}
	db.First(&chapter, chapter.ID)
	if chapter.Title != "并发编程" || chapter.Status != 0 || chapter.Sort != 73 {
		t.Fatal(chapter)
	}
	if !reflect.DeepEqual(beforeArticle, rolloutLoadArticle(t, db, id)) || !reflect.DeepEqual(beforePage, rolloutLoadPage(t, db)) {
		t.Fatal("catalog preparation touched content")
	}
	plan[0].ExpectedSectionTitle = chapter.Title
	_, next, err := service.PrepareSyncedTopicsWithReuse(context.Background(), r.Lease, rolloutSourceA, revision, "topic", options, plan)
	if err != nil || next != revision {
		t.Fatal("no-op revision", next, err)
	}
	plan = append(plan, catalog.TopicReuseInput{OptionID: "other", ExpectedOptionName: "Other", SectionCode: chapter.Code, ExpectedSectionTitle: chapter.Title, ExpectedSectionStatus: reviewedInt(0), ExpectedSectionSort: reviewedInt(73)})
	options = append(options, catalog.TopicOption{ID: "other", Name: "Other"})
	_, _, err = service.PrepareSyncedTopicsWithReuse(context.Background(), r.Lease, rolloutSourceA, revision, "topic", options, plan)
	if err == nil {
		t.Fatal("two themes occupied one reviewed section")
	}
	var bindingCount int64
	db.Model(&model.NotionCatalogBinding{}).Where("option_id = ?", "other").Count(&bindingCount)
	if bindingCount != 0 {
		t.Fatal("rejected map left a binding")
	}
}
func TestMySQLReviewedReuseCanonicalOccupancy(t *testing.T) {
	db, ds, r, _ := rolloutMySQLFixture(t)
	db.Model(&model.NotionSyncControl{}).Where("id=1").Updates(map[string]interface{}{"paused": true, "source_writes_paused": true})
	var section model.Section
	db.First(&section, "code = ?", "s1")
	db.Model(&model.NotionCatalogBinding{}).Where("source_id = ?", rolloutSourceA).Update("section_code", "S1")
	input := catalog.TopicReuseInput{OptionID: "fresh", ExpectedOptionName: "Fresh", SectionCode: section.Code, ExpectedSectionTitle: section.Title, ExpectedSectionStatus: reviewedInt(section.Status), ExpectedSectionSort: reviewedInt(section.Sort)}
	_, _, err := catalog.New(ds).PrepareSyncedTopicsWithReuse(context.Background(), r.Lease, rolloutSourceA, 1, "topic", []catalog.TopicOption{{ID: "fresh", Name: "Fresh"}}, []catalog.TopicReuseInput{input})
	if err == nil {
		t.Fatal("case variant occupancy was ignored")
	}
	// Self alias resolves to the same canonical chapter and is reusable.
	input.OptionID = "one"
	input.ExpectedOptionName = "One"
	if _, _, err = catalog.New(ds).PrepareSyncedTopicsWithReuse(context.Background(), r.Lease, rolloutSourceA, 1, "topic", []catalog.TopicOption{{ID: "one", Name: "One"}}, []catalog.TopicReuseInput{input}); err != nil {
		t.Fatal("canonical self binding rejected", err)
	}
}
func TestMySQLReviewedActivationExactVisibilityRollback(t *testing.T) {
	db, ds, r, _ := rolloutMySQLFixture(t)
	db.Model(&model.NotionSyncControl{}).Where("id=1").Updates(map[string]interface{}{"paused": true, "source_writes_paused": true})
	var module model.Module
	db.First(&module, "code = ?", "m2")
	db.Model(&module).Update("status", 2)
	module.Status = 2
	article := model.Article{ID: 9007199254740993, Title: "Manual retained article", SectionCode: "s2", Status: 2, Pos: 71}
	if err := db.Create(&article).Error; err != nil {
		t.Fatal(err)
	}
	input := catalog.CatalogActivationInput{Items: []catalog.CatalogActivationItem{{Kind: "module", Code: module.Code, ExpectedTitle: module.Title, ExpectedStatus: reviewedInt(module.Status), ExpectedSort: reviewedInt(module.Sort)}}, ExpectedNewlyPublicArticleIDs: []string{}}
	service := catalog.New(ds)
	if _, err := service.ActivateReviewedCatalog(context.Background(), r.Lease, rolloutSourceB, 1, input); err == nil {
		t.Fatal("unapproved new visibility allowed")
	}
	db.First(&module, module.ID)
	if module.Status != 2 {
		t.Fatal("failed approval did not roll back", module)
	}
	input.ExpectedNewlyPublicArticleIDs = []string{strconv.FormatUint(article.ID, 10)}
	result, err := service.ActivateReviewedCatalog(context.Background(), r.Lease, rolloutSourceB, 1, input)
	if err != nil || result.ConfigRevision != 2 || len(result.NewlyPublicArticleIDs) != 1 || result.NewlyPublicArticleIDs[0] != "9007199254740993" {
		t.Fatal(result, err)
	}
}

func TestMySQLReviewedReuseSQLFailureRollsBackBindingRenameAndRevision(t *testing.T) {
	db, ds, r, _ := rolloutMySQLFixture(t)
	db.Model(&model.NotionSyncControl{}).Where("id=1").Updates(map[string]interface{}{"paused": true, "source_writes_paused": true})
	chapter := model.Section{Code: "reviewed_old", Title: "Old", ModuleCode: "m1", Status: 2, Sort: 91}
	if err := db.Create(&chapter).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("CREATE TRIGGER reviewed_section_fail BEFORE UPDATE ON section FOR EACH ROW SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'reviewed catalog failure'").Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Exec("DROP TRIGGER IF EXISTS reviewed_section_fail") })
	input := catalog.TopicReuseInput{OptionID: "reviewed", ExpectedOptionName: "New", SectionCode: chapter.Code, ExpectedSectionTitle: chapter.Title, ExpectedSectionStatus: reviewedInt(chapter.Status), ExpectedSectionSort: reviewedInt(chapter.Sort)}
	_, _, err := catalog.New(ds).PrepareSyncedTopicsWithReuse(context.Background(), r.Lease, rolloutSourceA, 1, "topic", []catalog.TopicOption{{ID: "reviewed", Name: "New"}}, []catalog.TopicReuseInput{input})
	if err == nil {
		t.Fatal("database failure swallowed")
	}
	db.First(&chapter, chapter.ID)
	var src model.NotionSyncSource
	db.First(&src, "source_id = ?", rolloutSourceA)
	var count int64
	db.Model(&model.NotionCatalogBinding{}).Where("option_id = ?", "reviewed").Count(&count)
	if chapter.Title != "Old" || chapter.Status != 2 || chapter.Sort != 91 || count != 0 || src.ConfigRevision != 1 {
		t.Fatal("partial reviewed change survived SQL rollback", chapter, count, src)
	}
}
