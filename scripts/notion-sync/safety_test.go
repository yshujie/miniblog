package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/notionsync"
	"github.com/yshujie/miniblog/internal/miniblog/source"
	"github.com/yshujie/miniblog/pkg/db"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type cliSchemaClient struct {
	schemas   int
	forbidden int
	fail      error
}

func (c *cliSchemaClient) RetrieveDataSource(_ context.Context, id string) (source.NotionDataSource, error) {
	c.schemas++
	s := source.NotionDataSource{ID: id, Properties: map[string]source.NotionSchemaProperty{}}
	for name, p := range map[string]source.NotionSchemaProperty{"标题": {ID: "title", Type: "title"}, "博客状态": {ID: "state%3Aid", Type: "select"}, "主题": {ID: "topic", Type: "select"}, "知识点": {ID: "tags", Type: "multi_select"}} {
		p.Name = name
		if name == "博客状态" {
			p.Select.Options = []source.NotionOption{{ID: "s1", Name: "草稿"}, {ID: "s2", Name: "已发布"}, {ID: "s3", Name: "已下架"}, {ID: "s4", Name: "归档"}}
		}
		s.Properties[name] = p
	}
	return s, c.fail
}
func (c *cliSchemaClient) QueryDataSource(context.Context, string, bool, string) (source.NotionQueryResult, error) {
	c.forbidden++
	return source.NotionQueryResult{}, errors.New("forbidden page query")
}
func (c *cliSchemaClient) RetrievePage(context.Context, string) (source.NotionPage, error) {
	c.forbidden++
	return source.NotionPage{}, errors.New("forbidden page retrieve")
}

func sentinelEnvironment(t *testing.T) []string {
	t.Helper()
	sentinels := []string{"credential-token-SENTINEL", "credential-password-SENTINEL", "credential-dsn-SENTINEL", "credential-user-SENTINEL"}
	for i, key := range []string{"MINIBLOG_NOTION_TOKEN", "MYSQL_PASSWORD", "MYSQL_DSN", "MYSQL_USERNAME"} {
		t.Setenv(key, sentinels[i])
	}
	return sentinels
}
func assertNoSentinels(t *testing.T, value string, sentinels []string) {
	t.Helper()
	for _, sentinel := range sentinels {
		if strings.Contains(value, sentinel) {
			t.Fatal("credential sentinel exposed in output")
		}
	}
}
func TestSchemaCheckCLINeverConnectsDBAndReportIsPrivate(t *testing.T) {
	sentinels := sentinelEnvironment(t)
	for _, fails := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "transport_failure"}[fails], func(t *testing.T) {
			client := &cliSchemaClient{}
			if fails {
				client.fail = errors.New(strings.Join(sentinels, " "))
			}
			var out, errout bytes.Buffer
			report := filepath.Join(t.TempDir(), "schema.json")
			if err := os.WriteFile(report, []byte("old"), 0644); err != nil {
				t.Fatal(err)
			}
			code := runWithDependencies([]string{"--mode", "schema_check", "--report", report}, &out, &errout, dependencies{
				client: func(token string) source.SyncNotionClient {
					if token != sentinels[0] {
						t.Fatal("token was not loaded on schema branch")
					}
					return client
				},
				connect: func(*db.MySQLOptions) (*gorm.DB, error) {
					t.Fatal("schema_check connected a database")
					return nil, nil
				},
			})
			if code != map[bool]int{false: 0, true: 1}[fails] || client.schemas != 5 || client.forbidden != 0 {
				t.Fatal(code, errout.String(), client)
			}
			data, err := os.ReadFile(report)
			if err != nil {
				t.Fatal(err)
			}
			info, err := os.Stat(report)
			if err != nil || info.Mode().Perm() != 0600 {
				t.Fatal(info, err)
			}
			assertNoSentinels(t, string(data)+out.String()+errout.String(), sentinels)
			var result notionsync.SchemaCheckResult
			if err = json.Unmarshal(data, &result); err != nil || len(result.Sources) != 5 || result.Complete == fails {
				t.Fatal(result, err)
			}
		})
	}
}
func TestCLIHelpInvalidParametersAndUnsafeDestinationHaveNoSideEffects(t *testing.T) {
	sentinels := sentinelEnvironment(t)
	victim := filepath.Join(t.TempDir(), "victim")
	if err := os.WriteFile(victim, []byte("untouched"), 0644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(filepath.Dir(victim), "report-link")
	if err := os.Symlink(victim, link); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		args []string
		code int
	}{
		{[]string{"-h"}, 0},
		{[]string{"--unknown", sentinels[0]}, 2},
		{[]string{"--timeout", sentinels[0]}, 2},
		{[]string{"--mode", "catalog_prepare"}, 2},
		{[]string{"--mode", "catalog_prepare", "--source-id", notionsync.AllowedSources()[0], "--expected-config-revision", "0"}, 2},
		{[]string{"--mode", "catalog_prepare", "--source-id", "not-allowlisted", "--expected-config-revision", "2"}, 2},
		{[]string{"--mode", "schema_check", "--manual-matches", "unused.json"}, 2},
		{[]string{"--mode", "schema_check", "--confirmations", "unused.json"}, 2},
		{[]string{"--mode", "schema_check", "--allow-notion-write"}, 2},
		{[]string{"--mode", "schema_check", "--report", link}, 2},
	}
	for _, c := range cases {
		var out, errout bytes.Buffer
		code := runWithDependencies(c.args, &out, &errout, dependencies{
			client:  func(string) source.SyncNotionClient { t.Fatal("invalid command constructed client"); return nil },
			connect: func(*db.MySQLOptions) (*gorm.DB, error) { t.Fatal("invalid command connected DB"); return nil, nil },
		})
		if code != c.code {
			t.Fatal(c.args, code, errout.String())
		}
		assertNoSentinels(t, out.String()+errout.String(), sentinels)
	}
	data, err := os.ReadFile(victim)
	if err != nil || string(data) != "untouched" {
		t.Fatal(string(data), err)
	}
}
func TestCLIDatabaseConnectionFailureIsSanitized(t *testing.T) {
	sentinels := sentinelEnvironment(t)
	var out, errout bytes.Buffer
	code := runWithDependencies([]string{"--mode", "status"}, &out, &errout, dependencies{connect: func(*db.MySQLOptions) (*gorm.DB, error) { return nil, errors.New(strings.Join(sentinels, " ")) }})
	if code != 1 || !strings.Contains(errout.String(), "连接数据库失败") {
		t.Fatal(code, errout.String())
	}
	assertNoSentinels(t, out.String()+errout.String(), sentinels)
}
func TestCatalogArgumentsExplicitRevisionAndSourceScope(t *testing.T) {
	id := notionsync.AllowedSources()[0]
	if err := validateCatalogArguments("catalog_prepare", id, 2, true); err != nil {
		t.Fatal(err)
	}
	if err := validateCatalogArguments("catalog_prepare", strings.ToUpper(strings.ReplaceAll(id, "-", "")), 2, true); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"schema_check", "dry_run", "sync", "bootstrap_preview", "bootstrap_apply", "status"} {
		if err := validateCatalogArguments(mode, id, 2, true); err == nil {
			t.Fatal("accepted catalog scope in", mode)
		}
	}
}

