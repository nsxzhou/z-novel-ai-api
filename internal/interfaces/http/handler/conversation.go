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

type ConversationHandler struct {
	cfg *config.Config

	txMgr     repository.Transactor
	tenantCtx repository.TenantContextManager

	tenantRepo  repository.TenantRepository
	projectRepo repository.ProjectRepository
	jobRepo     repository.JobRepository

	sessionRepo  repository.ConversationSessionRepository
	turnRepo     repository.ConversationTurnRepository
	artifactRepo repository.ArtifactRepository

	quotaChecker *quota.TokenQuotaChecker
	generator    *story.ArtifactGenerator
}

func NewConversationHandler(
	cfg *config.Config,
	txMgr repository.Transactor,
	tenantCtx repository.TenantContextManager,
	tenantRepo repository.TenantRepository,
	projectRepo repository.ProjectRepository,
	jobRepo repository.JobRepository,
	sessionRepo repository.ConversationSessionRepository,
	turnRepo repository.ConversationTurnRepository,
	artifactRepo repository.ArtifactRepository,
	quotaChecker *quota.TokenQuotaChecker,
	generator *story.ArtifactGenerator,
) *ConversationHandler {
	return &ConversationHandler{
		cfg:          cfg,
		txMgr:        txMgr,
		tenantCtx:    tenantCtx,
		tenantRepo:   tenantRepo,
		projectRepo:  projectRepo,
		jobRepo:      jobRepo,
		sessionRepo:  sessionRepo,
		turnRepo:     turnRepo,
		artifactRepo: artifactRepo,
		quotaChecker: quotaChecker,
		generator:    generator,
	}
}

// CreateSession 创建会话
// @Summary 创建长期会话
// @Description 在指定项目下创建一个长期会话（支持 task 切换）
// @Tags Conversations
// @Accept json
// @Produce json
// @Param pid path string true "项目 ID"
// @Param body body dto.CreateSessionRequest false "创建会话请求"
// @Success 201 {object} dto.Response[dto.SessionResponse]
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /v1/projects/{pid}/sessions [post]
func (h *ConversationHandler) CreateSession(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID := middleware.GetTenantIDFromGin(c)
	projectID := dto.BindProjectID(c)

	var req dto.CreateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		dto.BadRequest(c, "invalid request body: "+err.Error())
		return
	}

	task, err := normalizeConversationTask(req.Task)
	if err != nil {
		dto.BadRequest(c, err.Error())
		return
	}

	var created *entity.ConversationSession
	if err := withTenantTx(ctx, h.txMgr, h.tenantCtx, tenantID, func(txCtx context.Context) error {
		project, getErr := h.projectRepo.GetByID(txCtx, projectID)
		if getErr != nil {
			return getErr
		}
		if project == nil {
			return errNotFound("project not found")
		}

		created = entity.NewConversationSession(tenantID, projectID, task)
		return h.sessionRepo.Create(txCtx, created)
	}); err != nil {
		if isNotFound(err) {
			dto.NotFound(c, err.Error())
			return
		}
		logger.Error(ctx, "failed to create session", err)
		dto.InternalError(c, "failed to create session")
		return
	}

	dto.Created(c, dto.ToSessionResponse(created))
}

// GetSession 获取会话详情
// @Summary 获取会话详情
// @Tags Conversations
// @Accept json
// @Produce json
// @Param pid path string true "项目 ID"
// @Param sid path string true "会话 ID"
// @Success 200 {object} dto.Response[dto.SessionResponse]
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /v1/projects/{pid}/sessions/{sid} [get]
func (h *ConversationHandler) GetSession(c *gin.Context) {
	ctx := c.Request.Context()
	projectID := dto.BindProjectID(c)
	sessionID := dto.BindSessionID(c)

	session, err := h.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		logger.Error(ctx, "failed to get session", err)
		dto.InternalError(c, "failed to get session")
		return
	}
	if session == nil || session.ProjectID != projectID {
		dto.NotFound(c, "session not found")
		return
	}

	dto.Success(c, dto.ToSessionResponse(session))
}

