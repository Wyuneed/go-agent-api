package redis

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/wyuneed/go-agent-api/internal/infrastructure/config"
)

func NewConnection(ctx context.Context, cfg config.RedisConfig) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr(),
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("ping: %w", err)
	}

	return client, nil
}
