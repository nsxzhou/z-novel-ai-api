// Package handler 提供 HTTP 请求处理器
package handler

import (
	"z-novel-ai-api/internal/domain/repository"
	"z-novel-ai-api/internal/interfaces/http/dto"
	"z-novel-ai-api/internal/interfaces/http/middleware"
	"z-novel-ai-api/pkg/logger"

	"github.com/gin-gonic/gin"
)

// TenantHandler 租户处理器
type TenantHandler struct {
	tenantRepo repository.TenantRepository
}

// NewTenantHandler 创建租户处理器
func NewTenantHandler(tenantRepo repository.TenantRepository) *TenantHandler {
	return &TenantHandler{
		tenantRepo: tenantRepo,
	}
}

// GetCurrentTenant 获取当前租户信息
// @Summary 获取当前租户资料
// @Description 获取请求上下文对应租户的详细信息
// @Tags Tenants
// @Accept json
// @Produce json
// @Success 200 {object} dto.Response[dto.TenantResponse]
// @Failure 401 {object} dto.ErrorResponse
// @Router /v1/tenants/current [get]
func (h *TenantHandler) GetCurrentTenant(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID := middleware.GetTenantIDFromGin(c)

	tenant, err := h.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		logger.Error(ctx, "failed to get tenant", err)
		dto.InternalError(c, "failed to get tenant info")
		return
	}

	if tenant == nil {
		dto.NotFound(c, "tenant not found")
		return
	}

	resp := dto.ToTenantResponse(tenant)
	dto.Success(c, resp)
}

// UpdateCurrentTenant 更新当前租户信息
// @Summary 更新当前租户配置
// @Description 修改当前租户的名称、设置等
// @Tags Tenants
// @Accept json
// @Produce json
// @Param body body dto.UpdateTenantRequest true "更新内容"
// @Success 200 {object} dto.Response[dto.TenantResponse]
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Router /v1/tenants/current [put]
func (h *TenantHandler) UpdateCurrentTenant(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID := middleware.GetTenantIDFromGin(c)

	var req dto.UpdateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequest(c, "invalid request body: "+err.Error())
		return
	}

	tenant, err := h.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		logger.Error(ctx, "failed to get tenant", err)
		dto.InternalError(c, "failed to get tenant info")
		return
	}

	if tenant == nil {
		dto.NotFound(c, "tenant not found")
		return
	}

	req.ApplyToTenant(tenant)

	if err := h.tenantRepo.Update(ctx, tenant); err != nil {
		logger.Error(ctx, "failed to update tenant", err)
		dto.InternalError(c, "failed to update tenant info")
		return
	}

	resp := dto.ToTenantResponse(tenant)
	dto.Success(c, resp)
}
