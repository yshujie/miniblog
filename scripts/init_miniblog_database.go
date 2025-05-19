package main

import (
	"context"
	"fmt"
	"log"

	"github.com/spf13/viper"
	"github.com/yshujie/miniblog/internal/miniblog/model"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"github.com/yshujie/miniblog/pkg/db"
)

func main() {
	// 设置配置文件路径
	viper.SetConfigFile("configs/miniblog.yaml")
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("读取配置文件失败: %v", err)
	}

	// 初始化数据库连接
	dbOpts := &db.MySQLOptions{
		Host:                  viper.GetString("database.host"),
		Port:                  viper.GetString("database.port"),
		Username:              viper.GetString("database.username"),
		Password:              viper.GetString("database.password"),
		Database:              viper.GetString("database.dbname"),
		MaxIdleConns:          viper.GetInt("database.max_idle_conns"),
		MaxOpenConns:          viper.GetInt("database.max_open_conns"),
		MaxConnectionLifeTime: viper.GetDuration("database.conn_max_lifetime"),
		LogLevel:              viper.GetInt("database.log_level"),
	}

	db, err := db.NewMySQL(dbOpts)
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	// 初始化 store
	store.NewStore(db)

	// 创建上下文
	ctx := context.Background()

	// 初始化 module
	initModule(ctx)

	// 初始化章节数据
	initSection(ctx)

	fmt.Println("数据库初始化完成！")
}

func initModule(ctx context.Context) {
	modules := []*model.Module{
		{Code: "ai", Title: "AI"},
		{Code: "golang", Title: "Golang"},
		{Code: "ddd", Title: "领域驱动设计"},
		{Code: "architecture", Title: "架构设计"},
		{Code: "refactoring", Title: "重构"},
		{Code: "database", Title: "数据库"},
	}

	for _, module := range modules {
		if err := store.S.Modules().Create(ctx, module); err != nil {
			log.Printf("创建模块失败: %v", err)
		}
	}

	fmt.Println("模块初始化完成！")
}

func initSection(ctx context.Context) {
	sections := []*model.Section{
		{Code: "ai_history", ModuleCode: "ai", Title: "AI 发展史"},
		{Code: "prompt", ModuleCode: "ai", Title: "Prompt 工程"},
		{Code: "llm", ModuleCode: "ai", Title: "LLM 模型"},

		{Code: "golang_basic", ModuleCode: "golang", Title: "Golang 基础"},
		{Code: "golang_object_oriented", ModuleCode: "golang", Title: "Golang 与面向对象"},
		{Code: "design_pattern", ModuleCode: "golang", Title: "Golang 中的设计模式"},
		{Code: "golang_interview", ModuleCode: "golang", Title: "Golang 面试题"},

		{Code: "ddd_analysis", ModuleCode: "ddd", Title: "需求分析"},
		{Code: "ddd_domain_modeling", ModuleCode: "ddd", Title: "领域建模"},

		{Code: "architecture_design", ModuleCode: "architecture", Title: "软件架构"},
		{Code: "design_principles", ModuleCode: "architecture", Title: "设计原则"},
		{Code: "design_pattern", ModuleCode: "architecture", Title: "设计模式"},

		{Code: "bad_smell", ModuleCode: "refactoring", Title: "坏味道"},
		{Code: "refactoring_techniques", ModuleCode: "refactoring", Title: "重构技巧"},

		{Code: "mysql", ModuleCode: "database", Title: "MySQL 基础"},
		{Code: "mysql_optimization", ModuleCode: "database", Title: "MySQL 性能优化"},
		{Code: "mysql_interview", ModuleCode: "database", Title: "MySQL 面试题"},

		{Code: "redis", ModuleCode: "database", Title: "Redis 基础"},
		{Code: "redis_interview", ModuleCode: "database", Title: "Redis 面试题"},
	}

	for _, section := range sections {
		if err := store.S.Sections().Create(ctx, section); err != nil {
			log.Printf("创建章节失败: %v", err)
		}
	}

	fmt.Println("章节初始化完成！")
}
