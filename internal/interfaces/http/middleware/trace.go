// Package middleware 提供 HTTP 中间件
package middleware

import (
	"z-novel-ai-api/pkg/logger"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel/trace"
)

// Trace OpenTelemetry 追踪中间件
func Trace(serviceName string) gin.HandlerFunc {
	return otelgin.Middleware(serviceName)
}

// TraceContext 自定义扩展：注入 trace_id 到 Context
func TraceContext() gin.HandlerFunc {
	return func(c *gin.Context) {
		span := trace.SpanFromContext(c.Request.Context())
		if span.SpanContext().IsValid() {
			traceID := span.SpanContext().TraceID().String()
			spanID := span.SpanContext().SpanID().String()

			// 设置到 Gin Context
			c.Set("trace_id", traceID)
			c.Set("span_id", spanID)

			// 设置到 Logger Context
			ctx := logger.WithContext(c.Request.Context(), logger.TraceIDKey, traceID)
			ctx = logger.WithContext(ctx, logger.SpanIDKey, spanID)
			c.Request = c.Request.WithContext(ctx)

			// 设置响应头
			c.Header("X-Trace-ID", traceID)
		}

		c.Next()
	}
}
