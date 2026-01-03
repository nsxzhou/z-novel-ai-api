// Package redis 提供 Redis 缓存实现
package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/singleflight"
)

var cacheTracer = otel.Tracer("redis.cache")

// Cache 缓存服务
type Cache struct {
	client *Client
	group  singleflight.Group
}

// NewCache 创建缓存服务
func NewCache(client *Client) *Cache {
	return &Cache{
		client: client,
	}
}

// Get 获取缓存值
func (c *Cache) Get(ctx context.Context, key string) ([]byte, error) {
	ctx, span := cacheTracer.Start(ctx, "cache.Get",
		trace.WithAttributes(attribute.String("cache.key", key)))
	defer span.End()

	val, err := c.client.rdb.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			span.SetAttributes(attribute.Bool("cache.hit", false))
			return nil, err
		}
		span.RecordError(err)
		return nil, err
	}

	span.SetAttributes(attribute.Bool("cache.hit", true))
	return val, nil
}

// Set 设置缓存值
func (c *Cache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	ctx, span := cacheTracer.Start(ctx, "cache.Set",
		trace.WithAttributes(
			attribute.String("cache.key", key),
			attribute.Int64("cache.ttl_ms", ttl.Milliseconds()),
		))
	defer span.End()

	bytes, err := json.Marshal(value)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	return c.client.rdb.Set(ctx, key, bytes, ttl).Err()
}

// GetOrLoad Read-Through 缓存模式
func (c *Cache) GetOrLoad(ctx context.Context, key string, ttl time.Duration, loader func() (interface{}, error)) ([]byte, error) {
	ctx, span := cacheTracer.Start(ctx, "cache.GetOrLoad",
		trace.WithAttributes(attribute.String("cache.key", key)))
	defer span.End()

	// 尝试从缓存获取
	val, err := c.client.rdb.Get(ctx, key).Bytes()
	if err == nil {
		span.SetAttributes(attribute.Bool("cache.hit", true))
		return val, nil
	}

	if err != redis.Nil {
		span.RecordError(err)
		return nil, err
	}

	span.SetAttributes(attribute.Bool("cache.hit", false))

	// 缓存未命中，加载数据
	data, err := loader()
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	// 序列化并缓存
	bytes, err := json.Marshal(data)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to marshal data: %w", err)
	}

	if err := c.client.rdb.Set(ctx, key, bytes, ttl).Err(); err != nil {
		// 缓存写入失败不影响返回结果
		span.RecordError(err)
	}

	return bytes, nil
}

// GetOrLoadSafe 使用 singleflight 防止缓存击穿
func (c *Cache) GetOrLoadSafe(ctx context.Context, key string, ttl time.Duration, loader func() (interface{}, error)) ([]byte, error) {
	ctx, span := cacheTracer.Start(ctx, "cache.GetOrLoadSafe",
		trace.WithAttributes(attribute.String("cache.key", key)))
	defer span.End()

	// 尝试从缓存获取
	val, err := c.client.rdb.Get(ctx, key).Bytes()
	if err == nil {
		span.SetAttributes(attribute.Bool("cache.hit", true))
		return val, nil
	}

	if err != redis.Nil {
		span.RecordError(err)
		return nil, err
	}

	span.SetAttributes(attribute.Bool("cache.hit", false))

	// 使用 singleflight 合并并发请求
	result, err, shared := c.group.Do(key, func() (interface{}, error) {
		// 再次检查缓存（可能已被其他请求填充）
		val, err := c.client.rdb.Get(ctx, key).Bytes()
		if err == nil {
			return val, nil
		}

		data, err := loader()
		if err != nil {
			return nil, err
		}

		bytes, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal data: %w", err)
		}

		if err := c.client.rdb.Set(ctx, key, bytes, ttl).Err(); err != nil {
			// 缓存写入失败不影响返回结果
		}

		return bytes, nil
	})

	span.SetAttributes(attribute.Bool("cache.shared", shared))

	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	return result.([]byte), nil
}

// SetWithDB Write-Through 缓存模式
func (c *Cache) SetWithDB(ctx context.Context, key string, value interface{}, ttl time.Duration, dbWriter func() error) error {
	ctx, span := cacheTracer.Start(ctx, "cache.SetWithDB",
		trace.WithAttributes(attribute.String("cache.key", key)))
	defer span.End()

	// 先写数据库
	if err := dbWriter(); err != nil {
		span.RecordError(err)
		return err
	}

	// 再更新缓存
	bytes, err := json.Marshal(value)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	if err := c.client.rdb.Set(ctx, key, bytes, ttl).Err(); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to set cache: %w", err)
	}

	return nil
}

// Delete 删除缓存
func (c *Cache) Delete(ctx context.Context, keys ...string) error {
	ctx, span := cacheTracer.Start(ctx, "cache.Delete",
		trace.WithAttributes(attribute.Int("cache.key_count", len(keys))))
	defer span.End()

	return c.client.rdb.Del(ctx, keys...).Err()
}

// InvalidatePattern 按模式使缓存失效
func (c *Cache) InvalidatePattern(ctx context.Context, pattern string) error {
	ctx, span := cacheTracer.Start(ctx, "cache.InvalidatePattern",
		trace.WithAttributes(attribute.String("cache.pattern", pattern)))
	defer span.End()

	iter := c.client.rdb.Scan(ctx, 0, pattern, 0).Iterator()
	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}

	if err := iter.Err(); err != nil {
		span.RecordError(err)
		return err
	}

	if len(keys) > 0 {
		span.SetAttributes(attribute.Int("cache.invalidated_count", len(keys)))
		return c.client.rdb.Del(ctx, keys...).Err()
	}

	return nil
}

// InvalidateEntity 使实体相关缓存失效
func (c *Cache) InvalidateEntity(ctx context.Context, tenantID, projectID, entityID string) error {
	pattern := fmt.Sprintf("entity:%s:%s:%s*", tenantID, projectID, entityID)
	return c.InvalidatePattern(ctx, pattern)
}

// InvalidateProject 使项目相关缓存失效
func (c *Cache) InvalidateProject(ctx context.Context, tenantID, projectID string) error {
	patterns := []string{
		fmt.Sprintf("entity:%s:%s:*", tenantID, projectID),
		fmt.Sprintf("summary:%s:%s:*", tenantID, projectID),
		fmt.Sprintf("ctx:%s:%s:*", tenantID, projectID),
	}

	for _, pattern := range patterns {
		if err := c.InvalidatePattern(ctx, pattern); err != nil {
			return err
		}
	}

	return nil
}
