package article

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	articlebiz "github.com/yshujie/miniblog/internal/miniblog/biz/article"
	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"net/http/httptest"
	"testing"
)

func controllerDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sql, _ := db.DB()
	sql.SetMaxOpenConns(1)
	t.Cleanup(func() { sql.Close() })
	if err = db.AutoMigrate(&model.Module{}, &model.Section{}, &model.Subsection{}, &model.Article{}); err != nil {
		t.Fatal(err)
	}
	db.Create(&model.Module{Code: "m", Title: "M", Status: 1})
	db.Create(&model.Section{Code: "s", Title: "S", ModuleCode: "m", Status: 1})
	return db
}
func testController(db *gorm.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	g := gin.New()
	c := New(store.NewStore(db))
	g.POST("/articles/register", c.Register)
	g.POST("/articles", c.Create)
	g.POST("/article-sources/preview", c.Preview)
	g.PUT("/articles/reorder", c.Reorder)
	g.GET("/articles/:id", c.GetOne)
	g.PUT("/articles/:id", c.Update)
	g.PUT("/articles/:id/archive", c.Archive)
	g.PUT("/articles/:id/restore", c.Restore)
	g.PUT("/articles/:id/move", c.Move)
	g.PATCH("/articles/:id/local-fields", c.PatchLocal)
	g.PUT("/articles/:id/publication-hold", c.PublicationHold)
	return g
}
func send(t *testing.T, g *gin.Engine, method, path, body string) (int, map[string]interface{}) {
	t.Helper()
	r := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	g.ServeHTTP(w, r)
	var out map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatal(w.Body.String())
	}
	return w.Code, out
}

const registerBody = `{"external_link":"https://example.com/one","title":"One","section_code":"s","publish":true}`

func TestRegistrationHTTPRequiresSwitchUniqueAndBackfill(t *testing.T) {
	db := controllerDB(t)
	g := testController(db)
	t.Setenv("MINIBLOG_CONTENT_REGISTER_ENABLED", "false")
	if status, _ := send(t, g, "POST", "/articles/register", registerBody); status != 503 {
		t.Fatal(status)
	}
	t.Setenv("MINIBLOG_CONTENT_REGISTER_ENABLED", "true")
	if err := db.Migrator().DropIndex(&model.Article{}, "uq_article_source_key"); err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("CREATE INDEX uq_article_source_key ON article(source_key)").Error; err != nil {
		t.Fatal(err)
	}
	if status, _ := send(t, g, "POST", "/articles/register", registerBody); status != 503 {
		t.Fatal("ordinary index permitted registration", status)
	}
	db.Migrator().DropIndex(&model.Article{}, "uq_article_source_key")
	db.Migrator().CreateIndex(&model.Article{}, "uq_article_source_key")
	legacy := &model.Article{ID: 10, Title: "Legacy", ExternalLink: "https://example.com/legacy", SectionCode: "s", Status: 1}
	db.Create(legacy)
	if status, _ := send(t, g, "POST", "/articles/register", registerBody); status != 503 {
		t.Fatal("incomplete backfill permitted registration", status)
	}
	db.Delete(legacy)
	status, out := send(t, g, "POST", "/articles/register", registerBody)
	if status != 200 {
		t.Fatal(status, out)
	}
	payload := out["payload"].(map[string]interface{})
	a := payload["article"].(map[string]interface{})
	id, ok := a["id"].(string)
	if !ok || id == "" || a["id_text"] != id || a["status"] != "Published" {
		t.Fatal(a)
	}
	status, out = send(t, g, "POST", "/articles/register", `{"external_link":"https://example.com/one","title":"Do not overwrite","section_code":"s","publish":false}`)
	payload = out["payload"].(map[string]interface{})
	duplicate := payload["article"].(map[string]interface{})
	if status != 200 || payload["outcome"] != "already_registered" || duplicate["title"] != "One" || duplicate["id"] != id || duplicate["status"] != "Published" {
		t.Fatal(status, out)
	}
}
func TestHTTPPathIDAndContentOmission(t *testing.T) {
	db := controllerDB(t)
	g := testController(db)
	t.Setenv("MINIBLOG_CONTENT_REGISTER_ENABLED", "true")
	status, out := send(t, g, "POST", "/articles/register", registerBody)
	if status != 200 {
		t.Fatal(out)
	}
	id := out["payload"].(map[string]interface{})["article"].(map[string]interface{})["id"].(string)
	numeric, _ := articlebiz.ParseID(id)
	db.Model(&model.Article{}).Where("id = ?", numeric).Update("content", "Keep body")
	mismatch := fmt.Sprintf(`{"id":"1","title":"Edited","module_code":"m","section_code":"s","external_link":"https://example.com/one"}`)
	if status, _ = send(t, g, "PUT", "/articles/"+id, mismatch); status != 400 {
		t.Fatal(status)
	}
	edit := `{"title":"Edited","module_code":"m","section_code":"s","external_link":"https://example.com/one"}`
	status, out = send(t, g, "PUT", "/articles/"+id, edit)
	if status != 200 {
		t.Fatal(status, out)
	}
	a := out["payload"].(map[string]interface{})["article"].(map[string]interface{})
	if a["content"] != "Keep body" || a["status"] != "Published" || a["id_text"] != id {
		t.Fatal(a)
	}
	for _, bad := range []string{"-1", "0", "9223372036854775808", "abc"} {
		if status, _ = send(t, g, "GET", "/articles/"+bad, ""); status != 400 {
			t.Fatalf("id %s status %d", bad, status)
		}
	}
	status, out = send(t, g, "PUT", "/articles/"+id+"/archive", "")
	if status != 200 || out["payload"].(map[string]interface{})["article"].(map[string]interface{})["status"] != "Deleted" {
		t.Fatal(status, out)
	}
	status, out = send(t, g, "PUT", "/articles/"+id+"/restore", "")
	if status != 200 || out["payload"].(map[string]interface{})["article"].(map[string]interface{})["status"] != "Draft" {
		t.Fatal(status, out)
	}
}
