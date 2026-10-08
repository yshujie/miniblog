package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	drivermysql "github.com/go-sql-driver/mysql"
	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/notionsync"
	"github.com/yshujie/miniblog/internal/miniblog/source"
	"github.com/yshujie/miniblog/pkg/db"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func fixtureReport(t *testing.T) notionsync.SchemaCheckResult {
	t.Helper()
	client := &reportClient{schemas: map[string]source.NotionDataSource{}}
	for i, target := range targets {
		s := source.NotionDataSource{ID: target.SourceID, Properties: map[string]source.NotionSchemaProperty{}}
		for name, p := range map[string]source.NotionSchemaProperty{
			"标题":   {ID: fmt.Sprintf("title-%d", i), Type: "title"},
			"博客状态": {ID: fmt.Sprintf("state%%3Aid-%d", i), Type: "select"},
			"主题":   {ID: fmt.Sprintf("topic-%d", i), Type: "select"},
			"知识点":  {ID: fmt.Sprintf("tags-%d", i), Type: "multi_select"},
			"描述":   {ID: fmt.Sprintf("description-%d", i), Type: "rich_text"},
		} {
			p.Name = name
			if name == "博客状态" {
				for j, n := range []string{"草稿", "已发布", "已下架", "归档", "额外选项"} {
					p.Select.Options = append(p.Select.Options, source.NotionOption{ID: fmt.Sprintf("option-%d-%d", i, j), Name: n})
				}
			}
			if name == "主题" {
				p.Select.Options = []source.NotionOption{{ID: "topic-option", Name: "主题一"}}
			}
			s.Properties[name] = p
		}
		client.schemas[target.SourceID] = s
	}
	r, err := notionsync.CheckSchemas(context.Background(), client)
	if err != nil {
		t.Fatal(err)
	}
	return *r
}
func fixtureInput(t *testing.T) *preparedInput {
	t.Helper()
	client, configs, err := validateReport(fixtureReport(t))
	if err != nil {
		t.Fatal(err)
	}
	revisions := map[string]uint64{}
	for _, target := range targets {
		revisions[target.SourceID] = 0
	}
	return &preparedInput{SchemaHash: "fixture-schema", RevisionsHash: "fixture-revisions", Configs: configs, Revisions: revisions, Client: client}
}
func fixtureDB(t *testing.T) *gorm.DB {
	t.Helper()
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := gdb.DB()
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { sqlDB.Close() })
	seedFixture(t, gdb)
	return gdb
}
func seedFixture(t *testing.T, gdb *gorm.DB) {
	t.Helper()
	if err := gdb.AutoMigrate(&model.Module{}, &model.Section{}, &model.Subsection{}, &model.Article{}, &model.NotionSyncControl{}, &model.NotionSyncSource{}, &model.NotionCatalogBinding{}, &model.NotionPageBinding{}, &model.NotionSyncRun{}, &model.NotionSyncRunItem{}); err != nil {
		t.Fatal(err)
	}
	if err := gdb.Create(&model.NotionSyncControl{ID: 1, Paused: true}).Error; err != nil {
		t.Fatal(err)
	}
	for i, target := range targets {
		if target.ModuleCode != "algorithm" {
			m := model.Module{Code: target.ModuleCode, Title: "Historical " + target.ModuleCode, Sort: 10 + i, Status: model.ModuleStatusNormal}
			if err := gdb.Create(&m).Error; err != nil {
				t.Fatal(err)
			}
			if target.ModuleCode == "database" {
				if err := gdb.Model(&m).UpdateColumn("status", 0).Error; err != nil {
					t.Fatal(err)
				}
			}
		}
		if err := gdb.Create(&model.NotionSyncSource{ID: target.SourceID, DataSourceID: target.SourceID, Label: target.Title, ConfigRevision: 0, Health: "unconfigured"}).Error; err != nil {
			t.Fatal(err)
		}
	}
	for i, code := range []string{"go", "database", "project"} {
		section := model.Section{Code: "section-" + code, ModuleCode: code, Title: "Legacy section", Status: 1, Sort: 3}
		if err := gdb.Create(&section).Error; err != nil {
			t.Fatal(err)
		}
		a := model.Article{ID: uint64(9007199254741000 + i), Title: "Legacy article", Content: "Private fixture body", ExternalLink: "https://fixture.example.test/article", SectionCode: section.Code, Author: "Legacy author", Tags: "old,tags", Status: 2, Pos: 4}
		if err := gdb.Create(&a).Error; err != nil {
			t.Fatal(err)
		}
	}
	stamp := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
	for _, row := range []interface{}{&model.Module{}, &model.Article{}, &model.Section{}} {
		if err := gdb.Session(&gorm.Session{AllowGlobalUpdate: true}).Model(row).UpdateColumns(map[string]interface{}{"created_at": stamp, "updated_at": stamp}).Error; err != nil {
			t.Fatal(err)
		}
	}
}
func snapshot(t *testing.T, gdb *gorm.DB) string {
	t.Helper()
	var modules []model.Module
	var sections []model.Section
	var subs []model.Subsection
	var articles []model.Article
	var sources []model.NotionSyncSource
	var bindings []model.NotionPageBinding
	var catalogs []model.NotionCatalogBinding
	var controls []model.NotionSyncControl
	for _, entry := range []struct {
		value interface{}
		order string
	}{{&modules, "id"}, {&sections, "id"}, {&subs, "id"}, {&articles, "id"}, {&sources, "source_id"}, {&bindings, "page_id"}, {&catalogs, "id"}, {&controls, "id"}} {
		if err := gdb.Order(entry.order).Find(entry.value).Error; err != nil {
			t.Fatal(err)
		}
	}
	// Plain structs' JSON omits source JSON fields, so include them separately.
	var rawSources []map[string]interface{}
	if err := gdb.Table("notion_sync_sources").Order("source_id").Find(&rawSources).Error; err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal([]interface{}{modules, sections, subs, articles, sources, rawSources, bindings, catalogs, controls})
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestReportRejectsIncompleteOrInconsistentSchemas(t *testing.T) {
	cases := map[string]func(*notionsync.SchemaCheckResult){
		"incomplete":        func(r *notionsync.SchemaCheckResult) { r.Complete = false },
		"version":           func(r *notionsync.SchemaCheckResult) { r.NotionVersion = "old" },
		"missing fifth":     func(r *notionsync.SchemaCheckResult) { r.Sources = r.Sources[:4] },
		"duplicate source":  func(r *notionsync.SchemaCheckResult) { r.Sources[4] = r.Sources[0] },
		"actual identity":   func(r *notionsync.SchemaCheckResult) { r.Sources[0].ActualDataSourceID = targets[1].SourceID },
		"unapproved source": func(r *notionsync.SchemaCheckResult) { r.Sources[0].SourceID = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa" },
		"invalid status":    func(r *notionsync.SchemaCheckResult) { r.Sources[0].Status = "invalid" },
		"error":             func(r *notionsync.SchemaCheckResult) { r.Sources[0].Error = "read failed" },
		"forged ID":         func(r *notionsync.SchemaCheckResult) { r.Sources[0].Config.TitlePropertyID = "not-in-schema" },
		"forged state":      func(r *notionsync.SchemaCheckResult) { r.Sources[0].Config.StateOptionIDs["draft"] = "not-in-schema" },
		"wrong type": func(r *notionsync.SchemaCheckResult) {
			for i := range r.Sources[0].Properties {
				if r.Sources[0].Properties[i].Name == "知识点" {
					r.Sources[0].Properties[i].Type = "rich_text"
				}
			}
		},
		"duplicate decoded ID": func(r *notionsync.SchemaCheckResult) {
			s := &r.Sources[0]
			s.Properties = append(s.Properties, notionsync.SchemaPropertyDTO{ID: strings.ReplaceAll(s.Config.StatePropertyID, "%3A", ":"), Name: "Duplicate", Type: "rich_text"})
		},
		"duplicate field name": func(r *notionsync.SchemaCheckResult) {
			r.Sources[0].Properties = append(r.Sources[0].Properties, notionsync.SchemaPropertyDTO{ID: "second-title", Name: "标题", Type: "title"})
		},
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			r := fixtureReport(t)
			change(&r)
			if _, _, err := validateReport(r); err == nil {
				t.Fatal("accepted inconsistent report")
			}
		})
	}
	_, configs, err := validateReport(fixtureReport(t))
	if err != nil || !strings.Contains(configs[targets[0].SourceID].StatePropertyID, "%3A") {
		t.Fatal("opaque IDs or additional state option were rejected")
	}
}

