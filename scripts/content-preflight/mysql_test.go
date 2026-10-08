package main

import (
	"context"
	"database/sql"
	"net"
	"os"
	"regexp"
	"testing"

	mysql "github.com/go-sql-driver/mysql"
)

// Explicitly opted-in read-only verification of an existing disposable local
// fixture. This test never creates, migrates, clears or repairs that database.
func TestExistingDisposableMySQLSchema(t *testing.T) {
	dsn := os.Getenv("MINIBLOG_PREFLIGHT_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("set MINIBLOG_PREFLIGHT_TEST_MYSQL_DSN to an existing disposable loopback schema")
	}
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		t.Fatal("invalid fixture configuration")
	}
	host, _, err := net.SplitHostPort(cfg.Addr)
	if err != nil || cfg.Net != "tcp" || (host != "127.0.0.1" && host != "::1" && host != "localhost") || !regexp.MustCompile(`^miniblog_refactor_test_[A-Za-z0-9_]+$`).MatchString(cfg.DBName) {
		t.Fatal("read-only fixture verification requires loopback TCP and miniblog_refactor_test_* schema")
	}
	cfg.MultiStatements = false
	_ = mysql.SetLogger(quietLogger{})
	connector, err := mysql.NewConnector(cfg)
	if err != nil {
		t.Fatal("fixture connection configuration unavailable")
	}
	database := sql.OpenDB(connector)
	defer database.Close()
	ctx, cancel := context.WithTimeout(context.Background(), checkTimeout)
	defer cancel()
	if err := checkDatabase(ctx, database, cfg.DBName); err != nil {
		t.Fatal(err)
	}
}
