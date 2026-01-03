// Package repository 定义数据访问层接口
package repository

import (
	"context"

	"z-novel-ai-api/internal/domain/entity"
)

// TenantRepository 租户仓储接口
type TenantRepository interface {
	// Create 创建租户
	Create(ctx context.Context, tenant *entity.Tenant) error

	// GetByID 根据 ID 获取租户
	GetByID(ctx context.Context, id string) (*entity.Tenant, error)

	// GetBySlug 根据 Slug 获取租户
	GetBySlug(ctx context.Context, slug string) (*entity.Tenant, error)

	// Update 更新租户
	Update(ctx context.Context, tenant *entity.Tenant) error

	// Delete 删除租户
	Delete(ctx context.Context, id string) error

	// List 获取租户列表
	List(ctx context.Context, pagination Pagination) (*PagedResult[*entity.Tenant], error)

	// UpdateStatus 更新租户状态
	UpdateStatus(ctx context.Context, id string, status entity.TenantStatus) error

	// ExistsBySlug 检查 Slug 是否存在
	ExistsBySlug(ctx context.Context, slug string) (bool, error)
}
