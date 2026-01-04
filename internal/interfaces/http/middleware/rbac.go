// Package middleware 提供 HTTP 中间件
package middleware

import (
	"net/http"

	"z-novel-ai-api/internal/domain/entity"

	"github.com/gin-gonic/gin"
)

// Permission 权限类型
type Permission string

// 权限常量定义
const (
	PermProjectRead     Permission = "project:read"
	PermProjectWrite    Permission = "project:write"
	PermChapterGenerate Permission = "chapter:generate"
	PermAdminAccess     Permission = "admin:access"
)

// rolePermissions 角色-权限映射表
var rolePermissions = map[entity.UserRole][]Permission{
	entity.UserRoleAdmin:  {PermProjectRead, PermProjectWrite, PermChapterGenerate, PermAdminAccess},
	entity.UserRoleMember: {PermProjectRead, PermProjectWrite, PermChapterGenerate},
	entity.UserRoleViewer: {PermProjectRead},
}

// HasPermission 检查角色是否具有指定权限
func HasPermission(role entity.UserRole, perm Permission) bool {
	permissions, ok := rolePermissions[role]
	if !ok {
		return false
	}
	for _, p := range permissions {
		if p == perm {
			return true
		}
	}
	return false
}

// RequirePermission 权限检查中间件
// 检查当前用户是否具有指定权限，否则返回 403
func RequirePermission(perm Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleStr := c.GetString("role")
		if roleStr == "" {
			abortForbidden(c, "missing role in context")
			return
		}

		role := entity.UserRole(roleStr)
		if !HasPermission(role, perm) {
			abortForbidden(c, "permission denied")
			return
		}

		c.Next()
	}
}

// RequireRole 角色检查中间件
// 检查当前用户是否为指定角色之一，否则返回 403
func RequireRole(roles ...entity.UserRole) gin.HandlerFunc {
	roleSet := make(map[entity.UserRole]bool)
	for _, r := range roles {
		roleSet[r] = true
	}

	return func(c *gin.Context) {
		roleStr := c.GetString("role")
		if roleStr == "" {
			abortForbidden(c, "missing role in context")
			return
		}

		role := entity.UserRole(roleStr)
		if !roleSet[role] {
			abortForbidden(c, "role not allowed")
			return
		}

		c.Next()
	}
}

// RequireAdmin 管理员权限检查中间件（便捷方法）
func RequireAdmin() gin.HandlerFunc {
	return RequirePermission(PermAdminAccess)
}

// abortForbidden 终止请求并返回 403
func abortForbidden(c *gin.Context, msg string) {
	c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
		"code":     403,
		"message":  msg,
		"trace_id": c.GetString("trace_id"),
	})
}
