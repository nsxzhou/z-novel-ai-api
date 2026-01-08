package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
)

// ProjectCreationHandler 对话式创建项目处理器
type ProjectCreationHandler struct {
	cfg *config.Config

	txMgr     repository.Transactor
	tenantCtx repository.TenantContextManager

	tenantRepo  repository.TenantRepository
	projectRepo repository.ProjectRepository
	sessionRepo repository.ConversationSessionRepository

	pcSessionRepo repository.ProjectCreationSessionRepository
	pcTurnRepo    repository.ProjectCreationTurnRepository

	jobRepo      repository.JobRepository
	llmUsageRepo repository.LLMUsageEventRepository

	quotaChecker *quota.TokenQuotaChecker
	generator    *story.ProjectCreationGenerator
}

func NewProjectCreationHandler(
	cfg *config.Config,
	txMgr repository.Transactor,
	tenantCtx repository.TenantContextManager,
	tenantRepo repository.TenantRepository,
	projectRepo repository.ProjectRepository,
	sessionRepo repository.ConversationSessionRepository,
	pcSessionRepo repository.ProjectCreationSessionRepository,
	pcTurnRepo repository.ProjectCreationTurnRepository,
	jobRepo repository.JobRepository,
	llmUsageRepo repository.LLMUsageEventRepository,
	quotaChecker *quota.TokenQuotaChecker,
	generator *story.ProjectCreationGenerator,
) *ProjectCreationHandler {
	return &ProjectCreationHandler{
		cfg:           cfg,
		txMgr:         txMgr,
		tenantCtx:     tenantCtx,
		tenantRepo:    tenantRepo,
		projectRepo:   projectRepo,
		sessionRepo:   sessionRepo,
		pcSessionRepo: pcSessionRepo,
		pcTurnRepo:    pcTurnRepo,
		jobRepo:       jobRepo,
		llmUsageRepo:  llmUsageRepo,
		quotaChecker:  quotaChecker,
		generator:     generator,
	}
}

// CreateSession 创建“对话式创建项目”会话
// @Summary 创建项目创建会话
// @Tags ProjectCreation
// @Accept json
// @Produce json
// @Param body body dto.CreateProjectCreationSessionRequest false "创建请求"
// @Success 201 {object} dto.Response[dto.ProjectCreationSessionResponse]
// @Router /v1/project-creation-sessions [post]
func (h *ProjectCreationHandler) CreateSession(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID := middleware.GetTenantIDFromGin(c)
	userID := middleware.GetUserIDFromGin(c)

	var req dto.CreateProjectCreationSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, http.ErrBodyReadAfterClose) && !errors.Is(err, fmt.Errorf("EOF")) {
		// 允许空 body
	}

	session := entity.NewProjectCreationSession(tenantID, userID)
	if err := h.pcSessionRepo.Create(ctx, session); err != nil {
		logger.Error(ctx, "failed to create pc session", err)
		dto.InternalError(c, "failed to create session")
		return
	}

	dto.Created(c, dto.ToProjectCreationSessionResponse(session))
}

// GetSession 获取会话详情
func (h *ProjectCreationHandler) GetSession(c *gin.Context) {
	ctx := c.Request.Context()
	sessionID := c.Param("sid")

	session, err := h.pcSessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		logger.Error(ctx, "failed to get pc session", err)
		dto.InternalError(c, "failed to get session")
		return
	}
	if session == nil {
		dto.NotFound(c, "session not found")
		return
	}

	dto.Success(c, dto.ToProjectCreationSessionResponse(session))
}

// ListTurns 获取会话轮次
func (h *ProjectCreationHandler) ListTurns(c *gin.Context) {
	ctx := c.Request.Context()
	sessionID := c.Param("sid")

	pageReq := dto.BindPage(c)
	result, err := h.pcTurnRepo.ListBySession(ctx, sessionID, repository.NewPagination(pageReq.Page, pageReq.PageSize))
	if err != nil {
		logger.Error(ctx, "failed to list pc turns", err)
		dto.InternalError(c, "failed to list turns")
		return
	}

	turns := make([]*dto.ProjectCreationTurnResponse, 0, len(result.Items))
	for i := range result.Items {
		turns = append(turns, dto.ToProjectCreationTurnResponse(result.Items[i]))
	}

	dto.SuccessWithPage(c, &dto.ProjectCreationTurnListResponse{Turns: turns}, dto.NewPageMeta(pageReq.Page, pageReq.PageSize, int(result.Total)))
}

