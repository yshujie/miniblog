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
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"github.com/yshujie/miniblog/pkg/db"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const maintenanceGoSource = "2bf330bd-ddf1-80a6-aa49-000bbd1e154b"

func TestMaintenanceScopeRejectedBeforeConnections(t *testing.T) {
	t.Setenv("MINIBLOG_NOTION_BOOTSTRAP_TOKEN", "write-secret-SENTINEL")
	invalid := [][]string{
		{"--mode", "source_update", "--input", "ignored.json"},
		{"--mode", "catalog_activate", "--source-id", maintenanceGoSource, "--input", "ignored.json"},
		{"--mode", "catalog_bind", "--source-id", "unlisted", "--expected-config-revision", "1", "--input", "ignored.json"},
		{"--mode", "control_update", "--source-id", maintenanceGoSource, "--input", "ignored.json"},
		{"--mode", "bootstrap_apply", "--allow-notion-write", "--confirmations", "ignored.json", "--source-id", maintenanceGoSource},
		{"--mode", "bootstrap_apply", "--allow-notion-write", "--confirmations", "ignored.json", "--expected-config-revision", "1"},
		{"--mode", "run_status", "--run-id", "../../file"},
		{"--mode", "status", "--run-id", "00000000-0000-0000-0000-000000000000"},
	}
	for _, args := range invalid {
		var out, errout bytes.Buffer
		code := runWithDependencies(args, &out, &errout, dependencies{
			connect: func(*db.MySQLOptions) (*gorm.DB, error) { t.Fatal("invalid maintenance reached DB"); return nil, nil },
		})
		if code != 2 || strings.Contains(out.String()+errout.String(), "write-secret-SENTINEL") {
			t.Fatal(code, out.String(), errout.String())
		}
	}
	for _, mode := range []string{"source_update", "catalog_bind", "catalog_activate", "bootstrap_apply", "catalog_prepare"} {
		input := "input.json"
		if mode == "bootstrap_apply" || mode == "catalog_prepare" {
			input = ""
		}
		if err := validateMaintenanceArguments(mode, maintenanceGoSource, 1, true, input, ""); err != nil {
			t.Fatal(mode, err)
		}
	}
}

func TestMaintenanceStrictInputAndVersion(t *testing.T) {
	for _, tt := range []struct {
		mode, body string
		ok         bool
	}{
		{"control_update", `{"paused":true,"source_writes_paused":true}`, true},
		{"control_update", `{"enabled":true}`, false},
		{"control_update", `{}`, false},
		{"control_update", `{"paused":true,"unknown":true}`, false},
		{"control_update", `{"paused":true} {}`, false},
		{"source_update", `{"enabled":true}`, true},
		{"source_update", `{"enabled":false,"expected_config_revision":2}`, false},
		{"source_update", `{}`, false},
		{"catalog_bind", `{"binding_id":"9223372036854775807","section_code":"s1"}`, true},
		{"catalog_bind", `{"binding_id":"9223372036854775808","section_code":"s1"}`, false},
		{"catalog_bind", `{"binding_id":"0","section_code":"s1"}`, false},
	} {
		path := filepath.Join(t.TempDir(), "command.json")
		if err := os.WriteFile(path, []byte(tt.body), 0600); err != nil {
			t.Fatal(err)
		}
		c, err := readMaintenanceCommand(tt.mode, maintenanceGoSource, 1, path, "")
		if (err == nil) != tt.ok {
			t.Fatal(tt.mode, tt.body, err)
		}
		if err == nil && tt.mode == "source_update" && (c.Source.ExpectedConfigRevision == nil || *c.Source.ExpectedConfigRevision != 1) {
			t.Fatal("revision not fenced", c)
		}
	}
}

func TestRunStatusIsReadOnlyModeAndSanitizesConnectionFailure(t *testing.T) {
	var out, errout bytes.Buffer
	if err := validateMode("run_status", false, ""); err != nil {
		t.Fatal(err)
	}
	code := runWithDependencies([]string{"--mode", "run_status", "--run-id", "00000000-0000-0000-0000-000000000000"}, &out, &errout, dependencies{
		connect: func(*db.MySQLOptions) (*gorm.DB, error) { return nil, errors.New("secret-connection-SENTINEL") },
	})
	if code != 1 || strings.Contains(out.String()+errout.String(), "secret-connection-SENTINEL") {
		t.Fatal(code, out.String(), errout.String())
	}
}

