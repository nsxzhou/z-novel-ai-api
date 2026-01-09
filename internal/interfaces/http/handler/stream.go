// Package handler 提供 HTTP 请求处理器
package handler

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"z-novel-ai-api/internal/application/quota"
	"z-novel-ai-api/internal/application/story"
	"z-novel-ai-api/internal/config"
	"z-novel-ai-api/internal/domain/entity"
	"z-novel-ai-api/internal/domain/repository"
	"z-novel-ai-api/internal/interfaces/http/dto"
	"z-novel-ai-api/internal/interfaces/http/middleware"
	"z-novel-ai-api/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// StreamHandler 流式响应处理器
type StreamHandler struct {
	cfg *config.Config

	chapterRepo repository.ChapterRepository
	projectRepo repository.ProjectRepository
	jobRepo     repository.JobRepository

	txMgr     repository.Transactor
	tenantCtx repository.TenantContextManager

	quotaChecker *quota.TokenQuotaChecker
	generator    *story.ChapterGenerator
}

// NewStreamHandler 创建流式响应处理器
func NewStreamHandler(
	cfg *config.Config,
	chapterRepo repository.ChapterRepository,
	projectRepo repository.ProjectRepository,
	jobRepo repository.JobRepository,
	txMgr repository.Transactor,
	tenantCtx repository.TenantContextManager,
	quotaChecker *quota.TokenQuotaChecker,
	generator *story.ChapterGenerator,
) *StreamHandler {
	return &StreamHandler{
		cfg:          cfg,
		chapterRepo:  chapterRepo,
		projectRepo:  projectRepo,
		jobRepo:      jobRepo,
		txMgr:        txMgr,
		tenantCtx:    tenantCtx,
		quotaChecker: quotaChecker,
		generator:    generator,
	}
}

