package main

import (
	"github.com/yshujie/miniblog/internal/config"
	"github.com/yshujie/miniblog/pkg/app"
)

func main() {
	// 加载配置文件
	cfg, err := config.LoadConfig()
	if err != nil {
		return
	}

	// 启动 app
	app.NewApp(cfg).Run()
}
