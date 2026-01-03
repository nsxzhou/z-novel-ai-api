// Package entity 定义领域实体
package entity

import (
	"time"
)

// UserRole 用户角色
type UserRole string

const (
	UserRoleAdmin  UserRole = "admin"
	UserRoleMember UserRole = "member"
	UserRoleViewer UserRole = "viewer"
)

// UserSettings 用户设置
type UserSettings struct {
	Theme            string `json:"theme,omitempty"`
	Language         string `json:"language,omitempty"`
	NotifyOnComplete bool   `json:"notify_on_complete,omitempty"`
}

// User 用户实体
type User struct {
	ID          string        `json:"id"`
	TenantID    string        `json:"tenant_id"`
	ExternalID  string        `json:"external_id,omitempty"`
	Email       string        `json:"email"`
	Name        string        `json:"name"`
	AvatarURL   string        `json:"avatar_url,omitempty"`
	Role        UserRole      `json:"role"`
	Settings    *UserSettings `json:"settings,omitempty"`
	LastLoginAt *time.Time    `json:"last_login_at,omitempty"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

// NewUser 创建新用户
func NewUser(tenantID, email, name string) *User {
	now := time.Now()
	return &User{
		TenantID:  tenantID,
		Email:     email,
		Name:      name,
		Role:      UserRoleMember,
		Settings:  &UserSettings{},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// IsAdmin 检查用户是否为管理员
func (u *User) IsAdmin() bool {
	return u.Role == UserRoleAdmin
}

// CanEdit 检查用户是否有编辑权限
func (u *User) CanEdit() bool {
	return u.Role == UserRoleAdmin || u.Role == UserRoleMember
}
