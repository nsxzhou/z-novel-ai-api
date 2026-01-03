// Package handler 提供 HTTP 请求处理器
package handler

import (
	"net/http"

	"z-novel-ai-api/internal/domain/repository"
	"z-novel-ai-api/internal/interfaces/http/dto"
	"z-novel-ai-api/internal/interfaces/http/middleware"
	"z-novel-ai-api/pkg/errors"
	"z-novel-ai-api/pkg/logger"

	"github.com/gin-gonic/gin"
)

// ProjectHandler 项目处理器
type ProjectHandler struct {
	projectRepo repository.ProjectRepository
}

// NewProjectHandler 创建项目处理器
func NewProjectHandler(projectRepo repository.ProjectRepository) *ProjectHandler {
	return &ProjectHandler{
		projectRepo: projectRepo,
	}
}

// ListProjects 获取项目列表
// @Summary 获取项目列表
// @Description 获取当前租户的项目列表
// @Tags Projects
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页条数" default(20)
// @Success 200 {object} dto.Response[dto.ProjectListResponse]
// @Failure 500 {object} dto.ErrorResponse
// @Router /v1/projects [get]
func (h *ProjectHandler) ListProjects(c *gin.Context) {
	ctx := c.Request.Context()
	_ = middleware.GetTenantIDFromGin(c)

	pageReq := dto.BindPage(c)

	result, err := h.projectRepo.List(ctx, nil, repository.NewPagination(pageReq.Page, pageReq.PageSize))
	if err != nil {
		logger.Error(ctx, "failed to list projects", err)
		dto.InternalError(c, "failed to list projects")
		return
	}

	resp := dto.ToProjectListResponse(result.Items)
	meta := dto.NewPageMeta(pageReq.Page, pageReq.PageSize, int(result.Total))
	dto.SuccessWithPage(c, resp, meta)
}

// CreateProject 创建项目
// @Summary 创建项目
// @Description 创建新的小说项目
// @Tags Projects
// @Accept json
// @Produce json
// @Param body body dto.CreateProjectRequest true "项目信息"
// @Success 201 {object} dto.Response[dto.ProjectResponse]
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /v1/projects [post]
func (h *ProjectHandler) CreateProject(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID := middleware.GetTenantIDFromGin(c)
	userID := middleware.GetUserIDFromGin(c)

	var req dto.CreateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequest(c, "invalid request body: "+err.Error())
		return
	}

	project := req.ToProjectEntity(tenantID, userID)

	if err := h.projectRepo.Create(ctx, project); err != nil {
		logger.Error(ctx, "failed to create project", err)
		dto.InternalError(c, "failed to create project")
		return
	}

	resp := dto.ToProjectResponse(project)
	dto.Created(c, resp)
}

// GetProject 获取项目详情
// @Summary 获取项目详情
// @Description 获取指定项目的详细信息
// @Tags Projects
// @Accept json
// @Produce json
// @Param pid path string true "项目 ID"
// @Success 200 {object} dto.Response[dto.ProjectResponse]
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /v1/projects/{pid} [get]
func (h *ProjectHandler) GetProject(c *gin.Context) {
	ctx := c.Request.Context()
	projectID := dto.BindProjectID(c)

	project, err := h.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		if errors.IsAppError(err) {
			appErr := errors.AsAppError(err)
			c.JSON(appErr.HTTPStatus, dto.ErrorResponse{
				Code:    appErr.HTTPStatus,
				Message: appErr.Message,
				TraceID: c.GetString("trace_id"),
			})
			return
		}
		logger.Error(ctx, "failed to get project", err)
		dto.InternalError(c, "failed to get project")
		return
	}

	if project == nil {
		dto.NotFound(c, "project not found")
		return
	}

	resp := dto.ToProjectResponse(project)
	dto.Success(c, resp)
}

