// Package redis 提供 Redis 限流器实现
package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/attribute"
)

// RateLimiter 滑动窗口限流器
type RateLimiter struct {
	client *Client
}

// NewRateLimiter 创建限流器
func NewRateLimiter(client *Client) *RateLimiter {
	return &RateLimiter{client: client}
}

// Allow 检查是否允许请求（滑动窗口算法）
func (l *RateLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	ctx, span := tracer.Start(ctx, "ratelimit.Allow")
	span.SetAttributes(
		attribute.String("ratelimit.key", key),
		attribute.Int("ratelimit.limit", limit),
		attribute.Int64("ratelimit.window_ms", window.Milliseconds()),
	)
	defer span.End()

	now := time.Now().UnixMilli()
	windowStart := now - window.Milliseconds()

	pipe := l.client.rdb.Pipeline()

	// 移除窗口外的请求
	pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart))

	// 获取当前窗口内的请求数
	countCmd := pipe.ZCard(ctx, key)

	_, err := pipe.Exec(ctx)
	if err != nil {
		span.RecordError(err)
		return false, err
	}

	count := countCmd.Val()
	span.SetAttributes(attribute.Int64("ratelimit.current_count", count))

	if count >= int64(limit) {
		span.SetAttributes(attribute.Bool("ratelimit.allowed", false))
		return false, nil
	}

	// 添加当前请求
	l.client.rdb.ZAdd(ctx, key, redis.Z{
		Score:  float64(now),
		Member: fmt.Sprintf("%d", now),
	})
	l.client.rdb.Expire(ctx, key, window*2)

	span.SetAttributes(attribute.Bool("ratelimit.allowed", true))
	return true, nil
}

// AllowN 检查是否允许 N 个请求
func (l *RateLimiter) AllowN(ctx context.Context, key string, limit, n int, window time.Duration) (bool, error) {
	ctx, span := tracer.Start(ctx, "ratelimit.AllowN")
	span.SetAttributes(
		attribute.String("ratelimit.key", key),
		attribute.Int("ratelimit.limit", limit),
		attribute.Int("ratelimit.n", n),
	)
	defer span.End()

	now := time.Now().UnixMilli()
	windowStart := now - window.Milliseconds()

	pipe := l.client.rdb.Pipeline()
	pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart))
	countCmd := pipe.ZCard(ctx, key)

	_, err := pipe.Exec(ctx)
	if err != nil {
		span.RecordError(err)
		return false, err
	}

	count := countCmd.Val()

	if count+int64(n) > int64(limit) {
		span.SetAttributes(attribute.Bool("ratelimit.allowed", false))
		return false, nil
	}

	// 添加 N 个请求
	pipe = l.client.rdb.Pipeline()
	for i := 0; i < n; i++ {
		pipe.ZAdd(ctx, key, redis.Z{
			Score:  float64(now),
			Member: fmt.Sprintf("%d-%d", now, i),
		})
	}
	pipe.Expire(ctx, key, window*2)

	_, err = pipe.Exec(ctx)
	if err != nil {
		span.RecordError(err)
		return false, err
	}

	span.SetAttributes(attribute.Bool("ratelimit.allowed", true))
	return true, nil
}

// Remaining 获取剩余配额
func (l *RateLimiter) Remaining(ctx context.Context, key string, limit int, window time.Duration) (int, error) {
	ctx, span := tracer.Start(ctx, "ratelimit.Remaining")
	span.SetAttributes(attribute.String("ratelimit.key", key))
	defer span.End()

	now := time.Now().UnixMilli()
	windowStart := now - window.Milliseconds()

	pipe := l.client.rdb.Pipeline()
	pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart))
	countCmd := pipe.ZCard(ctx, key)

	_, err := pipe.Exec(ctx)
	if err != nil {
		span.RecordError(err)
		return 0, err
	}

	count := countCmd.Val()
	remaining := limit - int(count)
	if remaining < 0 {
		remaining = 0
	}

	span.SetAttributes(attribute.Int("ratelimit.remaining", remaining))
	return remaining, nil
}

// Reset 重置限流计数
func (l *RateLimiter) Reset(ctx context.Context, key string) error {
	ctx, span := tracer.Start(ctx, "ratelimit.Reset")
	span.SetAttributes(attribute.String("ratelimit.key", key))
	defer span.End()

	return l.client.rdb.Del(ctx, key).Err()
}

// BuildRateLimitKey 构建限流键
func BuildRateLimitKey(tenantID, endpoint string) string {
	return fmt.Sprintf("ratelimit:%s:%s", tenantID, endpoint)
}

// BuildUserRateLimitKey 构建用户限流键
func BuildUserRateLimitKey(tenantID, userID, endpoint string) string {
	return fmt.Sprintf("ratelimit:%s:%s:%s", tenantID, userID, endpoint)
}
