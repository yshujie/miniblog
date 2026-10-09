// Local-only UI acceptance service. Production installRouters is deliberately
// not exported or modified; this dedicated assembly uses real controllers,
// Authn/Authz, business services and MySQL repositories with synthetic data.
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	drivermysql "github.com/go-sql-driver/mysql"
	articlebiz "github.com/yshujie/miniblog/internal/miniblog/biz/article"
	articlecontroller "github.com/yshujie/miniblog/internal/miniblog/controller/v1/article"
	authcontroller "github.com/yshujie/miniblog/internal/miniblog/controller/v1/auth"
	blogcontroller "github.com/yshujie/miniblog/internal/miniblog/controller/v1/blog"
	catalogcontroller "github.com/yshujie/miniblog/internal/miniblog/controller/v1/catalog"
	modulecontroller "github.com/yshujie/miniblog/internal/miniblog/controller/v1/module"
	synccontroller "github.com/yshujie/miniblog/internal/miniblog/controller/v1/notionsync"
	sectioncontroller "github.com/yshujie/miniblog/internal/miniblog/controller/v1/section"
	subsectioncontroller "github.com/yshujie/miniblog/internal/miniblog/controller/v1/subsection"
	usercontroller "github.com/yshujie/miniblog/internal/miniblog/controller/v1/user"
	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/notionsync"
	"github.com/yshujie/miniblog/internal/miniblog/source"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"github.com/yshujie/miniblog/internal/pkg/core"
	"github.com/yshujie/miniblog/internal/pkg/errno"
	"github.com/yshujie/miniblog/internal/pkg/known"
	"github.com/yshujie/miniblog/internal/pkg/log"
	"github.com/yshujie/miniblog/internal/pkg/middleware"
	"github.com/yshujie/miniblog/pkg/auth"
	"github.com/yshujie/miniblog/pkg/token"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const adminFixtureSource = "2bf330bd-ddf1-80a6-aa49-000bbd1e154b"
