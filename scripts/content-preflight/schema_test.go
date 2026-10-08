package main

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"strings"
	"testing"
)

func readyTables() map[string]*table {
	result := map[string]*table{}
	for name, names := range requiredColumns {
		value := &table{kind: "BASE TABLE", engine: "InnoDB", columns: map[string]column{}, indexes: map[string][]indexColumn{}}
		for _, col := range names {
			value.columns[col] = column{dataType: "varchar", collation: "ascii_bin", length: 128}
		}
		result[name] = value
	}
	result["article"].columns["source_key"] = column{dataType: "char", collation: "ascii_bin", length: 64}
	result["article"].columns["tags_json"] = column{dataType: "longtext", collation: "utf8mb4_general_ci"}
	for _, required := range requiredIndexes {
		name := required.name
		if name == "" {
			name = "test_" + required.table + "_" + strings.Join(required.columns, "_")
		}
		for pos, col := range required.columns {
			result[required.table].indexes[name] = append(result[required.table].indexes[name], indexColumn{name: col, position: pos + 1})
		}
	}
	return result
}

func TestRequiredSchemaRejectsIncompleteAndUnsafeStructures(t *testing.T) {
	cases := []struct {
		name, want string
		mutate     func(map[string]*table)
	}{
		{"missing tags", "article.tags_json", func(m map[string]*table) { delete(m["article"].columns, "tags_json") }},
		{"missing sort", "module.sort", func(m map[string]*table) { delete(m["module"].columns, "sort") }},
		{"missing sync table", "missing table notion_sync_run_items", func(m map[string]*table) { delete(m, "notion_sync_run_items") }},
		{"partial sync table", "notion_page_bindings.needs_revalidation", func(m map[string]*table) { delete(m["notion_page_bindings"].columns, "needs_revalidation") }},
		{"nontransactional", "InnoDB base table", func(m map[string]*table) { m["notion_sync_control"].engine = "MyISAM" }},
		{"case insensitive identity", "binary collation", func(m map[string]*table) {
			value := m["notion_catalog_bindings"].columns["option_id"]
			value.collation = "utf8mb4_general_ci"
			m["notion_catalog_bindings"].columns["option_id"] = value
		}},
		{"source prefix", "article.uq_article_source_key", func(m map[string]*table) { m["article"].indexes["uq_article_source_key"][0].prefix = true }},
		{"source nonunique", "article.uq_article_source_key", func(m map[string]*table) { m["article"].indexes["uq_article_source_key"][0].nonUnique = true }},
		{"source wrong name", "article.uq_article_source_key", func(m map[string]*table) {
			value := m["article"].indexes["uq_article_source_key"]
			delete(m["article"].indexes, "uq_article_source_key")
			m["article"].indexes["renamed"] = value
		}},
		{"source extra column", "article.uq_article_source_key", func(m map[string]*table) {
			m["article"].indexes["uq_article_source_key"] = append(m["article"].indexes["uq_article_source_key"], indexColumn{name: "id", position: 2})
		}},
		{"page unique removed", "notion_page_bindings.(article_id)", func(m map[string]*table) {
			delete(m["notion_page_bindings"].indexes, "test_notion_page_bindings_article_id")
		}},
		{"composite missing part", "notion_catalog_bindings.(data_source_id,theme_property_id,option_id)", func(m map[string]*table) {
			m["notion_catalog_bindings"].indexes["test_notion_catalog_bindings_data_source_id_theme_property_id_option_id"] = []indexColumn{{name: "data_source_id", position: 1}}
		}},
		{"short tags", "must be LONGTEXT", func(m map[string]*table) { m["article"].columns["tags_json"] = column{dataType: "text"} }},
		{"short source key", "must be CHAR(64)", func(m map[string]*table) {
			m["article"].columns["source_key"] = column{dataType: "char", collation: "ascii_bin", length: 32}
		}},
	}
	if err := validateSchema(readyTables()); err != nil {
		t.Fatal(err)
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fixture := readyTables()
			tc.mutate(fixture)
			err := validateSchema(fixture)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("wanted %q, got %v", tc.want, err)
			}
		})
	}
}

type fakeMetadata struct {
	versions []migration
	objects  map[string]*table
	control  bool
	fail     string
}

