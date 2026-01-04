// Package dto 提供 HTTP 层数据传输对象
package dto

import (
	"time"

	"z-novel-ai-api/internal/domain/entity"
)

// UserResponse 用户响应
type UserResponse struct {
	ID          string               `json:"id"`
	TenantID    string               `json:"tenant_id"`
	Email       string               `json:"email"`
	Name        string               `json:"name"`
	AvatarURL   string               `json:"avatar_url,omitempty"`
	Role        entity.UserRole      `json:"role"`
	Settings    *entity.UserSettings `json:"settings,omitempty"`
	LastLoginAt *time.Time           `json:"last_login_at,omitempty"`
	CreatedAt   time.Time            `json:"created_at"`
	UpdatedAt   time.Time            `json:"updated_at"`
}

// UserListResponse 用户列表响应
type UserListResponse struct {
	Items []*UserResponse `json:"items"`
}

// UpdateUserRequest 更新用户请求
type UpdateUserRequest struct {
	Name      *string              `json:"name"`
	AvatarURL *string              `json:"avatar_url"`
	Settings  *entity.UserSettings `json:"settings"`
}

// UpdateUserRoleRequest 更新用户角色请求
type UpdateUserRoleRequest struct {
	Role entity.UserRole `json:"role" binding:"required,oneof=admin member viewer"`
}

// ToUserResponse 实体转换为响应
func ToUserResponse(u *entity.User) *UserResponse {
	if u == nil {
		return nil
	}
	return &UserResponse{
		ID:          u.ID,
		TenantID:    u.TenantID,
		Email:       u.Email,
		Name:        u.Name,
		AvatarURL:   u.AvatarURL,
		Role:        u.Role,
		Settings:    u.Settings,
		LastLoginAt: u.LastLoginAt,
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
	}
}

// ToUserListResponse 实体列表转换为响应
func ToUserListResponse(users []*entity.User) *UserListResponse {
	items := make([]*UserResponse, len(users))
	for i, u := range users {
		items[i] = ToUserResponse(u)
	}
	return &UserListResponse{Items: items}
}

// ApplyToUser 更新实体
func (r *UpdateUserRequest) ApplyToUser(u *entity.User) {
	if r.Name != nil {
		u.Name = *r.Name
	}
	if r.AvatarURL != nil {
		u.AvatarURL = *r.AvatarURL
	}
	if r.Settings != nil {
		u.Settings = r.Settings
	}
	u.UpdatedAt = time.Now()
}
