// Package handler 提供 HTTP 请求处理器
package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"z-novel-ai-api/internal/infrastructure/persistence/milvus"
	"z-novel-ai-api/internal/infrastructure/persistence/postgres"
	"z-novel-ai-api/internal/infrastructure/persistence/redis"
)

// HealthHandler 健康检查处理器
type HealthHandler struct {
	pg     *postgres.Client
	redis  *redis.Client
	milvus *milvus.Client
}

// NewHealthHandler 创建健康检查处理器
func NewHealthHandler(pg *postgres.Client, redisClient *redis.Client, milvusClient *milvus.Client) *HealthHandler {
	return &HealthHandler{
		pg:     pg,
		redis:  redisClient,
		milvus: milvusClient,
	}
}

// HealthResponse 健康检查响应
type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version,omitempty"`
}

type readinessCheck struct {
	Status    string `json:"status"`
	Error     string `json:"error,omitempty"`
	LatencyMs int64  `json:"latency_ms,omitempty"`
}

type readinessResponse struct {
	Status string                     `json:"status"`
	Checks map[string]*readinessCheck `json:"checks,omitempty"`
}

// Health 健康检查接口
// @Summary 健康检查
// @Description 检查服务健康状态
// @Tags System
// @Produce json
// @Success 200 {object} HealthResponse
// @Router /health [get]
func (h *HealthHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, HealthResponse{
		Status: "ok",
	})
}

// Ready 就绪检查接口
// @Summary 就绪检查
// @Description 检查服务是否可以接收流量
// @Tags System
// @Produce json
// @Success 200 {object} HealthResponse
// @Router /ready [get]
func (h *HealthHandler) Ready(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	checks := map[string]*readinessCheck{
		"postgres": {Status: "unknown"},
		"redis":    {Status: "unknown"},
		"milvus":   {Status: "disabled"},
	}

	ready := true

	// Postgres（必需）
	if h == nil || h.pg == nil {
		checks["postgres"].Status = "missing"
		checks["postgres"].Error = "postgres client not configured"
		ready = false
	} else {
		start := time.Now()
		err := h.pg.HealthCheck(ctx)
		checks["postgres"].LatencyMs = time.Since(start).Milliseconds()
		if err != nil {
			checks["postgres"].Status = "error"
			checks["postgres"].Error = err.Error()
			ready = false
		} else {
			checks["postgres"].Status = "ok"
		}
	}

	// Redis（必需）
	if h == nil || h.redis == nil {
		checks["redis"].Status = "missing"
		checks["redis"].Error = "redis client not configured"
		ready = false
	} else {
		start := time.Now()
		err := h.redis.HealthCheck(ctx)
		checks["redis"].LatencyMs = time.Since(start).Milliseconds()
		if err != nil {
			checks["redis"].Status = "error"
			checks["redis"].Error = err.Error()
			ready = false
		} else {
			checks["redis"].Status = "ok"
		}
	}

	// Milvus（可选，不影响就绪态）
	if h != nil && h.milvus != nil {
		checks["milvus"] = &readinessCheck{Status: "unknown"}
		start := time.Now()
		err := h.milvus.HealthCheck(ctx)
		checks["milvus"].LatencyMs = time.Since(start).Milliseconds()
		if err != nil {
			checks["milvus"].Status = "degraded"
			checks["milvus"].Error = err.Error()
		} else {
			checks["milvus"].Status = "ok"
		}
	}

	resp := readinessResponse{
		Status: "ok",
		Checks: checks,
	}
	if !ready {
		resp.Status = "not_ready"
		c.JSON(http.StatusServiceUnavailable, resp)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// Live 存活检查接口
// @Summary 存活检查
// @Description 检查服务是否存活
// @Tags System
// @Produce json
// @Success 200 {object} HealthResponse
// @Router /live [get]
func (h *HealthHandler) Live(c *gin.Context) {
	c.JSON(http.StatusOK, HealthResponse{
		Status: "ok",
	})
}
