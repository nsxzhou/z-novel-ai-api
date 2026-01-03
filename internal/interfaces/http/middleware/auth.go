// Package middleware 提供 HTTP 中间件
package middleware

import (
	"net/http"
	"strings"

	"z-novel-ai-api/pkg/utils"

	"github.com/gin-gonic/gin"
)

// AuthConfig 认证配置
type AuthConfig struct {
	// Secret JWT 密钥
	Secret string
	// Issuer JWT 签发者
	Issuer string
	// SkipPaths 跳过认证的路径
	SkipPaths []string
	// Enabled 是否启用认证
	Enabled bool
}

// Auth 认证中间件
func Auth(cfg AuthConfig) gin.HandlerFunc {
	// 初始化 JWT 管理器
	jwtManager := utils.NewJWTManager(cfg.Secret, cfg.Issuer)

	// 构建跳过路径映射
	skipMap := make(map[string]bool)
	for _, path := range cfg.SkipPaths {
		skipMap[path] = true
	}

	return func(c *gin.Context) {
		// 如果未启用认证，直接放行
		if !cfg.Enabled {
			c.Next()
			return
		}

		// 检查是否跳过路径
		if skipMap[c.Request.URL.Path] {
			c.Next()
			return
		}

		// 检查路径前缀匹配（支持 /health, /ready, /live, /metrics）
		for path := range skipMap {
			if strings.HasPrefix(c.Request.URL.Path, path) {
				c.Next()
				return
			}
		}

		// 获取 Authorization Header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			abortUnauthorized(c, "missing authorization header")
			return
		}

		// 解析 Bearer Token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			abortUnauthorized(c, "invalid authorization format")
			return
		}

		token := parts[1]

		// 使用 JWT 验证 Token
		claims, err := jwtManager.ParseToken(token)
		if err != nil {
			msg := "invalid token"
			if err == utils.ErrExpiredToken {
				msg = "token expired"
			}
			abortUnauthorized(c, msg)
			return
		}

		// 确保是 AccessToken
		if claims.Type != "access" {
			abortUnauthorized(c, "invalid token type")
			return
		}

		// 注入用户信息到 Context
		c.Set("tenant_id", claims.TenantID)
		c.Set("user_id", claims.UserID)
		c.Set("role", claims.Role)

		c.Next()
	}
}

// abortUnauthorized 终止请求并返回 401
func abortUnauthorized(c *gin.Context, msg string) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
		"code":     401,
		"message":  msg,
		"trace_id": c.GetString("trace_id"),
	})
}

// 错误定义
var (
	ErrTokenMissing = &AuthError{Message: "token missing"}
	ErrTokenInvalid = &AuthError{Message: "token invalid"}
	ErrTokenExpired = &AuthError{Message: "token expired"}
)

// AuthError 认证错误
type AuthError struct {
	Message string
}

func (e *AuthError) Error() string {
	return e.Message
}

// DefaultSkipPaths 默认跳过认证的路径
var DefaultSkipPaths = []string{
	"/health",
	"/ready",
	"/live",
	"/metrics",
}
