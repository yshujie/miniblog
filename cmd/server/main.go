package main

import (
	"log"

	"github.com/yshujie/blog-serve/internal/config"
	"github.com/yshujie/blog-serve/pkg/app"
)

func main() {
	// 加载配置文件
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 初始化 app
	app := app.NewApp(cfg)

	// 启动 app
	app.Run()
}
