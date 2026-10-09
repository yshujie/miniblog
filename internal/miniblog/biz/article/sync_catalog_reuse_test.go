package article

import (
	"context"
	"reflect"
	"testing"

	"github.com/yshujie/miniblog/internal/miniblog/biz/catalog"
	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/store"
)

func reuseInt(v int) *int { return &v }
func reviewedReuse(code, title string, status, sort int) catalog.TopicReuseInput {
	return catalog.TopicReuseInput{OptionID: "opaque/topic%2F", ExpectedOptionName: "并发编程", SectionCode: code, ExpectedSectionTitle: title, ExpectedSectionStatus: reuseInt(status), ExpectedSectionSort: reuseInt(sort)}
}
func TestReviewedTopicReusePreservesLegacyAndNoop(t *testing.T) {
	db, b, r := syncFixture(t)
	first := syncApply(t, db, b, r)
	beforeArticle, beforePage := syncArticle(t, db, first.ArticleID), syncBinding(t, db)
	chapter := model.Section{Code: "concurrency&sync", Title: "并发&同步", ModuleCode: "m1", Status: 2, Sort: 91}
	if err := db.Create(&chapter).Error; err != nil {
		t.Fatal(err)
	}
	db.Model(&model.NotionSyncControl{}).Where("id = 1").Updates(map[string]interface{}{"paused": true, "source_writes_paused": true, "baseline_frozen": true})
	db.Model(&model.NotionSyncSource{}).Where("source_id = ?", "a").Update("enabled", false)
	options := []catalog.TopicOption{{ID: "o1", Name: "S1"}, {ID: "opaque/topic%2F", Name: "并发编程"}}
	plan := []catalog.TopicReuseInput{reviewedReuse(chapter.Code, chapter.Title, chapter.Status, chapter.Sort)}
	service := catalog.New(store.NewStore(db))
	results, revision, err := service.PrepareSyncedTopicsWithReuse(context.Background(), r.Lease, "a", 1, "topic", options, plan)
	if err != nil || revision != 2 || len(results) != 3 || results[1].SectionCode != chapter.Code || results[1].Outcome != "bound" {
		t.Fatalf("%+v rev=%d err=%v", results, revision, err)
	}
	var after model.Section
	if err := db.First(&after, chapter.ID).Error; err != nil {
		t.Fatal(err)
	}
	if after.Title != "并发编程" || after.Code != chapter.Code || after.Sort != 91 || after.Status != 2 {
		t.Fatal(after)
	}
	var count int64
	db.Model(&model.Section{}).Count(&count)
	if count != 4 {
		t.Fatal("duplicate legacy chapter", count)
	}
	if !reflect.DeepEqual(beforeArticle, syncArticle(t, db, first.ArticleID)) || !reflect.DeepEqual(beforePage, syncBinding(t, db)) {
		t.Fatal("directory prepare changed article/page")
	}
	plan[0].ExpectedSectionTitle = "并发编程"
	_, next, err := service.PrepareSyncedTopicsWithReuse(context.Background(), r.Lease, "a", revision, "topic", options, plan)
	if err != nil || next != revision {
		t.Fatal("noop changed revision", next, err)
	}
}
func TestReviewedTopicReuseRejectsWholePlan(t *testing.T) {
	cases := []string{"unknown_option", "option_name", "title", "status", "sort", "missing_status", "missing_sort", "duplicate_option", "duplicate_section", "other_module", "occupied", "title_collision", "second_invalid"}
	for _, test := range cases {
		t.Run(test, func(t *testing.T) {
			db, _, r := syncFixture(t)
			db.Model(&model.NotionSyncControl{}).Where("id = 1").Updates(map[string]interface{}{"paused": true, "source_writes_paused": true, "baseline_frozen": true})
			chapter := model.Section{Code: "legacy", Title: "Old", ModuleCode: "m1", Status: 2, Sort: 8}
			if err := db.Create(&chapter).Error; err != nil {
				t.Fatal(err)
			}
			options := []catalog.TopicOption{{ID: "opaque/topic%2F", Name: "并发编程"}, {ID: "second", Name: "Second"}}
			plan := []catalog.TopicReuseInput{reviewedReuse(chapter.Code, chapter.Title, chapter.Status, chapter.Sort)}
			switch test {
			case "unknown_option":
				plan[0].OptionID = "unknown"
			case "option_name":
				plan[0].ExpectedOptionName = "Changed remotely"
			case "title":
				plan[0].ExpectedSectionTitle = "stale"
			case "status":
				plan[0].ExpectedSectionStatus = reuseInt(1)
			case "sort":
				plan[0].ExpectedSectionSort = reuseInt(9)
			case "missing_status":
				plan[0].ExpectedSectionStatus = nil
			case "missing_sort":
				plan[0].ExpectedSectionSort = nil
			case "duplicate_option":
				plan = append(plan, plan[0])
			case "duplicate_section":
				second := plan[0]
				second.OptionID = "second"
				second.ExpectedOptionName = "Second"
				plan = append(plan, second)
			case "other_module":
				plan[0].SectionCode = "s2"
				plan[0].ExpectedSectionTitle = "S2"
				plan[0].ExpectedSectionStatus = reuseInt(1)
				plan[0].ExpectedSectionSort = reuseInt(0)
			case "occupied":
				plan[0].SectionCode = "s1"
				plan[0].ExpectedSectionTitle = "S1"
				plan[0].ExpectedSectionStatus = reuseInt(1)
				plan[0].ExpectedSectionSort = reuseInt(0)
			case "title_collision":
				collision := model.Section{Code: "collision", Title: "并发编程", ModuleCode: "m1"}
				if err := db.Create(&collision).Error; err != nil {
					t.Fatal(err)
				}
			case "second_invalid":
				second := plan[0]
				second.OptionID = "second"
				second.ExpectedOptionName = "Second"
				second.SectionCode = "missing"
				plan = append(plan, second)
			}
			var beforeSections []model.Section
			var beforeBindings []model.NotionCatalogBinding
			db.Order("id").Find(&beforeSections)
			db.Order("id").Find(&beforeBindings)
			_, _, err := catalog.New(store.NewStore(db)).PrepareSyncedTopicsWithReuse(context.Background(), r.Lease, "a", 1, "topic", options, plan)
			if err == nil {
				t.Fatal("invalid reviewed plan accepted", test)
			}
			errorHTTP(t, err, 409)
			var afterSections []model.Section
			var afterBindings []model.NotionCatalogBinding
			db.Order("id").Find(&afterSections)
			db.Order("id").Find(&afterBindings)
			var source model.NotionSyncSource
			db.First(&source, "source_id = ?", "a")
			if !reflect.DeepEqual(beforeSections, afterSections) || !reflect.DeepEqual(beforeBindings, afterBindings) || source.ConfigRevision != 1 {
				t.Fatal("rejected plan mutated catalog", test)
			}
		})
	}
}
