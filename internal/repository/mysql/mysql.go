package mysql

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
	"github.com/yshujie/blog-serve/internal/config"
	"github.com/yshujie/blog-serve/pkg/log"
)

var db *sql.DB

func Init(cfg *config.Database) {
	logger := log.NewLogger()
	logger.Info("Connecting to MySQL...")
	// 连接数据库
	var err error
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.Username,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.DBName,
	)
	logger.Info("connect dsn: %s", dsn)

	// 打开数据库
	db, err = sql.Open(cfg.Driver, dsn)
	if err != nil {
		panic(err)
	}

	// 设置最大连接数
	db.SetMaxOpenConns(100)

	// 设置最大空闲连接数
	db.SetMaxIdleConns(10)

	// 测试连接
	err = db.Ping()
	if err != nil {
		panic(err)
	}

	logger.Info("mysql connect success")
}

func Close() {
	db.Close()
}
