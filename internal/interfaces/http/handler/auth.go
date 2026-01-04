// Package handler 提供 HTTP 请求处理器
package handler

import (
	"time"

	"z-novel-ai-api/internal/domain/entity"
	"z-novel-ai-api/internal/domain/repository"
	"z-novel-ai-api/internal/interfaces/http/dto"
	"z-novel-ai-api/internal/interfaces/http/middleware"
	"z-novel-ai-api/pkg/logger"
	"z-novel-ai-api/pkg/utils"

	"github.com/gin-gonic/gin"
)

// AuthHandler 认证处理器
type AuthHandler struct {
	jwtManager *utils.JWTManager
	cfg        middleware.AuthConfig
	userRepo   repository.UserRepository
	tenantRepo repository.TenantRepository
}

// NewAuthHandler 创建认证处理器
func NewAuthHandler(cfg middleware.AuthConfig, userRepo repository.UserRepository, tenantRepo repository.TenantRepository) *AuthHandler {
	return &AuthHandler{
		jwtManager: utils.NewJWTManager(cfg.Secret, cfg.Issuer),
		cfg:        cfg,
		userRepo:   userRepo,
		tenantRepo: tenantRepo,
	}
}

// Register 注册
// @Summary 用户注册
// @Description 创建新用户，如果未指定租户则关联到默认租户
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body dto.RegisterRequest true "注册信息"
// @Success 201 {object} dto.Response[dto.AuthResponse]
// @Failure 400 {object} dto.ErrorResponse
// @Failure 409 {object} dto.ErrorResponse
// @Router /v1/auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	ctx := c.Request.Context()

	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequest(c, "invalid request body: "+err.Error())
		return
	}

	// 确定租户 ID
	tenantID := req.TenantID
	if tenantID == "" {
		dto.BadRequest(c, "tenant_id is required")
		return
	}

	// 检查租户是否存在及其注册策略
	tenant, err := h.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		logger.Error(ctx, "failed to check tenant", err)
		dto.InternalError(c, "registration failed")
		return
	}
	if tenant == nil {
		dto.BadRequest(c, "tenant not found")
		return
	}

	// 检查租户是否允许公开注册
	if tenant.Settings == nil || !tenant.Settings.AllowPublicRegistration {
		logger.Warn(ctx, "registration attempt on restricted tenant", "tenant_id", tenantID, "email", req.Email)
		dto.Forbidden(c, "registration is not allowed for this tenant")
		return
	}

	// 检查邮箱是否已存在
	exists, err := h.userRepo.ExistsByEmail(ctx, tenantID, req.Email)
	if err != nil {
		logger.Error(ctx, "failed to check email existence", err)
		dto.InternalError(c, "registration failed")
		return
	}
	if exists {
		dto.BadRequest(c, "email already registered")
		return
	}

	// 创建用户实体
	user := entity.NewUser(tenantID, req.Email, req.Name)
	if err := user.SetPassword(req.Password); err != nil {
		logger.Error(ctx, "failed to hash password", err)
		dto.InternalError(c, "registration failed")
		return
	}

	// 保存用户
	if err := h.userRepo.Create(ctx, user); err != nil {
		logger.Error(ctx, "failed to create user", err)
		dto.InternalError(c, "registration failed")
		return
	}

	// 生成 Token
	tokens, err := h.jwtManager.GenerateTokenPair(user.TenantID, user.ID, string(user.Role), 15*time.Minute, 7*24*time.Hour)
	if err != nil {
		dto.InternalError(c, "user created but failed to generate tokens")
		return
	}

	// 设置 RefreshToken 到 Cookie
	c.SetCookie("refresh_token", tokens.RefreshToken, int(7*24*time.Hour.Seconds()), "/v1/auth/refresh", "", false, true)

	dto.Created(c, &dto.AuthResponse{
		AccessToken: tokens.AccessToken,
		ExpiresIn:   900,
		User:        dto.ToAuthUserDTO(user),
	})
}

// Login 登录重构
// @Summary 用户登录
// @Description 验证邮箱密码并返回双 Token
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body dto.LoginRequest true "登录信息"
// @Success 200 {object} dto.Response[dto.AuthResponse]
// @Failure 401 {object} dto.ErrorResponse
// @Router /v1/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	ctx := c.Request.Context()

	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequest(c, "invalid request body: "+err.Error())
		return
	}

	tenantID := req.TenantID
	if tenantID == "" {
		dto.BadRequest(c, "tenant_id is required")
		return
	}

	// 查询用户
	user, err := h.userRepo.GetByEmail(ctx, tenantID, req.Email)
	if err != nil {
		logger.Error(ctx, "failed to get user", err)
		dto.InternalError(c, "login failed")
		return
	}

	// 校验存在性及密码
	if user == nil || !user.CheckPassword(req.Password) {
		dto.Unauthorized(c, "invalid email or password")
		return
	}

	// 更新登录状态
	if err := h.userRepo.UpdateLastLogin(ctx, user.ID); err != nil {
		logger.Warn(ctx, "failed to update last login time", "error", err, "user_id", user.ID)
	}

	// 生成 Token
	tokens, err := h.jwtManager.GenerateTokenPair(user.TenantID, user.ID, string(user.Role), 15*time.Minute, 7*24*time.Hour)
	if err != nil {
		dto.InternalError(c, "failed to generate tokens")
		return
	}

	c.SetCookie("refresh_token", tokens.RefreshToken, int(7*24*time.Hour.Seconds()), "/v1/auth/refresh", "", false, true)

	dto.Success(c, &dto.AuthResponse{
		AccessToken: tokens.AccessToken,
		ExpiresIn:   900,
		User:        dto.ToAuthUserDTO(user),
	})
}

// RefreshToken 刷新 Token (保持原有逻辑但适配 DTO)
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		dto.Unauthorized(c, "missing refresh token")
		return
	}

	claims, err := h.jwtManager.ParseToken(refreshToken)
	if err != nil {
		dto.Unauthorized(c, "invalid refresh token")
		return
	}

	newAccessToken, err := h.jwtManager.GenerateToken(claims.TenantID, claims.UserID, claims.Role, "access", 15*time.Minute)
	if err != nil {
		dto.InternalError(c, "failed to generate access token")
		return
	}

	dto.Success(c, gin.H{
		"access_token": newAccessToken,
		"expires_in":   900,
	})
}

// Logout 登出
func (h *AuthHandler) Logout(c *gin.Context) {
	c.SetCookie("refresh_token", "", -1, "/v1/auth/refresh", "", false, true)
	dto.Success(c, gin.H{"message": "logged out success"})
}
