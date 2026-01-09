// Package handler 提供 HTTP 请求处理器
package handler

import (
	"encoding/json"
	stderrors "errors"
	"net/http"
	"strings"

	"z-novel-ai-api/internal/application/quota"
	"z-novel-ai-api/internal/config"
	"z-novel-ai-api/internal/domain/entity"
	"z-novel-ai-api/internal/domain/repository"
	"z-novel-ai-api/internal/infrastructure/messaging"
	"z-novel-ai-api/internal/interfaces/http/dto"
	"z-novel-ai-api/internal/interfaces/http/middleware"
	"z-novel-ai-api/pkg/errors"
	"z-novel-ai-api/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ChapterHandler 章节处理器
type ChapterHandler struct {
	cfg *config.Config

	chapterRepo repository.ChapterRepository
	projectRepo repository.ProjectRepository
	jobRepo     repository.JobRepository
	producer    *messaging.Producer

	quotaChecker *quota.TokenQuotaChecker
}

// NewChapterHandler 创建章节处理器
func NewChapterHandler(
	cfg *config.Config,
	chapterRepo repository.ChapterRepository,
	projectRepo repository.ProjectRepository,
	jobRepo repository.JobRepository,
	producer *messaging.Producer,
	quotaChecker *quota.TokenQuotaChecker,
) *ChapterHandler {
	return &ChapterHandler{
		cfg:          cfg,
		chapterRepo:  chapterRepo,
		projectRepo:  projectRepo,
		jobRepo:      jobRepo,
		producer:     producer,
		quotaChecker: quotaChecker,
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
	tenantID := middleware.GetTenantIDFromGin(c)
	projectID := dto.BindProjectID(c)

	var req dto.GenerateChapterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequest(c, "invalid request body: "+err.Error())
		return
	}

	provider, model, err := resolveProviderModel(h.cfg, "", pickOptionModel(req.Options))
	if err != nil {
		dto.BadRequest(c, err.Error())
		return
	}

	idempotencyKey := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
	if len(idempotencyKey) > 128 {
		dto.BadRequest(c, "Idempotency-Key too long")
		return
	}
	if idempotencyKey != "" {
		existing, err := h.jobRepo.GetByIdempotencyKey(ctx, idempotencyKey)
		if err != nil {
			logger.Error(ctx, "failed to check idempotency key", err)
			dto.InternalError(c, "failed to create job")
			return
		}
		if existing != nil {
			if existing.ProjectID != projectID || existing.JobType != entity.JobTypeChapterGen {
				dto.Conflict(c, "idempotency key already used")
				return
			}
			dto.Accepted(c, dto.ToJobResponse(existing))
			return
		}
	}

	if h.quotaChecker != nil {
		if _, err := h.quotaChecker.CheckBalance(ctx, tenantID, 1000); err != nil {
			var exceeded quota.TokenBalanceExceededError
			if stderrors.As(err, &exceeded) {
				dto.Error(c, http.StatusTooManyRequests, "token balance insufficient")
				return
			}
			logger.Error(ctx, "quota check failed", err)
			dto.InternalError(c, "quota check failed")
			return
		}
	}

	targetWordCount := req.TargetWordCount
	if targetWordCount <= 0 {
		project, err := h.projectRepo.GetByID(ctx, projectID)
		if err != nil {
			logger.Error(ctx, "failed to load project", err)
			dto.InternalError(c, "failed to load project")
			return
		}
		if project == nil {
			dto.NotFound(c, "project not found")
			return
		}
		if project.Settings != nil && project.Settings.DefaultChapterLength > 0 {
			targetWordCount = project.Settings.DefaultChapterLength
		} else {
			targetWordCount = 2000
		}
	}

	jobID := uuid.NewString()
	inputParams := map[string]any{
		"mode":              "async_generate",
		"project_id":        projectID,
		"volume_id":         strings.TrimSpace(req.VolumeID),
		"title":             strings.TrimSpace(req.Title),
		"outline":           strings.TrimSpace(req.Outline),
		"target_word_count": targetWordCount,
		"story_time_start":  req.StoryTimeStart,
		"notes":             strings.TrimSpace(req.Notes),
		"provider":          provider,
		"model":             model,
	}
	if req.Options != nil {
		if req.Options.Temperature != 0 {
			inputParams["temperature"] = req.Options.Temperature
		}
		if req.Options.MaxRetries > 0 {
			inputParams["max_retries"] = req.Options.MaxRetries
		}
		if req.Options.SkipValidation {
			inputParams["skip_validation"] = true
		}
	}
	inputBytes, _ := json.Marshal(inputParams)

	job := entity.NewGenerationJob(tenantID, projectID, entity.JobTypeChapterGen, inputBytes)
	job.ID = jobID
	if idempotencyKey != "" {
		job.IdempotencyKey = &idempotencyKey
	}
	if err := h.jobRepo.Create(ctx, job); err != nil {
		if idempotencyKey != "" {
			existing, getErr := h.jobRepo.GetByIdempotencyKey(ctx, idempotencyKey)
			if getErr == nil && existing != nil {
				dto.Accepted(c, dto.ToJobResponse(existing))
				return
			}
		}
		logger.Error(ctx, "failed to create generation job", err)
		dto.InternalError(c, "failed to create job")
		return
	}

	seqNum, err := h.chapterRepo.GetNextSeqNum(ctx, projectID, strings.TrimSpace(req.VolumeID))
	if err != nil {
		logger.Error(ctx, "failed to get next seq num", err)
		dto.InternalError(c, "failed to create chapter")
		return
	}

	chapter := entity.NewChapter(projectID, strings.TrimSpace(req.VolumeID), seqNum)
	chapter.Title = strings.TrimSpace(req.Title)
	chapter.Outline = strings.TrimSpace(req.Outline)
	chapter.Notes = strings.TrimSpace(req.Notes)
	chapter.StoryTimeStart = req.StoryTimeStart
	chapter.Status = entity.ChapterStatusGenerating

	if err := h.chapterRepo.Create(ctx, chapter); err != nil {
		logger.Error(ctx, "failed to create chapter", err)
		dto.InternalError(c, "failed to create chapter")
		return
	}

	job.ChapterID = &chapter.ID
	inputParams["chapter_id"] = chapter.ID
	inputParams["chapter_seq_num"] = chapter.SeqNum
	inputBytes, _ = json.Marshal(inputParams)
	job.InputParams = inputBytes
	if err := h.jobRepo.Update(ctx, job); err != nil {
		logger.Error(ctx, "failed to update job with chapter id", err)
		dto.InternalError(c, "failed to create job")
		return
	}

	temp := pickOptionTemperature(req.Options)
	msg := &messaging.GenerationJobMessage{
		JobID:          jobID,
		TenantID:       tenantID,
		ProjectID:      projectID,
		ChapterID:      &chapter.ID,
		JobType:        string(entity.JobTypeChapterGen),
		Priority:       job.Priority,
		IdempotencyKey: job.IdempotencyKey,
		Params: map[string]interface{}{
			"outline":           chapter.Outline,
			"target_word_count": targetWordCount,
			"provider":          provider,
			"model":             model,
		},
	}
	if temp != nil {
		msg.Params["temperature"] = float64(*temp)
	}

	if _, err := h.producer.PublishGenJob(ctx, msg); err != nil {
		logger.Error(ctx, "failed to publish chapter generation job", err)
		dto.InternalError(c, "failed to enqueue job")
		return
	}

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
	tenantID := middleware.GetTenantIDFromGin(c)
	chapterID := dto.BindChapterID(c)

	var req dto.RegenerateChapterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequest(c, "invalid request body: "+err.Error())
		return
	}

	chapter, err := h.chapterRepo.GetByID(ctx, chapterID)
	if err != nil {
		logger.Error(ctx, "failed to get chapter", err)
		dto.InternalError(c, "failed to regenerate chapter")
		return
	}
	if chapter == nil {
		dto.NotFound(c, "chapter not found")
		return
	}

	project, err := h.projectRepo.GetByID(ctx, chapter.ProjectID)
	if err != nil {
		logger.Error(ctx, "failed to load project", err)
		dto.InternalError(c, "failed to regenerate chapter")
		return
	}
	if project == nil {
		dto.NotFound(c, "project not found")
		return
	}

	if h.quotaChecker != nil {
		if _, err := h.quotaChecker.CheckBalance(ctx, tenantID, 1000); err != nil {
			var exceeded quota.TokenBalanceExceededError
			if stderrors.As(err, &exceeded) {
				dto.Error(c, http.StatusTooManyRequests, "token balance insufficient")
				return
			}
			logger.Error(ctx, "quota check failed", err)
			dto.InternalError(c, "quota check failed")
			return
		}
	}

	outline := strings.TrimSpace(req.Outline)
	if outline == "" {
		outline = strings.TrimSpace(chapter.Outline)
	}
	if outline == "" {
		dto.BadRequest(c, "outline is required")
		return
	}

	targetWordCount := req.TargetWordCount
	if targetWordCount <= 0 {
		if project.Settings != nil && project.Settings.DefaultChapterLength > 0 {
			targetWordCount = project.Settings.DefaultChapterLength
		} else {
			targetWordCount = 2000
		}
	}

	provider, model, err := resolveProviderModel(h.cfg, "", pickOptionModel(req.Options))
	if err != nil {
		dto.BadRequest(c, err.Error())
		return
	}

	idempotencyKey := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
	if len(idempotencyKey) > 128 {
		dto.BadRequest(c, "Idempotency-Key too long")
		return
	}
	if idempotencyKey != "" {
		existing, err := h.jobRepo.GetByIdempotencyKey(ctx, idempotencyKey)
		if err != nil {
			logger.Error(ctx, "failed to check idempotency key", err)
			dto.InternalError(c, "failed to create job")
			return
		}
		if existing != nil {
			if existing.ProjectID != chapter.ProjectID || existing.JobType != entity.JobTypeChapterGen || existing.ChapterID == nil || *existing.ChapterID != chapterID {
				dto.Conflict(c, "idempotency key already used")
				return
			}
			dto.Accepted(c, dto.ToJobResponse(existing))
			return
		}
	}

	jobID := uuid.NewString()
	inputParams := map[string]any{
		"mode":              "async_regenerate",
		"project_id":        chapter.ProjectID,
		"chapter_id":        chapter.ID,
		"outline":           outline,
		"target_word_count": targetWordCount,
		"provider":          provider,
		"model":             model,
	}
	if req.Options != nil {
		if req.Options.Temperature != 0 {
			inputParams["temperature"] = req.Options.Temperature
		}
		if req.Options.MaxRetries > 0 {
			inputParams["max_retries"] = req.Options.MaxRetries
		}
		if req.Options.SkipValidation {
			inputParams["skip_validation"] = true
		}
	}
	inputBytes, _ := json.Marshal(inputParams)

	job := entity.NewGenerationJob(tenantID, chapter.ProjectID, entity.JobTypeChapterGen, inputBytes)
	job.ID = jobID
	job.ChapterID = &chapterID
	if idempotencyKey != "" {
		job.IdempotencyKey = &idempotencyKey
	}
	if err := h.jobRepo.Create(ctx, job); err != nil {
		if idempotencyKey != "" {
			existing, getErr := h.jobRepo.GetByIdempotencyKey(ctx, idempotencyKey)
			if getErr == nil && existing != nil {
				dto.Accepted(c, dto.ToJobResponse(existing))
				return
			}
		}
		logger.Error(ctx, "failed to create generation job", err)
		dto.InternalError(c, "failed to create job")
		return
	}

	chapter.Outline = outline
	chapter.Status = entity.ChapterStatusGenerating
	if err := h.chapterRepo.Update(ctx, chapter); err != nil {
		logger.Error(ctx, "failed to update chapter status", err)
		dto.InternalError(c, "failed to regenerate chapter")
		return
	}

	temp := pickOptionTemperature(req.Options)
	msg := &messaging.GenerationJobMessage{
		JobID:          jobID,
		TenantID:       tenantID,
		ProjectID:      chapter.ProjectID,
		ChapterID:      &chapter.ID,
		JobType:        string(entity.JobTypeChapterGen),
		Priority:       job.Priority,
		IdempotencyKey: job.IdempotencyKey,
		Params: map[string]interface{}{
			"outline":           outline,
			"target_word_count": targetWordCount,
			"provider":          provider,
			"model":             model,
		},
	}
	if temp != nil {
		msg.Params["temperature"] = float64(*temp)
	}

	if _, err := h.producer.PublishGenJob(ctx, msg); err != nil {
		logger.Error(ctx, "failed to publish chapter regeneration job", err)
		dto.InternalError(c, "failed to enqueue job")
		return
	}

	dto.Accepted(c, dto.ToJobResponse(job))
}

func pickOptionModel(opt *dto.GenerationOptions) string {
	if opt == nil {
		return ""
	}
	return strings.TrimSpace(opt.Model)
}

func pickOptionTemperature(opt *dto.GenerationOptions) *float32 {
	if opt == nil {
		return nil
	}
	if opt.Temperature == 0 {
		return nil
	}
	f := float32(opt.Temperature)
	return &f
}
