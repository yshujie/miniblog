// notion-rollout-init initializes only the five approved source/module mappings.
// It is offline with respect to Notion and never enables publishing or sync.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"syscall"
	"time"

	drivermysql "github.com/go-sql-driver/mysql"
	"github.com/yshujie/miniblog/internal/miniblog/biz/catalog"
	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/notionsync"
	"github.com/yshujie/miniblog/internal/miniblog/source"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"github.com/yshujie/miniblog/pkg/db"
	"github.com/yshujie/miniblog/scripts/internal/safereport"
	"golang.org/x/sys/unix"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

type target struct{ SourceID, ModuleCode, Title string }

// Order is intentional: the project source is the fifth write in rollback tests.
var targets = []target{
	{"2bf330bd-ddf1-80a6-aa49-000bbd1e154b", "go", "Go"},
	{"d579f619-4f1c-4e7e-9593-ab6529fc63d0", "ddd", "DDD"},
	{"cfa962f3-12be-40aa-95eb-9122186bd4ba", "database", "数据库"},
	{"03d24108-5172-4e8f-8cfd-e3bd52e437af", "algorithm", "数据结构&算法"},
	{"d13097e3-b78a-4b6e-8848-979749d4b2f3", "project", "项目开发"},
}

type revisionsInput struct {
	ExpectedConfigRevisions map[string]uint64 `json:"expected_config_revisions"`
}
type preparedInput struct {
	SchemaHash, RevisionsHash string
	Configs                   map[string]notionsync.SourceConfig
	Revisions                 map[string]uint64
	Client                    *reportClient
}
type modulePlan struct {
	Code        string `json:"code"`
	Action      string `json:"action"`
	ID          string `json:"id,omitempty"`
	BeforeTitle string `json:"before_title,omitempty"`
	Title       string `json:"title"`
	Status      int    `json:"status"`
	Sort        int    `json:"sort"`
}
type sourcePlan struct {
	SourceID         string                  `json:"source_id"`
	ModuleCode       string                  `json:"module_code"`
	BeforeModuleCode string                  `json:"before_module_code"`
	Action           string                  `json:"action"`
	ExpectedRevision uint64                  `json:"expected_config_revision"`
	NextRevision     uint64                  `json:"next_config_revision"`
	Enabled          bool                    `json:"enabled"`
	Config           notionsync.SourceConfig `json:"config"`
}
type contentProof struct {
	Articles      int64  `json:"articles"`
	Sections      int64  `json:"sections"`
	Subsections   int64  `json:"subsections"`
	PageBindings  int64  `json:"page_bindings"`
	CatalogLinks  int64  `json:"catalog_bindings"`
	PublicCount   int    `json:"public_article_count"`
	PublicIDsHash string `json:"public_article_ids_sha256"`
}
type result struct {
	Mode          string        `json:"mode"`
	Outcome       string        `json:"outcome"`
	Reason        string        `json:"reason,omitempty"`
	SchemaHash    string        `json:"schema_report_sha256"`
	RevisionsHash string        `json:"revisions_sha256"`
	Modules       []modulePlan  `json:"modules,omitempty"`
	Sources       []sourcePlan  `json:"sources,omitempty"`
	Before        *contentProof `json:"before,omitempty"`
	After         *contentProof `json:"after,omitempty"`
}
type blocked string

func (b blocked) Error() string { return string(b) }
func reason(err error) string {
	var b blocked
	if errors.As(err, &b) {
		return string(b)
	}
	return "database_operation_failed"
}

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr, db.NewMySQL)) }

type quietDriverLogger struct{}

func (quietDriverLogger) Print(...interface{}) {}

