// Package middleware 提供 HTTP 中间件
package middleware

import (
	"strconv"
	"time"

	"z-novel-ai-api/pkg/metrics"

	"github.com/gin-gonic/gin"
)

// Metrics Prometheus 指标采集中间件
func Metrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.FullPath()
		if path == "" {
			path = "unknown"
		}
		method := c.Request.Method

		// 记录请求大小
		reqSize := float64(c.Request.ContentLength)
		if reqSize > 0 {
			metrics.HTTPRequestSize.WithLabelValues(method, path).Observe(reqSize)
		}

		c.Next()

		// 请求完成后记录指标
		status := strconv.Itoa(c.Writer.Status())
		duration := time.Since(start).Seconds()
		respSize := float64(c.Writer.Size())

		metrics.HTTPRequestsTotal.WithLabelValues(method, path, status).Inc()
		metrics.HTTPRequestDuration.WithLabelValues(method, path).Observe(duration)
		if respSize > 0 {
			metrics.HTTPResponseSize.WithLabelValues(method, path).Observe(respSize)
		}
	}
}
