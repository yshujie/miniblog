// Package mysqlconfig centralizes MySQL connection flags used by operational scripts.
package mysqlconfig

import (
	"flag"
	"os"

	"github.com/yshujie/miniblog/pkg/db"
)

// Config contains the shared MySQL connection inputs for operational scripts.
type Config struct {
	dsn      string
	host     string
	port     string
	username string
	password string
	database string
}

// Bind registers the legacy connection flags and reads MYSQL_DSN.
// MYSQL_DSN is intentionally environment-only so credentials do not need to be
// passed as a command-line argument.
func Bind(flags *flag.FlagSet) *Config {
	config := &Config{dsn: os.Getenv("MYSQL_DSN")}

	flags.StringVar(&config.host, "host", envOr("MYSQL_HOST", "localhost"), "MySQL 主机；MYSQL_DSN 非空时忽略")
	flags.StringVar(&config.port, "port", envOr("MYSQL_PORT", "3306"), "MySQL 端口；MYSQL_DSN 非空时忽略")
	flags.StringVar(&config.username, "user", envOr("MYSQL_USERNAME", "miniblog"), "MySQL 用户名；MYSQL_DSN 非空时忽略")
	flags.StringVar(&config.password, "db-password", envOr("MYSQL_PASSWORD", "miniblog123"), "MySQL 密码；MYSQL_DSN 非空时忽略")
	flags.StringVar(&config.database, "database", envOr("MYSQL_DATABASE", "miniblog"), "MySQL 数据库名；MYSQL_DSN 非空时忽略")

	return config
}

// DBOptions builds the database options consumed by pkg/db.
func (c *Config) DBOptions(logLevel int) *db.MySQLOptions {
	return &db.MySQLOptions{
		DSN:      c.dsn,
		Host:     c.host,
		Port:     c.port,
		Username: c.username,
		Password: c.password,
		Database: c.database,
		LogLevel: logLevel,
	}
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