// Connection injection is for local fixtures; runtime accepts credentials only from env.
func run(args []string, out, errout io.Writer, connect func(*db.MySQLOptions) (*gorm.DB, error)) int {
	fs := flag.NewFlagSet("notion-rollout-init", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	schemaPath := fs.String("schema-report", "", "本轮完整 schema_check JSON 文件（私有常规文件）")
	revisionPath := fs.String("expected-revisions", "", "五库 expected_config_revisions JSON 文件（私有常规文件）")
	reportPath := fs.String("report", "notion-rollout-init-report.json", "0600 审核结果文件，不支持标准输出")
	apply := fs.Bool("apply", false, "显式执行固定目录和来源配置；默认仅读预检")
	timeout := fs.Duration("timeout", time.Minute, "数据库预检／事务最大时间")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fs.SetOutput(errout)
			fs.PrintDefaults()
			return 0
		}
		fmt.Fprintln(errout, "blocked: invalid_arguments")
		return 2
	}
	if fs.NArg() != 0 || *schemaPath == "" || *revisionPath == "" || *timeout <= 0 || *reportPath == "-" {
		fmt.Fprintln(errout, "blocked: invalid_arguments")
		return 2
	}
	for _, inputPath := range []string{*schemaPath, *revisionPath} {
		a, ea := filepath.Abs(inputPath)
		b, eb := filepath.Abs(*reportPath)
		inputInfo, inputErr := os.Stat(inputPath)
		reportInfo, reportErr := os.Stat(*reportPath)
		sameFile := inputErr == nil && reportErr == nil && os.SameFile(inputInfo, reportInfo)
		if ea != nil || eb != nil || a == b || sameFile {
			fmt.Fprintln(errout, "blocked: invalid_report_destination")
			return 2
		}
	}
	if safereport.CheckDestination(*reportPath) != nil {
		fmt.Fprintln(errout, "blocked: invalid_report_destination")
		return 2
	}
	in, err := loadInput(*schemaPath, *revisionPath)
	if err != nil {
		fmt.Fprintln(errout, "blocked: "+reason(err))
		return 2
	}
	if v := strings.ToLower(strings.TrimSpace(os.Getenv("MINIBLOG_NOTION_SYNC_ENABLED"))); v != "false" && v != "0" {
		fmt.Fprintln(errout, "blocked: runtime_sync_must_be_explicitly_disabled")
		return 2
	}
	opts, err := environmentDBOptions()
	if err != nil {
		fmt.Fprintln(errout, "blocked: "+reason(err))
		return 2
	}
	// The driver has a separate logger which can include raw hosts/network errors.
	_ = drivermysql.SetLogger(quietDriverLogger{})
	gdb, err := connect(opts)
	if err != nil {
		fmt.Fprintln(errout, "blocked: database_connection_failed")
		return 1
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		fmt.Fprintln(errout, "blocked: database_handle_unavailable")
		return 1
	}
	defer sqlDB.Close()
	sqlDB.SetMaxOpenConns(1)
	gdb = gdb.Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)})
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, *timeout)
	defer cancel()
	r, err := execute(ctx, gdb, in, *apply)
	if err != nil {
		r.Outcome, r.Reason = "blocked", reason(err)
	}
	encoded, encodeErr := json.MarshalIndent(r, "", "  ")
	if encodeErr != nil || safereport.Write(*reportPath, append(encoded, '\n')) != nil {
		if r.Outcome == "applied" {
			fmt.Fprintln(errout, "apply committed; report_write_failed; inspect database before any retry")
		} else {
			fmt.Fprintln(errout, "blocked: report_write_failed")
		}
		return 1
	}
	if err != nil {
		fmt.Fprintln(errout, "blocked: "+r.Reason)
		return 1
	}
	fmt.Fprintf(out, "mode=%s outcome=%s sources=5 schema_sha256=%s\n", r.Mode, r.Outcome, r.SchemaHash)
	return 0
}

func loadInput(schemaPath, revisionPath string) (*preparedInput, error) {
	var report notionsync.SchemaCheckResult
	schemaBytes, err := readPrivateJSON(schemaPath, &report)
	if err != nil {
		return nil, blocked("invalid_schema_report_file")
	}
	client, configs, err := validateReport(report)
	if err != nil {
		return nil, err
	}
	var revisions revisionsInput
	revisionBytes, err := readPrivateJSON(revisionPath, &revisions)
	if err != nil || len(revisions.ExpectedConfigRevisions) != len(targets) {
		return nil, blocked("invalid_expected_revisions")
	}
	for _, t := range targets {
		if _, ok := revisions.ExpectedConfigRevisions[t.SourceID]; !ok {
			return nil, blocked("invalid_expected_revisions")
		}
	}
	return &preparedInput{SchemaHash: digest(schemaBytes), RevisionsHash: digest(revisionBytes), Configs: configs, Revisions: revisions.ExpectedConfigRevisions, Client: client}, nil
}
func digest(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }

