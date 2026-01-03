// Package middleware 提供 HTTP 中间件
package middleware

import (
	"time"

	"z-novel-ai-api/pkg/logger"

	"github.com/gin-gonic/gin"
)

// Audit 审计日志中间件
// 记录请求的详细信息，用于审计和监控
func Audit() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 记录请求开始时间
		start := time.Now()

		// 处理请求
		c.Next()

		// 计算请求耗时
		duration := time.Since(start)

		// 记录审计日志
		logger.Info(c.Request.Context(), "api request",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"query", c.Request.URL.RawQuery,
			"status", c.Writer.Status(),
			"duration_ms", duration.Milliseconds(),
			"ip", c.ClientIP(),
			"user_agent", c.Request.UserAgent(),
			"tenant_id", c.GetString("tenant_id"),
			"user_id", c.GetString("user_id"),
			"request_id", c.GetString("request_id"),
			"trace_id", c.GetString("trace_id"),
			"body_size", c.Writer.Size(),
		)
	}
}

// AuditConfig 审计配置
type AuditConfig struct {
	// Enabled 是否启用审计
	Enabled bool
	// SkipPaths 跳过审计的路径
	SkipPaths []string
	// LogRequestBody 是否记录请求体
	LogRequestBody bool
	// LogResponseBody 是否记录响应体
	LogResponseBody bool
	// MaxBodyLogSize 最大记录的请求体大小
	MaxBodyLogSize int
}

// AuditWithConfig 带配置的审计中间件
func AuditWithConfig(cfg AuditConfig) gin.HandlerFunc {
	if !cfg.Enabled {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	// 构建跳过路径映射
	skipMap := make(map[string]bool)
	for _, path := range cfg.SkipPaths {
		skipMap[path] = true
	}

	return func(c *gin.Context) {
		// 检查是否跳过
		if skipMap[c.Request.URL.Path] {
			c.Next()
			return
		}

		// 记录请求开始时间
		start := time.Now()

		// 处理请求
		c.Next()

		// 计算请求耗时
		duration := time.Since(start)

		// 构建日志字段
		fields := []interface{}{
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"duration_ms", duration.Milliseconds(),
			"ip", c.ClientIP(),
			"tenant_id", c.GetString("tenant_id"),
			"user_id", c.GetString("user_id"),
			"request_id", c.GetString("request_id"),
		}

		// 记录审计日志
		logger.Info(c.Request.Context(), "api audit", fields...)
	}
}

// DefaultAuditSkipPaths 默认跳过审计的路径
var DefaultAuditSkipPaths = []string{
	"/health",
	"/ready",
	"/live",
	"/metrics",
}