func TestReviewedSchemaReportWhenProvided(t *testing.T) {
	path := os.Getenv("MINIBLOG_ROLLOUT_SCHEMA_TEST_REPORT")
	if path == "" {
		t.Skip("set MINIBLOG_ROLLOUT_SCHEMA_TEST_REPORT to check a private schema-only report offline")
	}
	var report notionsync.SchemaCheckResult
	if _, err := readPrivateJSON(path, &report); err != nil {
		t.Fatal("reviewed report could not be read safely")
	}
	if _, configs, err := validateReport(report); err != nil || len(configs) != 5 {
		t.Fatal("reviewed five-source schema report validation failed")
	}
}

func TestPreflightIsReadOnlyAndApplyPreservesHistory(t *testing.T) {
	gdb, in := fixtureDB(t), fixtureInput(t)
	before := snapshot(t, gdb)
	r, err := execute(context.Background(), gdb, in, false)
	if err != nil || r.Outcome != "ready" || len(r.Sources) != 5 || r.Before.PublicCount != 2 {
		t.Fatalf("preflight failed: %v", err)
	}
	if snapshot(t, gdb) != before {
		t.Fatal("preflight wrote data")
	}
	var beforeModules []model.Module
	var beforeArticles []model.Article
	var beforeSections []model.Section
	gdb.Order("id").Find(&beforeModules)
	gdb.Order("id").Find(&beforeArticles)
	gdb.Order("id").Find(&beforeSections)
	r, err = execute(context.Background(), gdb, in, true)
	if err != nil || r.Outcome != "applied" || !reflect.DeepEqual(r.Before, r.After) {
		t.Fatalf("apply failed: %v", err)
	}
	var algorithm model.Module
	if err := gdb.Where("code = ?", "algorithm").First(&algorithm).Error; err != nil || algorithm.Status != 2 || algorithm.Sort != 15 {
		t.Fatal("algorithm not inactive at tail")
	}
	for _, before := range beforeModules {
		var after model.Module
		if err := gdb.Where("id = ?", before.ID).First(&after).Error; err != nil {
			t.Fatal(err)
		}
		if after.ID != before.ID || after.Code != before.Code || after.Status != before.Status || after.Sort != before.Sort || !after.CreatedAt.Equal(before.CreatedAt) {
			t.Fatal("changed existing module fields")
		}
		if before.Code == "database" || before.Code == "project" {
			if after.Title != map[string]string{"database": "数据库", "project": "项目开发"}[before.Code] {
				t.Fatal("rename missing")
			}
		} else if !reflect.DeepEqual(before, after) {
			t.Fatal("unrequested module changed")
		}
	}
	var afterArticles []model.Article
	var afterSections []model.Section
	gdb.Order("id").Find(&afterArticles)
	gdb.Order("id").Find(&afterSections)
	if !reflect.DeepEqual(beforeArticles, afterArticles) || !reflect.DeepEqual(beforeSections, afterSections) {
		t.Fatal("article/body/author/status/position/timestamps or sections changed")
	}
	var control model.NotionSyncControl
	gdb.First(&control, 1)
	if !control.Paused || control.BaselineFrozen || control.SourceWritesPaused || control.LeaseEpoch != 0 {
		t.Fatal("initialization changed maintenance or baseline")
	}
	var sources []model.NotionSyncSource
	gdb.Find(&sources)
	for _, s := range sources {
		if s.Enabled || s.ConfigRevision != 1 {
			t.Fatal("source enabled or revision incorrect")
		}
		in.Revisions[s.ID] = 1
	}
	beforeRerun := snapshot(t, gdb)
	r, err = execute(context.Background(), gdb, in, true)
	if err != nil || r.Outcome != "applied" || snapshot(t, gdb) != beforeRerun {
		t.Fatalf("exact repeat not idempotent: %v", err)
	}
}