// UpdateProject 更新项目
// @Summary 更新项目
// @Description 更新指定项目的信息
// @Tags Projects
// @Accept json
// @Produce json
// @Param pid path string true "项目 ID"
// @Param body body dto.UpdateProjectRequest true "更新内容"
// @Success 200 {object} dto.Response[dto.ProjectResponse]
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /v1/projects/{pid} [put]
func (h *ProjectHandler) UpdateProject(c *gin.Context) {
	ctx := c.Request.Context()
	projectID := dto.BindProjectID(c)

	var req dto.UpdateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequest(c, "invalid request body: "+err.Error())
		return
	}

	// 获取现有项目
	project, err := h.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		logger.Error(ctx, "failed to get project", err)
		dto.InternalError(c, "failed to get project")
		return
	}

	if project == nil {
		dto.NotFound(c, "project not found")
		return
	}

	// 应用更新
	req.ApplyToProject(project)

	// 保存更新
	if err := h.projectRepo.Update(ctx, project); err != nil {
		logger.Error(ctx, "failed to update project", err)
		dto.InternalError(c, "failed to update project")
		return
	}

	resp := dto.ToProjectResponse(project)
	dto.Success(c, resp)
}

// DeleteProject 删除项目
// @Summary 删除项目
// @Description 删除指定项目
// @Tags Projects
// @Accept json
// @Produce json
// @Param pid path string true "项目 ID"
// @Success 204 "No Content"
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /v1/projects/{pid} [delete]
func (h *ProjectHandler) DeleteProject(c *gin.Context) {
	ctx := c.Request.Context()
	projectID := dto.BindProjectID(c)

	if err := h.projectRepo.Delete(ctx, projectID); err != nil {
		if errors.IsAppError(err) {
			appErr := errors.AsAppError(err)
			c.JSON(appErr.HTTPStatus, dto.ErrorResponse{
				Code:    appErr.HTTPStatus,
				Message: appErr.Message,
				TraceID: c.GetString("trace_id"),
			})
			return
		}
		logger.Error(ctx, "failed to delete project", err)
		dto.InternalError(c, "failed to delete project")
		return
	}

	c.Status(http.StatusNoContent)
}

// GetProjectSettings 获取项目设置
// @Summary 获取项目设置
// @Description 获取指定项目的设置
// @Tags Projects
// @Accept json
// @Produce json
// @Param pid path string true "项目 ID"
// @Success 200 {object} dto.Response[dto.ProjectSettingsResponse]
// @Failure 404 {object} dto.ErrorResponse
// @Router /v1/projects/{pid}/settings [get]
func (h *ProjectHandler) GetProjectSettings(c *gin.Context) {
	ctx := c.Request.Context()
	projectID := dto.BindProjectID(c)

	project, err := h.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		logger.Error(ctx, "failed to get project", err)
		dto.InternalError(c, "failed to get project")
		return
	}

	if project == nil {
		dto.NotFound(c, "project not found")
		return
	}

	settings := &dto.ProjectSettingsResponse{}
	if project.Settings != nil {
		settings = &dto.ProjectSettingsResponse{
			DefaultChapterLength: project.Settings.DefaultChapterLength,
			WritingStyle:         project.Settings.WritingStyle,
			POV:                  project.Settings.POV,
			Temperature:          project.Settings.Temperature,
		}
	}

	dto.Success(c, settings)
}

// UpdateProjectSettings 更新项目设置
// @Summary 更新项目设置
// @Description 更新指定项目的设置
// @Tags Projects
// @Accept json
// @Produce json
// @Param pid path string true "项目 ID"
// @Param body body dto.ProjectSettingsRequest true "设置内容"
// @Success 200 {object} dto.Response[dto.ProjectSettingsResponse]
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Router /v1/projects/{pid}/settings [put]
func (h *ProjectHandler) UpdateProjectSettings(c *gin.Context) {
	ctx := c.Request.Context()
	projectID := dto.BindProjectID(c)

	var req dto.ProjectSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequest(c, "invalid request body: "+err.Error())
		return
	}

	// 获取现有项目
	project, err := h.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		logger.Error(ctx, "failed to get project", err)
		dto.InternalError(c, "failed to get project")
		return
	}

	if project == nil {
		dto.NotFound(c, "project not found")
		return
	}

	// 更新设置
	updateReq := &dto.UpdateProjectRequest{
		Settings: &req,
	}
	updateReq.ApplyToProject(project)

	if err := h.projectRepo.Update(ctx, project); err != nil {
		logger.Error(ctx, "failed to update project", err)
		dto.InternalError(c, "failed to update project")
		return
	}

	settings := &dto.ProjectSettingsResponse{
		DefaultChapterLength: project.Settings.DefaultChapterLength,
		WritingStyle:         project.Settings.WritingStyle,
		POV:                  project.Settings.POV,
		Temperature:          project.Settings.Temperature,
	}

	dto.Success(c, settings)
}
