// Package repository 定义数据访问层接口
package repository

import (
	"context"

	"z-novel-ai-api/internal/domain/entity"
)

// UserRepository 用户仓储接口
type UserRepository interface {
	// Create 创建用户
	Create(ctx context.Context, user *entity.User) error

	// GetByID 根据 ID 获取用户
	GetByID(ctx context.Context, id string) (*entity.User, error)

	// GetByEmail 根据邮箱获取用户
	GetByEmail(ctx context.Context, tenantID, email string) (*entity.User, error)

	// GetByExternalID 根据外部 ID 获取用户
	GetByExternalID(ctx context.Context, externalID string) (*entity.User, error)

	// Update 更新用户
	Update(ctx context.Context, user *entity.User) error

	// Delete 删除用户
	Delete(ctx context.Context, id string) error

	// ListByTenant 获取租户用户列表
	ListByTenant(ctx context.Context, tenantID string, pagination Pagination) (*PagedResult[*entity.User], error)

	// UpdateRole 更新用户角色
	UpdateRole(ctx context.Context, id string, role entity.UserRole) error

	// UpdateLastLogin 更新最后登录时间
	UpdateLastLogin(ctx context.Context, id string) error

	// ExistsByEmail 检查邮箱是否存在
	ExistsByEmail(ctx context.Context, tenantID, email string) (bool, error)
}
