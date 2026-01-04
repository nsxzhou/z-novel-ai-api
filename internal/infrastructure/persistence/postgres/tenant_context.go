// Package postgres 提供 PostgreSQL 数据库访问层实现
package postgres

import (
	"context"
	"database/sql"
	"fmt"
)

// TenantContext 租户上下文管理
type TenantContext struct {
	client *Client
}

// NewTenantContext 创建租户上下文管理器
func NewTenantContext(client *Client) *TenantContext {
	return &TenantContext{client: client}
}

// SetTenant 设置当前租户上下文（用于 RLS）
func (tc *TenantContext) SetTenant(ctx context.Context, tenantID string) error {
	db := getDB(ctx, tc.client.db)
	err := db.Exec("SELECT set_config('app.current_tenant_id', ?, TRUE)", tenantID).Error
	if err != nil {
		return fmt.Errorf("failed to set tenant context: %w", err)
	}
	return nil
}

// GetCurrentTenant 获取当前租户 ID
func (tc *TenantContext) GetCurrentTenant(ctx context.Context) (string, error) {
	db := getDB(ctx, tc.client.db)
	var tenantID sql.NullString
	err := db.Raw("SELECT current_setting('app.current_tenant_id', TRUE)").Scan(&tenantID).Error
	if err != nil {
		return "", fmt.Errorf("failed to get tenant context: %w", err)
	}
	return tenantID.String, nil
}

// ClearTenant 清除租户上下文
func (tc *TenantContext) ClearTenant(ctx context.Context) error {
	db := getDB(ctx, tc.client.db)
	err := db.Exec("SELECT set_config('app.current_tenant_id', '', TRUE)").Error
	if err != nil {
		return fmt.Errorf("failed to clear tenant context: %w", err)
	}
	return nil
}

// WithTenant 在指定租户上下文中执行操作
func (tc *TenantContext) WithTenant(ctx context.Context, tenantID string, fn func(ctx context.Context) error) error {
	// 设置租户上下文
	if err := tc.SetTenant(ctx, tenantID); err != nil {
		return err
	}

	// 执行操作
	return fn(ctx)
}
