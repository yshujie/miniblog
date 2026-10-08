package article

import (
	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/source"
	"testing"
)

func TestManagedLocalHTTPStrictCommandsAndLegacyConflict(t *testing.T) {
	db := controllerDB(t)
	if err := db.AutoMigrate(&model.NotionPageBinding{}, &model.NotionSyncControl{}); err != nil {
		t.Fatal(err)
	}
	db.Create(&model.NotionSyncControl{ID: 1})
	const id uint64 = 9007199254740993
	identity, _ := source.NotionIdentity("1234567890abcdef1234567890abcdef")
	row := model.Article{ID: id, Title: "Managed", Author: "old", SectionCode: "s", Status: 2, ExternalLink: identity.CanonicalURL, SourceKey: &identity.SourceKey, Provider: &identity.Provider}
	if err := db.Create(&row).Error; err != nil {
		t.Fatal(err)
	}
	public := "https://public.notion.site/1234567890abcdef1234567890abcdef"
	binding := model.NotionPageBinding{PageID: "1234567890abcdef1234567890abcdef", ArticleID: &row.ID, ManagementState: model.NotionManagementManaged, PublicURL: &public, DesiredState: 2}
	if err := db.Create(&binding).Error; err != nil {
		t.Fatal(err)
	}
	g := testController(db)
	path := "/articles/9007199254740993"
	for _, test := range []struct{ method, path, body string }{
		{"PATCH", path + "/local-fields", `{}`},
		{"PATCH", path + "/local-fields", `{"author":"new","title":"bad"}`},
		{"PATCH", path + "/local-fields", `{"author":"new"} {}`},
		{"PUT", path + "/publication-hold", `{}`},
		{"PUT", "/articles/9223372036854775808/publication-hold", `{"held":true}`},
	} {
		if code, out := send(t, g, test.method, test.path, test.body); code != 400 {
			t.Fatal(code, out)
		}
	}
	code, out := send(t, g, "PATCH", path+"/local-fields", `{"author":""}`)
	if code != 200 {
		t.Fatal(code, out)
	}
	if out["payload"].(map[string]interface{})["article"].(map[string]interface{})["id_text"] != "9007199254740993" {
		t.Fatal(out)
	}
	db.First(&row, id)
	if row.Author != "" || row.Title != "Managed" || row.Status != 2 {
		t.Fatal("local patch changed managed fields", row)
	}
	for _, held := range []string{"true", "false"} {
		if code, out = send(t, g, "PUT", path+"/publication-hold", `{"held":`+held+`}`); code != 200 {
			t.Fatal(code, out)
		}
	}
	code, out = send(t, g, "PUT", path, `{"title":"Managed","module_code":"m","section_code":"s","external_link":"`+identity.CanonicalURL+`"}`)
	if code != 200 {
		t.Fatal("unchanged managed legacy payload rejected", code, out)
	}
	// Feature registration disabled must not mask a managed conflict as 503.
	t.Setenv("MINIBLOG_CONTENT_REGISTER_ENABLED", "false")
	code, out = send(t, g, "PUT", path, `{"title":"Changed","module_code":"m","section_code":"s","external_link":"https://example.com/other"}`)
	if code != 409 || out["payload"] == nil {
		t.Fatal("expected conflict with current record", code, out)
	}
	db.First(&row, id)
	if row.Title != "Managed" || row.ExternalLink != identity.CanonicalURL {
		t.Fatal("conflict was not atomic")
	}
}