// SendMessage 处理对话消息，驱动“对话式创建项目”的状态机流转。
//
// 核心流程：
// 1. **参数解析与校验**: 获取租户/用户 ID，解析请求体，确定使用的 LLM 模型。
// 2. **预处理 (事务 1)**:
//   - 检查租户状态与 Token 配额。
//   - 锁定 Session 行 (GetByIDForUpdate)。
//   - 持久化用户的输入消息 (User Turn)，确保即使后续 LLM 失败，用户的发言也被记录。
//
// 3. **LLM 生成 (非事务)**:
//   - 调用 generator.Generate 执行核心业务逻辑（状态机判断 + 内容生成）。
//   - 注意：此步骤耗时较长，必须在数据库事务之外执行，避免阻塞连接池。
//
// 4. **后处理 (事务 2)**:
//   - 再次锁定 Session。
//   - 更新 Session 的状态 (Stage) 和草稿内容 (Draft)。
//   - **动作执行**: 如果 LLM 决定“创建项目”，则执行实际的 Project/Conversation 创建逻辑。
//   - 包含“确定性门控”检查，防止模型幻觉误创建。
//   - 持久化 AI 的回复消息 (Assistant Turn)。
//
// @Summary 发送消息
// @Tags ProjectCreation
// @Accept json
// @Produce json
// @Param sid path string true "会话ID"
// @Param body body dto.SendProjectCreationMessageRequest true "消息内容"
// @Success 200 {object} dto.Response[dto.SendProjectCreationMessageResponse]
// @Router /v1/project-creation-sessions/{sid}/messages [post]
func (h *ProjectCreationHandler) SendMessage(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID := middleware.GetTenantIDFromGin(c)
	userID := middleware.GetUserIDFromGin(c)
	sessionID := c.Param("sid")

	// 1. 参数绑定与模型解析
	var req dto.SendProjectCreationMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequest(c, "invalid request body: "+err.Error())
		return
	}

	provider, model, err := resolveProviderModel(h.cfg, req.Provider, req.Model)
	if err != nil {
		dto.BadRequest(c, err.Error())
		return
	}

	var session *entity.ProjectCreationSession
	var tenant *entity.Tenant
	var userTurnID string

	// 2. 预处理事务：检查配额并保存用户消息
	if err := withTenantTx(ctx, h.txMgr, h.tenantCtx, tenantID, func(txCtx context.Context) error {
		var loadErr error
		tenant, loadErr = h.tenantRepo.GetByID(txCtx, tenantID)
		if loadErr != nil || tenant == nil {
			return loadErr
		}

		// 检查租户 AI 配额
		if quotaErr := precheckQuota(txCtx, h.quotaChecker, tenant); quotaErr != nil {
			return quotaErr
		}

		// 锁定会话，防止并发写入冲突
		session, loadErr = h.pcSessionRepo.GetByIDForUpdate(txCtx, sessionID)
		if loadErr != nil {
			return loadErr
		}
		if session == nil {
			return fmt.Errorf("session not found")
		}
		if session.Status != entity.ProjectCreationStatusActive {
			return fmt.Errorf("session is not active")
		}

		// 保存用户 Turn (User Input)
		userTurn := entity.NewProjectCreationTurn(sessionID, entity.RoleUser, req.Prompt, nil)
		if err := h.pcTurnRepo.Create(txCtx, userTurn); err != nil {
			return err
		}
		userTurnID = userTurn.ID
		return nil
	}); err != nil {
		logger.Error(ctx, "failed to prepare message", err)
		dto.InternalError(c, "failed to send message")
		return
	}

	// 3. LLM 生成 (无事务，耗时操作)
	input := &story.ProjectCreationGenerateInput{
		Stage:       string(session.Stage),
		Draft:       session.Draft,
		Prompt:      req.Prompt,
		Attachments: req.ToStoryAttachments(),
		Provider:    provider,
		Model:       model,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}

	start := time.Now()
	out, err := h.generator.Generate(ctx, input)
	durationMs := int(time.Since(start).Milliseconds())

	if err != nil {
		logger.Error(ctx, "llm generation failed", err)
		dto.InternalError(c, "generation failed")
		return
	}

	// 4. 后处理事务：更新状态并保存结果
	var projectID, projectSessionID *string
	var assistantTurnID string

	if err := withTenantTx(ctx, h.txMgr, h.tenantCtx, tenantID, func(txCtx context.Context) error {
		// 再次锁定会话 (跨事务需重新获取锁)
		session, err = h.pcSessionRepo.GetByIDForUpdate(txCtx, sessionID)
		if err != nil {
			return err
		}

		// 更新 Session 的核心状态
		session.Stage = entity.ProjectCreationStage(out.NextStage)
		session.Draft = out.Draft

		// 处理“创建项目”动作
		if out.Action == "create_project" && out.ProposedProject != nil {
			// **关键安全检查**：确定性门控 (Deterministic Gate)
			// 即使 LLM 认为应该创建项目，我们也要再次检查用户的 Prompt 是否包含明确的“确认”关键词。
			// 这是为了防止 LLM 在用户只是询问细节时误触发创建操作。
			if session.Stage != entity.ProjectCreationStageConfirm || !isDeterministicProjectCreateConfirm(req.Prompt) {
				// 未通过确认检查：降级为“提议创建”
				out.Action = "propose_creation"
				out.RequiresConfirmation = true
				out.AssistantMessage = strings.TrimSpace(out.AssistantMessage + "\n\n（系统提示：未检测到明确“确认创建”，因此未创建项目；如需创建，请回复“确认创建”。）")
			} else {
				// 通过确认检查：执行真实创建逻辑
				// 1. 创建 Project 实体
				newProject := &entity.Project{
					TenantID:    tenantID,
					OwnerID:     userID,
					Title:       out.ProposedProject.Title,
					Description: out.ProposedProject.Description,
					Genre:       out.ProposedProject.Genre,
					Status:      entity.ProjectStatusActive,
				}
				if err := h.projectRepo.Create(txCtx, newProject); err != nil {
					return err
				}
				pid := newProject.ID
				projectID = &pid

				// 2. 创建初始小说会话 (ConversationSession)
				newSession := entity.NewConversationSession(tenantID, pid, entity.ConversationTaskNovelFoundation)
				if err := h.sessionRepo.Create(txCtx, newSession); err != nil {
					return err
				}
				psid := newSession.ID
				projectSessionID = &psid

				// 3. 标记当前创建流程已完成
				session.Status = entity.ProjectCreationStatusCompleted
				session.CreatedProjectID = projectID
				session.CreatedProjectSessionID = projectSessionID
			}
		}

		// 持久化 Session 变更
		if err := h.pcSessionRepo.Update(txCtx, session); err != nil {
			return err
		}

		// 保存 Assistant Turn (AI Output)
		meta, _ := json.Marshal(map[string]any{
			"stage":                 out.NextStage,
			"action":                out.Action,
			"requires_confirmation": out.RequiresConfirmation,
			"usage":                 out.Meta,
			"duration_ms":           durationMs,
		})
		assistantTurn := entity.NewProjectCreationTurn(sessionID, entity.RoleAssistant, out.AssistantMessage, meta)
		if err := h.pcTurnRepo.Create(txCtx, assistantTurn); err != nil {
			return err
		}
		assistantTurnID = assistantTurn.ID

		return nil
	}); err != nil {
		logger.Error(ctx, "failed to persist session state", err)
		dto.InternalError(c, "failed to finalize message")
		return
	}

	dto.Success(c, &dto.SendProjectCreationMessageResponse{
		Session:          dto.ToProjectCreationSessionResponse(session),
		UserTurnID:       userTurnID,
		AssistantTurnID:  assistantTurnID,
		AssistantMessage: out.AssistantMessage,
		ProjectID:        projectID,
		ProjectSessionID: projectSessionID,
		Usage: &dto.FoundationUsageResponse{
			Provider:         out.Meta.Provider,
			Model:            out.Meta.Model,
			PromptTokens:     out.Meta.PromptTokens,
			CompletionTokens: out.Meta.CompletionTokens,
			GeneratedAt:      out.Meta.GeneratedAt.Format(time.RFC3339),
			DurationMs:       durationMs,
		},
	})
}

func isDeterministicProjectCreateConfirm(prompt string) bool {
	p := strings.ToLower(strings.TrimSpace(prompt))
	if p == "" {
		return false
	}

	// 明确否定优先
	for _, kw := range []string{
		"不创建", "不要创建", "别创建", "取消", "暂不", "先不", "不需要创建", "不想创建", "否", "不是",
	} {
		if strings.Contains(p, kw) {
			return false
		}
	}

	// 明确创建/确认意图
	for _, kw := range []string{
		"确认创建", "确定创建", "创建项目", "开始创建", "立即创建", "创建吧", "就创建", "创建",
		"confirm", "create project", "create",
	} {
		if strings.Contains(p, kw) {
			return true
		}
	}

	// 允许极短确认（仅在 confirm 阶段才会生效）
	switch strings.Trim(p, " \t\r\n。．.！!？?，,;；:：") {
	case "确认", "确定", "同意", "是", "是的", "好的", "好", "可以", "行", "ok", "okay", "yes":
		return true
	default:
		return false
	}
}