// ListTurns 获取会话轮次列表
// @Summary 获取会话轮次列表
// @Tags Conversations
// @Accept json
// @Produce json
// @Param pid path string true "项目 ID"
// @Param sid path string true "会话 ID"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页条数" default(20)
// @Success 200 {object} dto.Response[dto.TurnListResponse]
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /v1/projects/{pid}/sessions/{sid}/turns [get]
func (h *ConversationHandler) ListTurns(c *gin.Context) {
	ctx := c.Request.Context()
	projectID := dto.BindProjectID(c)
	sessionID := dto.BindSessionID(c)

	session, err := h.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		logger.Error(ctx, "failed to get session", err)
		dto.InternalError(c, "failed to list turns")
		return
	}
	if session == nil || session.ProjectID != projectID {
		dto.NotFound(c, "session not found")
		return
	}

	pageReq := dto.BindPage(c)
	result, err := h.turnRepo.ListBySession(ctx, sessionID, repository.NewPagination(pageReq.Page, pageReq.PageSize))
	if err != nil {
		logger.Error(ctx, "failed to list conversation turns", err)
		dto.InternalError(c, "failed to list turns")
		return
	}

	turns := make([]*dto.TurnResponse, 0, len(result.Items))
	for i := range result.Items {
		turns = append(turns, dto.ToTurnResponse(result.Items[i]))
	}
	dto.SuccessWithPage(c, &dto.TurnListResponse{Turns: turns}, dto.NewPageMeta(pageReq.Page, pageReq.PageSize, int(result.Total)))
}

