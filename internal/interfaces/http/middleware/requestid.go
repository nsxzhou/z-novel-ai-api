// Package middleware 提供 HTTP 中间件
package middleware

import (
	"z-novel-ai-api/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	// RequestIDHeader 请求 ID 头
	RequestIDHeader = "X-Request-ID"
)

// RequestID 请求 ID 注入中间件
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 尝试从请求头获取
		requestID := c.GetHeader(RequestIDHeader)

		// 如果没有则生成新的
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// 设置到 Gin Context
		c.Set("request_id", requestID)

		// 设置到 Logger Context
		ctx := logger.WithContext(c.Request.Context(), logger.RequestIDKey, requestID)
		c.Request = c.Request.WithContext(ctx)

		// 设置响应头
		c.Header(RequestIDHeader, requestID)

		c.Next()
	}
}
