package db

import (
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// MySQLOptions 定义MySQL数据库的配置选项
type MySQLOptions struct {
	Host                  string        // 主机
	Port                  string        // 端口
	Username              string        // 用户名
	Password              string        // 密码
	Database              string        // 数据库名称
	MaxIdleConns          int           // 最大空闲连接数
	MaxOpenConns          int           // 最大打开连接数
	MaxConnectionLifeTime time.Duration // 最大连接生命周期
	LogLevel              int           // 日志级别
}

// DNS 生成MySQL的连接字符串
func (o *MySQLOptions) DNS() string {
	return fmt.Sprintf(`%s:%s@tcp(%s)/%s?charset=utf8&parseTime=%t&loc=%s`,
		o.Username,
		o.Password,
		o.Host,
		o.Database,
		true,
		"Local")
}

func NewMySQL(opts *MySQLOptions) (*gorm.DB, error) {
	logLevel := logger.Silent
	if opts.LogLevel != 0 {
		logLevel = logger.LogLevel(opts.LogLevel)
	}

	db, err := gorm.Open(mysql.Open(opts.DNS()), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	// 配置数据库
	sqlDB.SetMaxIdleConns(opts.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(opts.MaxConnectionLifeTime)
	sqlDB.SetMaxIdleConns(opts.MaxIdleConns)

	return db, nil
}
