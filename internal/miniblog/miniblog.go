package miniblog

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yshujie/miniblog/internal/pkg/known"
	"github.com/yshujie/miniblog/internal/pkg/log"
	mw "github.com/yshujie/miniblog/internal/pkg/middleware"
	"github.com/yshujie/miniblog/pkg/token"
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
			log.Infow("log system initialized successfully")
			defer log.Sync() // Sync 将缓存中的日志刷新到磁盘文件中

			// 运行服务
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

	// 返回 cobra.Command 对象
	return cmd
}

// run 函数，实际的业务代码入口
func run() error {
	log.Infow("miniblog serve is running...")

	// 初始化 store
	if err := initStore(); err != nil {
		return err
	}

	// 初始化 token
	token.Init(viper.GetString("jwt.secret"), known.XUsernameKey)

	// 设置 Gin 模式
	gin.SetMode(viper.GetString("server.run-mode"))

	// 创建 Gin 引擎
	g := gin.New()

	// 注册中间件
	mws := []gin.HandlerFunc{gin.Recovery(), mw.NoCache, mw.Cors, mw.Secure, mw.RequestID(), mw.Logger()}
	g.Use(mws...)

	// 安装路由
	if err := installRouters(g); err != nil {
		return err
	}

	// 启动 HTTP 服务器
	httpSrv := startInsecureServer(g)

	// 启动 HTTPS 服务器
	// 启动 HTTPS 服务（如果证书存在则启动，否则跳过，由 infra 负责 TLS 终端）
	httpsSrv := startSecureServer(g)

	// 等待中断信号优雅地关闭服务器（10 秒超时)。
	quit := make(chan os.Signal, 1)
	// kill 默认会发送 syscall.SIGTERM 信号
	// kill -2 发送 syscall.SIGINT 信号，我们常用的 CTRL + C 就是触发系统 SIGINT 信号
	// kill -9 发送 syscall.SIGKILL 信号，但是不能被捕获，所以不需要添加它
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM) // 此处不会阻塞
	<-quit                                               // 阻塞在此，当接收到上述两种信号时才会往下执行
	log.Infow("Shutting down server ...")

	// 创建 ctx 用于通知服务器 goroutine, 它有 10 秒时间完成当前正在处理的请求
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 10 秒内优雅关闭服务（将未处理完的请求处理完再关闭服务），超过 10 秒就超时退出
	if err := httpSrv.Shutdown(ctx); err != nil {
		log.Errorw("Insecure Server forced to shutdown", "err", err)
		return err
	}
	if httpsSrv != nil {
		if err := httpsSrv.Shutdown(ctx); err != nil {
			log.Errorw("Secure Server forced to shutdown", "err", err)
			return err
		}
	}

	log.Infow("Server exiting")

	return nil
}

// startInsecureServer 启动 HTTP 服务器
func startInsecureServer(g *gin.Engine) *http.Server {
	// 创建 HTTP Server 实例
	httpSrv := &http.Server{
		Addr:    viper.GetString("server.address") + ":" + viper.GetString("server.port"),
		Handler: g,
	}

	// 打印日志
	log.Infow("Start to listening the incoming requests on port " + viper.GetString("server.address"))

	// 运行 HTTP Server
	go func() {
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalw(err.Error())
		}
	}()

	return httpSrv
}

// startSecureServer 启动 HTTPS 服务器
func startSecureServer(g *gin.Engine) *http.Server {
	// 创建 HTTPS Server 实例
	httpsSrv := &http.Server{
		Addr:    viper.GetString("server.address") + ":" + viper.GetString("server.port-ssl"),
		Handler: g,
	}

	// 打印日志
	log.Infow("Start to listening the incoming requests on port " + viper.GetString("server.address"))

	// 检查证书和密钥文件是否存在；若不存在，则根据 TLS_STRICT 控制是 fatal 还是跳过 HTTPS
	certPath := viper.GetString("tls.cert")
	keyPath := viper.GetString("tls.key")

	// TLS_STRICT 优先从 viper 配置读取（tls.strict），否则从环境变量 TLS_STRICT 读取（true/1 表示严格）
	tlsStrict := false
	if viper.IsSet("tls.strict") {
		tlsStrict = viper.GetBool("tls.strict")
	} else {
		if val := strings.ToLower(strings.TrimSpace(os.Getenv("TLS_STRICT"))); val == "1" || val == "true" || val == "yes" {
			tlsStrict = true
		}
	}

	certInfo, certErr := os.Stat(certPath)
	keyInfo, keyErr := os.Stat(keyPath)

	if certErr != nil || keyErr != nil {
		// 如果严格模式，停止启动并记录错误；否则仅发出警告并跳过 HTTPS
		if tlsStrict {
			log.Fatalw("TLS certificate or key missing (TLS_STRICT enabled)", "cert", certErr, "key", keyErr, "certPath", certPath, "keyPath", keyPath)
			return nil
		}
		log.Warnw("TLS certificate/key not found, skipping HTTPS server startup", "certErr", certErr, "keyErr", keyErr, "certPath", certPath, "keyPath", keyPath)
		return nil
	}

	// 记录证书/密钥的大小和权限信息，便于排查
	certPerm := certInfo.Mode().Perm()
	keyPerm := keyInfo.Mode().Perm()
	log.Infow("TLS certificate and key found, starting HTTPS server", "cert", certPath, "cert_size", certInfo.Size(), "cert_perm", certPerm, "key", keyPath, "key_size", keyInfo.Size(), "key_perm", keyPerm)

	// 私钥权限不应过于宽松（建议 0600），如权限包含 group/others 可读则警告
	if keyPerm&0077 != 0 {
		log.Warnw("TLS private key permissions are too permissive, consider setting to 0600", "key", keyPath, "perm", keyPerm)
	}

	// 运行 HTTPS Server
	go func() {
		if err := httpsSrv.ListenAndServeTLS(viper.GetString("tls.cert"), viper.GetString("tls.key")); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalw(err.Error())
		}
	}()

	return httpsSrv
}
