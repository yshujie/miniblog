package config

// config 文件地址
var configFilePath = "/configs/config.yaml"

// 服务器配置
type Server struct {
	Name    string `mapstructure:"name"`
	Version string `mapstructure:"version"`
	Address string `mapstructure:"address"`
	Port    int    `mapstructure:"port"`
}

// 数据库配置
type Database struct {
	Driver   string `mapstructure:"driver"`
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	MaxIdle  int    `mapstructure:"max_idle"`
	MaxOpen  int    `mapstructure:"max_open"`
	ConnMax  int    `mapstructure:"conn_max_lifetime"`
}

// Redis 配置
type Redis struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type Config struct {
	Server   Server   // 修改为Server类型
	Database Database // 修改为Database类型
	Redis    Redis    // 修改为Redis类型
}
