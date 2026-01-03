// Package middleware 提供 HTTP 中间件
package middleware

import (
	"context"

	"github.com/gin-gonic/gin"
)

// TenantContextKey 租户上下文 Key 类型
type TenantContextKey string

const (
	// TenantIDKey 租户 ID 上下文 Key
	TenantIDKey TenantContextKey = "tenant_id"
	// UserIDKey 用户 ID 上下文 Key
	UserIDKey TenantContextKey = "user_id"
)

// TenantConfig 租户中间件配置
type TenantConfig struct {
	// Enabled 是否启用租户隔离
	Enabled bool
	// HeaderName 从 Header 中获取租户 ID 的字段名
	HeaderName string
	// DefaultTenantID 默认租户 ID（用于开发环境）
	DefaultTenantID string
}

// Tenant 多租户上下文中间件
// 确保请求上下文中包含租户信息，用于 PostgreSQL RLS
func Tenant(cfg TenantConfig) gin.HandlerFunc {
	// 设置默认值
	if cfg.HeaderName == "" {
		cfg.HeaderName = "X-Tenant-ID"
	}

	return func(c *gin.Context) {
		// 优先从 Auth 中间件获取（JWT 解析后设置）
		tenantID := c.GetString("tenant_id")

		// 如果没有，尝试从 Header 获取
		if tenantID == "" {
			tenantID = c.GetHeader(cfg.HeaderName)
		}

		// 如果还没有，使用默认值（仅开发环境）
		if tenantID == "" && cfg.DefaultTenantID != "" {
			tenantID = cfg.DefaultTenantID
		}

		// 设置到 Gin Context
		if tenantID != "" {
			c.Set("tenant_id", tenantID)

			// 同时设置到 request context，便于 Repository 层使用
			ctx := context.WithValue(c.Request.Context(), TenantIDKey, tenantID)

			// 如果有 user_id，也设置到 context
			if userID := c.GetString("user_id"); userID != "" {
				ctx = context.WithValue(ctx, UserIDKey, userID)
			}

			c.Request = c.Request.WithContext(ctx)
		}

		c.Next()
	}
}

// GetTenantID 从 context 中获取租户 ID
func GetTenantID(ctx context.Context) string {
	if v := ctx.Value(TenantIDKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// GetUserID 从 context 中获取用户 ID
func GetUserID(ctx context.Context) string {
	if v := ctx.Value(UserIDKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// GetTenantIDFromGin 从 Gin Context 中获取租户 ID
func GetTenantIDFromGin(c *gin.Context) string {
	return c.GetString("tenant_id")
}

// GetUserIDFromGin 从 Gin Context 中获取用户 ID
func GetUserIDFromGin(c *gin.Context) string {
	return c.GetString("user_id")
}
