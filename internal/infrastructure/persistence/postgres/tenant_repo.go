// Package postgres 提供 PostgreSQL Repository 实现
package postgres

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"z-novel-ai-api/internal/domain/entity"
	"z-novel-ai-api/internal/domain/repository"
)

// TenantRepository 租户仓储实现
type TenantRepository struct {
	client *Client
}

// NewTenantRepository 创建租户仓储
func NewTenantRepository(client *Client) *TenantRepository {
	return &TenantRepository{client: client}
}

// Create 创建租户
func (r *TenantRepository) Create(ctx context.Context, tenant *entity.Tenant) error {
	ctx, span := tracer.Start(ctx, "postgres.TenantRepository.Create")
	defer span.End()

	db := getDB(ctx, r.client.db)
	if err := db.Create(tenant).Error; err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to create tenant: %w", err)
	}
	return nil
}

// GetByID 根据 ID 获取租户
func (r *TenantRepository) GetByID(ctx context.Context, id string) (*entity.Tenant, error) {
	ctx, span := tracer.Start(ctx, "postgres.TenantRepository.GetByID")
	defer span.End()

	db := getDB(ctx, r.client.db)
	var tenant entity.Tenant
	if err := db.First(&tenant, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		span.RecordError(err)
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}
	return &tenant, nil
}

// GetBySlug 根据 Slug 获取租户
func (r *TenantRepository) GetBySlug(ctx context.Context, slug string) (*entity.Tenant, error) {
	ctx, span := tracer.Start(ctx, "postgres.TenantRepository.GetBySlug")
	defer span.End()

	db := getDB(ctx, r.client.db)
	var tenant entity.Tenant
	if err := db.First(&tenant, "slug = ?", slug).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		span.RecordError(err)
		return nil, fmt.Errorf("failed to get tenant by slug: %w", err)
	}
	return &tenant, nil
}

// Update 更新租户
func (r *TenantRepository) Update(ctx context.Context, tenant *entity.Tenant) error {
	ctx, span := tracer.Start(ctx, "postgres.TenantRepository.Update")
	defer span.End()

	db := getDB(ctx, r.client.db)
	if err := db.Save(tenant).Error; err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to update tenant: %w", err)
	}
	return nil
}

// Delete 删除租户
func (r *TenantRepository) Delete(ctx context.Context, id string) error {
	ctx, span := tracer.Start(ctx, "postgres.TenantRepository.Delete")
	defer span.End()

	db := getDB(ctx, r.client.db)
	if err := db.Delete(&entity.Tenant{}, "id = ?", id).Error; err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to delete tenant: %w", err)
	}
	return nil
}

// List 获取租户列表
func (r *TenantRepository) List(ctx context.Context, pagination repository.Pagination) (*repository.PagedResult[*entity.Tenant], error) {
	ctx, span := tracer.Start(ctx, "postgres.TenantRepository.List")
	defer span.End()

	db := getDB(ctx, r.client.db)

	// 获取总数
	var total int64
	if err := db.Model(&entity.Tenant{}).Count(&total).Error; err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to count tenants: %w", err)
	}

	// 获取列表
	var tenants []*entity.Tenant
	if err := db.Order("created_at DESC").
		Offset(pagination.Offset()).
		Limit(pagination.Limit()).
		Find(&tenants).Error; err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to list tenants: %w", err)
	}

	return repository.NewPagedResult(tenants, total, pagination), nil
}

// UpdateStatus 更新租户状态
func (r *TenantRepository) UpdateStatus(ctx context.Context, id string, status entity.TenantStatus) error {
	ctx, span := tracer.Start(ctx, "postgres.TenantRepository.UpdateStatus")
	defer span.End()

	db := getDB(ctx, r.client.db)
	if err := db.Model(&entity.Tenant{}).Where("id = ?", id).Update("status", status).Error; err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to update tenant status: %w", err)
	}
	return nil
}

// ExistsBySlug 检查 Slug 是否存在
func (r *TenantRepository) ExistsBySlug(ctx context.Context, slug string) (bool, error) {
	ctx, span := tracer.Start(ctx, "postgres.TenantRepository.ExistsBySlug")
	defer span.End()

	db := getDB(ctx, r.client.db)
	var count int64
	if err := db.Model(&entity.Tenant{}).Where("slug = ?", slug).Count(&count).Error; err != nil {
		span.RecordError(err)
		return false, fmt.Errorf("failed to check slug exists: %w", err)
	}
	return count > 0, nil
}
