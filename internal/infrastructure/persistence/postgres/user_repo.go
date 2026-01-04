// Package postgres 提供 PostgreSQL Repository 实现
package postgres

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"z-novel-ai-api/internal/domain/entity"
	"z-novel-ai-api/internal/domain/repository"
)

// UserRepository 用户仓储实现
type UserRepository struct {
	client *Client
}

// NewUserRepository 创建用户仓储
func NewUserRepository(client *Client) *UserRepository {
	return &UserRepository{client: client}
}

// Create 创建用户
func (r *UserRepository) Create(ctx context.Context, user *entity.User) error {
	ctx, span := tracer.Start(ctx, "postgres.UserRepository.Create")
	defer span.End()

	db := getDB(ctx, r.client.db)
	if err := db.Create(user).Error; err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

// GetByID 根据 ID 获取用户
func (r *UserRepository) GetByID(ctx context.Context, id string) (*entity.User, error) {
	ctx, span := tracer.Start(ctx, "postgres.UserRepository.GetByID")
	defer span.End()

	db := getDB(ctx, r.client.db)
	var user entity.User
	if err := db.First(&user, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		span.RecordError(err)
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}

// GetByEmail 根据邮箱获取用户
func (r *UserRepository) GetByEmail(ctx context.Context, tenantID, email string) (*entity.User, error) {
	ctx, span := tracer.Start(ctx, "postgres.UserRepository.GetByEmail")
	defer span.End()

	db := getDB(ctx, r.client.db)
	var user entity.User
	if err := db.First(&user, "tenant_id = ? AND email = ?", tenantID, email).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		span.RecordError(err)
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}
	return &user, nil
}

// GetByExternalID 根据外部 ID 获取用户
func (r *UserRepository) GetByExternalID(ctx context.Context, externalID string) (*entity.User, error) {
	ctx, span := tracer.Start(ctx, "postgres.UserRepository.GetByExternalID")
	defer span.End()

	db := getDB(ctx, r.client.db)
	var user entity.User
	if err := db.First(&user, "external_id = ?", externalID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		span.RecordError(err)
		return nil, fmt.Errorf("failed to get user by external id: %w", err)
	}
	return &user, nil
}

// Update 更新用户
func (r *UserRepository) Update(ctx context.Context, user *entity.User) error {
	ctx, span := tracer.Start(ctx, "postgres.UserRepository.Update")
	defer span.End()

	db := getDB(ctx, r.client.db)
	if err := db.Save(user).Error; err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to update user: %w", err)
	}
	return nil
}

// Delete 删除用户
func (r *UserRepository) Delete(ctx context.Context, id string) error {
	ctx, span := tracer.Start(ctx, "postgres.UserRepository.Delete")
	defer span.End()

	db := getDB(ctx, r.client.db)
	if err := db.Delete(&entity.User{}, "id = ?", id).Error; err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to delete user: %w", err)
	}
	return nil
}

// ListByTenant 获取租户用户列表
func (r *UserRepository) ListByTenant(ctx context.Context, tenantID string, pagination repository.Pagination) (*repository.PagedResult[*entity.User], error) {
	ctx, span := tracer.Start(ctx, "postgres.UserRepository.ListByTenant")
	defer span.End()

	db := getDB(ctx, r.client.db)
	query := db.Model(&entity.User{}).Where("tenant_id = ?", tenantID)

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to count users: %w", err)
	}

	// 获取列表
	var users []*entity.User
	if err := query.Order("created_at DESC").
		Offset(pagination.Offset()).
		Limit(pagination.Limit()).
		Find(&users).Error; err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	return repository.NewPagedResult(users, total, pagination), nil
}

// UpdateRole 更新用户角色
func (r *UserRepository) UpdateRole(ctx context.Context, id string, role entity.UserRole) error {
	ctx, span := tracer.Start(ctx, "postgres.UserRepository.UpdateRole")
	defer span.End()

	db := getDB(ctx, r.client.db)
	if err := db.Model(&entity.User{}).Where("id = ?", id).Update("role", role).Error; err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to update user role: %w", err)
	}
	return nil
}

// UpdateLastLogin 更新最后登录时间
func (r *UserRepository) UpdateLastLogin(ctx context.Context, id string) error {
	ctx, span := tracer.Start(ctx, "postgres.UserRepository.UpdateLastLogin")
	defer span.End()

	db := getDB(ctx, r.client.db)
	now := time.Now()
	if err := db.Model(&entity.User{}).Where("id = ?", id).Update("last_login_at", now).Error; err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to update last login: %w", err)
	}
	return nil
}

// ExistsByEmail 检查邮箱是否存在
func (r *UserRepository) ExistsByEmail(ctx context.Context, tenantID, email string) (bool, error) {
	ctx, span := tracer.Start(ctx, "postgres.UserRepository.ExistsByEmail")
	defer span.End()

	db := getDB(ctx, r.client.db)
	var count int64
	if err := db.Model(&entity.User{}).Where("tenant_id = ? AND email = ?", tenantID, email).Count(&count).Error; err != nil {
		span.RecordError(err)
		return false, fmt.Errorf("failed to check email exists: %w", err)
	}
	return count > 0, nil
}
