// Package postgres 提供 PostgreSQL 数据库访问层实现
package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"z-novel-ai-api/internal/config"
)

var tracer = otel.Tracer("postgres")

// Client PostgreSQL 客户端（GORM 版本）
type Client struct {
	db     *gorm.DB
	config *config.PostgresConfig
}

// NewClient 创建 PostgreSQL 客户端
func NewClient(cfg *config.PostgresConfig) (*Client, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Database, cfg.SSLMode,
	)

	// 配置 GORM 日志
	gormLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Info,
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		},
	)

	gormConfig := &gorm.Config{
		Logger: gormLogger,
	}

	db, err := gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// 配置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	// 验证连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Client{
		db:     db,
		config: cfg,
	}, nil
}

// DB 获取 GORM DB 实例
func (c *Client) DB() *gorm.DB {
	return c.db
}

// SqlDB 获取底层 sql.DB（用于健康检查等）
func (c *Client) SqlDB() (*sql.DB, error) {
	return c.db.DB()
}

// Close 关闭数据库连接
func (c *Client) Close() error {
	sqlDB, err := c.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// Ping 检查数据库连接
func (c *Client) Ping(ctx context.Context) error {
	ctx, span := tracer.Start(ctx, "postgres.Ping")
	defer span.End()

	sqlDB, err := c.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

// Stats 获取连接池统计信息
func (c *Client) Stats() (sql.DBStats, error) {
	sqlDB, err := c.db.DB()
	if err != nil {
		return sql.DBStats{}, err
	}
	return sqlDB.Stats(), nil
}

// HealthCheck 健康检查
func (c *Client) HealthCheck(ctx context.Context) error {
	ctx, span := tracer.Start(ctx, "postgres.HealthCheck")
	defer span.End()

	var result int
	err := c.db.WithContext(ctx).Raw("SELECT 1").Scan(&result).Error
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("health check failed: %w", err)
	}
	return nil
}
