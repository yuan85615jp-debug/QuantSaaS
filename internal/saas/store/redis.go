package store

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/saas/config"
)

// Redis is a thin wrapper. Use only for cache (champion genes, sessions).
// Never use Redis as a signal or command bus.
type Redis struct {
	client *redis.Client
}

func NewRedis(cfg *config.Config) (*Redis, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}
	return &Redis{client: rdb}, nil
}

func (r *Redis) Get(ctx context.Context, key string) (string, error) {
	return r.client.Get(ctx, key).Result()
}

func (r *Redis) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return r.client.Set(ctx, key, value, ttl).Err()
}

func (r *Redis) Del(ctx context.Context, keys ...string) error {
	return r.client.Del(ctx, keys...).Err()
}

func ChampionKey(strategyID string) string {
	return fmt.Sprintf("champion:%s", strategyID)
}

func (r *Redis) Close() error {
	return r.client.Close()
}

func (r *Redis) Client() *redis.Client {
	return r.client
}
