package config

// 服务器配置
type Server struct {
	Name    string `mapstructure:"name"`
	Version string `mapstructure:"version"`
	Address string `mapstructure:"address"`
	Port    int    `mapstructure:"port"`
}

// 数据库配置
type Database struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Name     string `mapstructure:"name"`
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
