package main

import (
	"github.com/yshujie/blog-serve/internal/config"
	"github.com/yshujie/blog-serve/pkg/app"
)

func main() {
	// 加载配置文件
	cfg, err := config.LoadConfig("internal/config/config.yaml")
	if err != nil {
		return
	}

	// 启动 app
	app.NewApp(cfg).Run()
}
