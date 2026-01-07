// Package postgres 提供 PostgreSQL Repository 实现
package postgres

import (
	"context"
	"fmt"

	"z-novel-ai-api/internal/domain/entity"
	"z-novel-ai-api/internal/domain/repository"
)

type ConversationTurnRepository struct {
	client *Client
}

func NewConversationTurnRepository(client *Client) *ConversationTurnRepository {
	return &ConversationTurnRepository{client: client}
}

func (r *ConversationTurnRepository) Create(ctx context.Context, turn *entity.ConversationTurn) error {
	ctx, span := tracer.Start(ctx, "postgres.ConversationTurnRepository.Create")
	defer span.End()

	db := getDB(ctx, r.client.db)
	if err := db.Create(turn).Error; err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to create conversation turn: %w", err)
	}
	return nil
}

func (r *ConversationTurnRepository) ListBySession(ctx context.Context, sessionID string, pagination repository.Pagination) (*repository.PagedResult[*entity.ConversationTurn], error) {
	ctx, span := tracer.Start(ctx, "postgres.ConversationTurnRepository.ListBySession")
	defer span.End()

	db := getDB(ctx, r.client.db)
	query := db.Model(&entity.ConversationTurn{}).Where("session_id = ?", sessionID)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to count conversation turns: %w", err)
	}

	var turns []*entity.ConversationTurn
	if err := query.Order("created_at ASC").
		Offset(pagination.Offset()).
		Limit(pagination.Limit()).
		Find(&turns).Error; err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to list conversation turns: %w", err)
	}

	return repository.NewPagedResult(turns, total, pagination), nil
}
