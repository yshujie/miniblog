// Package mysqlconfig centralizes MySQL connection flags used by operational scripts.
package mysqlconfig

import (
	"flag"
	"os"

	"github.com/yshujie/miniblog/pkg/db"
)

// Config defers environment access until a command actually needs a database.
// Flag defaults deliberately contain no environment values: flag's help and
// parse-error usage print defaults even when the command never connects.
type Config struct {
	flags                                    *flag.FlagSet
	host, port, username, password, database string
}

func Bind(flags *flag.FlagSet) *Config {
	config := &Config{flags: flags}
	flags.StringVar(&config.host, "host", "", "MySQL 主机；默认读取 MYSQL_HOST；MYSQL_DSN 非空时忽略")
	flags.StringVar(&config.port, "port", "", "MySQL 端口；默认读取 MYSQL_PORT；MYSQL_DSN 非空时忽略")
	flags.StringVar(&config.username, "user", "", "MySQL 用户名；默认读取 MYSQL_USERNAME；MYSQL_DSN 非空时忽略")
	flags.StringVar(&config.password, "db-password", "", "兼容参数；请优先用环境 MYSQL_PASSWORD 或 MYSQL_DSN 传递密码")
	flags.StringVar(&config.database, "database", "", "MySQL 数据库名；默认读取 MYSQL_DATABASE；MYSQL_DSN 非空时忽略")
	return config
}

// DBOptions preserves explicit flags (including an explicitly empty password),
// legacy MYSQL_* fallbacks and MYSQL_DSN precedence. Credentials are read only
// here, after help/validation and database-free schema checks have branched off.
func (c *Config) DBOptions(logLevel int) *db.MySQLOptions {
	explicit := map[string]bool{}
	c.flags.Visit(func(value *flag.Flag) { explicit[value.Name] = true })
	resolve := func(flagName, value, envName, fallback string) string {
		if explicit[flagName] {
			return value
		}
		return envOr(envName, fallback)
	}
	return &db.MySQLOptions{
		DSN:      os.Getenv("MYSQL_DSN"),
		Host:     resolve("host", c.host, "MYSQL_HOST", "localhost"),
		Port:     resolve("port", c.port, "MYSQL_PORT", "3306"),
		Username: resolve("user", c.username, "MYSQL_USERNAME", "miniblog"),
		Password: resolve("db-password", c.password, "MYSQL_PASSWORD", "miniblog123"),
		Database: resolve("database", c.database, "MYSQL_DATABASE", "miniblog"),
		LogLevel: logLevel,
	}
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
