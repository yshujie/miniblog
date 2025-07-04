package miniblog

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yshujie/miniblog/internal/miniblog/store"
	"github.com/yshujie/miniblog/internal/pkg/log"
	"github.com/yshujie/miniblog/pkg/db"
)

const (
	// 定义放置 miniblog 服务配置的默认目录
	recommendedHomeDir = ".miniblog"

	// 定义放置 miniblog 服务配置的默认目录
	recommendedConfigDir = "configs"

	// 默认的配置文件名称
	defaultConfigName = "miniblog.yaml"
)

// initConfig 初始化配置
// Viper 的优先级机制：
// 第一优先级：显式调用 Set 设置的值（优先级最高，本项目不使用）
// 第二优先级：命令行参数（本项目不使用）
// 第三优先级：环境变量
// 第四优先级：配置文件
// 第五优先级：默认值（优先级最低）
func initConfig() {
	log.Infow("init config")

	// 从环境变量中读取配置
	loadConfigFromEnv()

	// 从配置文件中读取配置
	loadConfigFromFile()

	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		fmt.Printf("Error reading config file, %s\n", err)
	} else {
		fmt.Printf("Using config file: %s\n", viper.ConfigFileUsed())
	}

	// 打印配置
	log.Infow("config initialized successfully", "config", viper.AllSettings())
}

// loadConfigFromEnv 从环境变量中读取配置
func loadConfigFromEnv() {
	// 1. 所有 env 都以 MINIBLOG_ 开头
	viper.SetEnvPrefix("MINIBLOG")

	// 2. key path 用点分隔，自动变成下划线
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// 3. 开启自动读取
	viper.AutomaticEnv()

	// 打印环境变量
	log.Infow("environment variables",
		"MYSQL_PORT", os.Getenv("MINIBLOG_DATABASE_PORT"),
		"MYSQL_USERNAME", os.Getenv("MINIBLOG_DATABASE_USERNAME"),
		"MYSQL_PASSWORD", os.Getenv("MINIBLOG_DATABASE_PASSWORD"),
		"MYSQL_DBNAME", os.Getenv("MINIBLOG_DATABASE_DBNAME"),
		"MYSQL_HOST", os.Getenv("MINIBLOG_DATABASE_HOST"),
		"REDIS_HOST", os.Getenv("MINIBLOG_REDIS_HOST"),
		"REDIS_PORT", os.Getenv("MINIBLOG_REDIS_PORT"),
		"REDIS_PASSWORD", os.Getenv("MINIBLOG_REDIS_PASSWORD"),
		"REDIS_DB", os.Getenv("MINIBLOG_REDIS_DB"),
		"JWT_SECRET", os.Getenv("MINIBLOG_JWT_SECRET"),
		"FEISHU_DOCREADER_APPID", os.Getenv("MINIBLOG_FEISHU_DOCREADER_APPID"),
		"FEISHU_DOCREADER_APPSECRET", os.Getenv("MINIBLOG_FEISHU_DOCREADER_APPSECRET"),
	)
}

// loadConfigFromFile 从配置文件中读取配置
func loadConfigFromFile() {
	// 第一优先级：命令行参数指定的配置文件
	if isConfigFileSpecified() {
		loadConfigFromCmd()
		return
	}

	// 第二优先级：根据默认配置文件名称进行查找
	loadConfigFromDefaultDir()
}

// 是否指定了配置文件
func isConfigFileSpecified() bool {
	return cfgFile != ""
}

// loadConfigFromCmd 从命令行参数中读取配置
func loadConfigFromCmd() {
	viper.SetConfigFile(cfgFile)
}

// loadConfigFromDefaultDir 从默认目录中读取配置
func loadConfigFromDefaultDir() {
	// 设置配置文件的搜索路径
	// 第一优先级，当前用户主目录下的 .miniblog 目录
	home, err := os.UserHomeDir()
	cobra.CheckErr(err)
	viper.AddConfigPath(filepath.Join(home, recommendedHomeDir))

	// 第二优先级，当前工作目录
	viper.AddConfigPath(recommendedConfigDir)

	// 设置配置文件的类型
	viper.SetConfigType("yaml")

	// 根据默认配置文件名称进行查找
	viper.SetConfigName(defaultConfigName)
}

func logOptions() *log.Options {
	return &log.Options{
		DisableCaller:     viper.GetBool("log.disable-caller"),
		DisableStacktrace: viper.GetBool("log.disable-stacktrace"),
		Level:             viper.GetString("log.level"),
		Format:            viper.GetString("log.format"),
		OutputPaths:       viper.GetStringSlice("log.output-paths"),
	}
}

// initStore 读取 db 配置，创建 gorm.DB 实例，并初始化 miniblog store 层
func initStore() error {
	dbOpts := &db.MySQLOptions{
		Host:                  viper.GetString("database.host"),
		Port:                  viper.GetString("database.port"),
		Username:              viper.GetString("database.username"),
		Password:              viper.GetString("database.password"),
		Database:              viper.GetString("database.dbname"),
		MaxIdleConns:          viper.GetInt("database.max-idle-conns"),
		MaxOpenConns:          viper.GetInt("database.max-open-conns"),
		MaxConnectionLifeTime: viper.GetDuration("database.conn-max-lifetime"),
		LogLevel:              viper.GetInt("database.log-level"),
	}

	db, err := db.NewMySQL(dbOpts)
	if err != nil {
		return err
	}

	store.NewStore(db)

	return nil
}
