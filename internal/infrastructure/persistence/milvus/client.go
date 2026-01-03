// Package milvus 提供 Milvus 向量数据库访问层实现
package milvus

import (
	"context"
	"fmt"

	"github.com/milvus-io/milvus-sdk-go/v2/client"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"z-novel-ai-api/internal/config"
)

var tracer = otel.Tracer("milvus")

// Client Milvus 客户端
type Client struct {
	milvus client.Client
	config *config.MilvusConfig
}

// NewClient 创建 Milvus 客户端
func NewClient(ctx context.Context, cfg *config.MilvusConfig) (*Client, error) {
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	var milvusClient client.Client
	var err error

	if cfg.User != "" && cfg.Password != "" {
		milvusClient, err = client.NewClient(ctx, client.Config{
			Address:  addr,
			Username: cfg.User,
			Password: cfg.Password,
		})
	} else {
		milvusClient, err = client.NewClient(ctx, client.Config{
			Address: addr,
		})
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to milvus: %w", err)
	}

	return &Client{
		milvus: milvusClient,
		config: cfg,
	}, nil
}

// Milvus 获取底层 Milvus 客户端
func (c *Client) Milvus() client.Client {
	return c.milvus
}

// Close 关闭 Milvus 连接
func (c *Client) Close() error {
	return c.milvus.Close()
}

// HealthCheck 健康检查
func (c *Client) HealthCheck(ctx context.Context) error {
	ctx, span := tracer.Start(ctx, "milvus.HealthCheck")
	defer span.End()

	_, err := c.milvus.HasCollection(ctx, "health_check")
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("health check failed: %w", err)
	}
	return nil
}

// CollectionName 获取带前缀的集合名称
func (c *Client) CollectionName(name string) string {
	if c.config.CollectionPrefix != "" {
		return c.config.CollectionPrefix + "_" + name
	}
	return name
}

// HasCollection 检查集合是否存在
func (c *Client) HasCollection(ctx context.Context, name string) (bool, error) {
	ctx, span := tracer.Start(ctx, "milvus.HasCollection",
		trace.WithAttributes(attribute.String("collection", name)))
	defer span.End()

	return c.milvus.HasCollection(ctx, c.CollectionName(name))
}

// LoadCollection 加载集合到内存
func (c *Client) LoadCollection(ctx context.Context, name string) error {
	ctx, span := tracer.Start(ctx, "milvus.LoadCollection",
		trace.WithAttributes(attribute.String("collection", name)))
	defer span.End()

	return c.milvus.LoadCollection(ctx, c.CollectionName(name), false)
}

// ReleaseCollection 释放集合内存
func (c *Client) ReleaseCollection(ctx context.Context, name string) error {
	ctx, span := tracer.Start(ctx, "milvus.ReleaseCollection",
		trace.WithAttributes(attribute.String("collection", name)))
	defer span.End()

	return c.milvus.ReleaseCollection(ctx, c.CollectionName(name))
}
