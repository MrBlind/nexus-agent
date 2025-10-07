package client

import (
	"context"

	"github.com/mrblind/nexus-agent/internal/config"
	"github.com/redis/go-redis/v9"
)

// RedisClient wraps the Redis client.
type RedisClient struct {
	Client *redis.Client
}

// NewRedisClient constructs a Redis client from configuration.
func NewRedisClient(cfg config.RedisConfig) (*RedisClient, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}

	return &RedisClient{Client: client}, nil
}
