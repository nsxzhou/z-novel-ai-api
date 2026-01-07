// Package postgres 提供 PostgreSQL Repository 实现
package postgres

import (
	"context"
	"fmt"
	"time"

	"z-novel-ai-api/internal/domain/entity"
)

type LLMUsageEventRepository struct {
	client *Client
}

func NewLLMUsageEventRepository(client *Client) *LLMUsageEventRepository {
	return &LLMUsageEventRepository{client: client}
}

func (r *LLMUsageEventRepository) Create(ctx context.Context, event *entity.LLMUsageEvent) error {
	ctx, span := tracer.Start(ctx, "postgres.LLMUsageEventRepository.Create")
	defer span.End()

	db := getDB(ctx, r.client.db)
	if err := db.Create(event).Error; err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to create llm usage event: %w", err)
	}
	return nil
}

func (r *LLMUsageEventRepository) GetTokenUsage(ctx context.Context, tenantID string, startInclusive, endExclusive time.Time) (int64, error) {
	ctx, span := tracer.Start(ctx, "postgres.LLMUsageEventRepository.GetTokenUsage")
	defer span.End()

	db := getDB(ctx, r.client.db)

	var total int64
	if err := db.Model(&entity.LLMUsageEvent{}).
		Where("tenant_id = ? AND created_at >= ? AND created_at < ?", tenantID, startInclusive, endExclusive).
		Select("COALESCE(SUM(COALESCE(tokens_prompt,0) + COALESCE(tokens_completion,0)),0)").
		Scan(&total).Error; err != nil {
		span.RecordError(err)
		return 0, fmt.Errorf("failed to get llm usage: %w", err)
	}
	return total, nil
}

