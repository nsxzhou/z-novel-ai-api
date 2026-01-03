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

// EventHandler 事件处理器
type EventHandler struct {
	eventRepo repository.EventRepository
}

// NewEventHandler 创建事件处理器
func NewEventHandler(eventRepo repository.EventRepository) *EventHandler {
	return &EventHandler{
		eventRepo: eventRepo,
	}
}

// ListEvents 获取项目事件列表
// @Summary 获取项目事件列表
// @Description 根据项目 ID 获取事件，支持按类型、重要性及时间范围过滤
// @Tags Events
// @Accept json
// @Produce json
// @Param pid path string true "项目 ID"
// @Param type query string false "事件类型 (plot, dialogue, action, description)"
// @Param importance query string false "重要性 (critical, major, normal, minor)"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页条数" default(20)
// @Success 200 {object} dto.Response[dto.EventListResponse]
// @Failure 500 {object} dto.ErrorResponse
// @Router /v1/projects/{pid}/events [get]
func (h *EventHandler) ListEvents(c *gin.Context) {
	ctx := c.Request.Context()
	projectID := dto.BindProjectID(c)
	pageReq := dto.BindPage(c)

	filter := &repository.EventFilter{
		EventType:  entity.EventType(c.Query("type")),
		Importance: entity.EventImportance(c.Query("importance")),
	}

	result, err := h.eventRepo.ListByProject(ctx, projectID, filter, repository.NewPagination(pageReq.Page, pageReq.PageSize))
	if err != nil {
		logger.Error(ctx, "failed to list events", err)
		dto.InternalError(c, "failed to list events")
		return
	}

	resp := dto.ToEventListResponse(result.Items)
	meta := dto.NewPageMeta(pageReq.Page, pageReq.PageSize, int(result.Total))
	dto.SuccessWithPage(c, resp, meta)
}

// CreateEvent 创建事件
// @Summary 创建事件
// @Description 在指定项目下创建新事件
// @Tags Events
// @Accept json
// @Produce json
// @Param pid path string true "项目 ID"
// @Param body body dto.CreateEventRequest true "事件信息"
// @Success 201 {object} dto.Response[dto.EventResponse]
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /v1/projects/{pid}/events [post]
func (h *EventHandler) CreateEvent(c *gin.Context) {
	ctx := c.Request.Context()
	projectID := dto.BindProjectID(c)

	var req dto.CreateEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequest(c, "invalid request body: "+err.Error())
		return
	}

	event := req.ToEventEntity(projectID)

	if err := h.eventRepo.Create(ctx, event); err != nil {
		logger.Error(ctx, "failed to create event", err)
		dto.InternalError(c, "failed to create event")
		return
	}

	resp := dto.ToEventResponse(event)
	dto.Created(c, resp)
}

// GetEvent 获取事件详情
// @Summary 获取事件详情
// @Description 获取指定事件的详细信息
// @Tags Events
// @Accept json
// @Produce json
// @Param evid path string true "事件 ID"
// @Success 200 {object} dto.Response[dto.EventResponse]
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /v1/events/{evid} [get]
func (h *EventHandler) GetEvent(c *gin.Context) {
	ctx := c.Request.Context()
	eventID := c.Param("evid")

	event, err := h.eventRepo.GetByID(ctx, eventID)
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
		logger.Error(ctx, "failed to get event", err)
		dto.InternalError(c, "failed to get event")
		return
	}

	if event == nil {
		dto.NotFound(c, "event not found")
		return
	}

	resp := dto.ToEventResponse(event)
	dto.Success(c, resp)
}

// UpdateEvent 更新事件
// @Summary 更新事件
// @Description 更新指定事件的信息
// @Tags Events
// @Accept json
// @Produce json
// @Param evid path string true "事件 ID"
// @Param body body dto.UpdateEventRequest true "更新内容"
// @Success 200 {object} dto.Response[dto.EventResponse]
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /v1/events/{evid} [put]
func (h *EventHandler) UpdateEvent(c *gin.Context) {
	ctx := c.Request.Context()
	eventID := c.Param("evid")

	var req dto.UpdateEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequest(c, "invalid request body: "+err.Error())
		return
	}

	// 获取现有事件
	event, err := h.eventRepo.GetByID(ctx, eventID)
	if err != nil {
		logger.Error(ctx, "failed to get event", err)
		dto.InternalError(c, "failed to get event")
		return
	}

	if event == nil {
		dto.NotFound(c, "event not found")
		return
	}

	// 应用更新
	req.ApplyToEvent(event)

	// 保存更新
	if err := h.eventRepo.Update(ctx, event); err != nil {
		logger.Error(ctx, "failed to update event", err)
		dto.InternalError(c, "failed to update event")
		return
	}

	resp := dto.ToEventResponse(event)
	dto.Success(c, resp)
}

// DeleteEvent 删除事件
// @Summary 删除事件
// @Description 删除指定事件
// @Tags Events
// @Accept json
// @Produce json
// @Param evid path string true "事件 ID"
// @Success 204 "No Content"
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /v1/events/{evid} [delete]
func (h *EventHandler) DeleteEvent(c *gin.Context) {
	ctx := c.Request.Context()
	eventID := c.Param("evid")

	if err := h.eventRepo.Delete(ctx, eventID); err != nil {
		if errors.IsAppError(err) {
			appErr := errors.AsAppError(err)
			c.JSON(appErr.HTTPStatus, dto.ErrorResponse{
				Code:    appErr.HTTPStatus,
				Message: appErr.Message,
				TraceID: c.GetString("trace_id"),
			})
			return
		}
		logger.Error(ctx, "failed to delete event", err)
		dto.InternalError(c, "failed to delete event")
		return
	}

	c.Status(http.StatusNoContent)
}
