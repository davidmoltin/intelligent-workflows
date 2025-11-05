package database

import (
	"context"
	"fmt"
	"time"

	"github.com/davidmoltin/intelligent-workflows/pkg/config"
	"github.com/davidmoltin/intelligent-workflows/pkg/logger"
	"github.com/redis/go-redis/v9"
)

// RedisClient wraps the Redis connection
type RedisClient struct {
	Client *redis.Client
}

// NewRedisClient creates a new Redis client connection
func NewRedisClient(cfg *config.Config, log *logger.Logger) (*RedisClient, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr(),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping redis: %w", err)
	}

	log.Info("Redis connection established",
		logger.String("host", cfg.Redis.Host),
		logger.Int("port", cfg.Redis.Port),
		logger.Int("db", cfg.Redis.DB),
	)

	return &RedisClient{Client: client}, nil
}

// Close closes the Redis connection
func (r *RedisClient) Close() error {
	return r.Client.Close()
}

// HealthCheck performs a health check on Redis
func (r *RedisClient) HealthCheck(ctx context.Context) error {
	return r.Client.Ping(ctx).Err()
}

// Set sets a key-value pair with optional expiration
func (r *RedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return r.Client.Set(ctx, key, value, expiration).Err()
}

// Get gets a value by key
func (r *RedisClient) Get(ctx context.Context, key string) (string, error) {
	return r.Client.Get(ctx, key).Result()
}

// Delete deletes a key
func (r *RedisClient) Delete(ctx context.Context, keys ...string) error {
	return r.Client.Del(ctx, keys...).Err()
}

// Exists checks if a key exists
func (r *RedisClient) Exists(ctx context.Context, keys ...string) (int64, error) {
	return r.Client.Exists(ctx, keys...).Result()
}
