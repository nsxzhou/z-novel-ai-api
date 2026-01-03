// Package middleware 提供 HTTP 中间件
package middleware

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	// Enabled 是否启用限流
	Enabled bool
	// RequestsPerSecond 每秒请求数
	RequestsPerSecond int
	// Burst 突发容量
	Burst int
	// KeyPrefix Redis Key 前缀
	KeyPrefix string
}

// RateLimiter 限流器接口
type RateLimiter interface {
	Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error)
}

// RateLimit 限流中间件
func RateLimit(cfg RateLimitConfig, limiter RateLimiter) gin.HandlerFunc {
	// 如果未启用限流，返回空中间件
	if !cfg.Enabled || limiter == nil {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	// 设置默认值
	if cfg.RequestsPerSecond <= 0 {
		cfg.RequestsPerSecond = 100
	}
	if cfg.KeyPrefix == "" {
		cfg.KeyPrefix = "ratelimit"
	}

	return func(c *gin.Context) {
		// 构建限流 Key：prefix:tenant_id:path
		tenantID := c.GetString("tenant_id")
		if tenantID == "" {
			tenantID = "anonymous"
		}

		key := cfg.KeyPrefix + ":" + tenantID + ":" + c.Request.URL.Path

		// 检查限流
		allowed, err := limiter.Allow(c.Request.Context(), key, cfg.RequestsPerSecond, time.Second)
		if err != nil {
			// 限流器故障时放行，避免影响业务
			c.Next()
			return
		}

		if !allowed {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"code":     429,
				"message":  "rate limit exceeded",
				"trace_id": c.GetString("trace_id"),
			})
			return
		}

		c.Next()
	}
}

// NewRateLimitMiddleware 创建限流中间件（使用 Redis 客户端）
// 这是一个工厂函数，用于 Wire 依赖注入
func NewRateLimitMiddleware(cfg RateLimitConfig, redisClient *redis.Client) gin.HandlerFunc {
	if !cfg.Enabled || redisClient == nil {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	limiter := &redisRateLimiter{client: redisClient}
	return RateLimit(cfg, limiter)
}

// redisRateLimiter Redis 实现的限流器
type redisRateLimiter struct {
	client *redis.Client
}

// Allow 使用滑动窗口算法检查是否允许请求
func (r *redisRateLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	now := time.Now().UnixNano()
	windowStart := now - window.Nanoseconds()

	pipe := r.client.Pipeline()

	// 移除窗口外的请求
	pipe.ZRemRangeByScore(ctx, key, "0", formatInt64(windowStart))

	// 添加当前请求
	pipe.ZAdd(ctx, key, redis.Z{
		Score:  float64(now),
		Member: now,
	})

	// 获取当前窗口内的请求数
	countCmd := pipe.ZCard(ctx, key)

	// 设置过期时间
	pipe.Expire(ctx, key, window*2)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return false, err
	}

	count := countCmd.Val()
	return count <= int64(limit), nil
}

func formatInt64(n int64) string {
	return strconv.FormatInt(n, 10)
}
