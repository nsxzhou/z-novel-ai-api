// Package handler 提供 HTTP 请求处理器
package handler

import (
	"encoding/json"
	"net/http"

	"z-novel-ai-api/internal/domain/entity"
	"z-novel-ai-api/internal/domain/repository"
	"z-novel-ai-api/internal/infrastructure/messaging"
	"z-novel-ai-api/internal/interfaces/http/dto"
	"z-novel-ai-api/internal/interfaces/http/middleware"
	"z-novel-ai-api/pkg/errors"
	"z-novel-ai-api/pkg/logger"

	"github.com/gin-gonic/gin"
)

// ChapterHandler 章节处理器
type ChapterHandler struct {
	chapterRepo repository.ChapterRepository
	projectRepo repository.ProjectRepository
	jobRepo     repository.JobRepository
	producer    *messaging.Producer
}

// NewChapterHandler 创建章节处理器
func NewChapterHandler(
	chapterRepo repository.ChapterRepository,
	projectRepo repository.ProjectRepository,
	jobRepo repository.JobRepository,
	producer *messaging.Producer,
) *ChapterHandler {
	return &ChapterHandler{
		chapterRepo: chapterRepo,
		projectRepo: projectRepo,
		jobRepo:     jobRepo,
		producer:    producer,
	}
}

// ListChapters 获取章节列表
// @Summary 获取章节列表
// @Description 获取指定项目的章节列表
// @Tags Chapters
// @Accept json
// @Produce json
// @Param pid path string true "项目 ID"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页条数" default(20)
// @Success 200 {object} dto.Response[dto.ChapterListResponse]
// @Failure 500 {object} dto.ErrorResponse
// @Router /v1/projects/{pid}/chapters [get]
func (h *ChapterHandler) ListChapters(c *gin.Context) {
	ctx := c.Request.Context()
	projectID := dto.BindProjectID(c)
	pageReq := dto.BindPage(c)

	result, err := h.chapterRepo.ListByProject(ctx, projectID, nil, repository.NewPagination(pageReq.Page, pageReq.PageSize))
	if err != nil {
		logger.Error(ctx, "failed to list chapters", err)
		dto.InternalError(c, "failed to list chapters")
		return
	}

	resp := dto.ToChapterListResponse(result.Items)
	meta := dto.NewPageMeta(pageReq.Page, pageReq.PageSize, int(result.Total))
	dto.SuccessWithPage(c, resp, meta)
}

// CreateChapter 创建章节
// @Summary 创建章节
// @Description 创建新的章节
// @Tags Chapters
// @Accept json
// @Produce json
// @Param pid path string true "项目 ID"
// @Param body body dto.CreateChapterRequest true "章节信息"
// @Success 201 {object} dto.Response[dto.ChapterResponse]
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /v1/projects/{pid}/chapters [post]
func (h *ChapterHandler) CreateChapter(c *gin.Context) {
	ctx := c.Request.Context()
	projectID := dto.BindProjectID(c)

	var req dto.CreateChapterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequest(c, "invalid request body: "+err.Error())
		return
	}

	// 获取当前最大序号
	maxSeq, err := h.chapterRepo.GetNextSeqNum(ctx, projectID, req.VolumeID)
	if err != nil {
		logger.Error(ctx, "failed to get next seq num", err)
		dto.InternalError(c, "failed to create chapter")
		return
	}

	chapter := req.ToChapterEntity(projectID, maxSeq)

	if err := h.chapterRepo.Create(ctx, chapter); err != nil {
		logger.Error(ctx, "failed to create chapter", err)
		dto.InternalError(c, "failed to create chapter")
		return
	}

	resp := dto.ToChapterResponse(chapter)
	dto.Created(c, resp)
}

// GetChapter 获取章节详情
// @Summary 获取章节详情
// @Description 获取指定章节的详细信息
// @Tags Chapters
// @Accept json
// @Produce json
// @Param cid path string true "章节 ID"
// @Success 200 {object} dto.Response[dto.ChapterResponse]
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /v1/chapters/{cid} [get]
func (h *ChapterHandler) GetChapter(c *gin.Context) {
	ctx := c.Request.Context()
	chapterID := dto.BindChapterID(c)

	chapter, err := h.chapterRepo.GetByID(ctx, chapterID)
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
		logger.Error(ctx, "failed to get chapter", err)
		dto.InternalError(c, "failed to get chapter")
		return
	}

	if chapter == nil {
		dto.NotFound(c, "chapter not found")
		return
	}

	resp := dto.ToChapterResponse(chapter)
	dto.Success(c, resp)
}

