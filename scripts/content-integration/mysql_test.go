// These tests only run against an explicitly named, disposable local database.
package contentintegration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	drivermysql "github.com/go-sql-driver/mysql"
	articlebiz "github.com/yshujie/miniblog/internal/miniblog/biz/article"
	"github.com/yshujie/miniblog/internal/miniblog/biz/blog"
	"github.com/yshujie/miniblog/internal/miniblog/biz/catalog"
	articlecontroller "github.com/yshujie/miniblog/internal/miniblog/controller/v1/article"
	blogcontroller "github.com/yshujie/miniblog/internal/miniblog/controller/v1/blog"
	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/source"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	v1 "github.com/yshujie/miniblog/pkg/api/miniblog/v1"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func scratchDB(t *testing.T) (*gorm.DB, string) {
	t.Helper()
	dsn := os.Getenv("MINIBLOG_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("set MINIBLOG_TEST_MYSQL_DSN to a disposable loopback test database")
	}
	cfg, err := drivermysql.ParseDSN(dsn)
	if err != nil {
		t.Fatal("invalid test DSN")
	}
	host, _, err := net.SplitHostPort(cfg.Addr)
	if err != nil || cfg.Net != "tcp" || (host != "127.0.0.1" && host != "::1" && host != "localhost") || !regexp.MustCompile(`^miniblog_refactor_test_[A-Za-z0-9_]+$`).MatchString(cfg.DBName) {
		t.Fatal("refusing destructive fixture setup: expected loopback TCP and miniblog_refactor_test_* database")
	}
	cfg.MultiStatements = true
	db, err := gorm.Open(mysql.Open(cfg.FormatDSN()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal("cannot open disposable test database", err)
	}
	sql, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sql.SetMaxOpenConns(20)
	t.Cleanup(func() { sql.Close() })
	for _, table := range []string{"article", "subsection", "section", "module", "users", "user"} {
		if err := db.Exec("DROP TABLE IF EXISTS " + table).Error; err != nil {
			t.Fatal(err)
		}
	}
	migration(t, db, "000001_init.up.sql", cfg.DBName)
	migration(t, db, "000003_add_subsection.up.sql", cfg.DBName)
	migration(t, db, "000004_content_sources.up.sql", cfg.DBName)
	for _, statement := range []string{
		"INSERT INTO module(id,code,title,status,sort) VALUES(1,'m1','One',1,1),(2,'m2','Two',1,2)",
		"INSERT INTO section(id,code,title,module_code,status,sort) VALUES(1,'s1','One','m1',1,1),(2,'s2','Two','m2',1,1)",
		"INSERT INTO subsection(id,code,title,section_code,status,sort) VALUES(1,'sub1','Child','s1',1,1)",
	} {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}
	return db, cfg.FormatDSN()
}
func migration(t *testing.T, db *gorm.DB, file, name string) {
	t.Helper()
	if err := execMigration(db, file, name); err != nil {
		t.Fatalf("migration %s: %v", file, err)
	}
}
func execMigration(db *gorm.DB, file, name string) error {
	data, err := os.ReadFile(filepath.Join("../../db/migrations/sql", file))
	if err != nil {
		return err
	}
	sql, _ := db.DB()
	_, err = sql.Exec(strings.ReplaceAll(string(data), "USE "+string(rune(96))+"miniblog"+string(rune(96))+";", "USE "+string(rune(96))+name+string(rune(96))+";"))
	return err
}
func TestMySQLMigrationAuditAndSafeRollback(t *testing.T) {
	db, dsn := scratchDB(t)
	originalURL := "https://notion.so/aabbccddeeff00112233445566778899?view=old#original"
	const id uint64 = 9007199254740993
	if err := db.Exec("INSERT INTO article(id,title,content,external_link,section_code,pos,status,created_at,updated_at) VALUES(?, 'Legacy','Body',?,'s1',7,2,'2020-01-01','2020-01-02')", id, originalURL).Error; err != nil {
		t.Fatal(err)
	}
	if err := execMigration(db, "000005_source_uniqueness.up.sql", ""); err == nil {
		t.Fatal("cutover accepted missing source identity")
	}
	var before model.Article
	db.First(&before, id)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	report := filepath.Join(t.TempDir(), "audit.json")
	cmd := exec.CommandContext(ctx, "go", "run", "../audit-content", "-apply", "-report", report)
	cmd.Env = append(os.Environ(), "MYSQL_DSN="+dsn)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("audit backfill: %v %s", err, output)
	}
	var after model.Article
	db.First(&after, id)
	if after.ID != id || after.ExternalLink != originalURL || after.Content != "Body" || after.Status != 2 || after.Pos != 7 || !after.UpdatedAt.Equal(before.UpdatedAt) || after.SourceKey == nil {
		t.Fatalf("history changed: %+v", after)
	}
	migration(t, db, "000005_source_uniqueness.up.sql", "")
	ready, err := store.RegistrationReady(context.Background(), db)
	if err != nil || !ready {
		t.Fatal("registration not ready", ready, err)
	}
	for _, file := range []string{"000005_source_uniqueness.down.sql", "000004_content_sources.down.sql", "000004_content_sources.up.sql", "000005_source_uniqueness.up.sql"} {
		migration(t, db, file, "")
	}
	var key string
	db.Raw("SELECT source_key FROM article WHERE id=?", id).Scan(&key)
	if key != *after.SourceKey {
		t.Fatal("rollback erased source identity")
	}
	err = db.Exec("INSERT INTO article(id,title,external_link,section_code,source_key) VALUES(?, 'duplicate',?,'s1',?)", id+1, originalURL, key).Error
	if err == nil {
		t.Fatal("unique index did not reject duplicate")
	}
	// The binary source key must not inherit the case-insensitive legacy collation.
	var collation string
	db.Raw("SELECT COLLATION_NAME FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='article' AND COLUMN_NAME='source_key'").Scan(&collation)
	if collation != "ascii_bin" {
		t.Fatal(collation)
	}
}
func TestMySQLConcurrentRegistrationAndPositions(t *testing.T) {
	db, _ := scratchDB(t)
	migration(t, db, "000005_source_uniqueness.up.sql", "")
	b := articlebiz.NewWithNotionClient(store.NewStore(db), nil)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var wg sync.WaitGroup
	results := make(chan *v1.RegisterArticleResponse, 16)
	failures := make(chan error, 16)
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			section := "s1"
			if i%2 == 1 {
				section = "s2"
			}
			r, err := b.Register(ctx, &v1.RegisterArticleRequest{ExternalLink: "https://notion.so/aabbccddeeff00112233445566778899?variant=" + strconv.Itoa(i), Title: "Concurrent", SectionCode: section, Publish: true})
			if err != nil {
				failures <- err
				return
			}
			results <- r
		}(i)
	}
	wg.Wait()
	close(results)
	close(failures)
	for err := range failures {
		t.Error(err)
	}
	if t.Failed() {
		return
	}
	created := 0
	identity := ""
	for r := range results {
		if r.Outcome == "created" {
			created++
		}
		if identity != "" && identity != r.Article.ID {
			t.Fatal("different article IDs")
		}
		identity = r.Article.ID
	}
	var count int64
	db.Model(&model.Article{}).Count(&count)
	if created != 1 || count != 1 {
		t.Fatal("registration not idempotent", created, count)
	}
	// Independent documents in one module serialize placement allocation.
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := b.Register(ctx, &v1.RegisterArticleRequest{ExternalLink: fmt.Sprintf("https://example.com/concurrent/%d", i), Title: "Position", SectionCode: "s1"})
			if err != nil {
				t.Error(err)
			}
		}(i)
	}
	wg.Wait()
	var positions []int
	db.Model(&model.Article{}).Where("section_code='s1'").Order("pos").Pluck("pos", &positions)
	for i, p := range positions {
		if p != i+1 {
			t.Fatal("duplicate or missing position", positions)
		}
	}
	imports := make(chan *articlebiz.ImportResult, 12)
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			section := "s1"
			if i%2 == 1 {
				section = "s2"
			}
			result, err := b.Import(ctx, articlebiz.ImportRequest{Title: "Concurrent import", ExternalLink: "https://example.com/import-one", SectionCode: section}, false)
			if err != nil {
				t.Error(err)
				return
			}
			imports <- result
		}(i)
	}
	wg.Wait()
	close(imports)
	created = 0
	identity = ""
	for result := range imports {
		if result.Outcome == "created" {
			created++
		}
		value := strconv.FormatUint(result.ID, 10)
		if identity != "" && identity != value {
			t.Fatal("concurrent imports returned different IDs")
		}
		identity = value
	}
	if created != 1 {
		t.Fatal("concurrent imports not idempotent", created)
	}
}
func TestMySQLCommandsVisibilityAndFailure(t *testing.T) {
	db, _ := scratchDB(t)
	migration(t, db, "000005_source_uniqueness.up.sql", "")
	ds := store.NewStore(db)
	b := articlebiz.NewWithNotionClient(ds, nil)
	ctx := context.Background()
	r, err := b.Register(ctx, &v1.RegisterArticleRequest{Title: "Original", ExternalLink: "https://example.com/article", SectionCode: "s1", Publish: true})
	if err != nil {
		t.Fatal(err)
	}
	id, _ := strconv.ParseUint(r.Article.ID, 10, 64)
	out, err := b.Update(ctx, &v1.UpdateArticleRequest{ID: r.Article.ID, Title: "Edited", SectionCode: "s1", ExternalLink: "https://example.com/article"})
	if err != nil || out.Article.Status != "Published" {
		t.Fatal(out, err)
	}
	moved, err := b.Move(ctx, id, &v1.MoveArticleRequest{SectionCode: "s2"})
	if err != nil || moved.Article.ID != r.Article.ID {
		t.Fatal(moved, err)
	}
	detail, err := blog.New(ds).GetArticleDetail(ctx, &v1.GetArticleDetailRequest{ArticleID: id})
	if err != nil || detail.ArticleDetail.ModuleCode != "m2" {
		t.Fatal(detail, err)
	}
	if err := catalog.New(ds).DeleteSection(ctx, "s2"); err == nil {
		t.Fatal("deleted dependent section")
	}
	if _, err = b.Archive(ctx, id); err != nil {
		t.Fatal(err)
	}
	duplicate, err := b.Register(ctx, &v1.RegisterArticleRequest{ExternalLink: "https://example.com/article", Title: "Ignored", SectionCode: "s1", Publish: true})
	if err != nil || duplicate.Outcome != "already_registered" || duplicate.Article.Status != "Deleted" {
		t.Fatal(duplicate, err)
	}
	restored, err := b.Restore(ctx, id)
	if err != nil || restored.Article.Status != "Draft" {
		t.Fatal(restored, err)
	}
	if err = b.Publish(ctx, id); err != nil {
		t.Fatal(err)
	}
	if err = b.Publish(ctx, id); err != nil {
		t.Fatal("repeat publish", err)
	}
	if _, err = catalog.New(ds).ModuleStatus(ctx, "m2", 2); err != nil {
		t.Fatal(err)
	}
	if _, err = blog.New(ds).GetArticleDetail(ctx, &v1.GetArticleDetailRequest{ArticleID: id}); err == nil {
		t.Fatal("hidden module exposed detail")
	}
	// A real database failure must abort registration and preserve its position.
	if err = db.Exec("CREATE TRIGGER reject_article BEFORE INSERT ON article FOR EACH ROW SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'injected insert failure'").Error; err != nil {
		t.Fatal(err)
	}
	if _, err = b.Register(ctx, &v1.RegisterArticleRequest{ExternalLink: "https://example.com/fail", Title: "Failure", SectionCode: "s1", Publish: true}); err == nil {
		t.Fatal("database failure swallowed")
	}
	db.Exec("DROP TRIGGER reject_article")
	var count int64
	db.Model(&model.Article{}).Count(&count)
	if count != 1 {
		t.Fatal("failed transaction leaked", count)
	}
	r2, err := b.Register(ctx, &v1.RegisterArticleRequest{ExternalLink: "https://example.com/next", Title: "Next", SectionCode: "s1"})
	if err != nil || r2.Article.Pos != 1 {
		t.Fatal(r2, err)
	}
}
func TestMySQLPublicPaginationAndReorder(t *testing.T) {
	db, _ := scratchDB(t)
	migration(t, db, "000005_source_uniqueness.up.sql", "")
	ds := store.NewStore(db)
	b := articlebiz.NewWithNotionClient(ds, nil)
	ctx := context.Background()
	ids := []string{}
	for i := 0; i < 105; i++ {
		identity, _ := source.Parse(fmt.Sprintf("https://example.com/batch/%d", i))
		row := model.Article{ID: uint64(9007199254740993 + i), Title: fmt.Sprintf("A%d", i), ExternalLink: identity.CanonicalURL, SectionCode: "s1", Pos: i + 1, Status: 2, Provider: &identity.Provider, CanonicalURL: &identity.CanonicalURL, SourceKey: &identity.SourceKey}
		if err := db.Create(&row).Error; err != nil {
			t.Fatal(err)
		}
		ids = append(ids, strconv.FormatUint(row.ID, 10))
	}
	list, err := b.GetList(ctx, &v1.ArticleListRequest{ModuleCode: "m1", Page: 2, Limit: 10})
	if err != nil || list.Total != 105 || len(list.Articles) != 10 || list.Articles[0].IDText != "9007199254741003" {
		t.Fatal(list, err)
	}
	directory, err := blog.New(ds).GetModuleDetail(ctx, &v1.GetModuleDetailRequest{ModuleCode: "m1"})
	if err != nil || len(directory.ModuleDetail.Sections[0].Articles) != 105 {
		t.Fatal("public list truncated", err)
	}
	if err = b.Reorder(ctx, &v1.ReorderArticlesRequest{SectionCode: "s1", ArticleIDs: ids[:104]}); err == nil {
		t.Fatal("partial reorder accepted")
	}
	for i, j := 0, len(ids)-1; i < j; i, j = i+1, j-1 {
		ids[i], ids[j] = ids[j], ids[i]
	}
	if err = b.Reorder(ctx, &v1.ReorderArticlesRequest{SectionCode: "s1", ArticleIDs: ids}); err != nil {
		t.Fatal(err)
	}
	list, err = b.GetList(ctx, &v1.ArticleListRequest{ModuleCode: "m1", SectionCode: "s1", Page: 1, Limit: 10})
	if err != nil || list.Articles[0].IDText != ids[0] || list.Articles[0].Pos != 1 {
		t.Fatal(list, err)
	}
}

