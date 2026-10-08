// audit-content reports legacy content inconsistencies and optionally backfills
// source identities. It never fetches external pages or merges duplicate rows.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/source"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"github.com/yshujie/miniblog/pkg/db"
	"github.com/yshujie/miniblog/scripts/internal/mysqlconfig"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type issue struct {
	Kind   string `json:"kind"`
	ID     string `json:"id"`
	Reason string `json:"reason"`
}

type duplicate struct {
	SourceKey string   `json:"source_key"`
	IDs       []string `json:"article_ids"`
}

type report struct {
	Articles          int         `json:"articles"`
	WithSource        int         `json:"with_source"`
	LegacyContentOnly int         `json:"legacy_content_only"`
	Issues            []issue     `json:"issues"`
	Duplicates        []duplicate `json:"duplicates"`
	OrderingConflicts [][]string  `json:"ordering_conflicts"`
	Applied           bool        `json:"applied"`
}

func (r report) blocked() bool { return len(r.Issues) > 0 || len(r.Duplicates) > 0 }

type candidate struct {
	before   model.Article
	identity *source.Identity
	binding  *model.NotionPageBinding
}

type snapshot struct {
	modules     []model.Module
	sections    []model.Section
	subsections []model.Subsection
	candidates  []candidate
	bindings    []model.NotionPageBinding
	report      report
}

func main() { os.Exit(run()) }

