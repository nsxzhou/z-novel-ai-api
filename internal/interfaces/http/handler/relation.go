// Package handler 提供 HTTP 请求处理器
package handler

import (
	"net/http"

	"z-novel-ai-api/internal/domain/entity"
	"z-novel-ai-api/internal/domain/repository"
	"z-novel-ai-api/internal/interfaces/http/dto"
	"z-novel-ai-api/pkg/errors"
	"z-novel-ai-api/pkg/logger"

	"github.com/gin-gonic/gin"
)

// RelationHandler 关系处理器
type RelationHandler struct {
	relationRepo repository.RelationRepository
}

// NewRelationHandler 创建关系处理器
func NewRelationHandler(relationRepo repository.RelationRepository) *RelationHandler {
	return &RelationHandler{
		relationRepo: relationRepo,
	}
}

// ListRelations 获取项目关系列表
// @Summary 获取项目关系列表
// @Description 获取指定项目的实体关系列表
// @Tags Relations
// @Accept json
// @Produce json
// @Param pid path string true "项目 ID"
// @Param type query string false "关系类型 (friend, enemy, family, lover, subordinate, mentor, rival, ally)"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页条数" default(20)
// @Success 200 {object} dto.Response[dto.RelationListResponse]
// @Failure 500 {object} dto.ErrorResponse
// @Router /v1/projects/{pid}/relations [get]
func (h *RelationHandler) ListRelations(c *gin.Context) {
	ctx := c.Request.Context()
	projectID := dto.BindProjectID(c)
	pageReq := dto.BindPage(c)

	filter := &repository.RelationFilter{
		RelationType: entity.RelationType(c.Query("type")),
	}

	result, err := h.relationRepo.ListByProject(ctx, projectID, filter, repository.NewPagination(pageReq.Page, pageReq.PageSize))
	if err != nil {
		logger.Error(ctx, "failed to list relations", err)
		dto.InternalError(c, "failed to list relations")
		return
	}

	resp := dto.ToRelationListResponse(result.Items)
	meta := dto.NewPageMeta(pageReq.Page, pageReq.PageSize, int(result.Total))
	dto.SuccessWithPage(c, resp, meta)
}

// CreateRelation 创建关系
// @Summary 创建关系
// @Description 在指定项目下创建实体间关系
// @Tags Relations
// @Accept json
// @Produce json
// @Param pid path string true "项目 ID"
// @Param body body dto.CreateRelationRequest true "关系信息"
// @Success 201 {object} dto.Response[dto.RelationResponse]
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /v1/projects/{pid}/relations [post]
func (h *RelationHandler) CreateRelation(c *gin.Context) {
	ctx := c.Request.Context()
	projectID := dto.BindProjectID(c)

	var req dto.CreateRelationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequest(c, "invalid request body: "+err.Error())
		return
	}

	relation := req.ToRelationEntity(projectID)

	if err := h.relationRepo.Create(ctx, relation); err != nil {
		logger.Error(ctx, "failed to create relation", err)
		dto.InternalError(c, "failed to create relation")
		return
	}

	resp := dto.ToRelationResponse(relation)
	dto.Created(c, resp)
}

// GetRelation 获取关系详情
// @Summary 获取关系详情
// @Description 获取指定关系的详细信息
// @Tags Relations
// @Accept json
// @Produce json
// @Param rid path string true "关系 ID"
// @Success 200 {object} dto.Response[dto.RelationResponse]
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /v1/relations/{rid} [get]
func (h *RelationHandler) GetRelation(c *gin.Context) {
	ctx := c.Request.Context()
	relationID := c.Param("rid")

	relation, err := h.relationRepo.GetByID(ctx, relationID)
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
		logger.Error(ctx, "failed to get relation", err)
		dto.InternalError(c, "failed to get relation")
		return
	}

	if relation == nil {
		dto.NotFound(c, "relation not found")
		return
	}

	resp := dto.ToRelationResponse(relation)
	dto.Success(c, resp)
}

// UpdateRelation 更新关系
// @Summary 更新关系
// @Description 更新指定关系的信息
// @Tags Relations
// @Accept json
// @Produce json
// @Param rid path string true "关系 ID"
// @Param body body dto.UpdateRelationRequest true "更新内容"
// @Success 200 {object} dto.Response[dto.RelationResponse]
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /v1/relations/{rid} [put]
func (h *RelationHandler) UpdateRelation(c *gin.Context) {
	ctx := c.Request.Context()
	relationID := c.Param("rid")

	var req dto.UpdateRelationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequest(c, "invalid request body: "+err.Error())
		return
	}

	// 获取现有关系
	relation, err := h.relationRepo.GetByID(ctx, relationID)
	if err != nil {
		logger.Error(ctx, "failed to get relation", err)
		dto.InternalError(c, "failed to get relation")
		return
	}

	if relation == nil {
		dto.NotFound(c, "relation not found")
		return
	}

	// 应用更新
	req.ApplyToRelation(relation)

	// 保存更新
	if err := h.relationRepo.Update(ctx, relation); err != nil {
		logger.Error(ctx, "failed to update relation", err)
		dto.InternalError(c, "failed to update relation")
		return
	}

	resp := dto.ToRelationResponse(relation)
	dto.Success(c, resp)
}

// DeleteRelation 删除关系
// @Summary 删除关系
// @Description 删除指定关系
// @Tags Relations
// @Accept json
// @Produce json
// @Param rid path string true "关系 ID"
// @Success 204 "No Content"
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /v1/relations/{rid} [delete]
func (h *RelationHandler) DeleteRelation(c *gin.Context) {
	ctx := c.Request.Context()
	relationID := c.Param("rid")

	if err := h.relationRepo.Delete(ctx, relationID); err != nil {
		if errors.IsAppError(err) {
			appErr := errors.AsAppError(err)
			c.JSON(appErr.HTTPStatus, dto.ErrorResponse{
				Code:    appErr.HTTPStatus,
				Message: appErr.Message,
				TraceID: c.GetString("trace_id"),
			})
			return
		}
		logger.Error(ctx, "failed to delete relation", err)
		dto.InternalError(c, "failed to delete relation")
		return
	}

	c.Status(http.StatusNoContent)
}
