package mysql

import (
	"database/sql"
	"fmt"

	"github.com/yshujie/blog-serve/internal/config"
)

var db *sql.DB

func Init(cfg *config.Database) {
	// 连接数据库
	var err error
	db, err = sql.Open("mysql",
		fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			cfg.User,
			cfg.Password,
			cfg.Host,
			cfg.Port,
			cfg.Name,
		),
	)
	if err != nil {
		panic(err)
	}

	// 设置最大连接数
	db.SetMaxOpenConns(100)

	// 设置最大空闲连接数
	db.SetMaxIdleConns(10)

}

func Close() {
	db.Close()
}
