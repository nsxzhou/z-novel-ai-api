// Package handler 提供 HTTP 请求处理器
package handler

import (
	"net/http"

	"z-novel-ai-api/internal/domain/entity"
	"z-novel-ai-api/internal/domain/repository"
	"z-novel-ai-api/internal/interfaces/http/dto"
	"z-novel-ai-api/internal/interfaces/http/middleware"
	"z-novel-ai-api/pkg/errors"
	"z-novel-ai-api/pkg/logger"

	"github.com/gin-gonic/gin"
)

// EntityHandler 实体处理器
type EntityHandler struct {
	entityRepo   repository.EntityRepository
	relationRepo repository.RelationRepository
}

// NewEntityHandler 创建实体处理器
func NewEntityHandler(
	entityRepo repository.EntityRepository,
	relationRepo repository.RelationRepository,
) *EntityHandler {
	return &EntityHandler{
		entityRepo:   entityRepo,
		relationRepo: relationRepo,
	}
}

// ListEntities 获取实体列表
// @Summary 获取实体列表
// @Description 获取指定项目的实体列表
// @Tags Entities
// @Accept json
// @Produce json
// @Param pid path string true "项目 ID"
// @Param type query string false "实体类型" Enums(character, location, item, organization, concept)
// @Param importance query string false "重要性" Enums(protagonist, major, minor, background)
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页条数" default(20)
// @Success 200 {object} dto.Response[dto.EntityListResponse]
// @Failure 500 {object} dto.ErrorResponse
// @Router /v1/projects/{pid}/entities [get]
func (h *EntityHandler) ListEntities(c *gin.Context) {
	ctx := c.Request.Context()
	projectID := dto.BindProjectID(c)
	pageReq := dto.BindPage(c)

	// 获取过滤参数
	entityType := c.Query("type")
	importance := c.Query("importance")

	// 构建过滤条件
	var filter *repository.EntityFilter
	if entityType != "" || importance != "" {
		filter = &repository.EntityFilter{}
		if entityType != "" {
			filter.Type = entity.StoryEntityType(entityType)
		}
		if importance != "" {
			filter.Importance = entity.EntityImportance(importance)
		}
	}

	result, err := h.entityRepo.ListByProject(ctx, projectID, filter, repository.NewPagination(pageReq.Page, pageReq.PageSize))
	if err != nil {
		logger.Error(ctx, "failed to list entities", err)
		dto.InternalError(c, "failed to list entities")
		return
	}

	resp := dto.ToEntityListResponse(result.Items)
	meta := dto.NewPageMeta(pageReq.Page, pageReq.PageSize, int(result.Total))
	dto.SuccessWithPage(c, resp, meta)
}

// CreateEntity 创建实体
// @Summary 创建实体
// @Description 创建新的实体（角色、地点、物品等）
// @Tags Entities
// @Accept json
// @Produce json
// @Param pid path string true "项目 ID"
// @Param body body dto.CreateEntityRequest true "实体信息"
// @Success 201 {object} dto.Response[dto.EntityResponse]
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /v1/projects/{pid}/entities [post]
func (h *EntityHandler) CreateEntity(c *gin.Context) {
	ctx := c.Request.Context()
	projectID := dto.BindProjectID(c)

	var req dto.CreateEntityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequest(c, "invalid request body: "+err.Error())
		return
	}

	entity := req.ToStoryEntity(projectID)

	if err := h.entityRepo.Create(ctx, entity); err != nil {
		logger.Error(ctx, "failed to create entity", err)
		dto.InternalError(c, "failed to create entity")
		return
	}

	resp := dto.ToEntityResponse(entity)
	dto.Created(c, resp)
}

// GetEntity 获取实体详情
// @Summary 获取实体详情
// @Description 获取指定实体的详细信息
// @Tags Entities
// @Accept json
// @Produce json
// @Param eid path string true "实体 ID"
// @Success 200 {object} dto.Response[dto.EntityResponse]
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /v1/entities/{eid} [get]
func (h *EntityHandler) GetEntity(c *gin.Context) {
	ctx := c.Request.Context()
	entityID := dto.BindEntityID(c)

	entity, err := h.entityRepo.GetByID(ctx, entityID)
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
		logger.Error(ctx, "failed to get entity", err)
		dto.InternalError(c, "failed to get entity")
		return
	}

	if entity == nil {
		dto.NotFound(c, "entity not found")
		return
	}

	resp := dto.ToEntityResponse(entity)
	dto.Success(c, resp)
}

