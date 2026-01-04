// Package postgres 提供 PostgreSQL 数据库访问层实现
package postgres

import (
	"context"

	"gorm.io/gorm"
)

// gormTxKey GORM 事务上下文键
type gormTxKey struct{}

// TxManager 事务管理器
type TxManager struct {
	client *Client
}

// NewTxManager 创建事务管理器
func NewTxManager(client *Client) *TxManager {
	return &TxManager{client: client}
}

// WithTransaction 在事务中执行操作
func (m *TxManager) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	// 检查是否已在事务中
	if tx := GetTxFromContext(ctx); tx != nil {
		// 已在事务中，直接执行
		return fn(ctx)
	}

	// 开始新事务
	return m.client.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txCtx := context.WithValue(ctx, gormTxKey{}, tx)
		return fn(txCtx)
	})
}

// GetTxFromContext 从上下文获取事务
func GetTxFromContext(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(gormTxKey{}).(*gorm.DB); ok {
		return tx
	}
	return nil
}

// getDB 根据上下文获取数据库实例（事务或普通连接）
func getDB(ctx context.Context, db *gorm.DB) *gorm.DB {
	if tx := GetTxFromContext(ctx); tx != nil {
		return tx.WithContext(ctx)
	}
	return db.WithContext(ctx)
}
