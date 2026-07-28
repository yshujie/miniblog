// 一次性工具：按 user.id 重置管理员密码。
//
// 用法:
//
//	go run ./scripts/reset-user-password -id 1 -password 'NewPass123'
//	./scripts/reset-user-password.sh -id 1 -password 'NewPass123'
//
// 数据库连接默认读取 MYSQL_* 环境变量，与 docker-compose 一致。
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/pkg/auth"
	"github.com/yshujie/miniblog/pkg/db"
	"gorm.io/gorm"
)

func main() {
	os.Exit(run())
}

func run() int {
	var (
		userID   = flag.Uint64("id", 0, "要重置密码的用户 ID")
		password = flag.String("password", "", "新密码（6-18 位）；也可用环境变量 RESET_PASSWORD")
		host     = flag.String("host", envOr("MYSQL_HOST", "localhost"), "MySQL 主机")
		port     = flag.String("port", envOr("MYSQL_PORT", "3306"), "MySQL 端口")
		username = flag.String("user", envOr("MYSQL_USERNAME", "miniblog"), "MySQL 用户名")
		dbPass   = flag.String("db-password", envOr("MYSQL_PASSWORD", "miniblog123"), "MySQL 密码")
		database = flag.String("database", envOr("MYSQL_DATABASE", "miniblog"), "MySQL 数据库名")
	)
	flag.Parse()

	if *userID == 0 {
		fmt.Fprintln(os.Stderr, "error: 必须指定 -id")
		flag.Usage()
		return 1
	}

	if *password == "" {
		*password = os.Getenv("RESET_PASSWORD")
	}
	if len(*password) < 6 || len(*password) > 18 {
		fmt.Fprintln(os.Stderr, "error: 新密码长度须为 6-18 位")
		return 1
	}

	gdb, err := db.NewMySQL(&db.MySQLOptions{
		Host:     *host,
		Port:     *port,
		Username: *username,
		Password: *dbPass,
		Database: *database,
		LogLevel: 1,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: 连接数据库失败: %v\n", err)
		return 1
	}

	var user model.UserM
	if err := gdb.Where("id = ?", *userID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			fmt.Fprintf(os.Stderr, "error: 未找到 id=%d 的用户\n", *userID)
			return 1
		}
		fmt.Fprintf(os.Stderr, "error: 查询用户失败: %v\n", err)
		return 1
	}

	hashed, err := auth.Encrypt(*password)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: 密码加密失败: %v\n", err)
		return 1
	}

	result := gdb.Model(&model.UserM{}).Where("id = ?", *userID).Updates(map[string]any{
		"password":   hashed,
		"updated_at": time.Now(),
	})
	if result.Error != nil {
		fmt.Fprintf(os.Stderr, "error: 更新密码失败: %v\n", result.Error)
		return 1
	}
	if result.RowsAffected == 0 {
		fmt.Fprintf(os.Stderr, "error: 未更新任何记录（id=%d）\n", *userID)
		return 1
	}

	fmt.Printf("密码已重置: id=%d username=%s\n", user.ID, user.Username)
	return 0
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
