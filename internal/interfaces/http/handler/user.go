// Package handler 提供 HTTP 请求处理器
package handler

import (
	"z-novel-ai-api/internal/domain/repository"
	"z-novel-ai-api/internal/interfaces/http/dto"
	"z-novel-ai-api/internal/interfaces/http/middleware"
	"z-novel-ai-api/pkg/logger"

	"github.com/gin-gonic/gin"
)

// UserHandler 用户处理器
type UserHandler struct {
	userRepo repository.UserRepository
}

// NewUserHandler 创建用户处理器
func NewUserHandler(userRepo repository.UserRepository) *UserHandler {
	return &UserHandler{
		userRepo: userRepo,
	}
}

// GetMe 获取当前用户信息
// @Summary 获取当前用户信息
// @Description 获取登录用户的详细资料
// @Tags Users
// @Accept json
// @Produce json
// @Success 200 {object} dto.Response[dto.UserResponse]
// @Failure 401 {object} dto.ErrorResponse
// @Router /v1/users/me [get]
func (h *UserHandler) GetMe(c *gin.Context) {
	ctx := c.Request.Context()
	userID := middleware.GetUserIDFromGin(c)

	user, err := h.userRepo.GetByID(ctx, userID)
	if err != nil {
		logger.Error(ctx, "failed to get user", err)
		dto.InternalError(c, "failed to get user info")
		return
	}

	if user == nil {
		dto.NotFound(c, "user not found")
		return
	}

	resp := dto.ToUserResponse(user)
	dto.Success(c, resp)
}

// UpdateMe 更新当前用户信息
// @Summary 更新当前用户信息
// @Description 修改当前登录用户的昵称、头像或个人设置
// @Tags Users
// @Accept json
// @Produce json
// @Param body body dto.UpdateUserRequest true "更新内容"
// @Success 200 {object} dto.Response[dto.UserResponse]
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Router /v1/users/me [put]
func (h *UserHandler) UpdateMe(c *gin.Context) {
	ctx := c.Request.Context()
	userID := middleware.GetUserIDFromGin(c)

	var req dto.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequest(c, "invalid request body: "+err.Error())
		return
	}

	user, err := h.userRepo.GetByID(ctx, userID)
	if err != nil {
		logger.Error(ctx, "failed to get user", err)
		dto.InternalError(c, "failed to get user info")
		return
	}

	if user == nil {
		dto.NotFound(c, "user not found")
		return
	}

	req.ApplyToUser(user)

	if err := h.userRepo.Update(ctx, user); err != nil {
		logger.Error(ctx, "failed to update user", err)
		dto.InternalError(c, "failed to update user info")
		return
	}

	resp := dto.ToUserResponse(user)
	dto.Success(c, resp)
}

// ListTenantUsers 获取租户用户列表
// @Summary 获取租户用户列表
// @Description 获取当前租户下的所有用户（分页）
// @Tags Users
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页条数" default(20)
// @Success 200 {object} dto.Response[dto.UserListResponse]
// @Failure 403 {object} dto.ErrorResponse
// @Router /v1/users [get]
func (h *UserHandler) ListTenantUsers(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID := middleware.GetTenantIDFromGin(c)
	pageReq := dto.BindPage(c)

	result, err := h.userRepo.ListByTenant(ctx, tenantID, repository.NewPagination(pageReq.Page, pageReq.PageSize))
	if err != nil {
		logger.Error(ctx, "failed to list users", err)
		dto.InternalError(c, "failed to list users")
		return
	}

	resp := dto.ToUserListResponse(result.Items)
	meta := dto.NewPageMeta(pageReq.Page, pageReq.PageSize, int(result.Total))
	dto.SuccessWithPage(c, resp, meta)
}

// UpdateUserRole 更新用户角色
func (h *UserHandler) UpdateUserRole(c *gin.Context) {
	ctx := c.Request.Context()
	targetUserID := c.Param("id")

	var req dto.UpdateUserRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequest(c, "invalid request body: "+err.Error())
		return
	}

	if err := h.userRepo.UpdateRole(ctx, targetUserID, req.Role); err != nil {
		logger.Error(ctx, "failed to update user role", err)
		dto.InternalError(c, "failed to update user role")
		return
	}

	dto.Success(c, gin.H{"message": "user role updated"})
}

// DeleteUser 删除用户
func (h *UserHandler) DeleteUser(c *gin.Context) {
	ctx := c.Request.Context()
	targetUserID := c.Param("id")

	if err := h.userRepo.Delete(ctx, targetUserID); err != nil {
		logger.Error(ctx, "failed to delete user", err)
		dto.InternalError(c, "failed to delete user")
		return
	}

	dto.Success(c, gin.H{"message": "user deleted"})
}
