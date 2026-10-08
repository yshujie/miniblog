package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// This manifest follows migrations 3, 4, 5 and 6. Check actual objects instead
// of trusting a version marker: CREATE TABLE IF NOT EXISTS can mask drift.
var requiredColumns = map[string][]string{
	"module":                  {"id", "code", "title", "sort", "status", "created_at", "updated_at"},
	"section":                 {"id", "code", "title", "sort", "module_code", "status", "created_at", "updated_at"},
	"subsection":              {"id", "code", "title", "sort", "section_code", "status", "created_at", "updated_at"},
	"article":                 {"id", "title", "content", "external_link", "section_code", "subsection_code", "author", "tags", "tags_json", "pos", "status", "provider", "canonical_url", "source_key", "created_at", "updated_at"},
	"notion_sync_control":     {"id", "paused", "source_writes_paused", "baseline_frozen", "baseline_at", "lease_owner", "lease_epoch", "lease_until", "current_run_id", "next_run_at", "cooldown_until", "last_complete_scan_at", "last_success_at", "updated_at"},
	"notion_sync_sources":     {"source_id", "label", "data_source_id", "module_code", "property_mapping_json", "status_mapping_json", "config_revision", "enabled", "health", "last_attempt_at", "last_complete_scan_at", "last_success_at", "last_error", "created_at", "updated_at"},
	"notion_catalog_bindings": {"id", "source_id", "data_source_id", "theme_property_id", "option_id", "option_name", "section_code", "status", "reason", "created_at", "updated_at"},
	"notion_page_bindings":    {"page_id", "article_id", "legacy_source_key", "source_id", "management_state", "revision", "desired_state", "snapshot_json", "metadata_hash", "applied_hash", "page_url", "public_url", "publish_block_reason", "publication_held", "publication_hold_reason", "needs_revalidation", "native_archived", "in_trash", "notion_last_edited_at", "last_seen_run_id", "last_apply_run_id", "last_success_at", "last_error", "bootstrap_state", "bootstrap_expected_state", "bootstrap_expected_fingerprint", "confirmed_by", "confirmed_at", "local_before_json", "created_at", "updated_at"},
	"notion_sync_runs":        {"run_id", "mode", "phase", "status", "counts_json", "started_at", "finished_at", "error", "lease_epoch"},
	"notion_sync_run_items":   {"run_id", "item_id", "page_id", "article_id", "phase", "outcome", "reason", "before_json", "after_json", "error", "bootstrap_journal_json", "created_at", "updated_at"},
}

type indexRequirement struct {
	table, name string
	columns     []string
}

var requiredIndexes = []indexRequirement{
	{"module", "PRIMARY", []string{"id"}},
	{"module", "", []string{"code"}},
	{"section", "PRIMARY", []string{"id"}},
	{"section", "", []string{"code"}},
	{"subsection", "PRIMARY", []string{"id"}},
	{"subsection", "", []string{"code"}},
	{"article", "PRIMARY", []string{"id"}},
	// RegistrationReady uses this name as part of its cutover gate.
	{"article", "uq_article_source_key", []string{"source_key"}},
	{"notion_sync_control", "PRIMARY", []string{"id"}},
	{"notion_sync_sources", "PRIMARY", []string{"source_id"}},
	{"notion_sync_sources", "", []string{"data_source_id"}},
	{"notion_catalog_bindings", "PRIMARY", []string{"id"}},
	{"notion_catalog_bindings", "", []string{"data_source_id", "theme_property_id", "option_id"}},
	{"notion_page_bindings", "PRIMARY", []string{"page_id"}},
	{"notion_page_bindings", "", []string{"article_id"}},
	{"notion_page_bindings", "", []string{"legacy_source_key"}},
	{"notion_sync_runs", "PRIMARY", []string{"run_id"}},
	{"notion_sync_run_items", "PRIMARY", []string{"run_id", "item_id"}},
}

var binaryIdentityColumns = map[string][]string{
	"article":                 {"source_key"},
	"notion_sync_sources":     {"source_id", "data_source_id"},
	"notion_catalog_bindings": {"source_id", "data_source_id", "theme_property_id", "option_id"},
	"notion_page_bindings":    {"page_id", "legacy_source_key", "source_id"},
	"notion_sync_runs":        {"run_id"},
	"notion_sync_run_items":   {"run_id", "item_id", "page_id"},
}

type column struct {
	dataType, collation string
	length              int64
}

type indexColumn struct {
	name              string
	position          int
	nonUnique, prefix bool
}

type table struct {
	kind, engine string
	columns      map[string]column
	indexes      map[string][]indexColumn
}

type migration struct {
	version int64
	dirty   bool
}

type metadata interface {
	migrations(context.Context) ([]migration, error)
	tables(context.Context) (map[string]*table, error)
	controlExists(context.Context) (bool, error)
}

func inspect(ctx context.Context, source metadata) error {
	versions, err := source.migrations(ctx)
	if err != nil {
		return errors.New("migration metadata unavailable; complete approved migrations before deploying")
	}
	if len(versions) != 1 {
		return errors.New("migration metadata must contain exactly one version")
	}
	if versions[0].dirty {
		return errors.New("migration version is dirty; resolve the migration before deploying")
	}
	if versions[0].version < 6 {
		return errors.New("migration version is below 6; complete approved migrations before deploying")
	}
	tables, err := source.tables(ctx)
	if err != nil {
		return errors.New("required schema metadata unavailable")
	}
	if err := validateSchema(tables); err != nil {
		return err
	}
	exists, err := source.controlExists(ctx)
	if err != nil {
		return errors.New("sync control initialization cannot be verified")
	}
	if !exists {
		return errors.New("sync control singleton is missing; complete approved migration initialization")
	}
	return nil
}

