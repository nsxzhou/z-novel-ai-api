// Package handler 提供 HTTP 请求处理器
package handler

import (
	"z-novel-ai-api/internal/domain/entity"
	"z-novel-ai-api/internal/domain/repository"
	"z-novel-ai-api/internal/interfaces/http/dto"
	"z-novel-ai-api/pkg/errors"
	"z-novel-ai-api/pkg/logger"

	"github.com/gin-gonic/gin"
)

// JobHandler 任务处理器
type JobHandler struct {
	jobRepo repository.JobRepository
}

// NewJobHandler 创建任务处理器
func NewJobHandler(jobRepo repository.JobRepository) *JobHandler {
	return &JobHandler{
		jobRepo: jobRepo,
	}
}

// GetJob 获取任务详情
// @Summary 获取任务详情
// @Description 获取指定任务的详细信息和状态
// @Tags Jobs
// @Accept json
// @Produce json
// @Param jid path string true "任务 ID"
// @Success 200 {object} dto.Response[dto.JobResponse]
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /v1/jobs/{jid} [get]
func (h *JobHandler) GetJob(c *gin.Context) {
	ctx := c.Request.Context()
	jobID := dto.BindJobID(c)

	job, err := h.jobRepo.GetByID(ctx, jobID)
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
		logger.Error(ctx, "failed to get job", err)
		dto.InternalError(c, "failed to get job")
		return
	}

	if job == nil {
		dto.NotFound(c, "job not found")
		return
	}

	resp := dto.ToJobResponse(job)
	dto.Success(c, resp)
}

// CancelJob 取消任务
// @Summary 取消任务
// @Description 取消指定的任务
// @Tags Jobs
// @Accept json
// @Produce json
// @Param jid path string true "任务 ID"
// @Success 200 {object} dto.Response[dto.CancelJobResponse]
// @Failure 404 {object} dto.ErrorResponse
// @Failure 409 {object} dto.ErrorResponse "任务无法取消"
// @Failure 500 {object} dto.ErrorResponse
// @Router /v1/jobs/{jid} [delete]
func (h *JobHandler) CancelJob(c *gin.Context) {
	ctx := c.Request.Context()
	jobID := dto.BindJobID(c)

	// 获取任务
	job, err := h.jobRepo.GetByID(ctx, jobID)
	if err != nil {
		logger.Error(ctx, "failed to get job", err)
		dto.InternalError(c, "failed to get job")
		return
	}

	if job == nil {
		dto.NotFound(c, "job not found")
		return
	}

	// 检查任务状态
	if job.Status == entity.JobStatusCompleted || job.Status == entity.JobStatusFailed {
		dto.Conflict(c, "job already finished")
		return
	}

	if job.Status == entity.JobStatusCancelled {
		dto.Success(c, &dto.CancelJobResponse{
			ID:        jobID,
			Cancelled: true,
		})
		return
	}

	// 取消任务
	job.Status = entity.JobStatusCancelled
	if err := h.jobRepo.Update(ctx, job); err != nil {
		logger.Error(ctx, "failed to cancel job", err)
		dto.InternalError(c, "failed to cancel job")
		return
	}

	dto.Success(c, &dto.CancelJobResponse{
		ID:        jobID,
		Cancelled: true,
	})
}

// ListProjectJobs 获取项目任务列表（内部方法，可选暴露为 API）
func (h *JobHandler) ListProjectJobs(c *gin.Context) {
	ctx := c.Request.Context()
	projectID := dto.BindProjectID(c)
	pageReq := dto.BindPage(c)

	// 获取状态过滤
	status := c.Query("status")

	// 构建过滤条件
	var filter *repository.JobFilter
	if status != "" {
		filter = &repository.JobFilter{
			Status: entity.JobStatus(status),
		}
	}

	result, err := h.jobRepo.ListByProject(ctx, projectID, filter, repository.NewPagination(pageReq.Page, pageReq.PageSize))
	if err != nil {
		logger.Error(ctx, "failed to list jobs", err)
		dto.InternalError(c, "failed to list jobs")
		return
	}

	resp := dto.ToJobListResponse(result.Items)
	meta := dto.NewPageMeta(pageReq.Page, pageReq.PageSize, int(result.Total))
	dto.SuccessWithPage(c, resp, meta)
}
