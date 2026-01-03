// Package middleware 提供 HTTP 中间件
package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"z-novel-ai-api/internal/domain/repository"
	"z-novel-ai-api/pkg/logger"
)

type rollbackOnlyError struct {
	status int
}

func (e rollbackOnlyError) Error() string {
	return fmt.Sprintf("rollback only: status=%d", e.status)
}

// DBTransaction 为每个请求创建事务，并在事务内设置租户上下文，确保 PostgreSQL RLS 生效。
//
// 设计取舍：
// - 使用 set_config(..., is_local=TRUE) 时，租户上下文仅在“当前事务”内有效；
// - 因此需要将一次请求的 DB 访问绑定到同一个事务连接上。
func DBTransaction(tx repository.Transactor, tenantCtx repository.TenantContextManager) gin.HandlerFunc {
	if tx == nil || tenantCtx == nil {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	return func(c *gin.Context) {
		// SSE/长连接请求不应持有事务连接，避免占满连接池；此类请求由 Handler 自行做短事务读写。
		if strings.HasSuffix(c.Request.URL.Path, "/stream") {
			c.Next()
			return
		}

		ctx := c.Request.Context()
		tenantID := GetTenantID(ctx)

		err := tx.WithTransaction(ctx, func(txCtx context.Context) error {
			if tenantID != "" {
				if err := tenantCtx.SetTenant(txCtx, tenantID); err != nil {
					return err
				}
			}

			c.Request = c.Request.WithContext(txCtx)
			c.Next()

			status := c.Writer.Status()
			if status >= http.StatusBadRequest {
				return rollbackOnlyError{status: status}
			}
			if len(c.Errors) > 0 {
				return rollbackOnlyError{status: status}
			}
			return nil
		})

		if err == nil {
			return
		}

		var rbErr rollbackOnlyError
		if errors.As(err, &rbErr) {
			return
		}

		logger.Error(ctx, "db transaction failed", err)
		if !c.Writer.Written() && c.Writer.Status() < http.StatusBadRequest {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"code":     http.StatusInternalServerError,
				"message":  "internal server error",
				"trace_id": c.GetString("trace_id"),
			})
		}
	}
}
