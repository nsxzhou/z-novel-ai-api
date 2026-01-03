// Package repository 定义数据访问层接口
package repository

import "context"

// TenantContextManager 租户上下文管理接口（用于 PostgreSQL RLS）
type TenantContextManager interface {
	// SetTenant 设置当前租户上下文
	SetTenant(ctx context.Context, tenantID string) error
	// ClearTenant 清除当前租户上下文
	ClearTenant(ctx context.Context) error
}
