// Package handler 提供 HTTP 请求处理器
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// HealthHandler 健康检查处理器
type HealthHandler struct{}

// NewHealthHandler 创建健康检查处理器
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// HealthResponse 健康检查响应
type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version,omitempty"`
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
	// TODO: 检查数据库、Redis 等依赖的连接状态
	c.JSON(http.StatusOK, HealthResponse{
		Status: "ok",
	})
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
