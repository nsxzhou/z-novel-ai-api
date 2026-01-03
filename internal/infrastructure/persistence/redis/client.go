// Package redis 提供 Redis 缓存和消息队列实现
package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"z-novel-ai-api/internal/config"
)

var tracer = otel.Tracer("redis")

// Client Redis 客户端
type Client struct {
	rdb    *redis.Client
	config *config.RedisConfig
}

// NewClient 创建 Redis 客户端
func NewClient(cfg *config.RedisConfig) (*Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	})

	// 验证连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping redis: %w", err)
	}

	return &Client{
		rdb:    rdb,
		config: cfg,
	}, nil
}

// Redis 获取底层 Redis 客户端
func (c *Client) Redis() *redis.Client {
	return c.rdb
}

// Close 关闭 Redis 连接
func (c *Client) Close() error {
	return c.rdb.Close()
}

// Ping 检查 Redis 连接
func (c *Client) Ping(ctx context.Context) error {
	ctx, span := tracer.Start(ctx, "redis.Ping")
	defer span.End()

	return c.rdb.Ping(ctx).Err()
}

// HealthCheck 健康检查
func (c *Client) HealthCheck(ctx context.Context) error {
	ctx, span := tracer.Start(ctx, "redis.HealthCheck")
	defer span.End()

	result, err := c.rdb.Ping(ctx).Result()
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("health check failed: %w", err)
	}
	if result != "PONG" {
		return fmt.Errorf("unexpected ping response: %s", result)
	}
	return nil
}

// Get 获取值（带追踪）
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	ctx, span := tracer.Start(ctx, "redis.Get",
		trace.WithAttributes(attribute.String("redis.key", key)))
	defer span.End()

	result, err := c.rdb.Get(ctx, key).Result()
	if err != nil && err != redis.Nil {
		span.RecordError(err)
	}
	return result, err
}

// Set 设置值（带追踪）
func (c *Client) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	ctx, span := tracer.Start(ctx, "redis.Set",
		trace.WithAttributes(
			attribute.String("redis.key", key),
			attribute.Int64("redis.ttl_ms", expiration.Milliseconds()),
		))
	defer span.End()

	err := c.rdb.Set(ctx, key, value, expiration).Err()
	if err != nil {
		span.RecordError(err)
	}
	return err
}

// Del 删除键（带追踪）
func (c *Client) Del(ctx context.Context, keys ...string) error {
	ctx, span := tracer.Start(ctx, "redis.Del",
		trace.WithAttributes(attribute.Int("redis.key_count", len(keys))))
	defer span.End()

	err := c.rdb.Del(ctx, keys...).Err()
	if err != nil {
		span.RecordError(err)
	}
	return err
}

// Exists 检查键是否存在
func (c *Client) Exists(ctx context.Context, keys ...string) (int64, error) {
	ctx, span := tracer.Start(ctx, "redis.Exists")
	defer span.End()

	result, err := c.rdb.Exists(ctx, keys...).Result()
	if err != nil {
		span.RecordError(err)
	}
	return result, err
}

// Expire 设置过期时间
func (c *Client) Expire(ctx context.Context, key string, expiration time.Duration) error {
	ctx, span := tracer.Start(ctx, "redis.Expire",
		trace.WithAttributes(attribute.String("redis.key", key)))
	defer span.End()

	err := c.rdb.Expire(ctx, key, expiration).Err()
	if err != nil {
		span.RecordError(err)
	}
	return err
}

// TTL 获取剩余过期时间
func (c *Client) TTL(ctx context.Context, key string) (time.Duration, error) {
	ctx, span := tracer.Start(ctx, "redis.TTL",
		trace.WithAttributes(attribute.String("redis.key", key)))
	defer span.End()

	result, err := c.rdb.TTL(ctx, key).Result()
	if err != nil {
		span.RecordError(err)
	}
	return result, err
}

// IsNil 检查是否为 redis.Nil 错误
func IsNil(err error) bool {
	return err == redis.Nil
}
