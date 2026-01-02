// Package middleware 提供 HTTP 中间件
package middleware

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// CORSConfig CORS 配置
type CORSConfig struct {
	AllowedOrigins []string
	AllowedMethods []string
	AllowedHeaders []string
}

// CORS 跨域中间件
func CORS(cfg CORSConfig) gin.HandlerFunc {
	// 设置默认值
	if len(cfg.AllowedOrigins) == 0 {
		cfg.AllowedOrigins = []string{"*"}
	}
	if len(cfg.AllowedMethods) == 0 {
		cfg.AllowedMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	}
	if len(cfg.AllowedHeaders) == 0 {
		cfg.AllowedHeaders = []string{"Origin", "Content-Type", "Authorization", "X-Request-ID"}
	}

	return cors.New(cors.Config{
		AllowOrigins:     cfg.AllowedOrigins,
		AllowMethods:     cfg.AllowedMethods,
		AllowHeaders:     cfg.AllowedHeaders,
		ExposeHeaders:    []string{"X-Request-ID", "X-Trace-ID"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	})
}