// UpdateChapter 更新章节
// @Summary 更新章节
// @Description 更新指定章节的信息
// @Tags Chapters
// @Accept json
// @Produce json
// @Param cid path string true "章节 ID"
// @Param body body dto.UpdateChapterRequest true "更新内容"
// @Success 200 {object} dto.Response[dto.ChapterResponse]
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /v1/chapters/{cid} [put]
func (h *ChapterHandler) UpdateChapter(c *gin.Context) {
	ctx := c.Request.Context()
	chapterID := dto.BindChapterID(c)

	var req dto.UpdateChapterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequest(c, "invalid request body: "+err.Error())
		return
	}

	// 获取现有章节
	chapter, err := h.chapterRepo.GetByID(ctx, chapterID)
	if err != nil {
		logger.Error(ctx, "failed to get chapter", err)
		dto.InternalError(c, "failed to get chapter")
		return
	}

	if chapter == nil {
		dto.NotFound(c, "chapter not found")
		return
	}

	// 应用更新
	req.ApplyToChapter(chapter)

	// 保存更新
	if err := h.chapterRepo.Update(ctx, chapter); err != nil {
		logger.Error(ctx, "failed to update chapter", err)
		dto.InternalError(c, "failed to update chapter")
		return
	}

	resp := dto.ToChapterResponse(chapter)
	dto.Success(c, resp)
}

// DeleteChapter 删除章节
// @Summary 删除章节
// @Description 删除指定章节
// @Tags Chapters
// @Accept json
// @Produce json
// @Param cid path string true "章节 ID"
// @Success 204 "No Content"
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /v1/chapters/{cid} [delete]
func (h *ChapterHandler) DeleteChapter(c *gin.Context) {
	ctx := c.Request.Context()
	chapterID := dto.BindChapterID(c)

	if err := h.chapterRepo.Delete(ctx, chapterID); err != nil {
		if errors.IsAppError(err) {
			appErr := errors.AsAppError(err)
			c.JSON(appErr.HTTPStatus, dto.ErrorResponse{
				Code:    appErr.HTTPStatus,
				Message: appErr.Message,
				TraceID: c.GetString("trace_id"),
			})
			return
		}
		logger.Error(ctx, "failed to delete chapter", err)
		dto.InternalError(c, "failed to delete chapter")
		return
	}

	c.Status(http.StatusNoContent)
}

// GenerateChapter 生成章节（异步）
// @Summary 生成章节
// @Description 异步生成章节内容，返回任务 ID
// @Tags Chapters
// @Accept json
// @Produce json
// @Param pid path string true "项目 ID"
// @Param body body dto.GenerateChapterRequest true "生成请求"
// @Success 202 {object} dto.Response[dto.JobResponse]
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /v1/projects/{pid}/chapters/generate [post]
func (h *ChapterHandler) GenerateChapter(c *gin.Context) {
	ctx := c.Request.Context()
	projectID := dto.BindProjectID(c)

	var req dto.GenerateChapterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequest(c, "invalid request body: "+err.Error())
		return
	}

	tenantID := middleware.GetTenantID(ctx)
	if tenantID == "" {
		dto.BadRequest(c, "missing tenant_id")
		return
	}

	// 验证项目存在
	project, err := h.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		logger.Error(ctx, "failed to get project", err)
		dto.InternalError(c, "failed to create generation task")
		return
	}
	if project == nil {
		dto.NotFound(c, "project not found")
		return
	}

	// 创建章节记录（状态置为 generating）
	maxSeq, err := h.chapterRepo.GetNextSeqNum(ctx, projectID, req.VolumeID)
	if err != nil {
		logger.Error(ctx, "failed to get next seq num", err)
		dto.InternalError(c, "failed to create generation task")
		return
	}

	chapter := entity.NewChapter(projectID, req.VolumeID, maxSeq)
	chapter.Title = req.Title
	chapter.Outline = req.Outline
	chapter.Notes = req.Notes
	chapter.StoryTimeStart = req.StoryTimeStart
	chapter.Status = entity.ChapterStatusGenerating

	if err := h.chapterRepo.Create(ctx, chapter); err != nil {
		logger.Error(ctx, "failed to create chapter", err)
		dto.InternalError(c, "failed to create generation task")
		return
	}

	params := map[string]interface{}{
		"outline":           req.Outline,
		"target_word_count": req.TargetWordCount,
		"story_time_start":  req.StoryTimeStart,
	}
	if req.Options != nil {
		params["options"] = map[string]interface{}{
			"model":           req.Options.Model,
			"temperature":     req.Options.Temperature,
			"skip_validation": req.Options.SkipValidation,
			"max_retries":     req.Options.MaxRetries,
		}
	}
	inputParams, _ := json.Marshal(params)

	job := entity.NewGenerationJob(tenantID, projectID, entity.JobTypeChapterGen, inputParams)
	job.ChapterID = chapter.ID
	job.UpdateProgress(0)   

	if err := h.jobRepo.Create(ctx, job); err != nil {
		logger.Error(ctx, "failed to create job", err)
		dto.InternalError(c, "failed to create generation task")
		return
	}

	// 发送到消息队列（由 job-worker 执行）
	_, err = h.producer.PublishGenJob(ctx, &messaging.GenerationJobMessage{
		JobID:     job.ID,
		TenantID:  tenantID,
		ProjectID: projectID,
		ChapterID: chapter.ID,
		JobType:   string(job.JobType),
		Priority:  job.Priority,
		Params:    params,
	})
	if err != nil {
		logger.Error(ctx, "failed to publish generation job", err)
		dto.InternalError(c, "failed to enqueue generation task")
		return
	}

	logger.Info(ctx, "generate chapter request received",
		"project_id", projectID,
		"chapter_id", chapter.ID,
		"job_id", job.ID,
		"outline", req.Outline,
		"target_word_count", req.TargetWordCount,
	)

	dto.Accepted(c, dto.ToJobResponse(job))
}

