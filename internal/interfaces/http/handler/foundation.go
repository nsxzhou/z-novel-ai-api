// Package handler 提供 HTTP 请求处理器
package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"z-novel-ai-api/internal/application/quota"
	storyfoundation "z-novel-ai-api/internal/application/story/foundation"
	storymodel "z-novel-ai-api/internal/application/story/model"
	"z-novel-ai-api/internal/config"
	"z-novel-ai-api/internal/domain/entity"
	"z-novel-ai-api/internal/domain/repository"
	"z-novel-ai-api/internal/infrastructure/messaging"
	"z-novel-ai-api/internal/interfaces/http/dto"
	"z-novel-ai-api/internal/interfaces/http/middleware"
	wfmodel "z-novel-ai-api/internal/workflow/model"
	"z-novel-ai-api/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// FoundationHandler 设定集（世界观/角色/大纲）生成与落库
type FoundationHandler struct {
	cfg *config.Config

	txMgr     repository.Transactor
	tenantCtx repository.TenantContextManager

	tenantRepo  repository.TenantRepository
	projectRepo repository.ProjectRepository
	jobRepo     repository.JobRepository

	producer *messaging.Producer

	quotaChecker *quota.TokenQuotaChecker
	generator    *storyfoundation.FoundationGenerator
	applier      *storyfoundation.FoundationApplier
}

type applyPlanResolveErrorCode string

const (
	applyPlanResolveErrorInvalidRequest   applyPlanResolveErrorCode = "invalid_request"
	applyPlanResolveErrorJobNotFound      applyPlanResolveErrorCode = "job_not_found"
	applyPlanResolveErrorJobProjectMismat applyPlanResolveErrorCode = "job_project_mismatch"
	applyPlanResolveErrorJobNotCompleted  applyPlanResolveErrorCode = "job_not_completed"
)

type applyPlanResolveError struct {
	code applyPlanResolveErrorCode
	msg  string
}

func (e applyPlanResolveError) Error() string {
	return e.msg
}

func NewFoundationHandler(
	cfg *config.Config,
	txMgr repository.Transactor,
	tenantCtx repository.TenantContextManager,
	tenantRepo repository.TenantRepository,
	projectRepo repository.ProjectRepository,
	jobRepo repository.JobRepository,
	producer *messaging.Producer,
	quotaChecker *quota.TokenQuotaChecker,
	generator *storyfoundation.FoundationGenerator,
	applier *storyfoundation.FoundationApplier,
) *FoundationHandler {
	return &FoundationHandler{
		cfg:          cfg,
		txMgr:        txMgr,
		tenantCtx:    tenantCtx,
		tenantRepo:   tenantRepo,
		projectRepo:  projectRepo,
		jobRepo:      jobRepo,
		producer:     producer,
		quotaChecker: quotaChecker,
		generator:    generator,
		applier:      applier,
	}
}