// StreamChapter 流式获取章节内容
// @Summary 流式获取章节内容
// @Description 通过 SSE 流式获取章节生成内容
// @Tags Chapters
// @Accept json
// @Produce text/event-stream
// @Param cid path string true "章节 ID"
// @Success 200 "SSE stream"
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /v1/chapters/{cid}/stream [get]
func (h *StreamHandler) StreamChapter(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID := middleware.GetTenantIDFromGin(c)
	chapterID := dto.BindChapterID(c)

	provider, model, err := resolveProviderModel(h.cfg, strings.TrimSpace(c.Query("provider")), strings.TrimSpace(c.Query("model")))
	if err != nil {
		dto.BadRequest(c, err.Error())
		return
	}

	var temperature *float32
	if s := strings.TrimSpace(c.Query("temperature")); s != "" {
		f, err := strconv.ParseFloat(s, 32)
		if err != nil {
			dto.BadRequest(c, "invalid temperature")
			return
		}
		v := float32(f)
		temperature = &v
	}

	targetWordCount := 0
	if s := strings.TrimSpace(c.Query("target_word_count")); s != "" {
		i, err := strconv.Atoi(s)
		if err != nil {
			dto.BadRequest(c, "invalid target_word_count")
			return
		}
		targetWordCount = i
	}

	var chapter *entity.Chapter
	var project *entity.Project
	if err := withTenantTx(ctx, h.txMgr, h.tenantCtx, tenantID, func(txCtx context.Context) error {
		var loadErr error
		chapter, loadErr = h.chapterRepo.GetByID(txCtx, chapterID)
		if loadErr != nil || chapter == nil {
			return loadErr
		}
		project, loadErr = h.projectRepo.GetByID(txCtx, chapter.ProjectID)
		return loadErr
	}); err != nil {
		logger.Error(ctx, "failed to load chapter for stream", err)
		dto.InternalError(c, "failed to stream chapter")
		return
	}
	if chapter == nil {
		dto.NotFound(c, "chapter not found")
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

	outline := strings.TrimSpace(chapter.Outline)
	if outline == "" {
		dto.BadRequest(c, "chapter outline is empty")
		return
	}

	writingStyle := ""
	pov := ""
	if project.Settings != nil {
		writingStyle = strings.TrimSpace(project.Settings.WritingStyle)
		pov = strings.TrimSpace(project.Settings.POV)
		if temperature == nil && project.Settings.Temperature != 0 {
			t := float32(project.Settings.Temperature)
			temperature = &t
		}
		if targetWordCount <= 0 && project.Settings.DefaultChapterLength > 0 {
			targetWordCount = project.Settings.DefaultChapterLength
		}
	}
	if targetWordCount <= 0 {
		targetWordCount = 2000
	}

	if h.generator == nil {
		dto.InternalError(c, "chapter generator not configured")
		return
	}

	jobID := uuid.NewString()
	now := time.Now()
	inputParams, _ := json.Marshal(map[string]any{
		"mode":              "stream",
		"project_id":        chapter.ProjectID,
		"chapter_id":        chapter.ID,
		"outline":           outline,
		"target_word_count": targetWordCount,
		"provider":          provider,
		"model":             model,
		"temperature":       temperature,
	})
	job := entity.NewGenerationJob(tenantID, chapter.ProjectID, entity.JobTypeChapterGen, inputParams)
	job.ID = jobID
	job.JobType = entity.JobTypeChapterGen
	job.ChapterID = &chapter.ID
	job.Status = entity.JobStatusRunning
	job.StartedAt = &now
	job.Progress = 1

	if err := withTenantTx(ctx, h.txMgr, h.tenantCtx, tenantID, func(txCtx context.Context) error {
		if err := h.jobRepo.Create(txCtx, job); err != nil {
			return err
		}
		chapter.Status = entity.ChapterStatusGenerating
		return h.chapterRepo.Update(txCtx, chapter)
	}); err != nil {
		logger.Error(ctx, "failed to prepare chapter stream job", err)
		dto.InternalError(c, "failed to create job")
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	contentCh := make(chan string, 16)
	doneCh := make(chan *story.ChapterGenerateOutput, 1)
	errCh := make(chan error, 1)

	go func() {
		defer close(contentCh)
		defer close(doneCh)
		defer close(errCh)

		start := time.Now()
		reader, streamErr := h.generator.Stream(ctx, &story.ChapterGenerateInput{
			ProjectTitle:       project.Title,
			ProjectDescription: project.Description,
			ChapterTitle:       chapter.Title,
			ChapterOutline:     outline,
			TargetWordCount:    targetWordCount,
			WritingStyle:       writingStyle,
			POV:                pov,
			Provider:           provider,
			Model:              model,
			Temperature:        temperature,
		})
		if streamErr != nil {
			errCh <- streamErr
			_ = h.markJobFailed(ctx, tenantID, jobID, chapter.ID, streamErr, int(time.Since(start).Milliseconds()))
			return
		}
		defer reader.Close()

		var raw strings.Builder
		var usage *story.LLMUsageMeta

		for {
			msg, recvErr := reader.Recv()
			if stderrors.Is(recvErr, io.EOF) {
				break
			}
			if recvErr != nil {
				errCh <- recvErr
				_ = h.markJobFailed(ctx, tenantID, jobID, chapter.ID, recvErr, int(time.Since(start).Milliseconds()))
				return
			}

			if msg.Content != "" {
				raw.WriteString(msg.Content)
				contentCh <- msg.Content
			}

			if msg.ResponseMeta != nil && msg.ResponseMeta.Usage != nil {
				u := msg.ResponseMeta.Usage
				meta := story.LLMUsageMeta{
					Provider:         provider,
					Model:            model,
					PromptTokens:     u.PromptTokens,
					CompletionTokens: u.CompletionTokens,
					GeneratedAt:      time.Now().UTC(),
				}
				if temperature != nil {
					meta.Temperature = float64(*temperature)
				}
				usage = &meta
			}
		}

		out := &story.ChapterGenerateOutput{
			Content: strings.TrimSpace(raw.String()),
			Meta: story.LLMUsageMeta{
				Provider:    provider,
				Model:       model,
				GeneratedAt: time.Now().UTC(),
			},
		}
		if temperature != nil {
			out.Meta.Temperature = float64(*temperature)
		}
		if usage != nil {
			out.Meta = *usage
		}

		if err := h.markJobCompleted(ctx, tenantID, jobID, chapter.ID, out, int(time.Since(start).Milliseconds())); err != nil {
			errCh <- err
			return
		}

		doneCh <- out
	}()

	index := 0
	c.Stream(func(w io.Writer) bool {
		select {
		case chunk, ok := <-contentCh:
			if !ok {
				return false
			}
			c.SSEvent("content", gin.H{"chunk": chunk, "index": index})
			index++
			return true

		case out, ok := <-doneCh:
			if !ok || out == nil {
				return false
			}
			c.SSEvent("done", gin.H{
				"job_id":     jobID,
				"chapter_id": chapter.ID,
				"word_count": len([]rune(out.Content)),
			})
			return false

		case streamErr, ok := <-errCh:
			if ok && streamErr != nil {
				c.SSEvent("error", gin.H{"message": streamErr.Error()})
			}
			return false

		case <-ctx.Done():
			return false
		}
	})
}

// StreamGenerate 流式生成内容（内部方法）
// 用于实际的生成流程，由 GenerationService 调用
func (h *StreamHandler) StreamGenerate(c *gin.Context, contentChan <-chan string, metaChan <-chan map[string]interface{}, errChan <-chan error) {
	// 设置 SSE 响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	index := 0

	c.Stream(func(w io.Writer) bool {
		select {
		case chunk, ok := <-contentChan:
			if !ok {
				// 内容通道关闭
				return false
			}
			c.SSEvent("content", gin.H{
				"chunk": chunk,
				"index": index,
			})
			index++
			return true

		case meta, ok := <-metaChan:
			if !ok {
				return true // 元数据通道关闭，继续等待内容
			}
			c.SSEvent("metadata", meta)
			return true

		case err, ok := <-errChan:
			if !ok {
				return true // 错误通道关闭
			}
			c.SSEvent("error", gin.H{
				"message": err.Error(),
			})
			return false

		case <-c.Request.Context().Done():
			// 客户端断开
			return false
		}
	})
}

func (h *StreamHandler) markJobFailed(ctx context.Context, tenantID, jobID, chapterID string, err error, durationMs int) error {
	return withTenantTx(ctx, h.txMgr, h.tenantCtx, tenantID, func(txCtx context.Context) error {
		job, getErr := h.jobRepo.GetByID(txCtx, jobID)
		if getErr != nil || job == nil {
			return getErr
		}
		job.Status = entity.JobStatusFailed
		job.ErrorMessage = err.Error()
		now := time.Now()
		job.CompletedAt = &now
		job.DurationMs = durationMs
		job.Progress = 100
		if updateErr := h.jobRepo.Update(txCtx, job); updateErr != nil {
			return updateErr
		}

		if strings.TrimSpace(chapterID) == "" {
			return nil
		}
		ch, getErr := h.chapterRepo.GetByID(txCtx, chapterID)
		if getErr != nil {
			logger.Warn(txCtx, "failed to load chapter for stream failure", "error", getErr.Error())
			return nil
		}
		if ch == nil {
			return nil
		}
		if ch.Status == entity.ChapterStatusGenerating {
			ch.Status = entity.ChapterStatusDraft
			if err := h.chapterRepo.Update(txCtx, ch); err != nil {
				logger.Warn(txCtx, "failed to update chapter status for stream failure", "error", err.Error())
			}
		}
		return nil
	})
}

func (h *StreamHandler) markJobCompleted(ctx context.Context, tenantID, jobID, chapterID string, out *story.ChapterGenerateOutput, durationMs int) error {
	if out == nil {
		return fmt.Errorf("chapter output is nil")
	}
	return withTenantTx(ctx, h.txMgr, h.tenantCtx, tenantID, func(txCtx context.Context) error {
		job, err := h.jobRepo.GetByID(txCtx, jobID)
		if err != nil || job == nil {
			return err
		}

		resultBytes, _ := json.Marshal(map[string]any{
			"chapter_id": chapterID,
			"word_count": len([]rune(out.Content)),
		})
		job.OutputResult = resultBytes
		job.Status = entity.JobStatusCompleted
		now := time.Now()
		job.CompletedAt = &now
		job.DurationMs = durationMs
		job.Progress = 100
		job.SetLLMMetrics(out.Meta.Provider, out.Meta.Model, out.Meta.PromptTokens, out.Meta.CompletionTokens)
		if err := h.jobRepo.Update(txCtx, job); err != nil {
			return err
		}

		ch, err := h.chapterRepo.GetByID(txCtx, chapterID)
		if err != nil {
			return err
		}
		if ch == nil {
			return fmt.Errorf("chapter not found: %s", chapterID)
		}

		ch.SetContent(out.Content)
		ch.Status = entity.ChapterStatusCompleted
		ch.GenerationMetadata = &entity.GenerationMetadata{
			Model:            out.Meta.Model,
			Provider:         out.Meta.Provider,
			PromptTokens:     out.Meta.PromptTokens,
			CompletionTokens: out.Meta.CompletionTokens,
			Temperature:      out.Meta.Temperature,
			GeneratedAt:      out.Meta.GeneratedAt.Format(time.RFC3339),
		}

		if err := h.chapterRepo.Update(txCtx, ch); err != nil {
			return err
		}

		stats, err := h.projectRepo.GetStats(txCtx, ch.ProjectID)
		if err != nil || stats == nil {
			logger.Warn(txCtx, "failed to refresh project word count after chapter generation", "error", err)
			return nil
		}
		if err := h.projectRepo.UpdateWordCount(txCtx, ch.ProjectID, int(stats.TotalWordCount)); err != nil {
			logger.Warn(txCtx, "failed to update project word count after chapter generation", "error", err.Error())
		}
		return nil
	})
}
