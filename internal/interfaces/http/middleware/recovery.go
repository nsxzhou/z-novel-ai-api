// Package middleware 提供 HTTP 中间件
package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"z-novel-ai-api/pkg/errors"
	"z-novel-ai-api/pkg/logger"

	"github.com/gin-gonic/gin"
)

// Recovery Panic 恢复中间件
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// 获取堆栈信息
				stack := string(debug.Stack())

				// 记录错误日志
				logger.Error(c.Request.Context(), "panic recovered",
					fmt.Errorf("%v", err),
					"stack", stack,
					"path", c.Request.URL.Path,
					"method", c.Request.Method,
				)

				// 返回 500 错误
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"code":    errors.CodeInternalError,
					"message": "internal server error",
				})
			}
		}()

		c.Next()
	}
}
