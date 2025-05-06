package miniblog

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yshujie/miniblog/internal/pkg/log"
	mw "github.com/yshujie/miniblog/internal/pkg/middleware"
)

var (
	cfgFile string // 配置文件路径
)

// NewMiniBlogCommand 创建博客的 *cobra.Command 对象
// 之后可通过 Command 对象的 Execute 方法启动应用程序
func NewMiniBlogCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "miniblog",                                                                // 指定命令名字
		Short:        "A good Go pratical project",                                              // 命令的简述
		Long:         "A good Go pratical project, used to create user with basic information.", // 命令的详细描述
		SilenceUsage: true,                                                                      // 静默命令执行错误
		// cmd.Execute() 方法执行时，会调用 RunE 方法，执行 run() 方法
		RunE: func(cmd *cobra.Command, args []string) error {
			// 初始化日志
			log.Init(logOptions())
			defer log.Sync() // Sync 将缓存中的日志刷新到磁盘文件中

			return run()
		},
		// 命令运行时，不需要指定命令行参数
		Args: func(cmd *cobra.Command, args []string) error {
			for _, arg := range args {
				if len(arg) > 0 {
					return fmt.Errorf("%q does not take any arguments, got %q", cmd.CommandPath(), arg)
				}
			}
			return nil
		},
	}

	// 设定运行时执行 initConfig 方法
	cobra.OnInitialize(initConfig)

	// 设置 cobra 的持久化标志：设置 config 文件路径
	cmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "The path to the miniblog configuration file. Empty string for no configuration file.")

	return cmd
}

// run 函数，实际的业务代码入口
func run() error {
	log.Infow("miniblog serve is running...")

	// 设置 Gin 模式
	gin.SetMode(viper.GetString("runmode"))

	// 创建 Gin 引擎
	g := gin.New()

	// 注册中间件
	mws := []gin.HandlerFunc{mw.RequestID()}
	g.Use(mws...)

	// 注册 404 Handler
	g.NoRoute(func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{
			"code":    10003,
			"message": "Page not found",
		})
	})

	// 注册 /health 路由
	g.GET("/health", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// 创建 HTTP Server 实例
	httpSrv := &http.Server{Addr: viper.GetString("addr"), Handler: g}

	// 打印日志
	log.Infow("Start to listening the incoming requests on port " + viper.GetString("addr"))

	// 运行 HTTP Server
	if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalw("Failed to start HTTP server", "error", err)
	}

	return nil
}