func run() int {
	config := mysqlconfig.Bind(flag.CommandLine)
	apply := flag.Bool("apply", false, "回填来源身份；默认只读。须先停止旧写入程序")
	backfillTags := flag.Bool("backfill-tags", false, "配合 -apply，在维护窗口无损回填旧CSV标签为JSON")
	output := flag.String("report", "-", "JSON 审计报告路径；- 为标准输出（不包含原始链接或凭据）")
	flag.Parse()
	if *backfillTags && !*apply {
		fmt.Fprintln(os.Stderr, "-backfill-tags 须配合 -apply，默认审计始终只读")
		return 1
	}
	gdb, err := db.NewMySQL(config.DBOptions(1))
	if err != nil {
		fmt.Fprintln(os.Stderr, "连接数据库失败:", err)
		return 1
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer sqlDB.Close()
	state, err := inspect(context.Background(), gdb)
	if err != nil {
		fmt.Fprintln(os.Stderr, "审计失败:", err)
		return 1
	}
	if *apply && !state.report.blocked() {
		if err = backfill(context.Background(), gdb, state, *backfillTags); err != nil {
			fmt.Fprintln(os.Stderr, "回填失败，事务已回滚:", err)
			return 1
		}
		state.report.Applied = true
	}
	data, err := json.MarshalIndent(state.report, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	data = append(data, '\n')
	if *output == "-" {
		_, err = os.Stdout.Write(data)
	} else {
		err = os.WriteFile(*output, data, 0600)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "写报告失败:", err)
		return 1
	}
	if state.report.blocked() {
		fmt.Fprintln(os.Stderr, "存在阻断项；未进行任何回填或合并，请处理报告后重新审计")
		return 2
	}
	return 0
}

func inspect(ctx context.Context, gdb *gorm.DB) (*snapshot, error) {
	gdb = gdb.WithContext(ctx)
	state := &snapshot{report: report{Issues: []issue{}, Duplicates: []duplicate{}, OrderingConflicts: [][]string{}}}
	if err := gdb.Find(&state.modules).Error; err != nil {
		return nil, err
	}
	var sections []model.Section
	var subsections []model.Subsection
	var articles []model.Article
	if err := gdb.Order("id").Find(&sections).Error; err != nil {
		return nil, err
	}
	if err := gdb.Order("id").Find(&subsections).Error; err != nil {
		return nil, err
	}
	if err := gdb.Order("id").Find(&articles).Error; err != nil {
		return nil, err
	}
	bound := map[uint64]*model.NotionPageBinding{}
	if store.HasNotionSyncSchema(gdb) {
		if err := gdb.Order("page_id").Find(&state.bindings).Error; err != nil {
			return nil, err
		}
		if err := store.AuditSourceAliases(ctx, gdb); err != nil {
			return nil, err
		}
		for i := range state.bindings {
			p := &state.bindings[i]
			if p.ArticleID != nil {
				bound[*p.ArticleID] = p
			}
		}
	}
	state.sections, state.subsections = sections, subsections
	state.report.Articles = len(articles)
	add := func(kind string, id uint64, reason string) {
		state.report.Issues = append(state.report.Issues, issue{kind, fmt.Sprint(id), reason})
	}
	// SQL resolves codes according to the deployed database collation; comparing
	// them as Go strings would reject valid legacy case variants.
	moduleCache := map[string]*model.Module{}
	sectionCache := map[string]*model.Section{}
	subsectionCache := map[string]*model.Subsection{}
	moduleFor := func(code string) (*model.Module, error) {
		if row, ok := moduleCache[code]; ok {
			return row, nil
		}
		var row model.Module
		err := gdb.Where("code = ?", code).First(&row).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			moduleCache[code] = nil
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		moduleCache[code] = &row
		return &row, nil
	}
	sectionFor := func(code string) (*model.Section, error) {
		if row, ok := sectionCache[code]; ok {
			return row, nil
		}
		var row model.Section
		err := gdb.Where("code = ?", code).First(&row).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			sectionCache[code] = nil
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		sectionCache[code] = &row
		return &row, nil
	}
	subsectionFor := func(code string) (*model.Subsection, error) {
		if row, ok := subsectionCache[code]; ok {
			return row, nil
		}
		var row model.Subsection
		err := gdb.Where("code = ?", code).First(&row).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			subsectionCache[code] = nil
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		subsectionCache[code] = &row
		return &row, nil
	}
	// SQL seeds (000002 and section.sql) contain inactive module/section status 0.
	// Preserve that legacy value; public reads still require status 1. SQL NULL
	// is not a legacy status, even though GORM decodes it as the int zero value.
	nullStatuses := map[string]map[uint64]bool{}
	for _, kind := range []string{"module", "section"} {
		var ids []uint64
		if err := gdb.Table(kind).Where("status IS NULL").Pluck("id", &ids).Error; err != nil {
			return nil, err
		}
		nullStatuses[kind] = map[uint64]bool{}
		for _, id := range ids {
			nullStatuses[kind][id] = true
		}
	}
	for _, m := range state.modules {
		if strings.TrimSpace(m.Code) == "" {
			add("module", m.ID, "empty_code")
		}
		if nullStatuses["module"][m.ID] || (m.Status != 0 && m.Status != model.ModuleStatusNormal && m.Status != model.ModuleStatusDeleted) {
			add("module", m.ID, "invalid_status")
		}
	}
	for _, s := range sections {
		m, err := moduleFor(s.ModuleCode)
		if err != nil {
			return nil, err
		}
		if m == nil {
			add("section", s.ID, "missing_module")
		}
		if strings.TrimSpace(s.Code) == "" {
			add("section", s.ID, "empty_code")
		}
		if nullStatuses["section"][s.ID] || (s.Status != 0 && s.Status != model.SectionStatusNormal && s.Status != model.SectionStatusDeleted) {
			add("section", s.ID, "invalid_status")
		}
	}
	for _, s := range subsections {
		parent, err := sectionFor(s.SectionCode)
		if err != nil {
			return nil, err
		}
		if parent == nil {
			add("subsection", s.ID, "missing_section")
		}
		if strings.TrimSpace(s.Code) == "" {
			add("subsection", s.ID, "empty_code")
		}
		if s.Status != model.SubsectionStatusNormal && s.Status != model.SubsectionStatusDeleted {
			add("subsection", s.ID, "invalid_status")
		}
	}
	sources := map[string][]string{}
	order := map[string][]string{}
	for _, a := range articles {
		if a.Status < 1 || a.Status > 4 {
			add("article", a.ID, "invalid_status")
		}
		parent, err := sectionFor(a.SectionCode)
		if err != nil {
			return nil, err
		}
		var parentID uint64
		var childID uint64
		if parent == nil {
			add("article", a.ID, "missing_section")
		} else {
			parentID = parent.ID
			m, err := moduleFor(parent.ModuleCode)
			if err != nil {
				return nil, err
			}
			if m == nil {
				add("article", a.ID, "missing_module")
			}
		}
		if a.SubsectionCode != "" {
			child, err := subsectionFor(a.SubsectionCode)
			if err != nil {
				return nil, err
			}
			if child == nil {
				add("article", a.ID, "missing_subsection")
			} else {
				childID = child.ID
				childParent, err := sectionFor(child.SectionCode)
				if err != nil {
					return nil, err
				}
				if parent == nil || childParent == nil || parent.ID != childParent.ID {
					add("article", a.ID, "subsection_parent_mismatch")
				}
			}
		}
		if parent != nil {
			group := fmt.Sprintf("%d/%d/%d", parentID, childID, a.Pos)
			order[group] = append(order[group], fmt.Sprint(a.ID))
		}
		c := candidate{before: a, binding: bound[a.ID]}
		if strings.TrimSpace(a.ExternalLink) == "" {
			state.report.LegacyContentOnly++
			if a.SourceKey != nil {
				add("article", a.ID, "source_identity_without_link")
			}
		} else {
			identity, err := source.Parse(a.ExternalLink)
			if err != nil {
				add("article", a.ID, "invalid_external_link")
			} else {
				// A verified Page ID supersedes the historical URL digest. The alias
				// is audit evidence, never a replacement identity to backfill.
				if c.binding != nil {
					canonical, parseErr := source.NotionIdentity(c.binding.PageID)
					if parseErr != nil || a.SourceKey == nil || *a.SourceKey != canonical.SourceKey {
						add("article", a.ID, "invalid_verified_page_identity")
					} else if identity.SourceKey != canonical.SourceKey && (c.binding.LegacySourceKey == nil || *c.binding.LegacySourceKey != identity.SourceKey) {
						add("article", a.ID, "unverified_legacy_source_alias")
					} else {
						identity = canonical
					}
				}
				c.identity = &identity
				state.report.WithSource++
				sources[identity.SourceKey] = append(sources[identity.SourceKey], fmt.Sprint(a.ID))
				if a.SourceKey != nil && *a.SourceKey != identity.SourceKey {
					add("article", a.ID, "source_identity_mismatch")
				}
			}
		}
		state.candidates = append(state.candidates, c)
	}
	// Preserve report ordering for deterministic review and reruns.
	for _, c := range state.candidates {
		if c.identity == nil {
			continue
		}
		key := c.identity.SourceKey
		if ids := sources[key]; len(ids) > 1 {
			state.report.Duplicates = append(state.report.Duplicates, duplicate{key, ids})
			delete(sources, key)
		}
	}
	for _, c := range state.candidates {
		id := fmt.Sprint(c.before.ID)
		for key, ids := range order {
			if len(ids) > 1 && ids[0] == id {
				state.report.OrderingConflicts = append(state.report.OrderingConflicts, ids)
				delete(order, key)
			}
		}
	}
	return state, nil
}

func backfill(ctx context.Context, gdb *gorm.DB, state *snapshot, migrateTags ...bool) error {
	tags := len(migrateTags) > 0 && migrateTags[0]
	if tags && !gdb.Migrator().HasColumn(&model.Article{}, "tags_json") {
		return errors.New("apply expand migration 000006 first")
	}
	if state.report.blocked() {
		return errors.New("unresolved audit issues")
	}
	for _, column := range []string{"provider", "canonical_url", "source_key"} {
		if !gdb.Migrator().HasColumn(&model.Article{}, column) {
			return errors.New("apply expand migration 000004 first")
		}
	}
	if !gdb.Migrator().HasColumn(&model.Module{}, "sort") {
		return errors.New("apply expand migration 000004 first")
	}
	return gdb.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Match the source-write lock order and exclude a concurrent takeover.
		if store.HasNotionSyncSchema(tx) {
			var control model.NotionSyncControl
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&control, 1).Error; err != nil {
				return err
			}
			if tags && (!control.Paused || !control.SourceWritesPaused || control.LeaseOwner != "") {
				return errors.New("JSON tag backfill requires paused sync, drained source writes and no active lease")
			}
		}
		var modules []model.Module
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Order("id").Find(&modules).Error; err != nil {
			return err
		}
		var current []model.Article
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Order("id").Find(&current).Error; err != nil {
			return err
		}
		if len(current) != len(state.candidates) || len(modules) != len(state.modules) {
			return errors.New("content changed since audit; rerun")
		}
		beforeModules := make(map[uint64]model.Module, len(state.modules))
		for _, row := range state.modules {
			beforeModules[row.ID] = row
		}
		for _, row := range modules {
			if before, ok := beforeModules[row.ID]; !ok || !reflect.DeepEqual(row, before) {
				return errors.New("catalog changed since audit; rerun")
			}
		}
		var sections []model.Section
		var subsections []model.Subsection
		if err := tx.Order("id").Find(&sections).Error; err != nil {
			return err
		}
		if err := tx.Order("id").Find(&subsections).Error; err != nil {
			return err
		}
		// The int snapshots cannot distinguish a concurrent 0 -> NULL change.
		// Recheck SQL values while the catalog's module locks are held.
		for _, kind := range []string{"module", "section"} {
			var missing int64
			if err := tx.Table(kind).Where("status IS NULL").Count(&missing).Error; err != nil {
				return err
			}
			if missing != 0 {
				return errors.New("catalog status changed since audit; rerun")
			}
		}
		if !reflect.DeepEqual(sections, state.sections) || !reflect.DeepEqual(subsections, state.subsections) {
			return errors.New("catalog changed since audit; rerun")
		}
		if store.HasNotionSyncSchema(tx) {
			var bindings []model.NotionPageBinding
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Order("page_id").Find(&bindings).Error; err != nil {
				return err
			}
			if !reflect.DeepEqual(bindings, state.bindings) {
				return errors.New("source bindings changed since audit; rerun")
			}
		}
		for i, row := range current {
			c := state.candidates[i]
			if !reflect.DeepEqual(row, c.before) {
				return errors.New("article changed since audit; rerun")
			}
			if tags && row.TagsJSON == nil {
				values, err := model.ArticleTags(&row)
				if err != nil {
					return err
				}
				raw, err := model.EncodeArticleTags(values)
				if err != nil {
					return err
				}
				if err = tx.Model(&model.Article{}).Where("id=?", row.ID).UpdateColumns(map[string]interface{}{"tags_json": raw, "updated_at": row.UpdatedAt}).Error; err != nil {
					return err
				}
			}
			if c.identity == nil || c.binding != nil {
				continue
			}
			if err := tx.Model(&model.Article{}).Where("id = ?", row.ID).UpdateColumns(map[string]interface{}{
				"provider": c.identity.Provider, "canonical_url": c.identity.CanonicalURL, "source_key": c.identity.SourceKey,
				"updated_at": row.UpdatedAt,
			}).Error; err != nil {
				return err
			}
		}
		// A zero-only module order is the expand migration's initial state.
		// Keep an already configured order intact on subsequent runs.
		allZero := true
		for _, m := range modules {
			allZero = allZero && m.Sort == 0
		}
		if allZero {
			for i, m := range state.modules {
				if err := tx.Model(&model.Module{}).Where("id = ?", m.ID).UpdateColumns(map[string]interface{}{"sort": i + 1, "updated_at": m.UpdatedAt}).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}
