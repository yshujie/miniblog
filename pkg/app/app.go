package app

import (
	"github.com/yshujie/blog-serve/internal/config"
	router "github.com/yshujie/blog-serve/internal/delivery/http"
	redis "github.com/yshujie/blog-serve/internal/store/cache"
	mysql "github.com/yshujie/blog-serve/internal/store/mysql"
	"github.com/yshujie/blog-serve/pkg/log"
)

// App 应用结构体
type App struct {
	name    string
	version string
	ip      string
	port    int
	cfg     *config.Config
}

// 创建 App 实例
func NewApp(cfg *config.Config) *App {
	logger := log.NewLogger()
	logger.Info("Creating app instance...")

	return &App{
		name:    cfg.Server.Name,
		version: cfg.Server.Version,
		ip:      cfg.Server.Address,
		port:    cfg.Server.Port,
		cfg:     cfg,
	}
}

// 启动 app
func (a *App) Run() {
	// 日志
	logger := log.NewLogger()
	logger.Info("Starting app...")

	// 准备
	a.prepare()

	// 启动
	a.start()

	// 记录 app 成功启动
	logger.Info("%s app start success, version: %s, ip: %s, port: %d", a.name, a.version, a.ip, a.port)
}

// 启动前准备
func (a *App) prepare() {
	// 初始化 mysql
	mysql.Init(&a.cfg.Database)

	// 初始化 redis
	redis.Init(&a.cfg.Redis)
}

// 启动
func (a *App) start() {
	// 启动 router
	router.Start(&a.ip, &a.port)
}

// 关闭
func (a *App) Shutdown() {
	// // 关闭 router
	// a.router.Shutdown()

	// // 关闭 mysql
	// mysql.Close()

	// // 关闭 redis
	// redis.Close()
}
