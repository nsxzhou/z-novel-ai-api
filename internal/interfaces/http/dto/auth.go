// Package dto 提供 HTTP 层数据传输对象
package dto

import (
	"z-novel-ai-api/internal/domain/entity"
)

// RegisterRequest 注册请求
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6,max=32"`
	Name     string `json:"name" binding:"required,max=128"`
	TenantID string `json:"tenant_id" binding:"omitempty,uuid"`
}

// LoginRequest 登录请求
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
	TenantID string `json:"tenant_id" binding:"omitempty,uuid"`
}

// AuthUserDTO 认证响应中的用户信息
type AuthUserDTO struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url,omitempty"`
	Role      string `json:"role"`
}

// AuthResponse 认证响应
type AuthResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token,omitempty"` // 仅用于部分非 Cookie 场景
	ExpiresIn    int          `json:"expires_in"`              // 秒
	User         *AuthUserDTO `json:"user"`
}

// ToAuthUserDTO 将领域实体转换为 DTO
func ToAuthUserDTO(u *entity.User) *AuthUserDTO {
	if u == nil {
		return nil
	}
	return &AuthUserDTO{
		ID:        u.ID,
		Email:     u.Email,
		Name:      u.Name,
		AvatarURL: u.AvatarURL,
		Role:      string(u.Role),
	}
}
