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

// DBTransaction 为每个 HTTP 请求自动管理数据库事务，并设置多租户安全上下文。
//
// 核心功能：
//  1. **请求级事务 (Request-Scoped Transaction)**: 将整个请求的处理过程包裹在一个数据库事务中。
//  2. **多租户隔离 (RLS Context)**: 在事务开启后立即设置当前租户 ID，确保 PostgreSQL 的行级安全策略 (RLS) 生效。
//     注意：PostgreSQL 的 `set_config(..., is_local=TRUE)` 仅在当前事务内有效，因此必须绑定在事务中。
//  3. **自动提交/回滚**:
//     - 成功：HTTP 状态码 < 400 且无内部错误 -> 提交事务。
//     - 失败：HTTP 状态码 >= 400 或存在 Gin 错误 -> 回滚事务。
func DBTransaction(tx repository.Transactor, tenantCtx repository.TenantContextManager) gin.HandlerFunc {
	if tx == nil || tenantCtx == nil {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	return func(c *gin.Context) {
		// -------------------------------------------------------------------------
		// 1. 长连接/流式接口豁免策略
		// -------------------------------------------------------------------------
		// SSE (Server-Sent Events)、WebSocket 或长时间运行的流式接口 (Stream)
		// 不应持有全局的数据库事务连接。
		// 原因：这些请求持续时间长，如果一直占用事务，会迅速耗尽数据库连接池。
		// 方案：此类请求应在 Handler 内部按需创建短事务 (txMgr.WithTransaction)。
		path := c.Request.URL.Path
		if strings.HasSuffix(path, "/stream") || strings.HasSuffix(path, "/foundation/preview") || strings.HasSuffix(path, "/messages") {
			c.Next()
			return
		}

		ctx := c.Request.Context()
		tenantID := GetTenantID(ctx)

		// -------------------------------------------------------------------------
		// 2. 开启请求级事务
		// -------------------------------------------------------------------------
		err := tx.WithTransaction(ctx, func(txCtx context.Context) error {
			// A. 设置租户上下文 (RLS)
			// 必须在事务开启后、执行任何查询前设置，否则无法看到租户数据。
			if tenantID != "" {
				if err := tenantCtx.SetTenant(txCtx, tenantID); err != nil {
					return err
				}
			}

			// B. 将包含事务的 Context 注入 Gin，供后续 Handler 使用
			c.Request = c.Request.WithContext(txCtx)

			// C. 执行后续业务逻辑 (Controller/Handler)
			c.Next()

			// D. 决定提交还是回滚
			// 如果业务逻辑返回了错误状态码 (>=400) 或 Gin 记录了错误，触发回滚。
			status := c.Writer.Status()
			if status >= http.StatusBadRequest {
				return rollbackOnlyError{status: status}
			}
			if len(c.Errors) > 0 {
				return rollbackOnlyError{status: status}
			}
			// 返回 nil 表示事务可以提交
			return nil
		})

		// 事务成功提交或因豁免逻辑跳过
		if err == nil {
			return
		}

		// -------------------------------------------------------------------------
		// 3. 错误处理
		// -------------------------------------------------------------------------
		// 如果是 rollbackOnlyError，说明是业务逻辑主动要求回滚（例如验证失败），
		// 此时响应已经由 Handler 写入，不需要额外处理。
		var rbErr rollbackOnlyError
		if errors.As(err, &rbErr) {
			return
		}

		// 如果是数据库层面的系统错误（如提交失败、死锁等），记录日志并返回 500。
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