func (f fakeMetadata) migrations(context.Context) ([]migration, error) {
	if f.fail == "migration" {
		return nil, errors.New("password=private-token")
	}
	return f.versions, nil
}
func (f fakeMetadata) tables(context.Context) (map[string]*table, error) {
	if f.fail == "tables" {
		return nil, errors.New("private-token@private-host")
	}
	return f.objects, nil
}
func (f fakeMetadata) controlExists(context.Context) (bool, error) {
	if f.fail == "control" {
		return false, errors.New("private-token")
	}
	return f.control, nil
}

func TestMigrationAndInitializationGatesRedactUnderlyingErrors(t *testing.T) {
	for _, tc := range []struct {
		name, want string
		versions   []migration
		control    bool
		fail       string
	}{
		{"clean", "", []migration{{6, false}}, true, ""},
		{"future", "", []migration{{7, false}}, true, ""},
		{"old", "below 6", []migration{{5, false}}, true, ""},
		{"dirty", "dirty", []migration{{6, true}}, true, ""},
		{"missing", "exactly one", nil, true, ""},
		{"multiple", "exactly one", []migration{{6, false}, {7, false}}, true, ""},
		{"control absent", "singleton is missing", []migration{{6, false}}, false, ""},
		{"migration error", "migration metadata unavailable", nil, true, "migration"},
		{"schema error", "schema metadata unavailable", []migration{{6, false}}, true, "tables"},
		{"control error", "initialization cannot be verified", []migration{{6, false}}, true, "control"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := inspect(context.Background(), fakeMetadata{versions: tc.versions, objects: readyTables(), control: tc.control, fail: tc.fail})
			if tc.want == "" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.want) || strings.Contains(err.Error(), "private-") {
				t.Fatalf("unexpected gate error %v", err)
			}
		})
	}
}

func TestEnvironmentHasNoCredentialDefaultsAndUsesComposeNames(t *testing.T) {
	values := map[string]string{"MYSQL_HOST": "fallback", "MYSQL_PORT": "3307", "MYSQL_USERNAME": "fallback-user", "MYSQL_PASSWORD": "fallback-secret", "MYSQL_DATABASE": "fallback-db", "MINIBLOG_DATABASE_HOST": "primary", "MINIBLOG_DATABASE_USERNAME": "primary-user", "MINIBLOG_DATABASE_PASSWORD": "primary-secret", "MINIBLOG_DATABASE_DBNAME": "primary-db"}
	env := func(key string) string { return values[key] }
	cfg, err := configFromEnvironment(env)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Addr != "primary:3307" || cfg.User != "primary-user" || cfg.Passwd != "primary-secret" || cfg.DBName != "primary-db" || cfg.MultiStatements || cfg.Timeout.Seconds() != 10 || cfg.ReadTimeout.Seconds() != 10 {
		t.Fatal("wrong environment selection or unsafe driver options")
	}
	for _, key := range []string{"MINIBLOG_DATABASE_HOST", "MINIBLOG_DATABASE_USERNAME", "MINIBLOG_DATABASE_PASSWORD", "MINIBLOG_DATABASE_DBNAME"} {
		delete(values, key)
	}
	cfg, err = configFromEnvironment(env)
	if err != nil || cfg.DBName != "fallback-db" || cfg.Addr != "fallback:3307" {
		t.Fatal("MYSQL fallback unavailable")
	}
	delete(values, "MYSQL_PORT")
	cfg, err = configFromEnvironment(env)
	if err != nil || cfg.Addr != "fallback:3306" {
		t.Fatal("wrong default port")
	}
	values["MYSQL_PORT"] = "secret-invalid-port"
	_, err = configFromEnvironment(env)
	if err == nil || strings.Contains(err.Error(), "secret-invalid-port") {
		t.Fatal("invalid port not safely rejected")
	}
	values = map[string]string{"MYSQL_DSN": "private-token@tcp(private-host)/private-db"}
	var out, failed bytes.Buffer
	if code := run(env, &out, &failed); code != 1 || strings.Contains(failed.String(), "private-") || out.Len() != 0 {
		t.Fatal("missing environment did not fail safely")
	}
}

// A metadata-only driver exercises database/sql scanning without touching MySQL.
// Its connection rejects every query outside the fixed SELECT allowlist, checks
// bound schema parameters, and asserts the transaction is read-only.
type fixtureDriver struct {
	queries           []string
	begun, rolledBack bool
	pingError         bool
}

func (d *fixtureDriver) Connect(context.Context) (driver.Conn, error) { return fixtureConn{d}, nil }
func (d *fixtureDriver) Driver() driver.Driver                        { return d }
func (d *fixtureDriver) Open(string) (driver.Conn, error)             { return fixtureConn{d}, nil }

