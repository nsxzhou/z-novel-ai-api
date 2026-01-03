// Package postgres 提供 PostgreSQL Repository 实现
package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

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

	q := getQuerier(ctx, r.client.db)

	settingsJSON, _ := json.Marshal(tenant.Settings)
	quotaJSON, _ := json.Marshal(tenant.Quota)

	query := `
		INSERT INTO tenants (id, name, slug, settings, quota, status, created_at, updated_at)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`

	err := q.QueryRowContext(ctx, query,
		tenant.Name, tenant.Slug, settingsJSON, quotaJSON, tenant.Status,
	).Scan(&tenant.ID, &tenant.CreatedAt, &tenant.UpdatedAt)

	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to create tenant: %w", err)
	}

	return nil
}

// GetByID 根据 ID 获取租户
func (r *TenantRepository) GetByID(ctx context.Context, id string) (*entity.Tenant, error) {
	ctx, span := tracer.Start(ctx, "postgres.TenantRepository.GetByID")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	query := `
		SELECT id, name, slug, settings, quota, status, created_at, updated_at
		FROM tenants
		WHERE id = $1
	`

	var tenant entity.Tenant
	var settingsJSON, quotaJSON []byte

	err := q.QueryRowContext(ctx, query, id).Scan(
		&tenant.ID, &tenant.Name, &tenant.Slug,
		&settingsJSON, &quotaJSON, &tenant.Status,
		&tenant.CreatedAt, &tenant.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		span.RecordError(err)
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	json.Unmarshal(settingsJSON, &tenant.Settings)
	json.Unmarshal(quotaJSON, &tenant.Quota)

	return &tenant, nil
}

// GetBySlug 根据 Slug 获取租户
func (r *TenantRepository) GetBySlug(ctx context.Context, slug string) (*entity.Tenant, error) {
	ctx, span := tracer.Start(ctx, "postgres.TenantRepository.GetBySlug")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	query := `
		SELECT id, name, slug, settings, quota, status, created_at, updated_at
		FROM tenants
		WHERE slug = $1
	`

	var tenant entity.Tenant
	var settingsJSON, quotaJSON []byte

	err := q.QueryRowContext(ctx, query, slug).Scan(
		&tenant.ID, &tenant.Name, &tenant.Slug,
		&settingsJSON, &quotaJSON, &tenant.Status,
		&tenant.CreatedAt, &tenant.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		span.RecordError(err)
		return nil, fmt.Errorf("failed to get tenant by slug: %w", err)
	}

	json.Unmarshal(settingsJSON, &tenant.Settings)
	json.Unmarshal(quotaJSON, &tenant.Quota)

	return &tenant, nil
}

// Update 更新租户
func (r *TenantRepository) Update(ctx context.Context, tenant *entity.Tenant) error {
	ctx, span := tracer.Start(ctx, "postgres.TenantRepository.Update")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	settingsJSON, _ := json.Marshal(tenant.Settings)
	quotaJSON, _ := json.Marshal(tenant.Quota)

	query := `
		UPDATE tenants
		SET name = $1, slug = $2, settings = $3, quota = $4, status = $5, updated_at = NOW()
		WHERE id = $6
		RETURNING updated_at
	`

	err := q.QueryRowContext(ctx, query,
		tenant.Name, tenant.Slug, settingsJSON, quotaJSON, tenant.Status, tenant.ID,
	).Scan(&tenant.UpdatedAt)

	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to update tenant: %w", err)
	}

	return nil
}

// Delete 删除租户
func (r *TenantRepository) Delete(ctx context.Context, id string) error {
	ctx, span := tracer.Start(ctx, "postgres.TenantRepository.Delete")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	query := `DELETE FROM tenants WHERE id = $1`
	_, err := q.ExecContext(ctx, query, id)

	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to delete tenant: %w", err)
	}

	return nil
}

// List 获取租户列表
func (r *TenantRepository) List(ctx context.Context, pagination repository.Pagination) (*repository.PagedResult[*entity.Tenant], error) {
	ctx, span := tracer.Start(ctx, "postgres.TenantRepository.List")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	// 获取总数
	var total int64
	countQuery := `SELECT COUNT(*) FROM tenants`
	if err := q.QueryRowContext(ctx, countQuery).Scan(&total); err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to count tenants: %w", err)
	}

	// 获取列表
	query := `
		SELECT id, name, slug, settings, quota, status, created_at, updated_at
		FROM tenants
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := q.QueryContext(ctx, query, pagination.Limit(), pagination.Offset())
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to list tenants: %w", err)
	}
	defer rows.Close()

	var tenants []*entity.Tenant
	for rows.Next() {
		var tenant entity.Tenant
		var settingsJSON, quotaJSON []byte

		if err := rows.Scan(
			&tenant.ID, &tenant.Name, &tenant.Slug,
			&settingsJSON, &quotaJSON, &tenant.Status,
			&tenant.CreatedAt, &tenant.UpdatedAt,
		); err != nil {
			span.RecordError(err)
			return nil, fmt.Errorf("failed to scan tenant: %w", err)
		}

		json.Unmarshal(settingsJSON, &tenant.Settings)
		json.Unmarshal(quotaJSON, &tenant.Quota)
		tenants = append(tenants, &tenant)
	}

	return repository.NewPagedResult(tenants, total, pagination), nil
}

// UpdateStatus 更新租户状态
func (r *TenantRepository) UpdateStatus(ctx context.Context, id string, status entity.TenantStatus) error {
	ctx, span := tracer.Start(ctx, "postgres.TenantRepository.UpdateStatus")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	query := `UPDATE tenants SET status = $1, updated_at = NOW() WHERE id = $2`
	_, err := q.ExecContext(ctx, query, status, id)

	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to update tenant status: %w", err)
	}

	return nil
}

// ExistsBySlug 检查 Slug 是否存在
func (r *TenantRepository) ExistsBySlug(ctx context.Context, slug string) (bool, error) {
	ctx, span := tracer.Start(ctx, "postgres.TenantRepository.ExistsBySlug")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM tenants WHERE slug = $1)`
	err := q.QueryRowContext(ctx, query, slug).Scan(&exists)

	if err != nil {
		span.RecordError(err)
		return false, fmt.Errorf("failed to check slug exists: %w", err)
	}

	return exists, nil
}
