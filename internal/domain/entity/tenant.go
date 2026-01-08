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
	DefaultModel            string `json:"default_model,omitempty"`
	DefaultLanguage         string `json:"default_language,omitempty"`
	AllowPublicRegistration bool   `json:"allow_public_registration,omitempty"`
}

// Tenant 租户实体
type Tenant struct {
	ID           string          `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name         string          `json:"name" gorm:"type:varchar(255);not null"`
	Slug         string          `json:"slug" gorm:"type:varchar(100);uniqueIndex;not null"`
	Settings     *TenantSettings `json:"settings,omitempty" gorm:"type:jsonb;serializer:json"`
	Quota        *TenantQuota    `json:"quota,omitempty" gorm:"type:jsonb;serializer:json"`
	TokenBalance int64           `json:"token_balance" gorm:"not null;default:1000000"`
	Status       TenantStatus    `json:"status" gorm:"type:varchar(50);default:'active'"`
	CreatedAt    time.Time       `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt    time.Time       `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName 指定表名
func (Tenant) TableName() string {
	return "tenants"
}

// NewTenant 创建新租户
func NewTenant(name, slug string) *Tenant {
	now := time.Now()
	return &Tenant{
		Name: name,
		Slug: slug,
		Status: TenantStatusActive,
		Quota: &TenantQuota{
			MaxProjects:           100,
			MaxChaptersPerProject: 1000,
			MaxTokensPerDay:       1000000,
		},
		TokenBalance: 1000000,
		Settings:     &TenantSettings{},
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// HasSufficientBalance 检查余额是否充足
func (t *Tenant) HasSufficientBalance(required int64) bool {
	return t.TokenBalance >= required
}

// IsActive 检查租户是否活跃
func (t *Tenant) IsActive() bool {
	return t.Status == TenantStatusActive
}
