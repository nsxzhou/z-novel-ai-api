// Package postgres 提供 PostgreSQL 数据库访问层实现
package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"z-novel-ai-api/internal/domain/repository"
)

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
	if tx := getTxFromContext(ctx); tx != nil {
		// 已在事务中，直接执行
		return fn(ctx)
	}

	// 开始新事务
	tx, err := m.client.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// 将事务放入上下文
	txCtx := context.WithValue(ctx, repository.TxKey{}, tx)

	// 执行操作
	if err := fn(txCtx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("rollback failed: %v, original error: %w", rbErr, err)
		}
		return err
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// getTxFromContext 从上下文获取事务
func getTxFromContext(ctx context.Context) *sql.Tx {
	if tx, ok := ctx.Value(repository.TxKey{}).(*sql.Tx); ok {
		return tx
	}
	return nil
}

// Querier 查询接口（支持普通连接和事务）
type Querier interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

// getQuerier 根据上下文获取查询器
func getQuerier(ctx context.Context, db *sql.DB) Querier {
	if tx := getTxFromContext(ctx); tx != nil {
		return tx
	}
	return db
}