func TestMySQLLegacyCodesAndOppositeMoves(t *testing.T) {
	db, _ := scratchDB(t)
	migration(t, db, "000005_source_uniqueness.up.sql", "")
	ds := store.NewStore(db)
	b := articlebiz.NewWithNotionClient(ds, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	r1, err := b.Register(ctx, &v1.RegisterArticleRequest{Title: "One", ExternalLink: "https://example.com/one", SectionCode: "s1", SubsectionCode: "sub1", Publish: true})
	if err != nil {
		t.Fatal(err)
	}
	r2, err := b.Register(ctx, &v1.RegisterArticleRequest{Title: "Two", ExternalLink: "https://example.com/two", SectionCode: "s2", Publish: true})
	if err != nil {
		t.Fatal(err)
	}
	id1, _ := strconv.ParseUint(r1.Article.ID, 10, 64)
	id2, _ := strconv.ParseUint(r2.Article.ID, 10, 64)
	db.Exec("UPDATE article SET section_code='S1', subsection_code='SUB1' WHERE id=?", id1)
	db.Exec("UPDATE subsection SET section_code='S1' WHERE code='sub1'")
	module, err := blog.New(ds).GetModuleDetail(ctx, &v1.GetModuleDetailRequest{ModuleCode: "M1"})
	if err != nil || len(module.ModuleDetail.Sections[0].Subsections) != 1 || len(module.ModuleDetail.Sections[0].Subsections[0].Articles) != 1 {
		t.Fatal("legacy collation mismatch dropped visible article", module, err)
	}
	var wg sync.WaitGroup
	for _, move := range []struct {
		id      uint64
		section string
	}{{id1, "s2"}, {id2, "s1"}} {
		wg.Add(1)
		go func(id uint64, section string) {
			defer wg.Done()
			if _, err := b.Move(ctx, id, &v1.MoveArticleRequest{SectionCode: section}); err != nil {
				t.Error(err)
			}
		}(move.id, move.section)
	}
	wg.Wait()
	for _, want := range []struct {
		id     uint64
		module string
	}{{id1, "m2"}, {id2, "m1"}} {
		result, err := blog.New(ds).GetArticleDetail(ctx, &v1.GetArticleDetailRequest{ArticleID: want.id})
		if err != nil || result.ArticleDetail.ModuleCode != want.module {
			t.Fatal(result, err)
		}
	}
}

func TestMySQLHTTPContractAndError(t *testing.T) {
	db, _ := scratchDB(t)
	migration(t, db, "000005_source_uniqueness.up.sql", "")
	t.Setenv("MINIBLOG_CONTENT_REGISTER_ENABLED", "true")
	gin.SetMode(gin.TestMode)
	ds := store.NewStore(db)
	g := gin.New()
	controller := articlecontroller.New(ds)
	reader := blogcontroller.New(ds)
	// Authentication is unchanged and outside these content-contract fixtures.
	g.POST("/articles/register", controller.Register)
	g.GET("/articles/:id", controller.GetOne)
	g.PUT("/articles/:id", controller.Update)
	g.PUT("/articles/:id/move", controller.Move)
	g.GET("/articleDetail", reader.GetArticleDetail)
	g.GET("/modules", reader.GetModuleList)
	send := func(method, path, body string) (int, map[string]interface{}) {
		t.Helper()
		request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
		request.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()
		g.ServeHTTP(recorder, request)
		decoder := json.NewDecoder(recorder.Body)
		decoder.UseNumber()
		var result map[string]interface{}
		if err := decoder.Decode(&result); err != nil {
			t.Fatal(err)
		}
		return recorder.Code, result
	}
	payload := func(response map[string]interface{}) map[string]interface{} {
		return response["payload"].(map[string]interface{})
	}
	listStatus, listResponse := send("GET", "/modules", "")
	if listStatus != 200 {
		t.Fatal(listStatus, listResponse)
	}
	firstModule := payload(listResponse)["modules"].([]interface{})[0].(map[string]interface{})
	if firstModule["id"].(json.Number).String() != "1" || firstModule["code"] != "m1" || firstModule["status"].(json.Number).String() != "1" {
		t.Fatal("module summaries dropped stored identity", firstModule)
	}
	status, response := send("POST", "/articles/register", `{"external_link":"https://example.com/http","title":"HTTP published","section_code":"s1","publish":true}`)
	if status != 200 {
		t.Fatal(status, response)
	}
	article := payload(response)["article"].(map[string]interface{})
	id := article["id"].(string)
	if article["status"] != "Published" || article["id_text"] != id || payload(response)["outcome"] != "created" {
		t.Fatal(article)
	}
	status, response = send("GET", "/articles/"+id, "")
	if status != 200 {
		t.Fatal(status, response)
	}
	legacy := payload(response)["article"].(map[string]interface{})
	if legacy["id"].(json.Number).String() != id || legacy["id_text"] != id {
		t.Fatal("legacy ID contract", legacy)
	}
	status, response = send("PUT", "/articles/"+id, `{"title":"Edited","external_link":"https://example.com/http","module_code":"m1","section_code":"s1","author":"","tags":[]}`)
	if status != 200 || payload(response)["article"].(map[string]interface{})["status"] != "Published" {
		t.Fatal(status, response)
	}
	status, response = send("PUT", "/articles/"+id+"/move", `{"section_code":"s2"}`)
	if status != 200 || payload(response)["article"].(map[string]interface{})["id"] != id {
		t.Fatal(status, response)
	}
	status, response = send("GET", "/articleDetail?article_id="+id, "")
	if status != 200 || payload(response)["article_detail"].(map[string]interface{})["module_code"] != "m2" {
		t.Fatal(status, response)
	}
	if err := db.Exec("CREATE TRIGGER reject_http BEFORE INSERT ON article FOR EACH ROW SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT='injected HTTP failure'").Error; err != nil {
		t.Fatal(err)
	}
	status, response = send("POST", "/articles/register", `{"external_link":"https://example.com/http-failure","title":"Failure","section_code":"s1","publish":true}`)
	if status != 500 || response["code"] == "ok" {
		t.Fatal("database failure disguised as success", status, response)
	}
	db.Exec("DROP TRIGGER reject_http")
}