func TestEveryGateRejectsWithoutWrites(t *testing.T) {
	for name, alter := range map[string]func(*gorm.DB, *preparedInput) error{
		"not paused": func(db *gorm.DB, _ *preparedInput) error {
			return db.Model(&model.NotionSyncControl{}).Where("id = 1").Update("paused", false).Error
		},
		"baseline frozen": func(db *gorm.DB, _ *preparedInput) error {
			return db.Model(&model.NotionSyncControl{}).Where("id = 1").Update("baseline_frozen", true).Error
		},
		"current run": func(db *gorm.DB, _ *preparedInput) error {
			return db.Model(&model.NotionSyncControl{}).Where("id = 1").Update("current_run_id", "fixture-run").Error
		},
		"valid lease": func(db *gorm.DB, _ *preparedInput) error {
			return db.Model(&model.NotionSyncControl{}).Where("id = 1").Updates(map[string]interface{}{"lease_owner": "fixture", "lease_until": time.Now().UTC().Add(time.Hour)}).Error
		},
		"managed exists": func(db *gorm.DB, _ *preparedInput) error {
			return db.Create(&model.NotionPageBinding{PageID: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", SourceID: targets[0].SourceID, ManagementState: model.NotionManagementManaged}).Error
		},
		"source enabled": func(db *gorm.DB, _ *preparedInput) error {
			return db.Model(&model.NotionSyncSource{}).Where("source_id = ?", targets[4].SourceID).Update("enabled", true).Error
		},
		"fifth stale revision": func(_ *gorm.DB, in *preparedInput) error { in.Revisions[targets[4].SourceID] = 99; return nil },
		"wrong existing mapping": func(db *gorm.DB, _ *preparedInput) error {
			return db.Model(&model.NotionSyncSource{}).Where("source_id = ?", targets[1].SourceID).Update("module_code", "project").Error
		},
		"null module status": func(db *gorm.DB, _ *preparedInput) error {
			return db.Model(&model.Module{}).Where("code = ?", "database").UpdateColumn("status", nil).Error
		},
		"algorithm active": func(db *gorm.DB, _ *preparedInput) error {
			return db.Create(&model.Module{Code: "algorithm", Title: "数据结构&算法", Status: 1, Sort: 20}).Error
		},
		"missing fifth": func(db *gorm.DB, _ *preparedInput) error {
			return db.Delete(&model.NotionSyncSource{}, "source_id = ?", targets[4].SourceID).Error
		},
	} {
		t.Run(name, func(t *testing.T) {
			gdb, in := fixtureDB(t), fixtureInput(t)
			if err := alter(gdb, in); err != nil {
				t.Fatal(err)
			}
			before := snapshot(t, gdb)
			if _, err := execute(context.Background(), gdb, in, true); err == nil {
				t.Fatal("unsafe apply accepted")
			}
			if snapshot(t, gdb) != before {
				t.Fatal("blocked apply changed database")
			}
		})
	}
}

func TestFifthSourceFailureRollsBackAllEarlierChanges(t *testing.T) {
	gdb, in := fixtureDB(t), fixtureInput(t)
	query := "CREATE TRIGGER fail_project BEFORE UPDATE ON notion_sync_sources WHEN NEW.source_id = '" + targets[4].SourceID + "' BEGIN SELECT RAISE(ABORT, 'private fixture driver error'); END"
	if err := gdb.Exec(query).Error; err != nil {
		t.Fatal(err)
	}
	before := snapshot(t, gdb)
	if _, err := execute(context.Background(), gdb, in, true); err == nil || !strings.Contains(err.Error(), "private fixture driver error") {
		t.Fatal("did not reach the fifth-source failure fixture")
	}
	if snapshot(t, gdb) != before {
		t.Fatal("algorithm, renames or earlier source configs escaped rollback")
	}
}

func privateJSON(t *testing.T, path string, value interface{}) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
}
func fixtureFiles(t *testing.T) (string, string, string) {
	t.Helper()
	dir := t.TempDir()
	schema, rev, report := filepath.Join(dir, "schema.json"), filepath.Join(dir, "revisions.json"), filepath.Join(dir, "result.json")
	privateJSON(t, schema, fixtureReport(t))
	privateJSON(t, rev, revisionsInput{ExpectedConfigRevisions: fixtureInput(t).Revisions})
	return schema, rev, report
}
func TestPrivateStrictInputsAndSafeReport(t *testing.T) {
	schema, rev, report := fixtureFiles(t)
	in, err := loadInput(schema, rev)
	if err != nil || len(in.SchemaHash) != 64 || len(in.Revisions) != 5 {
		t.Fatal("valid private report rejected")
	}
	for _, text := range []string{`{"expected_config_revisions":{},"module_code":"arbitrary"}`, `{"expected_config_revisions":{},"expected_config_revisions":{}}`, `{} {}`, `{"expected_config_revisions":{"arbitrary":0}}`} {
		if err := os.WriteFile(rev, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := loadInput(schema, rev); err == nil {
			t.Fatal("accepted ambiguous or incomplete manifest")
		}
	}
	privateJSON(t, rev, revisionsInput{ExpectedConfigRevisions: fixtureInput(t).Revisions})
	if err := os.Chmod(schema, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadInput(schema, rev); err == nil {
		t.Fatal("accepted public schema file")
	}
	os.Chmod(schema, 0600)
	link := filepath.Join(t.TempDir(), "schema-link")
	if err := os.Symlink(schema, link); err != nil {
		t.Fatal(err)
	}
	if _, err := loadInput(link, rev); err == nil {
		t.Fatal("followed schema symlink")
	}
	t.Setenv("MINIBLOG_NOTION_SYNC_ENABLED", "false")
	t.Setenv("MYSQL_DSN", "fixture:private-password@tcp(127.0.0.1:3306)/fixture?parseTime=true")
	var out, errout bytes.Buffer
	gdb := fixtureDB(t)
	code := run([]string{"--schema-report", schema, "--expected-revisions", rev, "--report", report}, &out, &errout, func(*db.MySQLOptions) (*gorm.DB, error) { return gdb, nil })
	if code != 0 || !strings.Contains(out.String(), "mode=preflight outcome=ready") || errout.Len() != 0 {
		t.Fatalf("CLI failed: %d %s", code, errout.String())
	}
	info, err := os.Stat(report)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatal("report not private")
	}
	data, err := os.ReadFile(report)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"private-password", "Private fixture body", "Legacy author", "fixture.example.test", "source_writes_paused", "property_mapping_json"} {
		if strings.Contains(string(data)+out.String()+errout.String(), forbidden) {
			t.Fatal("unsafe report/output")
		}
	}
}

func TestCLIRejectsWriteModesCredentialsAndRuntimeBeforeConnecting(t *testing.T) {
	t.Setenv("MYSQL_DSN", "fixture:do-not-print@tcp(private.invalid:3306)/fixture")
	schema, rev, report := fixtureFiles(t)
	base := []string{"--schema-report", schema, "--expected-revisions", rev, "--report", report}
	for _, args := range [][]string{{"--db-password", "do-not-print"}, {"--mode", "bootstrap_apply"}, {"--module-code", "arbitrary"}, {"--allow-notion-write"}, {"--help"}} {
		var out, errout bytes.Buffer
		calls := 0
		code := run(append(append([]string{}, base...), args...), &out, &errout, func(*db.MySQLOptions) (*gorm.DB, error) { calls++; return nil, errors.New("driver-secret") })
		if calls != 0 || (code != 2 && code != 0) || strings.Contains(out.String()+errout.String(), "do-not-print") {
			t.Fatal("unsafe CLI flag or help")
		}
	}
	for _, value := range []string{"", "true", "1", "yes"} {
		t.Setenv("MINIBLOG_NOTION_SYNC_ENABLED", value)
		var out, errout bytes.Buffer
		calls := 0
		if code := run(base, &out, &errout, func(*db.MySQLOptions) (*gorm.DB, error) { calls++; return nil, nil }); code != 2 || calls != 0 {
			t.Fatal("daemon gate missing")
		}
	}
	t.Setenv("MINIBLOG_NOTION_SYNC_ENABLED", "false")
	var out, errout bytes.Buffer
	if code := run(base, &out, &errout, func(*db.MySQLOptions) (*gorm.DB, error) {
		return nil, errors.New("driver-secret do-not-print private.invalid")
	}); code != 1 || strings.Contains(errout.String(), "driver-secret") || strings.Contains(errout.String(), "private.invalid") {
		t.Fatal("echoed connection details")
	}
}

func TestCLIRejectsAliasedReportOverInputBeforeConnecting(t *testing.T) {
	schema, revisions, _ := fixtureFiles(t)
	before, err := os.ReadFile(schema)
	if err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(t.TempDir(), "alias-directory")
	if err := os.Symlink(filepath.Dir(schema), alias); err != nil {
		t.Fatal(err)
	}
	var out, errout bytes.Buffer
	calls := 0
	code := run([]string{"--schema-report", schema, "--expected-revisions", revisions, "--report", filepath.Join(alias, filepath.Base(schema))}, &out, &errout, func(*db.MySQLOptions) (*gorm.DB, error) { calls++; return nil, nil })
	after, err := os.ReadFile(schema)
	if err != nil || code != 2 || calls != 0 || !bytes.Equal(before, after) || !strings.Contains(errout.String(), "invalid_report_destination") {
		t.Fatal("aliased output overwrote or accepted the reviewed input")
	}
}

func TestDatabaseEnvironmentAliasAndCredentialsStayPrivate(t *testing.T) {
	for _, key := range []string{"MYSQL_DSN", "MYSQL_HOST", "MYSQL_PORT", "MYSQL_DATABASE", "MYSQL_USERNAME", "MYSQL_PASSWORD", "MINIBLOG_DATABASE_HOST", "MINIBLOG_DATABASE_PORT", "MINIBLOG_DATABASE_DBNAME", "MINIBLOG_DATABASE_USERNAME", "MINIBLOG_DATABASE_PASSWORD"} {
		t.Setenv(key, "")
	}
	if _, err := environmentDBOptions(); err == nil {
		t.Fatal("allowed implicit/default credentials")
	}
	for _, pair := range [][2]string{{"MYSQL_HOST", "127.0.0.1"}, {"MYSQL_PORT", "3306"}, {"MYSQL_DATABASE", "fixture"}, {"MYSQL_USERNAME", "fixture"}, {"MYSQL_PASSWORD", "private-password"}} {
		t.Setenv(pair[0], pair[1])
	}
	// Both families are explicit in Compose. Matching values are accepted, conflicting ones block.
	for _, pair := range [][2]string{{"MINIBLOG_DATABASE_HOST", "127.0.0.1"}, {"MINIBLOG_DATABASE_PORT", "3306"}, {"MINIBLOG_DATABASE_DBNAME", "fixture"}, {"MINIBLOG_DATABASE_USERNAME", "fixture"}, {"MINIBLOG_DATABASE_PASSWORD", "private-password"}} {
		t.Setenv(pair[0], pair[1])
	}
	opts, err := environmentDBOptions()
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := drivermysql.ParseDSN(opts.DSN)
	if err != nil || cfg.Passwd != "private-password" || cfg.Loc != time.UTC || !cfg.ParseTime {
		t.Fatal("incorrect database configuration")
	}
	t.Setenv("MINIBLOG_DATABASE_HOST", "different.invalid")
	if _, err := environmentDBOptions(); reason(err) != "database_environment_alias_conflict" {
		t.Fatal("ambiguous DB target accepted")
	}
}

type driverLogProbe struct{ calls int }

func (p *driverLogProbe) Print(...interface{}) { p.calls++ }

type failingFixtureConn struct{}

func (failingFixtureConn) Read([]byte) (int, error) {
	return 0, errors.New("private.invalid private-password raw driver error")
}
func (failingFixtureConn) Write([]byte) (int, error) {
	return 0, errors.New("private.invalid private-password raw driver error")
}
func (failingFixtureConn) Close() error                     { return nil }
func (failingFixtureConn) LocalAddr() net.Addr              { return fixtureAddress("local.invalid") }
func (failingFixtureConn) RemoteAddr() net.Addr             { return fixtureAddress("private.invalid") }
func (failingFixtureConn) SetDeadline(time.Time) error      { return nil }
func (failingFixtureConn) SetReadDeadline(time.Time) error  { return nil }
func (failingFixtureConn) SetWriteDeadline(time.Time) error { return nil }

type fixtureAddress string

func (fixtureAddress) Network() string  { return "fixture" }
func (v fixtureAddress) String() string { return string(v) }

func TestCLISuppressesIndependentMySQLDriverLogger(t *testing.T) {
	// The connector uses an in-memory failing transport, never a real socket.
	const network = "rollout-init-fixture"
	drivermysql.RegisterDialContext(network, func(context.Context, string) (net.Conn, error) { return failingFixtureConn{}, nil })
	cfg := drivermysql.NewConfig()
	cfg.Net, cfg.Addr, cfg.DBName = network, "private.invalid", "fixture"
	connector, err := drivermysql.NewConnector(cfg)
	if err != nil {
		t.Fatal("fixture connector failed")
	}
	connect := func(*db.MySQLOptions) (*gorm.DB, error) {
		sqlDB := sql.OpenDB(connector)
		defer sqlDB.Close()
		return nil, sqlDB.PingContext(context.Background())
	}
	probe := &driverLogProbe{}
	_ = drivermysql.SetLogger(probe)
	t.Cleanup(func() { _ = drivermysql.SetLogger(quietDriverLogger{}) })
	if _, err := connect(nil); err == nil || probe.calls == 0 {
		t.Fatal("fixture did not exercise driver error logging")
	}
	initialCalls := probe.calls
	t.Setenv("MINIBLOG_NOTION_SYNC_ENABLED", "false")
	t.Setenv("MYSQL_DSN", cfg.FormatDSN())
	schema, revisions, report := fixtureFiles(t)
	var out, errout bytes.Buffer
	code := run([]string{"--schema-report", schema, "--expected-revisions", revisions, "--report", report}, &out, &errout, connect)
	if code != 1 || probe.calls != initialCalls || !strings.Contains(errout.String(), "database_connection_failed") {
		t.Fatal("CLI did not silence the independent driver logger")
	}
	if strings.Contains(out.String()+errout.String(), "private.invalid") || strings.Contains(out.String()+errout.String(), "private-password") {
		t.Fatal("raw driver details were printed")
	}
}

// This test is opt-in and refuses every database except the dedicated loopback fixture.
func TestMySQLInitializerAtomicAndPreservesContent(t *testing.T) {
	dsn := os.Getenv("MINIBLOG_ROLLOUT_INIT_TEST_DSN")
	if dsn == "" {
		t.Skip("set MINIBLOG_ROLLOUT_INIT_TEST_DSN for the dedicated local fixture")
	}
	cfg, err := drivermysql.ParseDSN(dsn)
	if err != nil {
		t.Fatal("invalid test database configuration")
	}
	host, _, err := net.SplitHostPort(cfg.Addr)
	if err != nil || cfg.Net != "tcp" || (host != "127.0.0.1" && host != "::1" && host != "localhost") || cfg.DBName != "miniblog_rollout_init_test" {
		t.Fatal("refusing fixture reset outside dedicated loopback database")
	}
	cfg.ParseTime, cfg.Loc = true, time.UTC
	gdb, err := gorm.Open(mysql.Open(cfg.FormatDSN()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal("cannot connect to dedicated fixture")
	}
	sqlDB, _ := gdb.DB()
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { sqlDB.Close() })
	for _, table := range []string{"notion_sync_run_items", "notion_sync_runs", "notion_page_bindings", "notion_catalog_bindings", "notion_sync_sources", "notion_sync_control", "article", "subsection", "section", "module"} {
		if err := gdb.Exec("DROP TABLE IF EXISTS " + table).Error; err != nil {
			t.Fatal("fixture reset failed")
		}
	}
	seedFixture(t, gdb)
	in := fixtureInput(t)
	before := snapshot(t, gdb)
	if err := gdb.Exec("ALTER TABLE module ENGINE=MyISAM").Error; err != nil {
		t.Fatal("cannot install nontransactional fixture")
	}
	if _, err := execute(context.Background(), gdb, in, true); reason(err) != "transactional_schema_required" || snapshot(t, gdb) != before {
		t.Fatal("nontransactional table passed the atomicity gate")
	}
	if err := gdb.Exec("ALTER TABLE module ENGINE=InnoDB").Error; err != nil {
		t.Fatal("cannot restore transactional fixture")
	}
	trigger := "CREATE TRIGGER fail_project BEFORE UPDATE ON notion_sync_sources FOR EACH ROW BEGIN IF NEW.source_id = '" + targets[4].SourceID + "' THEN SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'private fixture error'; END IF; END"
	if err := gdb.Exec(trigger).Error; err != nil {
		t.Fatal("cannot install failure fixture")
	}
	if _, err := execute(context.Background(), gdb, in, true); err == nil || !strings.Contains(err.Error(), "private fixture error") || snapshot(t, gdb) != before {
		t.Fatal("MySQL fifth-source failure escaped atomic rollback")
	}
	if err := gdb.Exec("DROP TRIGGER fail_project").Error; err != nil {
		t.Fatal("cannot remove failure fixture")
	}
	r, err := execute(context.Background(), gdb, in, true)
	if err != nil || !reflect.DeepEqual(r.Before, r.After) || r.Before.PublicCount != 2 {
		t.Fatalf("MySQL apply failed: %s", reason(err))
	}
	var database, algorithm model.Module
	gdb.Where("code = ?", "database").First(&database)
	gdb.Where("code = ?", "algorithm").First(&algorithm)
	if database.Status != 0 || database.Sort != 12 || algorithm.Status != 2 || algorithm.Sort != 15 {
		t.Fatal("legacy inactivity/tail insertion changed")
	}
	var article model.Article
	gdb.Where("id = ?", uint64(9007199254741001)).First(&article)
	if article.Status != 2 || article.Author != "Legacy author" || article.Content != "Private fixture body" || article.Pos != 4 || article.UpdatedAt.Year() != 2020 {
		t.Fatal("MySQL article history changed")
	}
	for _, target := range targets {
		in.Revisions[target.SourceID] = 1
	}
	before = snapshot(t, gdb)
	if _, err := execute(context.Background(), gdb, in, true); err != nil || snapshot(t, gdb) != before {
		t.Fatal("MySQL repeat mutated data")
	}
}