type fixtureConn struct{ d *fixtureDriver }

func (c fixtureConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("unexpected Prepare")
}
func (c fixtureConn) Close() error { return nil }
func (c fixtureConn) Begin() (driver.Tx, error) {
	return nil, errors.New("read-only BeginTx is required")
}
func (c fixtureConn) BeginTx(_ context.Context, opts driver.TxOptions) (driver.Tx, error) {
	if !opts.ReadOnly {
		return nil, errors.New("write transaction is forbidden")
	}
	c.d.begun = true
	return fixtureTx{c.d}, nil
}
func (c fixtureConn) Ping(context.Context) error {
	if c.d.pingError {
		return errors.New("private-secret@private-host")
	}
	return nil
}

type fixtureTx struct{ d *fixtureDriver }

func (tx fixtureTx) Commit() error   { return errors.New("unexpected Commit") }
func (tx fixtureTx) Rollback() error { tx.d.rolledBack = true; return nil }

type fixtureRows struct {
	columns []string
	values  [][]driver.Value
}

func (r *fixtureRows) Columns() []string { return r.columns }
func (*fixtureRows) Close() error        { return nil }
func (r *fixtureRows) Next(dest []driver.Value) error {
	if len(r.values) == 0 {
		return io.EOF
	}
	copy(dest, r.values[0])
	r.values = r.values[1:]
	return nil
}
func (c fixtureConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	c.d.queries = append(c.d.queries, query)
	if query == "SELECT version, dirty FROM schema_migrations LIMIT 2" {
		return &fixtureRows{[]string{"version", "dirty"}, [][]driver.Value{{int64(6), false}}}, nil
	}
	if query == "SELECT EXISTS(SELECT 1 FROM notion_sync_control WHERE id = 1)" {
		return &fixtureRows{[]string{"exists"}, [][]driver.Value{{true}}}, nil
	}
	if !strings.HasPrefix(query, "SELECT ") || !strings.Contains(query, " FROM information_schema.") || len(args) != 1 || args[0].Value != "fixture-schema" {
		return nil, errors.New("forbidden query")
	}
	objects := readyTables()
	rows := &fixtureRows{}
	switch {
	case strings.Contains(query, "FROM information_schema.TABLES"):
		rows.columns = []string{"TABLE_NAME", "TABLE_TYPE", "ENGINE"}
		for name, value := range objects {
			rows.values = append(rows.values, []driver.Value{name, value.kind, value.engine})
		}
	case strings.Contains(query, "FROM information_schema.COLUMNS"):
		rows.columns = []string{"TABLE_NAME", "COLUMN_NAME", "DATA_TYPE", "CHARACTER_MAXIMUM_LENGTH", "COLLATION_NAME"}
		for name, value := range objects {
			for col, info := range value.columns {
				rows.values = append(rows.values, []driver.Value{name, col, info.dataType, info.length, info.collation})
			}
		}
	case strings.Contains(query, "FROM information_schema.STATISTICS"):
		rows.columns = []string{"TABLE_NAME", "INDEX_NAME", "COLUMN_NAME", "SEQ_IN_INDEX", "NON_UNIQUE", "SUB_PART"}
		for name, value := range objects {
			for index, cols := range value.indexes {
				for _, col := range cols {
					rows.values = append(rows.values, []driver.Value{name, index, col.name, int64(col.position), int64(0), nil})
				}
			}
		}
	default:
		return nil, errors.New("forbidden metadata query")
	}
	return rows, nil
}

func TestSQLInspectionOnlyReadsMetadataInReadOnlyTransaction(t *testing.T) {
	fixture := &fixtureDriver{}
	database := sql.OpenDB(fixture)
	defer database.Close()
	if err := checkDatabase(context.Background(), database, "fixture-schema"); err != nil {
		t.Fatal(err)
	}
	if !fixture.begun || !fixture.rolledBack || len(fixture.queries) != 5 {
		t.Fatal("read-only inspection boundary was not enforced")
	}
	// Only metadata reads, never the legacy article table or application values.
	for _, query := range fixture.queries {
		if strings.Contains(query, "FROM article") {
			t.Fatal("application rows were read")
		}
	}
	fixture.pingError = true
	if err := checkDatabase(context.Background(), database, "fixture-schema"); err == nil || err.Error() != "database connection unavailable" {
		t.Fatal("connection error is not sanitized")
	}
}