// RegenerateChapter 重新生成章节
// @Summary 重新生成章节
// @Description 重新生成指定章节的内容
// @Tags Chapters
// @Accept json
// @Produce json
// @Param cid path string true "章节 ID"
// @Param body body dto.RegenerateChapterRequest true "重新生成请求"
// @Success 202 {object} dto.Response[dto.JobResponse]
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /v1/chapters/{cid}/regenerate [post]
func (h *ChapterHandler) RegenerateChapter(c *gin.Context) {
	ctx := c.Request.Context()
	chapterID := dto.BindChapterID(c)

	var req dto.RegenerateChapterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequest(c, "invalid request body: "+err.Error())
		return
	}

	// 验证章节存在
	chapter, err := h.chapterRepo.GetByID(ctx, chapterID)
	if err != nil {
		logger.Error(ctx, "failed to get chapter", err)
		dto.InternalError(c, "failed to get chapter")
		return
	}

	if chapter == nil {
		dto.NotFound(c, "chapter not found")
		return
	}

	tenantID := middleware.GetTenantID(ctx)
	if tenantID == "" {
		dto.BadRequest(c, "missing tenant_id")
		return
	}

	// 应用更新（可选覆盖大纲）
	if req.Outline != "" {
		chapter.Outline = req.Outline
	}
	chapter.ContentText = ""
	chapter.WordCount = 0
	chapter.Status = entity.ChapterStatusGenerating

	if err := h.chapterRepo.Update(ctx, chapter); err != nil {
		logger.Error(ctx, "failed to update chapter", err)
		dto.InternalError(c, "failed to create regeneration task")
		return
	}

	params := map[string]interface{}{
		"outline":           chapter.Outline,
		"target_word_count": req.TargetWordCount,
		"story_time_start":  chapter.StoryTimeStart,
	}
	if req.Options != nil {
		params["options"] = map[string]interface{}{
			"model":           req.Options.Model,
			"temperature":     req.Options.Temperature,
			"skip_validation": req.Options.SkipValidation,
			"max_retries":     req.Options.MaxRetries,
		}
	}
	inputParams, _ := json.Marshal(params)

	job := entity.NewGenerationJob(tenantID, chapter.ProjectID, entity.JobTypeChapterGen, inputParams)
	job.ChapterID = chapter.ID
	job.UpdateProgress(0)

	if err := h.jobRepo.Create(ctx, job); err != nil {
		logger.Error(ctx, "failed to create job", err)
		dto.InternalError(c, "failed to create regeneration task")
		return
	}

	_, err = h.producer.PublishGenJob(ctx, &messaging.GenerationJobMessage{
		JobID:     job.ID,
		TenantID:  tenantID,
		ProjectID: chapter.ProjectID,
		ChapterID: chapter.ID,
		JobType:   string(job.JobType),
		Priority:  job.Priority,
		Params:    params,
	})
	if err != nil {
		logger.Error(ctx, "failed to publish regeneration job", err)
		dto.InternalError(c, "failed to enqueue regeneration task")
		return
	}

	logger.Info(ctx, "regenerate chapter request received",
		"chapter_id", chapterID,
		"job_id", job.ID,
	)

	dto.Accepted(c, dto.ToJobResponse(job))
}