const adminFixturePage = "abcdefabcdef4abc8abcabcdefabcdef"
const fixturePassword = "LocalFixture12"

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	listen := flag.String("listen", "127.0.0.1:0", "loopback HTTP address")
	readyFile := flag.String("ready", "", "write non-secret fixture metadata")
	flag.Parse()
	host, _, err := net.SplitHostPort(*listen)
	if err != nil || (host != "127.0.0.1" && host != "::1") {
		return errors.New("refusing a non-loopback HTTP listen address")
	}
	dist, err := filepath.Abs(os.Getenv("MINIBLOG_ADMIN_REAL_DIST"))
	if err != nil || os.Getenv("MINIBLOG_ADMIN_REAL_DIST") == "" {
		return errors.New("MINIBLOG_ADMIN_REAL_DIST must point to the new admin build")
	}
	if info, err := os.Stat(filepath.Join(dist, "index.html")); err != nil || info.IsDir() {
		return errors.New("admin dist has no index.html")
	}
	db, name, err := openScratch()
	if err != nil {
		return err
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()
	if err := seed(db, name); err != nil {
		return err
	}
	options := log.NewOptions()
	options.Level = "error"
	log.Init(options)
	gin.SetMode(gin.ReleaseMode)
	var key [32]byte
	if _, err := rand.Read(key[:]); err != nil {
		return err
	}
	token.Init(hex.EncodeToString(key[:]), known.XUsernameKey)
	os.Setenv("MINIBLOG_CONTENT_REGISTER_ENABLED", "true")
	os.Setenv("MINIBLOG_NOTION_TOKEN", "local-fixture-token")
	fake, client, cleanup := localNotion()
	defer cleanup()
	ds := store.NewStore(db)
	service := notionsync.New(ds, notionsync.Options{Enabled: true, Client: client, Author: "接口作者", Interval: time.Hour, OwnerID: "admin-ui-local-service"})
	managedID, err := seedManaged(ds, fake.page.LastEditedTime)
	if err != nil {
		return err
	}
	authz, err := auth.NewAuthz(db)
	if err != nil {
		return err
	}
	if _, err := authz.AddPolicy("fixtureadmin", "/v1/admin/*", ".*"); err != nil {
		return err
	}
	router := gin.New()
	router.Use(gin.Recovery())
	mount(router, ds, authz, service)
	router.GET("/health", func(c *gin.Context) { core.WriteResponse(c, nil, map[string]string{"status": "ok"}) })
	router.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/v1/") {
			core.WriteResponse(c, errno.ErrPageNotFound, nil)
			return
		}
		if c.Request.Method != "GET" && c.Request.Method != "HEAD" {
			c.Status(405)
			return
		}
		relative := strings.TrimPrefix(filepath.Clean("/"+c.Request.URL.Path), "/")
		file := filepath.Join(dist, relative)
		if info, err := os.Stat(file); err != nil || info.IsDir() {
			file = filepath.Join(dist, "index.html")
		}
		http.ServeFile(c.Writer, c.Request, file)
	})
	listener, err := net.Listen("tcp", *listen)
	if err != nil {
		return err
	}
	server := &http.Server{Handler: router, ReadHeaderTimeout: 5 * time.Second}
	ready := map[string]any{"url": "http://" + listener.Addr().String(), "pid": os.Getpid(), "scope": "new admin UI + real Go controller/biz/store + isolated MySQL; dedicated local route assembly", "notion": "local HTTP substitute, real provider not contacted", "module_code": "m1", "section_code": "s1", "subsection_code": "sub1", "manual_large_id": "9007199254740993", "managed_id": fmt.Sprint(managedID), "source_id": adminFixtureSource, "database": name, "dist": dist, "production_router_verified": false}
	if *readyFile != "" {
		data, _ := json.MarshalIndent(ready, "", "  ")
		if err := os.WriteFile(*readyFile, append(data, '\n'), 0600); err != nil {
			return err
		}
	}
	done := make(chan error, 1)
	go func() { done <- server.Serve(listener) }()
	fmt.Println("Local admin acceptance service ready at http://" + listener.Addr().String())
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signals)
	select {
	case <-signals:
	case err := <-done:
		if err != nil && err != http.ErrServerClosed {
			return errors.New("local acceptance HTTP server failed")
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := service.Stop(ctx); err != nil {
		return err
	}
	if err := server.Shutdown(ctx); err != nil {
		return err
	}
	fake.mu.Lock()
	evidence := map[string]any{"provider_requests": fake.requests, "provider_writes": fake.writes, "production_access": false}
	fake.mu.Unlock()
	if *readyFile != "" {
		data, _ := json.MarshalIndent(evidence, "", "  ")
		os.WriteFile(*readyFile+".shutdown.json", append(data, '\n'), 0600)
	}
	return nil
}
func openScratch() (*gorm.DB, string, error) {
	cfg, err := drivermysql.ParseDSN(os.Getenv("MINIBLOG_TEST_MYSQL_DSN"))
	if err != nil {
		return nil, "", errors.New("invalid scratch DSN")
	}
	host, _, err := net.SplitHostPort(cfg.Addr)
	if err != nil || cfg.Net != "tcp" || (host != "127.0.0.1" && host != "::1" && host != "localhost") || !regexp.MustCompile(`^miniblog_refactor_test_admin_ui_[A-Za-z0-9_]+$`).MatchString(cfg.DBName) {
		return nil, "", errors.New("refusing database: requires loopback TCP and miniblog_refactor_test_admin_ui_* schema")
	}
	cfg.MultiStatements = true
	db, err := gorm.Open(mysql.Open(cfg.FormatDSN()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		return nil, "", errors.New("cannot open local scratch database")
	}
	var count int64
	if err := db.Raw("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE()").Scan(&count).Error; err != nil {
		return nil, "", err
	}
	if count != 0 {
		return nil, "", errors.New("refusing fixture initialization: scratch database must be empty")
	}
	return db, cfg.DBName, nil
}
func seed(db *gorm.DB, name string) error {
	for _, file := range []string{"000001_init.up.sql", "000003_add_subsection.up.sql", "000004_content_sources.up.sql", "000005_source_uniqueness.up.sql", "000006_notion_sync.up.sql"} {
		data, err := os.ReadFile(filepath.Join("db/migrations/sql", file))
		if err != nil {
			return err
		}
		sqlDB, _ := db.DB()
		if _, err := sqlDB.Exec(strings.ReplaceAll(string(data), "USE `miniblog`;", "USE `"+name+"`;")); err != nil {
			return fmt.Errorf("scratch migration %s failed", file)
		}
	}
	for _, statement := range []string{"INSERT INTO module(id,code,title,status,sort) VALUES(1,'m1','接口主题',1,1),(2,'m2','另一主题',1,2)", "INSERT INTO section(id,code,title,module_code,status,sort) VALUES(1,'s1','接口章节','m1',1,1),(2,'s2','另一章节','m2',1,1),(3,'hidden','未上架章节','m1',2,2)", "INSERT INTO subsection(id,code,title,section_code,status,sort) VALUES(1,'sub1','接口子章节','s1',1,1)"} {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	if err := db.Create(&model.UserM{Username: "fixtureadmin", Password: fixturePassword, Nickname: "接口验收作者", Introduction: "仅在本机临时数据库中的测试账号", Email: "fixture@example.invalid"}).Error; err != nil {
		return err
	}
	for i, title := range []string{"真实API大ID文章", "真实API草稿", "真实API子章节文章"} {
		link := fmt.Sprintf("https://example.invalid/local-fixture/%d", i)
		identity, err := source.Parse(link)
		if err != nil {
			return err
		}
		article := model.Article{ID: 9007199254740993 + uint64(i), Title: title, Content: "历史正文仅供本机只读核对", ExternalLink: link, Provider: &identity.Provider, CanonicalURL: &identity.CanonicalURL, SourceKey: &identity.SourceKey, SectionCode: "s1", Author: "接口作者", Pos: i + 1, Status: 2}
		if i == 1 {
			article.Status = 1
		}
		if i == 2 {
			article.SubsectionCode = "sub1"
			article.Pos = 1
		}
		if err := db.Create(&article).Error; err != nil {
			return err
		}
	}
	config := notionsync.SourceConfig{TitlePropertyID: "title", StatePropertyID: "state", TopicPropertyID: "topic", TagsPropertyID: "tags", StateOptionIDs: map[string]string{"draft": "draft", "published": "published", "unpublished": "unpublished", "archived": "archived"}}
	cfgJSON, _ := json.Marshal(config)
	stateJSON, _ := json.Marshal(config.StateOptionIDs)
	if err := db.Create(&model.NotionSyncSource{ID: adminFixtureSource, DataSourceID: adminFixtureSource, Label: "本机 Notion 替身", ModuleCode: "m1", Enabled: true, Health: "complete", ConfigRevision: 1, PropertyMappingJSON: string(cfgJSON), StatusMappingJSON: string(stateJSON)}).Error; err != nil {
		return err
	}
	if err := db.Model(&model.NotionSyncControl{}).Where("id=1").Updates(map[string]any{"paused": false, "source_writes_paused": false, "baseline_frozen": true}).Error; err != nil {
		return err
	}
	section := "s1"
	return db.Create(&model.NotionCatalogBinding{SourceID: adminFixtureSource, DataSourceID: adminFixtureSource, ThemePropertyID: "topic", OptionID: "topic-one", OptionName: "接口章节", SectionCode: &section, Status: "bound"}).Error
}
func seedManaged(ds store.IStore, lastEdited time.Time) (uint64, error) {
	ctx := context.Background()
	repo := store.NewNotionSyncRepository(ds.DB())
	lease, ok, err := repo.AcquireLease(ctx, "local-fixture-seed", time.Minute)
	if err != nil || !ok {
		return 0, errors.New("cannot seed managed fixture")
	}
	defer repo.ReleaseLease(ctx, lease)
	public := "https://fixture.notion.site/" + adminFixturePage
	snapshot, err := json.Marshal(notionsync.Snapshot{PageID: adminFixturePage, SourceID: adminFixtureSource, Title: "真实API托管文章", Tags: []string{"接口标签"}, TopicOptionID: "topic-one", TopicOptionName: "接口章节", DesiredState: 2, StateOptionID: "published", PageURL: "https://notion.so/" + adminFixturePage, PublicURL: &public, LastEditedAt: lastEdited})
	if err != nil {
		return 0, err
	}
	out, err := articlebiz.New(ds).ApplySyncedSource(ctx, articlebiz.SyncInput{Lease: lease, RunID: "local-fixture-seed", SourceID: adminFixtureSource, DataSourceID: adminFixtureSource, PageID: adminFixturePage, ThemePropertyID: "topic", ThemeOptionID: "topic-one", ThemeOptionName: "接口章节", Title: "真实API托管文章", Tags: []string{"接口标签"}, PageURL: "https://notion.so/" + adminFixturePage, PublicURL: &public, DesiredState: 2, MetadataComplete: true, ExpectedConfigRevision: 1, MetadataHash: "fixture-seed", SnapshotJSON: string(snapshot), NotionLastEditedAt: &lastEdited})
	if err != nil {
		return 0, err
	}
	return out.ArticleID, nil
}

func mount(router *gin.Engine, ds store.IStore, authz *auth.Authz, service *notionsync.Service) {
	ac := authcontroller.New(ds)
	router.POST("/v1/auth/login", ac.Login)
	router.POST("/v1/auth/logout", ac.Logout)
	bc := blogcontroller.New(ds)
	router.GET("/v1/blog/articleDetail", bc.GetArticleDetail)
	protected := router.Group("/v1/admin", middleware.Authn(), middleware.Authz(authz))
	uc := usercontroller.New(ds, authz)
	protected.GET("/users/myinfo", uc.GetMyInfo)
	ar := articlecontroller.New(ds)
	protected.POST("/article-sources/preview", ar.Preview)
	protected.POST("/articles/register", ar.Register)
	protected.GET("/articles", ar.GetList)
	protected.GET("/articles/:id", ar.GetOne)
	protected.PUT("/articles/:id", ar.Update)
	protected.PUT("/articles/:id/move", ar.Move)
	protected.PUT("/articles/:id/publish", ar.Publish)
	protected.PUT("/articles/:id/unpublish", ar.Unpublish)
	protected.PUT("/articles/:id/archive", ar.Archive)
	protected.PUT("/articles/:id/restore", ar.Restore)
	protected.PUT("/articles/reorder", ar.Reorder)
	protected.PATCH("/articles/:id/local-fields", ar.PatchLocal)
	protected.PUT("/articles/:id/publication-hold", ar.PublicationHold)
	mc := modulecontroller.New(ds)
	protected.GET("/modules", mc.GetAll)
	protected.POST("/modules", mc.Create)
	protected.GET("/modules/:code", mc.GetOne)
	protected.PUT("/modules/:code", mc.Update)
	protected.PUT("/modules/:code/publish", mc.Publish)
	protected.PUT("/modules/:code/unpublish", mc.Unpublish)
	protected.DELETE("/modules/:code", mc.Delete)
	sc := sectioncontroller.New(ds)
	protected.GET("/sections/:module_code", sc.GetList)
	protected.GET("/sections/:module_code/:code", sc.GetOne)
	protected.POST("/sections", sc.Create)
	protected.PUT("/sections/:code", sc.Update)
	protected.PUT("/sections/:code/unpublish", sc.Unpublish)
	protected.PUT("/sections/:code/publish", sc.Publish)
	protected.DELETE("/sections/:code", sc.Delete)
	cc := catalogcontroller.New(ds)
	protected.POST("/catalog/reorder", cc.Reorder)
	nc := synccontroller.New(service)
	protected.GET("/notion-sync/status", nc.Status)
	protected.GET("/notion-sync/sources", nc.Sources)
	protected.GET("/notion-sync/pages", nc.Pages)
	protected.GET("/notion-sync/runs", nc.Runs)
	protected.GET("/notion-sync/runs/:run_id", nc.Run)
	protected.GET("/notion-sync/runs/:run_id/items", nc.Items)
	protected.POST("/notion-sync/runs", nc.Trigger)
	protected.PATCH("/notion-sync/control", nc.UpdateControl)
	protected.PATCH("/notion-sync/sources/:source_id", nc.UpdateSource)
	protected.PUT("/notion-sync/catalog-bindings/:binding_id", nc.BindCatalog)

	ssc := subsectioncontroller.New(ds)
	protected.GET("/subsections/:section_code", ssc.GetList)
	protected.GET("/subsections/:section_code/:code", ssc.GetOne)
	protected.POST("/subsections", ssc.Create)
	protected.PUT("/subsections/:code", ssc.Update)
	protected.PUT("/subsections/:code/publish", ssc.Publish)
	protected.PUT("/subsections/:code/unpublish", ssc.Unpublish)
	protected.DELETE("/subsections/:code", ssc.Delete)
}
