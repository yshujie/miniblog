package app

import (
	"github.com/yshujie/blog-serve/internal/config"
	redis "github.com/yshujie/blog-serve/internal/repository/cache"
	"github.com/yshujie/blog-serve/internal/repository/mysql"
	"github.com/yshujie/blog-serve/pkg/log"
)

type App struct {
	name    string
	version string
	ip      string
	port    int
	cfg     *config.Config
}

func NewApp(cfg *config.Config) *App {
	return &App{
		cfg: cfg,
	}
}

// 启动 app
func (a *App) Run() {
	// 日志
	logger := log.NewLogger()

	// 记录 app 开启
	logger.Info("%s app start, version: %s, ip: %s, port: %d", a.name, a.version, a.ip, a.port)

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
	// // 启动 router
	// router := gin.Default()

	// // 启动 server
	// router.Run(fmt.Sprintf("%s:%d", a.ip, a.port))
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
