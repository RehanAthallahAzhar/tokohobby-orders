package redis

import (
	"fmt"
	"log"
	"os"

	"github.com/RehanAthallahAzhar/tokohobby-orders/internal/configs"
	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
)

type RedisClient struct {
	Client *redis.Client
	log    *logrus.Logger
}

func NewRedisClient(cfg *configs.RedisConfig, log *logrus.Logger) (*RedisClient, error) {
	redisHost := os.Getenv("REDIS_HOST")
	if redisHost == "" {
		redisHost = "localhost"
	}

	redisPort := os.Getenv("REDIS_PORT")
	if redisPort == "" {
		redisPort = "6379"
	}

	redisAddr := fmt.Sprintf("%s:%s", redisHost, redisPort)

	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: "",
		DB:       0,
	})

	return &RedisClient{Client: rdb, log: log}, nil
}

func (rc *RedisClient) Close() {
	if rc.Client != nil {
		err := rc.Client.Close()
		if err != nil {
			log.Printf("Gagal menutup koneksi Redis: %v", err)
		}
	}
}
