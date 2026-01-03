// Package dto 提供 HTTP 层数据传输对象
package dto

import (
	"time"

	"z-novel-ai-api/internal/domain/entity"
)

// TenantResponse 租户响应
type TenantResponse struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Slug      string                 `json:"slug"`
	Settings  *entity.TenantSettings `json:"settings,omitempty"`
	Quota     *entity.TenantQuota    `json:"quota,omitempty"`
	Status    entity.TenantStatus    `json:"status"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// UpdateTenantRequest 更新租户请求
type UpdateTenantRequest struct {
	Name     *string                `json:"name"`
	Settings *entity.TenantSettings `json:"settings"`
}

// ToTenantResponse 实体转换为响应
func ToTenantResponse(t *entity.Tenant) *TenantResponse {
	if t == nil {
		return nil
	}
	return &TenantResponse{
		ID:        t.ID,
		Name:      t.Name,
		Slug:      t.Slug,
		Settings:  t.Settings,
		Quota:     t.Quota,
		Status:    t.Status,
		CreatedAt: t.CreatedAt,
		UpdatedAt: t.UpdatedAt,
	}
}

// ApplyToTenant 更新实体
func (r *UpdateTenantRequest) ApplyToTenant(t *entity.Tenant) {
	if r.Name != nil {
		t.Name = *r.Name
	}
	if r.Settings != nil {
		t.Settings = r.Settings
	}
	t.UpdatedAt = time.Now()
}