// PreviewFoundation 同步生成设定集 Plan（不落库）
// @Summary 同步生成设定集 Plan（预览）
// @Description 同步调用 LLM 生成 FoundationPlan，并写入 generation_jobs 记录 token 使用量
// @Tags Foundation
// @Accept json
// @Produce json
// @Param pid path string true "项目 ID"
// @Param body body dto.FoundationGenerateRequest true "生成请求"
// @Success 200 {object} dto.Response[dto.FoundationPreviewResponse]
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 429 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /v1/projects/{pid}/foundation/preview [post]
func (h *FoundationHandler) PreviewFoundation(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID := middleware.GetTenantIDFromGin(c)
	projectID := dto.BindProjectID(c)

	var req dto.FoundationGenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequest(c, "invalid request body: "+err.Error())
		return
	}

	provider, model, err := resolveProviderModel(h.cfg, req.Provider, req.Model)
	if err != nil {
		dto.BadRequest(c, err.Error())
		return
	}

	jobID := uuid.NewString()
	job := entity.NewGenerationJob(tenantID, projectID, entity.JobTypeFoundationGen, nil)
	job.ID = jobID
	job.JobType = entity.JobTypeFoundationGen
	job.Status = entity.JobStatusRunning
	now := time.Now()
	job.StartedAt = &now

	inputParams, _ := json.Marshal(map[string]any{
		"mode":        "preview",
		"project_id":  projectID,
		"prompt":      req.Prompt,
		"attachments": req.Attachments,
		"provider":    provider,
		"model":       model,
		"temperature": req.Temperature,
		"max_tokens":  req.MaxTokens,
	})
	job.InputParams = inputParams

	var tenant *entity.Tenant
	var project *entity.Project
	if err := withTenantTx(ctx, h.txMgr, h.tenantCtx, tenantID, func(txCtx context.Context) error {
		var loadErr error
		tenant, loadErr = h.tenantRepo.GetByID(txCtx, tenantID)
		if loadErr != nil || tenant == nil {
			return loadErr
		}

		if quotaErr := precheckQuota(txCtx, h.quotaChecker, tenant); quotaErr != nil {
			return quotaErr
		}

		project, loadErr = h.projectRepo.GetByID(txCtx, projectID)
		if loadErr != nil || project == nil {
			return loadErr
		}

		return h.jobRepo.Create(txCtx, job)
	}); err != nil {
		var exceeded quota.TokenBalanceExceededError
		if errors.As(err, &exceeded) {
			dto.Error(c, http.StatusTooManyRequests, "token balance insufficient")
			return
		}
		logger.Error(ctx, "failed to prepare foundation preview", err)
		dto.InternalError(c, "failed to prepare preview")
		return
	}
	if tenant == nil {
		dto.NotFound(c, "tenant not found")
		return
	}
	if project == nil {
		dto.NotFound(c, "project not found")
		return
	}

	start := time.Now()
	out, genErr := h.generator.Generate(ctx, req.ToStoryInput(project.Title, project.Description, provider, model))
	durationMs := int(time.Since(start).Milliseconds())

	if genErr != nil {
		_ = h.markJobFailed(ctx, tenantID, jobID, genErr, durationMs)
		logger.Error(ctx, "foundation generation failed", genErr)
		dto.InternalError(c, "foundation generation failed")
		return
	}

	if err := storyfoundation.ValidateFoundationPlan(out.Plan); err != nil {
		_ = h.markJobFailed(ctx, tenantID, jobID, err, durationMs)
		h.writePlanValidationError(c, err)
		return
	}

	if err := h.markJobCompleted(ctx, tenantID, jobID, out, durationMs); err != nil {
		logger.Error(ctx, "failed to persist job result", err)
		dto.InternalError(c, "failed to persist job result")
		return
	}

	resp := &dto.FoundationPreviewResponse{
		JobID: jobID,
		Plan:  out.Plan,
		Usage: &dto.FoundationUsageResponse{
			Provider:         out.Meta.Provider,
			Model:            out.Meta.Model,
			PromptTokens:     out.Meta.PromptTokens,
			CompletionTokens: out.Meta.CompletionTokens,
			Temperature:      out.Meta.Temperature,
			DurationMs:       durationMs,
			GeneratedAt:      out.Meta.GeneratedAt.Format(time.RFC3339),
		},
	}
	dto.Success(c, resp)
}

