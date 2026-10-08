// content-preflight checks deployment prerequisites without changing database data or schema.
package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	mysql "github.com/go-sql-driver/mysql"
)

const checkTimeout = 30 * time.Second

func main() { os.Exit(run(os.Getenv, os.Stdout, os.Stderr)) }

func run(env func(string) string, stdout, stderr io.Writer) int {
	cfg, err := configFromEnvironment(env)
	if err != nil {
		fmt.Fprintln(stderr, "content preflight failed:", err)
		return 1
	}
	// Driver errors can include the host, account or database name. They are never
	// printed; only fixed check failures or static schema identifiers are reported.
	// Disable driver logs as well as sanitizing returned errors.
	_ = mysql.SetLogger(quietLogger{})
	connector, err := mysql.NewConnector(cfg)
	if err != nil {
		fmt.Fprintln(stderr, "content preflight failed: database connection unavailable")
		return 1
	}
	database := sql.OpenDB(connector)
	defer database.Close()
	database.SetMaxOpenConns(1)
	database.SetMaxIdleConns(1)
	ctx, cancel := context.WithTimeout(context.Background(), checkTimeout)
	defer cancel()
	if err := checkDatabase(ctx, database, cfg.DBName); err != nil {
		fmt.Fprintln(stderr, "content preflight failed:", err)
		return 1
	}
	fmt.Fprintln(stdout, "content preflight passed: migration version >= 6 is clean; required schema and unique indexes are ready")
	return 0
}

func checkDatabase(ctx context.Context, database *sql.DB, schema string) error {
	if err := database.PingContext(ctx); err != nil {
		return errors.New("database connection unavailable")
	}
	// The server also enforces the read-only boundary. There is no Exec, DDL,
	// application-row scan, credential flag or migration fallback in this tool.
	tx, err := database.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return errors.New("read-only schema inspection unavailable")
	}
	defer tx.Rollback()
	return inspect(ctx, sqlMetadata{tx: tx, schema: schema})
}

func configFromEnvironment(env func(string) string) (*mysql.Config, error) {
	first := func(keys ...string) string {
		for _, key := range keys {
			if value := env(key); value != "" {
				return value
			}
		}
		return ""
	}
	host := first("MINIBLOG_DATABASE_HOST", "MYSQL_HOST")
	port := first("MINIBLOG_DATABASE_PORT", "MYSQL_PORT")
	user := first("MINIBLOG_DATABASE_USERNAME", "MYSQL_USERNAME", "MYSQL_USER")
	password := first("MINIBLOG_DATABASE_PASSWORD", "MYSQL_PASSWORD")
	name := first("MINIBLOG_DATABASE_DBNAME", "MINIBLOG_DATABASE_DATABASE", "MYSQL_DATABASE", "MYSQL_DBNAME")
	if host == "" || user == "" || password == "" || name == "" {
		return nil, errors.New("database environment is incomplete (host, username, password and database are required)")
	}
	if port == "" {
		port = "3306"
	}
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 {
		return nil, errors.New("database port is invalid")
	}
	if strings.ContainsAny(host, "/\x00\r\n") || strings.ContainsAny(name, "\x00\r\n") {
		return nil, errors.New("database environment is invalid")
	}
	cfg := mysql.NewConfig()
	cfg.Net, cfg.Addr = "tcp", net.JoinHostPort(host, port)
	cfg.User, cfg.Passwd, cfg.DBName = user, password, name
	cfg.Timeout, cfg.ReadTimeout, cfg.WriteTimeout = 10*time.Second, 10*time.Second, 10*time.Second
	cfg.Params = map[string]string{"charset": "utf8mb4"}
	return cfg, nil
}

type quietLogger struct{}

func (quietLogger) Print(...interface{}) {}
