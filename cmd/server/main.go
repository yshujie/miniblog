package main

import (
	"github.com/yshujie/blog-serve/internal/config"
	"github.com/yshujie/blog-serve/pkg/app"
	"github.com/yshujie/blog-serve/pkg/log"
)

func main() {
	// 初始化日志
	logger := log.NewLogger()
	logger.Info("Starting server...")

	// 加载配置文件
	logger.Info("Loading config...")
	cfg, err := config.LoadConfig("internal/config/config.yaml")
	if err != nil {
		logger.Error("Failed to load config: %v", err)
		return
	}
	logger.Info("Config loaded successfully")

	// 启动 app
	logger.Info("Starting app...")
	app.NewApp(cfg).Run()
	logger.Info("App started successfully")
}