// UpdateEntity 更新实体
// @Summary 更新实体
// @Description 更新指定实体的信息
// @Tags Entities
// @Accept json
// @Produce json
// @Param eid path string true "实体 ID"
// @Param body body dto.UpdateEntityRequest true "更新内容"
// @Success 200 {object} dto.Response[dto.EntityResponse]
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /v1/entities/{eid} [put]
func (h *EntityHandler) UpdateEntity(c *gin.Context) {
	ctx := c.Request.Context()
	entityID := dto.BindEntityID(c)

	var req dto.UpdateEntityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequest(c, "invalid request body: "+err.Error())
		return
	}

	// 获取现有实体
	entity, err := h.entityRepo.GetByID(ctx, entityID)
	if err != nil {
		logger.Error(ctx, "failed to get entity", err)
		dto.InternalError(c, "failed to get entity")
		return
	}

	if entity == nil {
		dto.NotFound(c, "entity not found")
		return
	}

	// 应用更新
	req.ApplyToEntity(entity)

	// 保存更新
	if err := h.entityRepo.Update(ctx, entity); err != nil {
		logger.Error(ctx, "failed to update entity", err)
		dto.InternalError(c, "failed to update entity")
		return
	}

	resp := dto.ToEntityResponse(entity)
	dto.Success(c, resp)
}

// DeleteEntity 删除实体
// @Summary 删除实体
// @Description 删除指定实体
// @Tags Entities
// @Accept json
// @Produce json
// @Param eid path string true "实体 ID"
// @Success 204 "No Content"
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /v1/entities/{eid} [delete]
func (h *EntityHandler) DeleteEntity(c *gin.Context) {
	ctx := c.Request.Context()
	entityID := dto.BindEntityID(c)

	if err := h.entityRepo.Delete(ctx, entityID); err != nil {
		if errors.IsAppError(err) {
			appErr := errors.AsAppError(err)
			c.JSON(appErr.HTTPStatus, dto.ErrorResponse{
				Code:    appErr.HTTPStatus,
				Message: appErr.Message,
				TraceID: c.GetString("trace_id"),
			})
			return
		}
		logger.Error(ctx, "failed to delete entity", err)
		dto.InternalError(c, "failed to delete entity")
		return
	}

	c.Status(http.StatusNoContent)
}

// UpdateEntityState 更新实体状态
// @Summary 更新实体状态
// @Description 更新实体的当前状态
// @Tags Entities
// @Accept json
// @Produce json
// @Param eid path string true "实体 ID"
// @Param body body dto.UpdateEntityStateRequest true "状态更新"
// @Success 200 {object} dto.Response[dto.EntityResponse]
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /v1/entities/{eid}/state [put]
func (h *EntityHandler) UpdateEntityState(c *gin.Context) {
	ctx := c.Request.Context()
	entityID := dto.BindEntityID(c)

	var req dto.UpdateEntityStateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequest(c, "invalid request body: "+err.Error())
		return
	}

	// 获取现有实体
	entity, err := h.entityRepo.GetByID(ctx, entityID)
	if err != nil {
		logger.Error(ctx, "failed to get entity", err)
		dto.InternalError(c, "failed to get entity")
		return
	}

	if entity == nil {
		dto.NotFound(c, "entity not found")
		return
	}

	// 更新状态
	entity.UpdateState(req.CurrentState, req.ChapterID, req.StoryTime)

	// 合并属性变更（保存到元数据）
	for k, v := range req.AttributeChanges {
		entity.Metadata[k] = v
	}

	// 保存更新
	if err := h.entityRepo.Update(ctx, entity); err != nil {
		logger.Error(ctx, "failed to update entity state", err)
		dto.InternalError(c, "failed to update entity state")
		return
	}

	resp := dto.ToEntityResponse(entity)
	dto.Success(c, resp)
}

// GetEntityRelations 获取实体关系
// @Summary 获取实体关系
// @Description 获取指定实体的所有关系
// @Tags Entities
// @Accept json
// @Produce json
// @Param eid path string true "实体 ID"
// @Success 200 {object} dto.Response[dto.EntityRelationsResponse]
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /v1/entities/{eid}/relations [get]
func (h *EntityHandler) GetEntityRelations(c *gin.Context) {
	ctx := c.Request.Context()
	entityID := dto.BindEntityID(c)
	_ = middleware.GetTenantIDFromGin(c)

	relations, err := h.relationRepo.ListByEntity(ctx, entityID)
	if err != nil {
		logger.Error(ctx, "failed to get entity relations", err)
		dto.InternalError(c, "failed to get entity relations")
		return
	}

	resp := dto.ToEntityRelationsResponse(entityID, relations)
	dto.Success(c, resp)
}