// Input files may include private schema metadata, but never credentials or bodies.
func readPrivateJSON(path string, value interface{}) ([]byte, error) {
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	f := os.NewFile(uintptr(fd), path)
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 || info.Size() > 4<<20 {
		return nil, errors.New("unsafe input")
	}
	if stat, ok := info.Sys().(*syscall.Stat_t); !ok || int(stat.Uid) != os.Getuid() {
		return nil, errors.New("unsafe input")
	}
	data, err := io.ReadAll(io.LimitReader(f, (4<<20)+1))
	if err != nil || len(data) > 4<<20 {
		return nil, errors.New("invalid input")
	}
	if err := uniqueJSONKeys(data); err != nil {
		return nil, err
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(value); err != nil {
		return nil, err
	}
	if err := d.Decode(new(interface{})); err != io.EOF {
		return nil, errors.New("extra JSON")
	}
	return data, nil
}

// encoding/json otherwise accepts ambiguous duplicate object keys.
func uniqueJSONKeys(data []byte) error {
	d := json.NewDecoder(bytes.NewReader(data))
	var walk func(int) error
	walk = func(depth int) error {
		if depth > 64 {
			return errors.New("JSON nesting too deep")
		}
		token, err := d.Token()
		if err != nil {
			return err
		}
		open, ok := token.(json.Delim)
		if !ok {
			return nil
		}
		if open != '{' && open != '[' {
			return errors.New("invalid JSON delimiter")
		}
		keys := map[string]bool{}
		for d.More() {
			if open == '{' {
				key, err := d.Token()
				if err != nil {
					return err
				}
				text, ok := key.(string)
				if !ok || keys[text] {
					return errors.New("duplicate JSON key")
				}
				keys[text] = true
			}
			if err := walk(depth + 1); err != nil {
				return err
			}
		}
		_, err = d.Token()
		return err
	}
	if err := walk(0); err != nil {
		return err
	}
	if _, err := d.Token(); err != io.EOF {
		return errors.New("extra JSON")
	}
	return nil
}

type reportClient struct {
	schemas map[string]source.NotionDataSource
}

func (c *reportClient) RetrieveDataSource(_ context.Context, id string) (source.NotionDataSource, error) {
	v, ok := c.schemas[id]
	if !ok {
		return v, errors.New("offline schema unavailable")
	}
	return v, nil
}
func (*reportClient) QueryDataSource(context.Context, string, bool, string) (source.NotionQueryResult, error) {
	return source.NotionQueryResult{}, errors.New("offline initializer forbids page queries")
}
func (*reportClient) RetrievePage(context.Context, string) (source.NotionPage, error) {
	return source.NotionPage{}, errors.New("offline initializer forbids page reads")
}

func validateReport(report notionsync.SchemaCheckResult) (*reportClient, map[string]notionsync.SourceConfig, error) {
	if !report.Complete || report.NotionVersion != source.NotionSyncVersion || len(report.Sources) != len(targets) {
		return nil, nil, blocked("schema_report_incomplete")
	}
	allowed := map[string]bool{}
	for _, t := range targets {
		allowed[t.SourceID] = true
	}
	client := &reportClient{schemas: map[string]source.NotionDataSource{}}
	configs := map[string]notionsync.SourceConfig{}
	for _, s := range report.Sources {
		if !allowed[s.SourceID] || s.ActualDataSourceID != s.SourceID || s.Status != "valid" || s.Error != "" || s.Config == nil {
			return nil, nil, blocked("schema_report_identity_or_status_invalid")
		}
		if _, exists := client.schemas[s.SourceID]; exists {
			return nil, nil, blocked("schema_report_duplicate_source")
		}
		schema := source.NotionDataSource{ID: s.ActualDataSourceID, Properties: map[string]source.NotionSchemaProperty{}}
		for i, p := range s.Properties {
			v := source.NotionSchemaProperty{ID: p.ID, Name: p.Name, Type: p.Type}
			switch p.Type {
			case "select":
				v.Select.Options = p.Options
			case "multi_select":
				v.MultiSelect.Options = p.Options
			default:
				if len(p.Options) != 0 {
					return nil, nil, blocked("schema_report_options_invalid")
				}
			}
			// Sequential keys preserve duplicate field names for CheckSchemas to reject.
			schema.Properties[strconv.Itoa(i)] = v
		}
		client.schemas[s.SourceID], configs[s.SourceID] = schema, *s.Config
	}
	checked, err := notionsync.CheckSchemas(context.Background(), client)
	if err != nil || !checked.Complete {
		return nil, nil, blocked("schema_report_fields_invalid")
	}
	for _, s := range checked.Sources {
		if s.Config == nil || !reflect.DeepEqual(*s.Config, configs[s.SourceID]) {
			return nil, nil, blocked("schema_report_mapping_mismatch")
		}
	}
	return client, configs, nil
}

func environmentDBOptions() (*db.MySQLOptions, error) {
	var cfg *drivermysql.Config
	if raw := os.Getenv("MYSQL_DSN"); raw != "" {
		var err error
		cfg, err = drivermysql.ParseDSN(raw)
		if err != nil {
			return nil, blocked("invalid_database_environment")
		}
	} else {
		values := map[string]string{}
		for _, pair := range [][2]string{{"MYSQL_HOST", "MINIBLOG_DATABASE_HOST"}, {"MYSQL_PORT", "MINIBLOG_DATABASE_PORT"}, {"MYSQL_DATABASE", "MINIBLOG_DATABASE_DBNAME"}, {"MYSQL_USERNAME", "MINIBLOG_DATABASE_USERNAME"}, {"MYSQL_PASSWORD", "MINIBLOG_DATABASE_PASSWORD"}} {
			a, aSet := os.LookupEnv(pair[0])
			b, bSet := os.LookupEnv(pair[1])
			if aSet && bSet && a != b {
				return nil, blocked("database_environment_alias_conflict")
			}
			if bSet {
				a, aSet = b, true
			}
			if !aSet || (a == "" && pair[0] != "MYSQL_PASSWORD") {
				return nil, blocked("incomplete_database_environment")
			}
			values[pair[0]] = a
		}
		port, err := strconv.Atoi(values["MYSQL_PORT"])
		if err != nil || port < 1 || port > 65535 {
			return nil, blocked("invalid_database_environment")
		}
		cfg = drivermysql.NewConfig()
		cfg.Net = "tcp"
		cfg.Addr = net.JoinHostPort(values["MYSQL_HOST"], values["MYSQL_PORT"])
		cfg.User, cfg.Passwd, cfg.DBName = values["MYSQL_USERNAME"], values["MYSQL_PASSWORD"], values["MYSQL_DATABASE"]
		cfg.Params = map[string]string{"charset": "utf8mb4"}
	}
	if cfg.DBName == "" {
		return nil, blocked("invalid_database_environment")
	}
	cfg.ParseTime, cfg.Loc = true, time.UTC
	return &db.MySQLOptions{DSN: cfg.FormatDSN(), LogLevel: int(logger.Silent), MaxIdleConns: 1, MaxOpenConns: 1}, nil
}

func execute(ctx context.Context, gdb *gorm.DB, in *preparedInput, apply bool) (*result, error) {
	r := &result{Mode: "preflight", Outcome: "ready", SchemaHash: in.SchemaHash, RevisionsHash: in.RevisionsHash}
	if apply {
		r.Mode = "apply"
	}
	tables := []string{"module", "section", "subsection", "article", "notion_sync_control", "notion_sync_sources", "notion_catalog_bindings", "notion_page_bindings", "notion_sync_runs", "notion_sync_run_items"}
	for _, table := range tables {
		if !gdb.Migrator().HasTable(table) {
			return r, blocked("migration_required")
		}
	}
	if !gdb.Migrator().HasColumn(&model.Module{}, "sort") || !gdb.Migrator().HasColumn(&model.Article{}, "tags_json") {
		return r, blocked("migration_required")
	}
	if gdb.Dialector.Name() == "mysql" {
		var rows []struct {
			Name   string `gorm:"column:table_name"`
			Type   string `gorm:"column:table_type"`
			Engine string `gorm:"column:engine"`
		}
		if err := gdb.WithContext(ctx).Raw("SELECT TABLE_NAME AS table_name,TABLE_TYPE AS table_type,ENGINE AS engine FROM information_schema.TABLES WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME IN ?", tables).Scan(&rows).Error; err != nil {
			return r, err
		}
		if len(rows) != len(tables) {
			return r, blocked("transactional_schema_required")
		}
		for _, row := range rows {
			if row.Type != "BASE TABLE" || !strings.EqualFold(row.Engine, "InnoDB") {
				return r, blocked("transactional_schema_required")
			}
		}
	}
	work := func(ds store.IStore) error {
		modules, err := plan(ctx, ds, in, r, apply)
		if err != nil {
			return err
		}
		if !apply {
			return nil
		}
		cat := catalog.New(ds)
		for _, m := range r.Modules {
			if m.Action == "create_inactive" {
				if err := ds.Modules().Create(&model.Module{Code: m.Code, Title: m.Title, Status: model.ModuleStatusDeleted, Sort: m.Sort}); err != nil {
					return err
				}
			} else if m.Action == "rename" {
				if _, err := cat.UpdateModule(ctx, m.Code, catalog.UpdateInput{Title: m.Title}); err != nil {
					return err
				}
			}
		}
		service := notionsync.New(ds, notionsync.Options{Enabled: false, Client: in.Client})
		disabled := false
		for _, s := range r.Sources {
			cfg, expected := s.Config, s.ExpectedRevision
			if _, err := service.UpdateSource(ctx, s.SourceID, notionsync.SourceInput{ModuleCode: s.ModuleCode, Enabled: &disabled, Config: &cfg, ExpectedConfigRevision: &expected}); err != nil {
				return err
			}
		}
		after, err := proveContent(ctx, ds.DB())
		if err != nil {
			return err
		}
		if !reflect.DeepEqual(*r.Before, after) {
			return blocked("content_invariant_changed")
		}
		r.After = &after
		// Verify unchanged module fields through SQL; NULL status is rejected in plan.
		for _, before := range modules {
			after, err := ds.Modules().GetByCode(before.Code)
			if err != nil {
				return err
			}
			if after == nil || after.ID != before.ID || after.Code != before.Code || after.Status != before.Status || after.Sort != before.Sort || !after.CreatedAt.Equal(before.CreatedAt) {
				return blocked("module_invariant_changed")
			}
		}
		return nil
	}
	var err error
	if apply {
		err = store.InTransaction(ctx, store.NewStore(gdb), work)
	} else {
		err = work(store.NewStore(gdb.WithContext(ctx)))
	}
	if err == nil && apply {
		r.Outcome = "applied"
	}
	return r, err
}

func plan(ctx context.Context, ds store.IStore, in *preparedInput, r *result, lock bool) ([]*model.Module, error) {
	db := ds.DB().WithContext(ctx)
	var c model.NotionSyncControl
	q := db
	if lock {
		q = q.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	if err := q.First(&c, 1).Error; err != nil {
		return nil, err
	}
	now, err := store.DatabaseNow(db)
	if err != nil {
		return nil, err
	}
	if !c.Paused || c.BaselineFrozen || c.CurrentRunID != "" || (c.LeaseUntil != nil && c.LeaseUntil.After(now)) {
		return nil, blocked("initialization_control_not_ready")
	}
	var managed int64
	if err := db.Model(&model.NotionPageBinding{}).Where("management_state = ?", model.NotionManagementManaged).Count(&managed).Error; err != nil {
		return nil, err
	}
	if managed != 0 {
		return nil, blocked("managed_articles_already_exist")
	}
	var modules []*model.Module
	if err := db.Order("id asc").Find(&modules).Error; err != nil {
		return nil, err
	}
	var nullFields int64
	if err := db.Model(&model.Module{}).Where("status IS NULL OR sort IS NULL").Count(&nullFields).Error; err != nil {
		return nil, err
	}
	if nullFields != 0 {
		return nil, blocked("module_fields_invalid")
	}
	if lock {
		codes := make([]string, 0, len(modules))
		for _, m := range modules {
			codes = append(codes, m.Code)
		}
		if err := catalog.LockModules(ds, codes...); err != nil {
			return nil, err
		}
		if err := db.Order("id asc").Find(&modules).Error; err != nil {
			return nil, err
		}
	}
	byCode := map[string]*model.Module{}
	maxSort := math.MinInt
	for _, m := range modules {
		byCode[m.Code] = m
		if m.Sort > maxSort {
			maxSort = m.Sort
		}
	}
	for _, t := range targets {
		m := byCode[t.ModuleCode]
		if t.ModuleCode != "algorithm" {
			if m == nil {
				return nil, blocked("required_module_missing")
			}
			if m.Status < 0 || m.Status > model.ModuleStatusDeleted {
				return nil, blocked("module_fields_invalid")
			}
			if t.ModuleCode != "database" && t.ModuleCode != "project" {
				continue
			}
			action := "no_op"
			if m.Title != t.Title {
				action = "rename"
			}
			r.Modules = append(r.Modules, modulePlan{Code: m.Code, Action: action, ID: strconv.FormatUint(m.ID, 10), BeforeTitle: m.Title, Title: t.Title, Status: m.Status, Sort: m.Sort})
			continue
		}
		if m == nil {
			if maxSort >= math.MaxInt32 {
				return nil, blocked("module_sort_exhausted")
			}
			r.Modules = append(r.Modules, modulePlan{Code: t.ModuleCode, Action: "create_inactive", Title: t.Title, Status: model.ModuleStatusDeleted, Sort: maxSort + 1})
		} else {
			var dependencies int64
			if err := db.Model(&model.Section{}).Where("module_code = ?", m.Code).Count(&dependencies).Error; err != nil {
				return nil, err
			}
			if m.Status != model.ModuleStatusDeleted || m.Title != t.Title || dependencies != 0 {
				return nil, blocked("algorithm_module_requires_review")
			}
			r.Modules = append(r.Modules, modulePlan{Code: m.Code, Action: "no_op", ID: strconv.FormatUint(m.ID, 10), BeforeTitle: m.Title, Title: m.Title, Status: m.Status, Sort: m.Sort})
		}
	}
	var sources []model.NotionSyncSource
	q = db
	if lock {
		q = q.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	if err := q.Order("source_id").Find(&sources).Error; err != nil {
		return nil, err
	}
	if len(sources) != len(targets) {
		return nil, blocked("five_sources_required")
	}
	bySource := map[string]model.NotionSyncSource{}
	for _, s := range sources {
		bySource[s.ID] = s
	}
	for _, t := range targets {
		s, ok := bySource[t.SourceID]
		if !ok || s.DataSourceID != t.SourceID || s.Enabled || (s.ModuleCode != "" && s.ModuleCode != t.ModuleCode) {
			return nil, blocked("source_initialization_not_ready")
		}
		expected, ok := in.Revisions[t.SourceID]
		if !ok || expected != s.ConfigRevision {
			return nil, blocked("source_revision_conflict")
		}
		var previous notionsync.SourceConfig
		if s.PropertyMappingJSON != "" && json.Unmarshal([]byte(s.PropertyMappingJSON), &previous) != nil {
			return nil, blocked("stored_source_config_invalid")
		}
		if s.StatusMappingJSON != "" && json.Unmarshal([]byte(s.StatusMappingJSON), &previous.StateOptionIDs) != nil {
			return nil, blocked("stored_source_config_invalid")
		}
		action, next := "no_op", expected
		if s.ModuleCode != t.ModuleCode || !reflect.DeepEqual(previous, in.Configs[t.SourceID]) {
			if next == math.MaxUint64 {
				return nil, blocked("source_revision_exhausted")
			}
			next++
			action = "configure"
		}
		r.Sources = append(r.Sources, sourcePlan{SourceID: s.ID, ModuleCode: t.ModuleCode, BeforeModuleCode: s.ModuleCode, Action: action, ExpectedRevision: expected, NextRevision: next, Config: in.Configs[t.SourceID]})
	}
	proof, err := proveContent(ctx, db)
	if err != nil {
		return nil, err
	}
	r.Before = &proof
	return modules, nil
}

func proveContent(ctx context.Context, db *gorm.DB) (contentProof, error) {
	var p contentProof
	for _, entry := range []struct {
		table string
		count *int64
	}{{"article", &p.Articles}, {"section", &p.Sections}, {"subsection", &p.Subsections}, {"notion_page_bindings", &p.PageBindings}, {"notion_catalog_bindings", &p.CatalogLinks}} {
		if err := db.WithContext(ctx).Table(entry.table).Count(entry.count).Error; err != nil {
			return p, err
		}
	}
	var ids []uint64
	if err := store.PublishedArticles(ctx, db).Order("article.id asc").Pluck("article.id", &ids).Error; err != nil {
		return p, err
	}
	h := sha256.New()
	for _, id := range ids {
		fmt.Fprintf(h, "%d\n", id)
	}
	p.PublicCount, p.PublicIDsHash = len(ids), hex.EncodeToString(h.Sum(nil))
	return p, nil
}