func validateSchema(tables map[string]*table) error {
	var issues []string
	for name, required := range requiredColumns {
		actual := tables[name]
		if actual == nil {
			issues = append(issues, "missing table "+name)
			continue
		}
		if actual.kind != "BASE TABLE" || !strings.EqualFold(actual.engine, "InnoDB") {
			issues = append(issues, "table "+name+" must be an InnoDB base table")
		}
		for _, col := range required {
			if _, exists := actual.columns[col]; !exists {
				issues = append(issues, "missing column "+name+"."+col)
			}
		}
		for _, col := range binaryIdentityColumns[name] {
			if value, exists := actual.columns[col]; exists && !strings.HasSuffix(strings.ToLower(value.collation), "_bin") && !strings.EqualFold(value.collation, "binary") {
				issues = append(issues, "identity column "+name+"."+col+" requires binary collation")
			}
		}
	}
	if actual := tables["article"]; actual != nil {
		if value, exists := actual.columns["tags_json"]; exists && value.dataType != "longtext" {
			issues = append(issues, "column article.tags_json must be LONGTEXT")
		}
		if value, exists := actual.columns["source_key"]; exists && (value.dataType != "char" || value.length != 64) {
			issues = append(issues, "column article.source_key must be CHAR(64)")
		}
	}
	for _, required := range requiredIndexes {
		actual := tables[required.table]
		if actual == nil {
			continue
		}
		matched := false
		for name, cols := range actual.indexes {
			if required.name != "" && name != required.name || len(cols) != len(required.columns) {
				continue
			}
			valid := true
			for i, col := range cols {
				if col.nonUnique || col.prefix || col.position != i+1 || col.name != required.columns[i] {
					valid = false
				}
			}
			if valid {
				matched = true
				break
			}
		}
		if !matched {
			label := required.name
			if label == "" {
				label = "(" + strings.Join(required.columns, ",") + ")"
			}
			issues = append(issues, "missing full-column unique index "+required.table+"."+label)
		}
	}
	if len(issues) > 0 {
		sort.Strings(issues)
		return fmt.Errorf("required schema is incomplete: %s", strings.Join(issues, "; "))
	}
	return nil
}

// All SQL is fixed, SELECT-only metadata. The configured schema is bound as a
// parameter, never interpolated into SQL, and no article contents are fetched.
type sqlMetadata struct {
	tx     *sql.Tx
	schema string
}

func (s sqlMetadata) migrations(ctx context.Context) ([]migration, error) {
	rows, err := s.tx.QueryContext(ctx, "SELECT version, dirty FROM schema_migrations LIMIT 2")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []migration
	for rows.Next() {
		var value migration
		if err := rows.Scan(&value.version, &value.dirty); err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, rows.Err()
}

func (s sqlMetadata) tables(ctx context.Context) (map[string]*table, error) {
	result := map[string]*table{}
	rows, err := s.tx.QueryContext(ctx, "SELECT TABLE_NAME, TABLE_TYPE, COALESCE(ENGINE, '') FROM information_schema.TABLES WHERE TABLE_SCHEMA = ?", s.schema)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var name, kind, engine string
		if err := rows.Scan(&name, &kind, &engine); err != nil {
			rows.Close()
			return nil, err
		}
		if _, required := requiredColumns[name]; required {
			result[name] = &table{kind: kind, engine: engine, columns: map[string]column{}, indexes: map[string][]indexColumn{}}
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	rows, err = s.tx.QueryContext(ctx, "SELECT TABLE_NAME, COLUMN_NAME, DATA_TYPE, COALESCE(CHARACTER_MAXIMUM_LENGTH, 0), COALESCE(COLLATION_NAME, '') FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = ?", s.schema)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var name, col string
		var value column
		if err := rows.Scan(&name, &col, &value.dataType, &value.length, &value.collation); err != nil {
			rows.Close()
			return nil, err
		}
		if actual := result[name]; actual != nil {
			actual.columns[col] = value
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	rows, err = s.tx.QueryContext(ctx, "SELECT TABLE_NAME, INDEX_NAME, COLUMN_NAME, SEQ_IN_INDEX, NON_UNIQUE, SUB_PART FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = ? ORDER BY TABLE_NAME, INDEX_NAME, SEQ_IN_INDEX", s.schema)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var name, index string
		var col sql.NullString
		var position, nonUnique int
		var prefix sql.NullInt64
		if err := rows.Scan(&name, &index, &col, &position, &nonUnique, &prefix); err != nil {
			return nil, err
		}
		if actual := result[name]; actual != nil {
			actual.indexes[index] = append(actual.indexes[index], indexColumn{name: col.String, position: position, nonUnique: nonUnique != 0, prefix: prefix.Valid})
		}
	}
	return result, rows.Err()
}

func (s sqlMetadata) controlExists(ctx context.Context) (bool, error) {
	var exists bool
	err := s.tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM notion_sync_control WHERE id = 1)").Scan(&exists)
	return exists, err
}