func TestAuthorResolveAndDrainEvidence(t *testing.T) {
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err = gdb.AutoMigrate(&model.UserM{}, &model.NotionSyncControl{}, &model.NotionPageBinding{}); err != nil {
		t.Fatal(err)
	}
	ds := store.NewStore(gdb)
	ctx := context.Background()
	if err = gdb.Model(&model.UserM{}).Create(map[string]interface{}{"id": 1, "username": "admin", "password": "private-hash", "nickname": "博客作者", "status": 1}).Error; err != nil {
		t.Fatal(err)
	}
	resolved, err := executeMaintenanceCommand(ctx, nil, ds, "author_resolve", maintenanceCommand{})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(resolved)
	if string(raw) != `{"author":"博客作者"}` {
		t.Fatal(string(raw))
	}
	if err = gdb.Model(&model.UserM{}).Create(map[string]interface{}{"id": 2, "username": "second", "password": "private-hash", "nickname": "其他作者", "status": 1}).Error; err != nil {
		t.Fatal(err)
	}
	if _, err = executeMaintenanceCommand(ctx, nil, ds, "author_resolve", maintenanceCommand{}); err == nil {
		t.Fatal("ambiguous administrator accepted")
	}
	if err = gdb.Create(&model.NotionSyncControl{ID: 1, Paused: true, SourceWritesPaused: true, BaselineFrozen: true}).Error; err != nil {
		t.Fatal(err)
	}
	for _, row := range []model.NotionPageBinding{
		{PageID: "one", ManagementState: model.NotionManagementBaselinePending, BootstrapState: "write_requested"},
		{PageID: "two", ManagementState: model.NotionManagementBaselinePending, BootstrapState: "verified"},
		{PageID: "three", ManagementState: model.NotionManagementManaged, BootstrapState: "adopted"},
		{PageID: "four", ManagementState: model.NotionManagementBaselinePending},
	} {
		if err = gdb.Create(&row).Error; err != nil {
			t.Fatal(err)
		}
	}
	evidence, err := executeMaintenanceCommand(ctx, nil, ds, "drain_status", maintenanceCommand{})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ = json.Marshal(evidence)
	var result map[string]interface{}
	if err = json.Unmarshal(raw, &result); err != nil {
		t.Fatal(err)
	}
	if result["unresolved_bootstrap_count"] != float64(2) || result["baseline_frozen"] != true || result["lease_epoch"] != "0" {
		t.Fatal(result)
	}
}

func TestBootstrapFileScopeCannotBeOverridden(t *testing.T) {
	for _, in := range []notionsync.BootstrapInput{
		{SourceID: "another-source"}, {ExpectedConfigRevision: 8},
	} {
		if applyBootstrapScope(&in, maintenanceGoSource, 7) == nil {
			t.Fatal("overrode reviewed scope", in)
		}
	}
	in := notionsync.BootstrapInput{}
	if err := applyBootstrapScope(&in, maintenanceGoSource, 7); err != nil || in.SourceID != maintenanceGoSource || in.ExpectedConfigRevision != 7 {
		t.Fatal(in, err)
	}
}

func TestMaintenanceMutationSharesLeaseAndPauseTransaction(t *testing.T) {
	for _, mode := range []string{"resumed", "writes_resumed", "unfrozen", "unknown_write", "lease_changed", "sql_failure", "success"} {
		t.Run(mode, func(t *testing.T) {
			gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
			if err != nil {
				t.Fatal(err)
			}
			if err = gdb.AutoMigrate(&model.NotionSyncControl{}, &model.NotionPageBinding{}, &model.Article{}); err != nil {
				t.Fatal(err)
			}
			c := model.NotionSyncControl{ID: 1, Paused: mode != "resumed", SourceWritesPaused: mode != "writes_resumed", BaselineFrozen: mode != "unfrozen"}
			if err = gdb.Create(&c).Error; err != nil {
				t.Fatal(err)
			}
			if mode == "unknown_write" {
				if err = gdb.Create(&model.NotionPageBinding{PageID: "unresolved", ManagementState: model.NotionManagementBaselinePending, BootstrapState: "write_requested"}).Error; err != nil {
					t.Fatal(err)
				}
			}
			called := false
			_, err = leasedMaintenance(context.Background(), store.NewStore(gdb), func(tx store.IStore) (interface{}, error) {
				called = true
				if err := tx.DB().Create(&model.Article{ID: 123, Title: "transaction marker"}).Error; err != nil {
					return nil, err
				}
				if mode == "lease_changed" {
					if err := tx.DB().Model(&model.NotionSyncControl{}).Where("id = ?", 1).Update("lease_epoch", 99).Error; err != nil {
						return nil, err
					}
				}
				if mode == "sql_failure" {
					return nil, errors.New("fixture mutation failure")
				}
				return "saved", nil
			})
			if (err == nil) != (mode == "success") {
				t.Fatal(mode, err)
			}
			if called != (mode == "lease_changed" || mode == "sql_failure" || mode == "success") {
				t.Fatal("mutation executed before gate", mode)
			}
			var count int64
			if err := gdb.Model(&model.Article{}).Count(&count).Error; err != nil {
				t.Fatal(err)
			}
			if (count == 1) != (mode == "success") {
				t.Fatal("failed maintenance escaped transaction", mode, count)
			}
			if err := gdb.First(&c, 1).Error; err != nil {
				t.Fatal(err)
			}
			if c.LeaseOwner != "" || c.LeaseUntil != nil || c.LeaseEpoch != 1 {
				t.Fatal("lease not released or rejected mutation persisted", c)
			}
		})
	}
}
