// Package repository 定义数据访问层接口
package repository

import (
	"context"
	"time"

	"z-novel-ai-api/internal/domain/entity"
)

type LLMUsageEventRepository interface {
	Create(ctx context.Context, event *entity.LLMUsageEvent) error
	GetTokenUsage(ctx context.Context, tenantID string, startInclusive, endExclusive time.Time) (int64, error)
}

