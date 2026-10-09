package main

import (
	"context"
	"database/sql"
	"os"
	"regexp"
	"strings"
	"testing"

	mysql "github.com/go-sql-driver/mysql"
	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/notionsync"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	driver "gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Use a separate namespace: content-integration resets tables concurrently when
// go test ./... runs packages in parallel. Never borrow its business fixture tables.
func TestMySQLScopedCLIMaintenance(t *testing.T) {
	dsn := os.Getenv("MINIBLOG_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("dedicated loopback fixture not configured")
	}
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil || cfg.Net != "tcp" || !strings.HasPrefix(cfg.Addr, "127.0.0.1:") || !regexp.MustCompile(`^miniblog_refactor_test_[a-zA-Z0-9_]{1,30}$`).MatchString(cfg.DBName) {
		t.Fatal("refusing non-isolated MySQL fixture")
	}
	namespace := cfg.DBName + "_cli"
	admin, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal("fixture connection unavailable")
	}
	defer admin.Close()
	// Every identifier above is from a constrained test-only name, never operator input.
	if _, err = admin.Exec("DROP DATABASE IF EXISTS `" + namespace + "`"); err != nil {
		t.Fatal("fixture namespace cleanup failed")
	}
	if _, err = admin.Exec("CREATE DATABASE `" + namespace + "` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"); err != nil {
		t.Fatal("fixture namespace creation failed")
	}
	defer func() {
		if _, err := admin.Exec("DROP DATABASE `" + namespace + "`"); err != nil {
			t.Error("fixture namespace cleanup failed")
		}
	}()
	cfg.DBName = namespace
	gdb, err := gorm.Open(driver.Open(cfg.FormatDSN()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal("fixture connection unavailable")
	}
	pool, err := gdb.DB()
	if err != nil {
		t.Fatal("fixture connection unavailable")
	}
	defer pool.Close()
	if err = gdb.AutoMigrate(&model.Module{}, &model.Section{}, &model.Subsection{}, &model.Article{}, &model.NotionSyncControl{}, &model.NotionSyncSource{}, &model.NotionPageBinding{}, &model.NotionCatalogBinding{}, &model.NotionSyncRun{}, &model.NotionSyncRunItem{}, &model.UserM{}); err != nil {
		t.Fatal("fixture schema unavailable")
	}
	if err = gdb.Create(&model.Module{Code: "go", Title: "Go", Status: 1}).Error; err != nil {
		t.Fatal("module fixture unavailable")
	}
	ds := store.NewStore(gdb)
	ctx := context.Background()
	for _, test := range []string{"resumed", "unknown", "success"} {
		t.Run(test, func(t *testing.T) {
			for _, v := range []interface{}{&model.NotionSyncControl{}, &model.NotionPageBinding{}, &model.NotionSyncSource{}} {
				if err := gdb.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(v).Error; err != nil {
					t.Fatal("fixture reset failed")
				}
			}
			if err := gdb.Create(&model.NotionSyncControl{ID: 1, Paused: test != "resumed", SourceWritesPaused: true, BaselineFrozen: true}).Error; err != nil {
				t.Fatal("control fixture unavailable")
			}
			row := model.NotionSyncSource{ID: maintenanceGoSource, DataSourceID: maintenanceGoSource, Label: "Before", ModuleCode: "go", ConfigRevision: 1, PropertyMappingJSON: `{}`, StatusMappingJSON: `{}`}
			if err := gdb.Create(&row).Error; err != nil {
				t.Fatal("source fixture unavailable")
			}
			if test == "unknown" {
				if err := gdb.Create(&model.NotionPageBinding{PageID: strings.Repeat("a", 32), SourceID: maintenanceGoSource, ManagementState: model.NotionManagementBaselinePending, BootstrapState: "verified"}).Error; err != nil {
					t.Fatal("journal fixture unavailable")
				}
			}
			revision := uint64(1)
			result, err := executeMaintenanceCommand(ctx, nil, ds, "source_update", maintenanceCommand{SourceID: maintenanceGoSource, Revision: revision, Source: notionsync.SourceInput{Label: "After", ExpectedConfigRevision: &revision}})
			if (err == nil) != (test == "success") {
				t.Fatal("unexpected maintenance result", test)
			}
			if err != nil && result != nil {
				t.Fatal("returned rolled-back source DTO")
			}
			var after model.NotionSyncSource
			if err := gdb.First(&after, "source_id = ?", maintenanceGoSource).Error; err != nil {
				t.Fatal("source readback failed")
			}
			if test == "success" {
				if after.Label != "After" || after.ConfigRevision != 2 {
					t.Fatal("nested business transaction did not commit")
				}
			} else if after.Label != "Before" || after.ConfigRevision != 1 {
				t.Fatal("rejected mutation changed source")
			}
			var control model.NotionSyncControl
			if err := gdb.First(&control, 1).Error; err != nil {
				t.Fatal("control readback failed")
			}
			if control.LeaseOwner != "" || control.LeaseUntil != nil {
				t.Fatal("maintenance lease not released")
			}
		})
	}
	for _, test := range authorResolutionCases() {
		t.Run("author_"+test.name, func(t *testing.T) { assertAuthorResolutionCase(t, gdb, test) })
	}
}
