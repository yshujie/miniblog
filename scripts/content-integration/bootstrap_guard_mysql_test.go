package contentintegration

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	articlebiz "github.com/yshujie/miniblog/internal/miniblog/biz/article"
	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/store"
)

func TestMySQLReviewedPublishedAdoptionRechecksCurrentConditionsAtomically(t *testing.T) {
	for _, gate := range []string{"source_disabled", "section_disabled", "publication_hold", "public_url_invalid"} {
		t.Run(gate, func(t *testing.T) {
			db, _ := scratchDB(t)
			migration(t, db, "000005_source_uniqueness.up.sql", "")
			const pageID = "1234567890abcdef1234567890abcdef"
			const sourceID = "2bf330bd-ddf1-80a6-aa49-000bbd1e154b"
			a := model.Article{ID: 9007199254740993, Title: "History", Content: "Local body", ExternalLink: "https://notion.so/" + pageID, SectionCode: "s1", SubsectionCode: "sub1", Status: 2, Pos: 71, Author: "Local author"}
			if err := db.Create(&a).Error; err != nil {
				t.Fatal(err)
			}
			src := model.NotionSyncSource{ID: sourceID, DataSourceID: sourceID, ModuleCode: "m1", Enabled: true, ConfigRevision: 1, PropertyMappingJSON: `{"topic_property_id":"topic"}`}
			if err := db.Create(&src).Error; err != nil {
				t.Fatal(err)
			}
			sectionCode := "s1"
			if err := db.Create(&model.NotionCatalogBinding{SourceID: sourceID, DataSourceID: sourceID, ThemePropertyID: "topic", OptionID: "reviewed", OptionName: "One", SectionCode: &sectionCode, Status: model.NotionCatalogBound}).Error; err != nil {
				t.Fatal(err)
			}
			db.First(&a, a.ID)
			beforeJSON, _ := articlebiz.ArticleBeforeJSON(&a)
			state := 2
			public := "https://fixture.notion.site/" + pageID
			held := false
			switch gate {
			case "source_disabled":
				db.Model(&src).Update("enabled", false)
			case "section_disabled":
				db.Model(&model.Section{}).Where("code = ?", "s1").Update("status", 2)
			case "publication_hold":
				held = true
			case "public_url_invalid":
				public = "javascript:alert(1)"
			}
			snapshot, _ := json.Marshal(map[string]interface{}{"page_id": pageID, "source_id": sourceID, "desired_state": 2, "title": "Fresh title", "tags": []string{"Go"}, "topic_option_id": "reviewed", "topic_option_name": "One", "public_url": public})
			p := model.NotionPageBinding{PageID: pageID, ArticleID: &a.ID, SourceID: sourceID, ManagementState: model.NotionManagementBaselinePending, Revision: 3, DesiredState: 2, PublicURL: &public, PublicationHeld: held, SnapshotJSON: string(snapshot), BootstrapState: "verified", BootstrapExpectedState: &state, BootstrapExpectedFingerprint: "reviewed", LocalBeforeJSON: beforeJSON}
			if err := db.Create(&p).Error; err != nil {
				t.Fatal(err)
			}
			db.Model(&model.NotionSyncControl{}).Where("id=1").Updates(map[string]interface{}{"paused": true, "source_writes_paused": true, "baseline_frozen": true})
			ds := store.NewStore(db)
			token, ok, err := ds.NotionSync().AcquireLease(context.Background(), "reviewed-guard", time.Minute)
			if err != nil || !ok {
				t.Fatal(err, ok)
			}
			db.First(&p, "page_id = ?", pageID)
			beforeArticle, beforePage := a, p
			_, err = articlebiz.New(ds).AdoptSyncedSource(context.Background(), articlebiz.AdoptInput{Lease: token, PageID: pageID, SourceID: sourceID, ExpectedArticleID: a.ID, ExpectedBindingRevision: p.Revision, ExpectedConfigRevision: 1, ExpectedFingerprint: "reviewed"})
			if err == nil {
				t.Fatal("published gate ignored", gate)
			}
			reason := gate
			if gate == "section_disabled" {
				reason = "catalog_disabled"
			}
			if !strings.Contains(err.Error(), reason) {
				t.Fatal("unexpected rejection reason", gate, err)
			}
			db.First(&a, a.ID)
			db.First(&p, "page_id = ?", pageID)
			if !reflect.DeepEqual(beforeArticle, a) || !reflect.DeepEqual(beforePage, p) {
				t.Fatal("blocked adoption wrote history or ownership", gate, a, p)
			}
		})
	}
}
