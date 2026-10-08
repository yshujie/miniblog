package article

import (
	"context"
	"errors"
	"github.com/yshujie/miniblog/internal/miniblog/model"
	"gorm.io/gorm"
	"reflect"
	"testing"
)

func TestSyncTopicCannotOccupyAnotherBindingSection(t *testing.T) {
	db, b, r := syncFixture(t)
	duplicate := model.NotionCatalogBinding{SourceID: "a", DataSourceID: syncDataSourceA, ThemePropertyID: "topic", OptionID: "other", OptionName: "Other", SectionCode: syncString("s1"), Status: model.NotionCatalogBound}
	if e := db.Create(&duplicate).Error; e != nil {
		t.Fatal(e)
	}
	results, e := b.EnsureSyncedTopics(context.Background(), r.Lease, "a", 1, "topic", []TopicOption{{ID: "other", Name: "Other"}})
	if e != nil {
		t.Fatal(e)
	}
	if len(results) == 0 || results[0].Outcome != "blocked" {
		t.Fatalf("occupied chapter renamed: %+v", results)
	}
	var section model.Section
	db.Where("code = ?", "s1").First(&section)
	if section.Title != "S1" {
		t.Fatal(section)
	}
}

func TestPrepareTopicsDisabledSourceNoopAndRevision(t *testing.T) {
	db, b, r := syncFixture(t)
	first := syncApply(t, db, b, r)
	db.Model(&model.NotionSyncControl{}).Where("id = 1").Updates(map[string]interface{}{"paused": true, "source_writes_paused": true, "baseline_frozen": true})
	db.Model(&model.NotionSyncSource{}).Where("source_id = ?", "a").Update("enabled", false)
	beforeArticle := syncArticle(t, db, first.ArticleID)
	beforePage := syncBinding(t, db)
	options := []TopicOption{{ID: "empty-topic", Name: "Empty topic"}}
	results, revision, e := b.PrepareSyncedTopics(context.Background(), r.Lease, "a", 1, "topic", options)
	if e != nil || revision != 2 || len(results) != 2 || results[0].Outcome != "created" || results[0].BindingID == 0 {
		t.Fatal(results, revision, e)
	}
	var created model.Section
	db.Where("code = ?", results[0].SectionCode).First(&created)
	db.Model(&created).Update("sort", 91)
	second, next, e := b.PrepareSyncedTopics(context.Background(), r.Lease, "a", revision, "topic", options)
	if e != nil || next != revision || second[0].Outcome != "bound" || second[0].BindingID != results[0].BindingID {
		t.Fatal(second, next, e)
	}
	db.First(&created, created.ID)
	if created.Sort != 91 {
		t.Fatal("prepare changed local order", created)
	}
	options[0].Name = "Renamed topic"
	_, next, e = b.PrepareSyncedTopics(context.Background(), r.Lease, "a", revision, "topic", options)
	if e != nil || next != 3 {
		t.Fatal(next, e)
	}
	if after := syncArticle(t, db, first.ArticleID); !reflect.DeepEqual(after, beforeArticle) {
		t.Fatal("prepare changed article", after)
	}
	if after := syncBinding(t, db); !reflect.DeepEqual(after, beforePage) {
		t.Fatal("prepare changed page projection", after)
	}
}
func TestPrepareTopicsMaintenanceAndCASGuards(t *testing.T) {
	for _, guard := range []string{"pause", "write_pause", "baseline", "lease", "config", "duplicate"} {
		t.Run(guard, func(t *testing.T) {
			db, b, r := syncFixture(t)
			db.Model(&model.NotionSyncControl{}).Where("id = 1").Updates(map[string]interface{}{"paused": true, "source_writes_paused": true, "baseline_frozen": true})
			expected := uint64(1)
			options := []TopicOption{{ID: "unused", Name: "Unused"}}
			switch guard {
			case "pause":
				db.Model(&model.NotionSyncControl{}).Where("id = 1").Update("paused", false)
			case "write_pause":
				db.Model(&model.NotionSyncControl{}).Where("id = 1").Update("source_writes_paused", false)
			case "baseline":
				db.Model(&model.NotionSyncControl{}).Where("id = 1").Update("baseline_frozen", false)
			case "lease":
				r.Lease.Epoch++
			case "config":
				expected++
			case "duplicate":
				options = append(options, options[0])
			}
			_, _, e := b.PrepareSyncedTopics(context.Background(), r.Lease, "a", expected, "topic", options)
			if e == nil {
				t.Fatal("guard accepted", guard)
			}
			var sections, bindings int64
			db.Model(&model.Section{}).Count(&sections)
			db.Model(&model.NotionCatalogBinding{}).Count(&bindings)
			var src model.NotionSyncSource
			db.First(&src, "source_id = ?", "a")
			if sections != 2 || bindings != 2 || src.ConfigRevision != 1 {
				t.Fatalf("rejected prepare wrote: %d %d %+v", sections, bindings, src)
			}
		})
	}
}
func TestPrepareTopicPersistenceFailureRollsBackRevisionAndCatalog(t *testing.T) {
	db, b, r := syncFixture(t)
	db.Model(&model.NotionSyncControl{}).Where("id = 1").Updates(map[string]interface{}{"paused": true, "source_writes_paused": true, "baseline_frozen": true})
	fail := errors.New("catalog persistence unavailable")
	db.Callback().Create().Before("gorm:create").Register("prepare_test_fail", func(tx *gorm.DB) {
		if tx.Statement.Table == "notion_catalog_bindings" {
			tx.AddError(fail)
		}
	})
	t.Cleanup(func() { db.Callback().Create().Remove("prepare_test_fail") })
	_, _, e := b.PrepareSyncedTopics(context.Background(), r.Lease, "a", 1, "topic", []TopicOption{{ID: "unused", Name: "Unused"}})
	if !errors.Is(e, fail) {
		t.Fatal(e)
	}
	var n int64
	db.Model(&model.Section{}).Count(&n)
	var src model.NotionSyncSource
	db.First(&src, "source_id = ?", "a")
	if n != 2 || src.ConfigRevision != 1 {
		t.Fatal("partial prepare survived rollback", n, src)
	}
}