// StreamFoundation SSE 流式生成设定集 Plan（不落库）
// @Summary SSE 流式生成设定集 Plan
// @Description 通过 SSE 事件流输出增量 content，结束时输出 done（包含 plan 与 job_id）
// @Tags Foundation
// @Accept json
// @Produce text/event-stream
// @Param pid path string true "项目 ID"
// @Success 200 "SSE stream"
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 429 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /v1/projects/{pid}/foundation/stream [get]
func (h *FoundationHandler) StreamFoundation(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID := middleware.GetTenantIDFromGin(c)
	projectID := dto.BindProjectID(c)

	req, err := h.bindStreamRequest(c)
	if err != nil {
		dto.BadRequest(c, err.Error())
		return
	}

	provider, model, err := resolveProviderModel(h.cfg, req.Provider, req.Model)
	if err != nil {
		dto.BadRequest(c, err.Error())
		return
	}

	var tenant *entity.Tenant
	if err := withTenantTx(ctx, h.txMgr, h.tenantCtx, tenantID, func(txCtx context.Context) error {
		var loadErr error
		tenant, loadErr = h.tenantRepo.GetByID(txCtx, tenantID)
		return loadErr
	}); err != nil {
		logger.Error(ctx, "failed to load tenant", err)
		dto.InternalError(c, "failed to load tenant")
		return
	}
	if tenant == nil {
		dto.NotFound(c, "tenant not found")
		return
	}

	if err := precheckQuota(ctx, h.quotaChecker, tenant); err != nil {
		h.writeQuotaError(c, err)
		return
	}

	var project *entity.Project
	if err := withTenantTx(ctx, h.txMgr, h.tenantCtx, tenantID, func(txCtx context.Context) error {
		var loadErr error
		project, loadErr = h.projectRepo.GetByID(txCtx, projectID)
		return loadErr
	}); err != nil {
		logger.Error(ctx, "failed to load project", err)
		dto.InternalError(c, "failed to load project")
		return
	}
	if project == nil {
		dto.NotFound(c, "project not found")
		return
	}

	jobID := uuid.NewString()
	job := entity.NewGenerationJob(tenantID, projectID, entity.JobTypeFoundationGen, nil)
	job.ID = jobID
	job.JobType = entity.JobTypeFoundationGen
	job.Status = entity.JobStatusRunning
	now := time.Now()
	job.StartedAt = &now
	inputParams, _ := json.Marshal(map[string]any{
		"mode":        "stream",
		"project_id":  projectID,
		"prompt":      req.Prompt,
		"attachments": req.Attachments,
		"provider":    provider,
		"model":       model,
		"temperature": req.Temperature,
		"max_tokens":  req.MaxTokens,
	})
	job.InputParams = inputParams

	if err := withTenantTx(ctx, h.txMgr, h.tenantCtx, tenantID, func(txCtx context.Context) error {
		return h.jobRepo.Create(txCtx, job)
	}); err != nil {
		logger.Error(ctx, "failed to create generation job", err)
		dto.InternalError(c, "failed to create job")
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	contentCh := make(chan string, 16)
	doneCh := make(chan *storyfoundation.FoundationGenerateOutput, 1)
	errCh := make(chan error, 1)

	go func() {
		defer close(contentCh)
		defer close(doneCh)
		defer close(errCh)

		start := time.Now()
		reader, streamErr := h.generator.Stream(ctx, req.ToStoryInput(project.Title, project.Description, provider, model))
		if streamErr != nil {
			errCh <- streamErr
			_ = h.markJobFailed(ctx, tenantID, jobID, streamErr, int(time.Since(start).Milliseconds()))
			return
		}
		defer reader.Close()

		var raw strings.Builder
		var usage *wfmodel.LLMUsageMeta

		for {
			msg, recvErr := reader.Recv()
			if errors.Is(recvErr, io.EOF) {
				break
			}
			if recvErr != nil {
				errCh <- recvErr
				_ = h.markJobFailed(ctx, tenantID, jobID, recvErr, int(time.Since(start).Milliseconds()))
				return
			}

			if msg.Content != "" {
				raw.WriteString(msg.Content)
				contentCh <- msg.Content
			}

			if msg.ResponseMeta != nil && msg.ResponseMeta.Usage != nil {
				u := msg.ResponseMeta.Usage
				meta := wfmodel.LLMUsageMeta{
					Provider:         provider,
					Model:            model,
					PromptTokens:     u.PromptTokens,
					CompletionTokens: u.CompletionTokens,
					GeneratedAt:      time.Now().UTC(),
				}
				if req.Temperature != nil {
					meta.Temperature = float64(*req.Temperature)
				}
				usage = &meta
			}
		}

		plan, jsonText, parseErr := storyfoundation.ParseFoundationPlan(raw.String())
		if parseErr != nil {
			errCh <- parseErr
			_ = h.markJobFailed(ctx, tenantID, jobID, parseErr, int(time.Since(start).Milliseconds()))
			return
		}

		if err := storyfoundation.ValidateFoundationPlan(plan); err != nil {
			errCh <- err
			_ = h.markJobFailed(ctx, tenantID, jobID, err, int(time.Since(start).Milliseconds()))
			return
		}

		out := &storyfoundation.FoundationGenerateOutput{
			Plan: plan,
			Raw:  jsonText,
			Meta: wfmodel.LLMUsageMeta{
				Provider:    provider,
				Model:       model,
				GeneratedAt: time.Now().UTC(),
			},
		}
		if req.Temperature != nil {
			out.Meta.Temperature = float64(*req.Temperature)
		}
		if usage != nil {
			out.Meta = *usage
		}

		if err := h.markJobCompleted(ctx, tenantID, jobID, out, int(time.Since(start).Milliseconds())); err != nil {
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
			if !ok {
				return false
			}
			c.SSEvent("done", gin.H{
				"job_id": jobID,
				"plan":   out.Plan,
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

// GenerateFoundation 创建异步 generation_job 并投递 Redis Stream
// @Summary 创建设定集异步生成任务
// @Description 创建 generation_jobs 记录并发布 foundation_gen 消息，返回 job_id
// @Tags Foundation
// @Accept json
// @Produce json
// @Param pid path string true "项目 ID"
// @Param body body dto.FoundationGenerateRequest true "生成请求"
// @Success 202 {object} dto.Response[dto.JobResponse]
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 409 {object} dto.ErrorResponse
// @Failure 429 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /v1/projects/{pid}/foundation/generate [post]
func (h *FoundationHandler) GenerateFoundation(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID := middleware.GetTenantIDFromGin(c)
	projectID := dto.BindProjectID(c)

	var req dto.FoundationGenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequest(c, "invalid request body: "+err.Error())
		return
	}

	provider, model, err := resolveProviderModel(h.cfg, req.Provider, req.Model)
	if err != nil {
		dto.BadRequest(c, err.Error())
		return
	}

	tenant, err := h.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		logger.Error(ctx, "failed to load tenant", err)
		dto.InternalError(c, "failed to load tenant")
		return
	}
	if tenant == nil {
		dto.NotFound(c, "tenant not found")
		return
	}

	if err := precheckQuota(ctx, h.quotaChecker, tenant); err != nil {
		h.writeQuotaError(c, err)
		return
	}

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
			if existing.ProjectID != projectID || existing.JobType != entity.JobTypeFoundationGen {
				dto.Conflict(c, "idempotency key already used")
				return
			}
			dto.Accepted(c, dto.ToJobResponse(existing))
			return
		}
	}

	jobID := uuid.NewString()
	inputParams, _ := json.Marshal(map[string]any{
		"mode":        "async",
		"project_id":  projectID,
		"project":     map[string]any{"title": project.Title, "description": project.Description},
		"prompt":      req.Prompt,
		"attachments": req.Attachments,
		"provider":    provider,
		"model":       model,
		"temperature": req.Temperature,
		"max_tokens":  req.MaxTokens,
	})

	job := entity.NewGenerationJob(tenantID, projectID, entity.JobTypeFoundationGen, inputParams)
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

	msg := &messaging.GenerationJobMessage{
		JobID:          jobID,
		TenantID:       tenantID,
		ProjectID:      projectID,
		JobType:        string(entity.JobTypeFoundationGen),
		Priority:       job.Priority,
		IdempotencyKey: job.IdempotencyKey,
		Params: map[string]interface{}{
			"prompt":      req.Prompt,
			"attachments": req.Attachments,
			"provider":    provider,
			"model":       model,
			"temperature": req.Temperature,
			"max_tokens":  req.MaxTokens,
		},
	}

	if _, err := h.producer.PublishFoundationJob(ctx, msg); err != nil {
		logger.Error(ctx, "failed to publish foundation job", err)
		dto.InternalError(c, "failed to enqueue job")
		return
	}

	dto.Accepted(c, dto.ToJobResponse(job))
}

// ApplyFoundation 应用 Plan 落库（单事务，幂等）
// @Summary 应用设定集 Plan（落库）
// @Description 将 FoundationPlan（或 job_id 对应结果）映射并写入 Project/Entity/Relation/Volume/Chapter
// @Tags Foundation
// @Accept json
// @Produce json
// @Param pid path string true "项目 ID"
// @Param body body dto.FoundationApplyRequest true "应用请求"
// @Success 200 {object} dto.Response[dto.FoundationApplyResponse]
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 409 {object} dto.ErrorResponse
// @Failure 422 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /v1/projects/{pid}/foundation/apply [post]
func (h *FoundationHandler) ApplyFoundation(c *gin.Context) {
	ctx := c.Request.Context()
	projectID := dto.BindProjectID(c)

	var req dto.FoundationApplyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequest(c, "invalid request body: "+err.Error())
		return
	}

	plan, err := h.resolveApplyPlan(ctx, projectID, &req)
	if err != nil {
		h.writeApplyResolveError(c, err)
		return
	}

	if err := storyfoundation.ValidateFoundationPlan(plan); err != nil {
		h.writePlanValidationError(c, err)
		return
	}

	if h.applier == nil {
		dto.InternalError(c, "foundation applier not configured")
		return
	}

	result, err := h.applier.Apply(ctx, projectID, plan)
	if err != nil {
		logger.Error(ctx, "failed to apply foundation plan", err)
		dto.InternalError(c, "failed to apply foundation plan")
		return
	}

	dto.Success(c, &dto.FoundationApplyResponse{
		ProjectID: projectID,
		Result:    result,
	})
}

func (h *FoundationHandler) writeQuotaError(c *gin.Context, err error) {
	var exceeded quota.TokenBalanceExceededError
	if errors.As(err, &exceeded) {
		dto.Error(c, http.StatusTooManyRequests, "token balance insufficient")
		return
	}
	dto.InternalError(c, "quota check failed")
}

func (h *FoundationHandler) writePlanValidationError(c *gin.Context, err error) {
	var ve storyfoundation.FoundationPlanValidationError
	if errors.As(err, &ve) {
		dto.UnprocessableEntity(c, "invalid foundation plan", &dto.ErrorDetail{
			ErrorCode: "foundation_plan_invalid",
			Details:   strings.Join(ve.Issues, "; "),
		})
		return
	}
	dto.UnprocessableEntity(c, "invalid foundation plan", &dto.ErrorDetail{
		ErrorCode: "foundation_plan_invalid",
		Details:   err.Error(),
	})
}

func (h *FoundationHandler) bindStreamRequest(c *gin.Context) (*dto.FoundationGenerateRequest, error) {
	if c.Request.Method == http.MethodPost {
		var req dto.FoundationGenerateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			return nil, fmt.Errorf("invalid request body: %w", err)
		}
		return &req, nil
	}

	// GET: 兼容 EventSource 场景（仅 query 参数，attachments 不建议走 query）
	prompt := strings.TrimSpace(c.Query("prompt"))
	if prompt == "" {
		return nil, fmt.Errorf("missing prompt")
	}

	req := &dto.FoundationGenerateRequest{
		Prompt:   prompt,
		Provider: strings.TrimSpace(c.Query("provider")),
		Model:    strings.TrimSpace(c.Query("model")),
	}
	return req, nil
}

