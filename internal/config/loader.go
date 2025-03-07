package config

import (
	"github.com/spf13/viper"
	"github.com/yshujie/blog-serve/pkg/log"
)

// LoadConfig 加载配置文件
func LoadConfig() (*Config, error) {
	logger := log.NewLogger()
	logger.Info("Loading config...")

	viper.SetConfigFile(configFilePath)
	if err := viper.ReadInConfig(); err != nil {
		logger.Error("Failed to load config: %v", err)
		return nil, err
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		logger.Error("Failed to unmarshal config: %v", err)
		return nil, err
	}

	logger.Info("Config loaded successfully")
	return &config, nil
}
