// Package entity 定义领域实体
package entity

import (
	"time"
)

// TenantStatus 租户状态
type TenantStatus string

const (
	TenantStatusActive    TenantStatus = "active"
	TenantStatusSuspended TenantStatus = "suspended"
	TenantStatusDeleted   TenantStatus = "deleted"
)

// TenantQuota 租户配额
type TenantQuota struct {
	MaxProjects           int   `json:"max_projects"`
	MaxChaptersPerProject int   `json:"max_chapters_per_project"`
	MaxTokensPerDay       int64 `json:"max_tokens_per_day"`
}

// TenantSettings 租户设置
type TenantSettings struct {
	DefaultModel    string `json:"default_model,omitempty"`
	DefaultLanguage string `json:"default_language,omitempty"`
}

// Tenant 租户实体
type Tenant struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Slug      string          `json:"slug"`
	Settings  *TenantSettings `json:"settings,omitempty"`
	Quota     *TenantQuota    `json:"quota,omitempty"`
	Status    TenantStatus    `json:"status"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// NewTenant 创建新租户
func NewTenant(name, slug string) *Tenant {
	now := time.Now()
	return &Tenant{
		Name:   name,
		Slug:   slug,
		Status: TenantStatusActive,
		Quota: &TenantQuota{
			MaxProjects:           100,
			MaxChaptersPerProject: 1000,
			MaxTokensPerDay:       1000000,
		},
		Settings:  &TenantSettings{},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// IsActive 检查租户是否活跃
func (t *Tenant) IsActive() bool {
	return t.Status == TenantStatusActive
}
