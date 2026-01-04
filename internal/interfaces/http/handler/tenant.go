// Package handler 提供 HTTP 请求处理器
package handler

import (
	"z-novel-ai-api/internal/domain/entity"
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

// ListTenants 获取租户列表
func (h *TenantHandler) ListTenants(c *gin.Context) {
	ctx := c.Request.Context()
	pageReq := dto.BindPage(c)

	result, err := h.tenantRepo.List(ctx, repository.NewPagination(pageReq.Page, pageReq.PageSize))
	if err != nil {
		logger.Error(ctx, "failed to list tenants", err)
		dto.InternalError(c, "failed to list tenants")
		return
	}

	items := make([]*dto.TenantResponse, len(result.Items))
	for i, t := range result.Items {
		items[i] = dto.ToTenantResponse(t)
	}

	meta := dto.NewPageMeta(pageReq.Page, pageReq.PageSize, int(result.Total))
	dto.SuccessWithPage(c, &dto.TenantListResponse{Items: items}, meta)
}

// CreateTenant 创建新租户
func (h *TenantHandler) CreateTenant(c *gin.Context) {
	ctx := c.Request.Context()

	var req dto.CreateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequest(c, "invalid request body: "+err.Error())
		return
	}

	// 检查 Slug 唯一性
	exists, err := h.tenantRepo.ExistsBySlug(ctx, req.Slug)
	if err != nil {
		logger.Error(ctx, "failed to check slug existence", err)
		dto.InternalError(c, "failed to create tenant")
		return
	}
	if exists {
		dto.BadRequest(c, "tenant slug already exists")
		return
	}

	tenant := entity.NewTenant(req.Name, req.Slug)
	if err := h.tenantRepo.Create(ctx, tenant); err != nil {
		logger.Error(ctx, "failed to create tenant", err)
		dto.InternalError(c, "failed to create tenant")
		return
	}

	dto.Created(c, dto.ToTenantResponse(tenant))
}
