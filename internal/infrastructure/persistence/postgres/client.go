// Package postgres 提供 PostgreSQL 数据库访问层实现
package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"z-novel-ai-api/internal/config"
)

var tracer = otel.Tracer("postgres")

// Client PostgreSQL 客户端
type Client struct {
	db     *sql.DB
	config *config.PostgresConfig
}

// NewClient 创建 PostgreSQL 客户端
func NewClient(cfg *config.PostgresConfig) (*Client, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Database, cfg.SSLMode,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// 配置连接池
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	// 验证连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Client{
		db:     db,
		config: cfg,
	}, nil
}

// DB 获取底层数据库连接
func (c *Client) DB() *sql.DB {
	return c.db
}

// Close 关闭数据库连接
func (c *Client) Close() error {
	return c.db.Close()
}

// Ping 检查数据库连接
func (c *Client) Ping(ctx context.Context) error {
	ctx, span := tracer.Start(ctx, "postgres.Ping")
	defer span.End()

	return c.db.PingContext(ctx)
}

// Stats 获取连接池统计信息
func (c *Client) Stats() sql.DBStats {
	return c.db.Stats()
}

// HealthCheck 健康检查
func (c *Client) HealthCheck(ctx context.Context) error {
	ctx, span := tracer.Start(ctx, "postgres.HealthCheck")
	defer span.End()

	var result int
	err := c.db.QueryRowContext(ctx, "SELECT 1").Scan(&result)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("health check failed: %w", err)
	}
	return nil
}

// ExecContext 执行 SQL（带追踪）
func (c *Client) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	ctx, span := tracer.Start(ctx, "postgres.Exec",
		trace.WithAttributes(attribute.String("db.statement", query)))
	defer span.End()

	result, err := c.db.ExecContext(ctx, query, args...)
	if err != nil {
		span.RecordError(err)
	}
	return result, err
}

// QueryContext 查询 SQL（带追踪）
func (c *Client) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	ctx, span := tracer.Start(ctx, "postgres.Query",
		trace.WithAttributes(attribute.String("db.statement", query)))
	defer span.End()

	rows, err := c.db.QueryContext(ctx, query, args...)
	if err != nil {
		span.RecordError(err)
	}
	return rows, err
}

// QueryRowContext 查询单行 SQL（带追踪）
func (c *Client) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	_, span := tracer.Start(ctx, "postgres.QueryRow",
		trace.WithAttributes(attribute.String("db.statement", query)))
	defer span.End()

	return c.db.QueryRowContext(ctx, query, args...)
}