func TestConfirmationLimitCannotHideTrailingData(t *testing.T) {
	path := filepath.Join(t.TempDir(), "oversize.json")
	data := []byte(`{"confirmations":[]}` + strings.Repeat(" ", 2<<20) + `{"unexpected":true}`)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := readConfirmations(path); err == nil || !strings.Contains(err.Error(), "2 MiB") {
		t.Fatal("oversized input accepted", err)
	}
}

// Exercise the real CLI branch and Service, not only a success-gate helper.
type cliPreviewClient struct {
	cliSchemaClient
	page source.NotionPage
}

func (c *cliPreviewClient) QueryDataSource(_ context.Context, id string, archived bool, _ string) (source.NotionQueryResult, error) {
	var result source.NotionQueryResult
	result.RequestStatus.Type = "complete"
	if id == c.page.Parent.DataSourceID && !archived {
		result.Results = []source.NotionPage{c.page}
	}
	return result, nil
}
func (c *cliPreviewClient) RetrievePage(context.Context, string) (source.NotionPage, error) {
	return c.page, nil
}

func TestBootstrapPreviewCLISuccessGate(t *testing.T) {
	for _, name := range []string{"page_failure", "run_read_failure", "public_url_missing_candidate"} {
		t.Run(name, func(t *testing.T) {
			sentinels := sentinelEnvironment(t)
			gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
			if err != nil {
				t.Fatal(err)
			}
			sqlDB, err := gdb.DB()
			if err != nil {
				t.Fatal(err)
			}
			sqlDB.SetMaxOpenConns(1)
			t.Cleanup(func() { _ = sqlDB.Close() })
			if err = gdb.AutoMigrate(&model.Module{}, &model.Section{}, &model.Subsection{}, &model.Article{}, &model.NotionSyncControl{}, &model.NotionSyncSource{}, &model.NotionCatalogBinding{}, &model.NotionPageBinding{}, &model.NotionSyncRun{}, &model.NotionSyncRunItem{}); err != nil {
				t.Fatal(err)
			}
			if err = gdb.Create(&model.NotionSyncControl{ID: 1, Paused: true, SourceWritesPaused: true}).Error; err != nil {
				t.Fatal(err)
			}
			if err = gdb.Create(&model.Module{Code: "m1", Title: "Module", Status: 1}).Error; err != nil {
				t.Fatal(err)
			}
			if err = gdb.Create(&model.Section{Code: "s1", Title: "Historical", ModuleCode: "m1", Status: 1}).Error; err != nil {
				t.Fatal(err)
			}
			for _, id := range notionsync.AllowedSources() {
				if err = gdb.Create(&model.NotionSyncSource{ID: id, DataSourceID: id, ModuleCode: "m1", ConfigRevision: 1}).Error; err != nil {
					t.Fatal(err)
				}
			}
			id := notionsync.AllowedSources()[0]
			pageID := "11111111111141118111111111111111"
			page := source.NotionPage{ID: pageID, Object: "page", URL: "https://www.notion.so/" + pageID, Properties: map[string]source.NotionProperty{
				"标题":   {ID: "title", Type: "title", Title: []source.NotionRichText{{PlainText: "Historical title"}}},
				"博客状态": {ID: "state%3Aid", Type: "select"},
				"主题":   {ID: "topic", Type: "select"},
				"知识点":  {ID: "tags", Type: "multi_select"},
			}}
			page.Parent.Type, page.Parent.DataSourceID = "data_source_id", id
			if name == "page_failure" {
				p := page.Properties["标题"]
				p.Title[0].PlainText = strings.Repeat("x", 256)
				page.Properties["标题"] = p
			}
			if err = gdb.Create(&model.Article{ID: 7001, Title: "Historical title", SectionCode: "s1", Status: 2, ExternalLink: page.URL}).Error; err != nil {
				t.Fatal(err)
			}
			if name == "run_read_failure" {
				if err = gdb.Callback().Query().Before("gorm:query").Register("preview_run_read_fail", func(tx *gorm.DB) {
					if tx.Statement.Table == "notion_sync_runs" {
						tx.AddError(errors.New(strings.Join(sentinels, " ")))
					}
				}); err != nil {
					t.Fatal(err)
				}
			}
			client := &cliPreviewClient{page: page}
			var out, errout bytes.Buffer
			code := runWithDependencies([]string{"--mode", "bootstrap_preview"}, &out, &errout, dependencies{
				client:  func(string) source.SyncNotionClient { return client },
				connect: func(*db.MySQLOptions) (*gorm.DB, error) { return gdb, nil },
			})
			want := 1
			if name == "public_url_missing_candidate" {
				want = 0
			}
			if code != want {
				t.Fatalf("actual CLI code=%d want=%d stderr=%s", code, want, errout.String())
			}
			assertNoSentinels(t, out.String()+errout.String(), sentinels)
			var result notionsync.BootstrapPreviewResult
			if err = json.Unmarshal(out.Bytes(), &result); err != nil || result.RunID == "" || len(result.Items) != 1 {
				t.Fatal(result, err)
			}
			if name == "public_url_missing_candidate" && (result.Items[0].PublishBlockReason != "public_url_missing" || result.Items[0].ExpectedFingerprint == "") {
				t.Fatal("public candidate lost", result)
			}
		})
	}
}

func TestPreviewRunRequiresCompletedWithoutFailuresAndAllowsReviewStates(t *testing.T) {
	for _, run := range []*notionsync.RunDTO{nil, {Status: "completed_with_errors"}, {Status: "completed", Counts: notionsync.RunCounts{Failed: 1}}, {Status: "completed", Counts: notionsync.RunCounts{Blocked: 1}}, {Status: "cancelled"}, {Status: "running"}} {
		if err := validatePreviewRun(run); err == nil {
			t.Fatal("accepted incomplete run", run)
		}
	}
	if err := validatePreviewRun(&notionsync.RunDTO{Status: "completed", Counts: notionsync.RunCounts{Pending: 47, Frozen: 12}}); err != nil {
		t.Fatal("normal review state rejected", err)
	}
}
