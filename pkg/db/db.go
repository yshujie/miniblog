package db

import (
	"context"
	"fmt"
	"time"

	"github.com/yshujie/miniblog/internal/pkg/log"
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
	return fmt.Sprintf(`%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=%t&loc=%s`,
		o.Username,
		o.Password,
		o.Host,
		o.Port,
		o.Database,
		true,
		"Local")
}

// 自定义 GORM 日志记录器
type gormLogger struct {
	LogLevel logger.LogLevel
}

func (l *gormLogger) LogMode(level logger.LogLevel) logger.Interface {
	newLogger := *l
	newLogger.LogLevel = level
	return &newLogger
}

func (l *gormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= logger.Info {
		log.Infow(msg, data...)
	}
}

func (l *gormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= logger.Warn {
		log.Warnw(msg, data...)
	}
}

func (l *gormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= logger.Error {
		log.Errorw(msg, data...)
	}
}

func (l *gormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.LogLevel <= logger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()

	// 构建日志消息
	msg := fmt.Sprintf("[%.3f ms] [rows:%v] %s", float64(elapsed.Microseconds())/1000, rows, sql)

	if err != nil {
		log.Errorw("SQL Error", "error", err, "sql", msg)
		return
	}

	if l.LogLevel >= logger.Info {
		log.Infow("SQL Query", "sql", msg)
	}
}

func NewMySQL(opts *MySQLOptions) (*gorm.DB, error) {
	logLevel := logger.Silent
	if opts.LogLevel != 0 {
		logLevel = logger.LogLevel(opts.LogLevel)
	}

	// 创建自定义日志记录器
	customLogger := &gormLogger{
		LogLevel: logLevel,
	}

	db, err := gorm.Open(mysql.Open(opts.DNS()), &gorm.Config{
		Logger: customLogger,
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
