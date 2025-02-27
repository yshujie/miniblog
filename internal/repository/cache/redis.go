package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/yshujie/blog-serve/internal/config"
	"github.com/yshujie/blog-serve/pkg/log"
)

var rdb *redis.Client

func Init(cfg *config.Redis) {
	logger := log.NewLogger()
	logger.Info("Connecting to Redis...")

	// 创建 redis 客户端，测试连接
	_, err := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	}).Ping(context.Background()).Result()

	if err != nil {
		logger.Error("Failed to connect to Redis: %v", err)
		panic(err)
	}

	logger.Info("Redis connected successfully")
}

func Close() {
	rdb.Close()
}

func Get(key string) (string, error) {
	return rdb.Get(context.Background(), key).Result()
}

func Set(key string, value interface{}, expiration time.Duration) error {
	return rdb.Set(context.Background(), key, value, expiration).Err()
}

func Del(key string) error {
	return rdb.Del(context.Background(), key).Err()
}

func Expire(key string, expiration time.Duration) error {
	return rdb.Expire(context.Background(), key, expiration).Err()
}
