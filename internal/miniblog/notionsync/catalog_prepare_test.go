package notionsync

import (
	"context"
	"reflect"
	"testing"

	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/source"
	"gorm.io/gorm"
)

func prepareFixture(t *testing.T) (*Service, *gorm.DB, *schemaOnlyClient, CatalogPrepareInput) {
	t.Helper()
	s, db, _ := syncFixture(t)
	if run := waitFixtureRun(t, s, "dry_run"); run.Status != "completed" {
		t.Fatal(run)
	}
	off := false
	row, err := s.UpdateSource(context.Background(), AllowedSources()[0], SourceInput{ModuleCode: "m1", Enabled: &off})
	if err != nil {
		t.Fatal(err)
	}
	on := true
	if _, err = s.UpdateControl(context.Background(), ControlInput{Paused: &on, SourceWritesPaused: &on}); err != nil {
		t.Fatal(err)
	}
	client := &schemaOnlyClient{}
	s.client = client
	return s, db, client, CatalogPrepareInput{SourceID: row.SourceID, ExpectedConfigRevision: row.ConfigRevision}
}

func TestCatalogPrepareSelectedDisabledSourceNoPagesAndNoopRevision(t *testing.T) {
	s, db, client, input := prepareFixture(t)
	a := model.Article{ID: 7001, Title: "Historical", Content: "Body retained", Author: "Original", Tags: "old", Pos: 72, SectionCode: "s1", Status: 2, ExternalLink: "https://example.com/history"}
	if err := db.Create(&a).Error; err != nil {
		t.Fatal(err)
	}
	p := model.NotionPageBinding{PageID: firstPage, ArticleID: &a.ID, SourceID: input.SourceID, Revision: 12, ManagementState: model.NotionManagementBaselinePending, DesiredState: 2, SnapshotJSON: "historical snapshot"}
	if err := db.Create(&p).Error; err != nil {
		t.Fatal(err)
	}
	beforeA, beforeP := a, p
	db.First(&beforeA, a.ID)
	db.First(&beforeP, "page_id = ?", p.PageID)
	var otherSources []model.NotionSyncSource
	db.Where("source_id <> ?", input.SourceID).Order("source_id").Find(&otherSources)
	// A removed option is retained in persistence for audit, not this fresh prepare result.
	removed := model.NotionCatalogBinding{SourceID: input.SourceID, DataSourceID: input.SourceID, ThemePropertyID: "topic", OptionID: "removed", OptionName: "Old option", Status: model.NotionCatalogBlocked}
	if err := db.Create(&removed).Error; err != nil {
		t.Fatal(err)
	}
	result, err := s.PrepareCatalog(context.Background(), input)
	if err != nil || result.ConfigRevision != input.ExpectedConfigRevision+1 || result.RunID == "" || len(result.Items) != 2 {
		t.Fatal(result, err)
	}
	for _, item := range result.Items {
		if item.ID == "" || item.Status != "bound" || item.OptionID == "removed" {
			t.Fatal(item)
		}
	}
	if len(client.calls) != 1 || client.calls[0] != input.SourceID || client.forbidden != 0 {
		t.Fatal(client.calls, client.forbidden)
	}
	var section model.Section
	if err = db.Where("code = ?", result.Items[0].SectionCode).First(&section).Error; err != nil {
		t.Fatal(err)
	}
	if err = db.Model(&section).Update("sort", 91).Error; err != nil {
		t.Fatal(err)
	}
	input.ExpectedConfigRevision = result.ConfigRevision
	again, err := s.PrepareCatalog(context.Background(), input)
	if err != nil || again.ConfigRevision != input.ExpectedConfigRevision || len(again.Items) != 2 {
		t.Fatal(again, err)
	}
	db.Where("code = ?", section.Code).First(&section)
	if section.Sort != 91 {
		t.Fatal("prepare changed local order", section)
	}
	var afterA model.Article
	var afterP model.NotionPageBinding
	db.First(&afterA, a.ID)
	db.First(&afterP, "page_id = ?", p.PageID)
	if !reflect.DeepEqual(afterA, beforeA) || !reflect.DeepEqual(afterP, beforeP) {
		t.Fatalf("prepare changed article/page: %+v %+v", afterA, afterP)
	}
	var afterSources []model.NotionSyncSource
	db.Where("source_id <> ?", input.SourceID).Order("source_id").Find(&afterSources)
	if !reflect.DeepEqual(otherSources, afterSources) {
		t.Fatal("prepare changed unselected sources")
	}
	run, err := s.Run(context.Background(), result.RunID)
	if err != nil || run.Mode != "catalog_prepare" || run.Status != "completed" || run.Counts.Seen != 2 || run.Counts.Created != 2 {
		t.Fatal(run, err)
	}
}