func (h *FoundationHandler) markJobFailed(ctx context.Context, tenantID, jobID string, err error, durationMs int) error {
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
		return h.jobRepo.Update(txCtx, job)
	})
}

func (h *FoundationHandler) markJobCompleted(ctx context.Context, tenantID, jobID string, out *storyfoundation.FoundationGenerateOutput, durationMs int) error {
	return withTenantTx(ctx, h.txMgr, h.tenantCtx, tenantID, func(txCtx context.Context) error {
		job, err := h.jobRepo.GetByID(txCtx, jobID)
		if err != nil || job == nil {
			return err
		}
		resultBytes, _ := json.Marshal(out.Plan)
		job.OutputResult = resultBytes
		job.Status = entity.JobStatusCompleted
		now := time.Now()
		job.CompletedAt = &now
		job.DurationMs = durationMs
		job.SetLLMMetrics(out.Meta.Provider, out.Meta.Model, out.Meta.PromptTokens, out.Meta.CompletionTokens)
		return h.jobRepo.Update(txCtx, job)
	})
}

func (h *FoundationHandler) resolveApplyPlan(ctx context.Context, projectID string, req *dto.FoundationApplyRequest) (*storymodel.FoundationPlan, error) {
	if req == nil {
		return nil, applyPlanResolveError{code: applyPlanResolveErrorInvalidRequest, msg: "empty request"}
	}
	if strings.TrimSpace(req.JobID) == "" && req.Plan == nil {
		return nil, applyPlanResolveError{code: applyPlanResolveErrorInvalidRequest, msg: "job_id or plan required"}
	}
	if strings.TrimSpace(req.JobID) != "" {
		job, err := h.jobRepo.GetByID(ctx, strings.TrimSpace(req.JobID))
		if err != nil {
			return nil, err
		}
		if job == nil {
			return nil, applyPlanResolveError{code: applyPlanResolveErrorJobNotFound, msg: "job not found"}
		}
		if job.ProjectID != projectID {
			return nil, applyPlanResolveError{code: applyPlanResolveErrorJobProjectMismat, msg: "job does not belong to project"}
		}
		if job.Status != entity.JobStatusCompleted {
			return nil, applyPlanResolveError{code: applyPlanResolveErrorJobNotCompleted, msg: "job not completed"}
		}

		var plan storymodel.FoundationPlan
		if err := json.Unmarshal(job.OutputResult, &plan); err != nil {
			return nil, fmt.Errorf("failed to parse job result plan: %w", err)
		}
		return &plan, nil
	}
	return req.Plan, nil
}

func (h *FoundationHandler) writeApplyResolveError(c *gin.Context, err error) {
	var ae applyPlanResolveError
	if errors.As(err, &ae) {
		switch ae.code {
		case applyPlanResolveErrorInvalidRequest:
			dto.BadRequest(c, ae.Error())
		case applyPlanResolveErrorJobNotFound:
			dto.NotFound(c, ae.Error())
		case applyPlanResolveErrorJobProjectMismat, applyPlanResolveErrorJobNotCompleted:
			dto.Conflict(c, ae.Error())
		default:
			dto.InternalError(c, ae.Error())
		}
		return
	}
	dto.InternalError(c, err.Error())
}