// SendMessage 发送消息并生成构件新版本
// @Summary 发送消息并生成构件新版本
// @Tags Conversations
// @Accept json
// @Produce json
// @Param pid path string true "项目 ID"
// @Param sid path string true "会话 ID"
// @Param body body dto.SendMessageRequest true "发送消息请求"
// @Success 200 {object} dto.Response[dto.SendMessageResponse]
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 429 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /v1/projects/{pid}/sessions/{sid}/messages [post]
func (h *ConversationHandler) SendMessage(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID := middleware.GetTenantIDFromGin(c)
	userID := middleware.GetUserIDFromGin(c)
	projectID := dto.BindProjectID(c)
	sessionID := dto.BindSessionID(c)

	var req dto.SendMessageRequest
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
	userTurnID := uuid.NewString()
	assistantTurnID := uuid.NewString()
	now := time.Now()

	var tenant *entity.Tenant
	var project *entity.Project
	var session *entity.ConversationSession

	var task entity.ConversationTask
	var artifactType entity.ArtifactType

	var currentWorldview json.RawMessage
	var currentCharacters json.RawMessage
	var currentOutline json.RawMessage
	var currentArtifact json.RawMessage

	requestID := c.GetString("request_id")
	traceID := c.GetString("trace_id")

	if err := withTenantTx(ctx, h.txMgr, h.tenantCtx, tenantID, func(txCtx context.Context) error {
		var loadErr error
		tenant, loadErr = h.tenantRepo.GetByID(txCtx, tenantID)
		if loadErr != nil {
			return loadErr
		}
		if tenant == nil {
			return errNotFound("tenant not found")
		}
		if quotaErr := precheckQuota(txCtx, h.quotaChecker, tenant); quotaErr != nil {
			return quotaErr
		}

		project, loadErr = h.projectRepo.GetByID(txCtx, projectID)
		if loadErr != nil {
			return loadErr
		}
		if project == nil {
			return errNotFound("project not found")
		}

		session, loadErr = h.sessionRepo.GetByIDForUpdate(txCtx, sessionID)
		if loadErr != nil {
			return loadErr
		}
		if session == nil || session.ProjectID != projectID {
			return errNotFound("session not found")
		}

		if strings.TrimSpace(req.Task) != "" {
			normalizedTask, taskErr := normalizeConversationTask(req.Task)
			if taskErr != nil {
				return taskErr
			}
			if session.CurrentTask != normalizedTask {
				session.CurrentTask = normalizedTask
				if err := h.sessionRepo.Update(txCtx, session); err != nil {
					return err
				}
			}
		}

		task = session.CurrentTask
		artifactType, loadErr = entity.TaskToArtifactType(task)
		if loadErr != nil {
			return loadErr
		}

		userMeta, _ := json.Marshal(map[string]any{
			"attachments": req.Attachments,
			"request_id":  requestID,
			"trace_id":    traceID,
		})
		userTurn := entity.NewConversationTurn(sessionID, entity.RoleUser, task, strings.TrimSpace(req.Prompt), userMeta)
		userTurn.ID = userTurnID
		if err := h.turnRepo.Create(txCtx, userTurn); err != nil {
			return err
		}

		inputParams, _ := json.Marshal(map[string]any{
			"mode":        "conversation",
			"project_id":  projectID,
			"session_id":  sessionID,
			"task":        task,
			"prompt":      strings.TrimSpace(req.Prompt),
			"attachments": req.Attachments,
			"provider":    provider,
			"model":       model,
			"temperature": req.Temperature,
			"max_tokens":  req.MaxTokens,
			"request_id":  requestID,
			"trace_id":    traceID,
		})
		job := entity.NewGenerationJob(tenantID, projectID, entity.JobTypeArtifactGen, inputParams)
		job.ID = jobID
		job.Status = entity.JobStatusRunning
		job.StartedAt = &now
		if err := h.jobRepo.Create(txCtx, job); err != nil {
			return err
		}

		arts, loadErr := h.artifactRepo.ListArtifactsByProject(txCtx, projectID)
		if loadErr != nil {
			return loadErr
		}

		typeKeyByArtifactType := make(map[entity.ArtifactType]*entity.ProjectArtifact, len(arts))
		for i := range arts {
			a := arts[i]
			typeKeyByArtifactType[a.Type] = a
		}

		loadActive := func(t entity.ArtifactType) (json.RawMessage, error) {
			a := typeKeyByArtifactType[t]
			if a == nil || a.ActiveVersionID == nil || strings.TrimSpace(*a.ActiveVersionID) == "" {
				return nil, nil
			}
			v, err := h.artifactRepo.GetVersionByID(txCtx, *a.ActiveVersionID)
			if err != nil {
				return nil, err
			}
			if v == nil {
				return nil, nil
			}
			return v.Content, nil
		}

		currentWorldview, loadErr = loadActive(entity.ArtifactTypeWorldview)
		if loadErr != nil {
			return loadErr
		}
		currentCharacters, loadErr = loadActive(entity.ArtifactTypeCharacters)
		if loadErr != nil {
			return loadErr
		}
		currentOutline, loadErr = loadActive(entity.ArtifactTypeOutline)
		if loadErr != nil {
			return loadErr
		}
		currentArtifact, loadErr = loadActive(artifactType)
		if loadErr != nil {
			return loadErr
		}

		return nil
	}); err != nil {
		var exceeded quota.TokenBalanceExceededError
		if errors.As(err, &exceeded) {
			h.writeQuotaError(c, err)
			return
		}
		if isNotFound(err) {
			dto.NotFound(c, err.Error())
			return
		}
		logger.Error(ctx, "failed to prepare conversation message", err)
		dto.InternalError(c, "failed to send message")
		return
	}

	start := time.Now()
	out, genErr := h.generator.Generate(ctx, &story.ArtifactGenerateInput{
		ProjectTitle:       project.Title,
		ProjectDescription: project.Description,
		Type:               artifactType,
		Prompt:             strings.TrimSpace(req.Prompt),
		Attachments:        req.ToStoryAttachments(),
		CurrentWorldview:   currentWorldview,
		CurrentCharacters:  currentCharacters,
		CurrentOutline:     currentOutline,
		CurrentArtifactRaw: currentArtifact,
		Provider:           provider,
		Model:              model,
		Temperature:        req.Temperature,
		MaxTokens:          req.MaxTokens,
	})
	durationMs := int(time.Since(start).Milliseconds())

	if genErr != nil {
		_ = h.markJobFailed(ctx, tenantID, jobID, genErr, durationMs)
		logger.Error(ctx, "artifact generation failed", genErr)
		dto.InternalError(c, "artifact generation failed")
		return
	}

	var snapshot *dto.ArtifactSnapshotResponse
	if err := withTenantTx(ctx, h.txMgr, h.tenantCtx, tenantID, func(txCtx context.Context) error {
		session, err := h.sessionRepo.GetByIDForUpdate(txCtx, sessionID)
		if err != nil {
			return err
		}
		if session == nil || session.ProjectID != projectID {
			return errNotFound("session not found")
		}

		art, err := h.artifactRepo.EnsureArtifact(txCtx, tenantID, projectID, out.Type)
		if err != nil {
			return err
		}

		latest, err := h.artifactRepo.GetLatestVersionNo(txCtx, art.ID)
		if err != nil {
			return err
		}
		nextNo := latest + 1

		versionID := uuid.NewString()
		createdBy := strings.TrimSpace(userID)
		sourceJobID := jobID

		version := &entity.ArtifactVersion{
			ID:          versionID,
			ArtifactID:  art.ID,
			VersionNo:   nextNo,
			Content:     out.Content,
			CreatedBy:   &createdBy,
			SourceJobID: &sourceJobID,
		}
		if err := h.artifactRepo.CreateVersion(txCtx, version); err != nil {
			return err
		}
		if err := h.artifactRepo.SetActiveVersion(txCtx, art.ID, version.ID); err != nil {
			return err
		}

		if out.Type == entity.ArtifactTypeNovelFoundation {
			var payload struct {
				Title       string `json:"title"`
				Description string `json:"description"`
				Genre       string `json:"genre,omitempty"`
			}
			if err := json.Unmarshal(out.Content, &payload); err != nil {
				return fmt.Errorf("invalid novel_foundation content: %w", err)
			}
			payload.Title = strings.TrimSpace(payload.Title)
			payload.Description = strings.TrimSpace(payload.Description)
			payload.Genre = strings.TrimSpace(payload.Genre)
			if payload.Title != "" {
				project.Title = payload.Title
			}
			if payload.Description != "" {
				project.Description = payload.Description
			}
			if payload.Genre != "" {
				project.Genre = payload.Genre
			}
			if err := h.projectRepo.Update(txCtx, project); err != nil {
				return err
			}
		}

		assistantMeta, _ := json.Marshal(map[string]any{
			"job_id":            jobID,
			"artifact_id":       art.ID,
			"version_id":        version.ID,
			"version_no":        version.VersionNo,
			"provider":          out.Meta.Provider,
			"model":             out.Meta.Model,
			"prompt_tokens":     out.Meta.PromptTokens,
			"completion_tokens": out.Meta.CompletionTokens,
			"temperature":       out.Meta.Temperature,
			"duration_ms":       durationMs,
			"generated_at":      out.Meta.GeneratedAt.Format(time.RFC3339),
			"request_id":        requestID,
			"trace_id":          traceID,
		})
		assistantTurn := entity.NewConversationTurn(sessionID, entity.RoleAssistant, task, out.Raw, assistantMeta)
		assistantTurn.ID = assistantTurnID
		if err := h.turnRepo.Create(txCtx, assistantTurn); err != nil {
			return err
		}

		job, err := h.jobRepo.GetByID(txCtx, jobID)
		if err != nil || job == nil {
			return err
		}
		job.OutputResult = out.Content
		job.Status = entity.JobStatusCompleted
		done := time.Now()
		job.CompletedAt = &done
		job.DurationMs = durationMs
		job.SetLLMMetrics(out.Meta.Provider, out.Meta.Model, out.Meta.PromptTokens, out.Meta.CompletionTokens)
		if err := h.jobRepo.Update(txCtx, job); err != nil {
			return err
		}

		snapshot = &dto.ArtifactSnapshotResponse{
			ArtifactID: art.ID,
			Type:       string(art.Type),
			VersionID:  version.ID,
			VersionNo:  version.VersionNo,
			Content:    version.Content,
		}

		return nil
	}); err != nil {
		if isNotFound(err) {
			dto.NotFound(c, err.Error())
			return
		}
		logger.Error(ctx, "failed to persist artifact version", err)
		dto.InternalError(c, "failed to persist result")
		return
	}

	dto.Success(c, &dto.SendMessageResponse{
		Session:          dto.ToSessionResponse(session),
		UserTurnID:       userTurnID,
		AssistantTurnID:  assistantTurnID,
		AssistantMessage: out.Raw,
		JobID:            jobID,
		ArtifactSnapshot: snapshot,
		Usage: &dto.FoundationUsageResponse{
			Provider:         out.Meta.Provider,
			Model:            out.Meta.Model,
			PromptTokens:     out.Meta.PromptTokens,
			CompletionTokens: out.Meta.CompletionTokens,
			Temperature:      out.Meta.Temperature,
			DurationMs:       durationMs,
			GeneratedAt:      out.Meta.GeneratedAt.Format(time.RFC3339),
		},
	})
}

func (h *ConversationHandler) writeQuotaError(c *gin.Context, err error) {
	var exceeded quota.TokenBalanceExceededError
	if errors.As(err, &exceeded) {
		dto.Error(c, http.StatusTooManyRequests, "token balance insufficient")
		return
	}
	dto.InternalError(c, "quota check failed")
}

func (h *ConversationHandler) markJobFailed(ctx context.Context, tenantID, jobID string, err error, durationMs int) error {
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

type notFoundError struct {
	msg string
}

func (e notFoundError) Error() string {
	return e.msg
}

func errNotFound(msg string) error {
	return notFoundError{msg: msg}
}

func isNotFound(err error) bool {
	var nf notFoundError
	return errors.As(err, &nf)
}

func normalizeConversationTask(task string) (entity.ConversationTask, error) {
	t := strings.TrimSpace(task)
	if t == "" {
		return entity.ConversationTaskNovelFoundation, nil
	}
	ct := entity.ConversationTask(t)
	switch ct {
	case entity.ConversationTaskNovelFoundation, entity.ConversationTaskWorldview, entity.ConversationTaskCharacters, entity.ConversationTaskOutline:
		return ct, nil
	default:
		return "", fmt.Errorf("invalid task: %s", t)
	}
}