func TestCatalogPrepareConflictIsAuditedAndOtherOptionsContinue(t *testing.T) {
	s, db, _, input := prepareFixture(t)
	if err := db.Create(&model.Section{Code: "conflict", Title: "主题一", ModuleCode: "m1", Status: 1}).Error; err != nil {
		t.Fatal(err)
	}
	result, err := s.PrepareCatalog(context.Background(), input)
	if err != nil || len(result.Items) != 2 {
		t.Fatal(result, err)
	}
	blocked, bound := 0, 0
	for _, item := range result.Items {
		if item.Status == "blocked" && item.ID != "" {
			blocked++
		}
		if item.Status == "bound" {
			bound++
		}
	}
	run, err := s.Run(context.Background(), result.RunID)
	if blocked != 1 || bound != 1 || err != nil || run.Status != "completed_with_errors" || run.Counts.Blocked != 1 {
		t.Fatal(result, run, err)
	}
}

func TestCatalogPrepareGuardsRejectBeforeCatalogWrites(t *testing.T) {
	for _, guard := range []string{"pause", "write_pause", "baseline", "revision", "changed_during_schema", "invalid_schema", "lease_revoked"} {
		t.Run(guard, func(t *testing.T) {
			s, db, client, input := prepareFixture(t)
			switch guard {
			case "pause":
				db.Model(&model.NotionSyncControl{}).Where("id = 1").Update("paused", false)
			case "write_pause":
				db.Model(&model.NotionSyncControl{}).Where("id = 1").Update("source_writes_paused", false)
			case "baseline":
				db.Model(&model.NotionSyncControl{}).Where("id = 1").Update("baseline_frozen", false)
			case "revision":
				input.ExpectedConfigRevision++
			case "changed_during_schema":
				client.beforeRead = func(string) {
					if err := db.Model(&model.NotionSyncSource{}).Where("source_id = ?", input.SourceID).Update("config_revision", input.ExpectedConfigRevision+1).Error; err != nil {
						t.Fatal(err)
					}
				}
			case "invalid_schema":
				client.mutate = func(_ string, schema *source.NotionDataSource) {
					p := schema.Properties["博客状态"]
					p.Select.Options = p.Select.Options[:3]
					schema.Properties["博客状态"] = p
				}
			case "lease_revoked":
				client.beforeRead = func(string) {
					on := true
					if _, err := s.UpdateControl(context.Background(), ControlInput{Paused: &on}); err != nil {
						t.Fatal(err)
					}
				}
			}
			var before int64
			db.Model(&model.Section{}).Count(&before)
			_, err := s.PrepareCatalog(context.Background(), input)
			if err == nil {
				t.Fatal("guard accepted", guard)
			}
			var after, bindings int64
			db.Model(&model.Section{}).Count(&after)
			db.Model(&model.NotionCatalogBinding{}).Count(&bindings)
			if after != before || bindings != 0 || client.forbidden != 0 {
				t.Fatal("rejected prepare wrote", guard, before, after, bindings)
			}
			if (guard == "pause" || guard == "write_pause" || guard == "baseline" || guard == "revision") && len(client.calls) != 0 {
				t.Fatal("guard read Notion before validation", guard, client.calls)
			}
		})
	}
}
