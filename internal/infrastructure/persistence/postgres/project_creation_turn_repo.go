// Package postgres 提供 PostgreSQL Repository 实现
package postgres

import (
	"context"
	"fmt"

	"z-novel-ai-api/internal/domain/entity"
	"z-novel-ai-api/internal/domain/repository"
)

type ProjectCreationTurnRepository struct {
	client *Client
}

func NewProjectCreationTurnRepository(client *Client) *ProjectCreationTurnRepository {
	return &ProjectCreationTurnRepository{client: client}
}

func (r *ProjectCreationTurnRepository) Create(ctx context.Context, turn *entity.ProjectCreationTurn) error {
	ctx, span := tracer.Start(ctx, "postgres.ProjectCreationTurnRepository.Create")
	defer span.End()

	db := getDB(ctx, r.client.db)
	if err := db.Create(turn).Error; err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to create project creation turn: %w", err)
	}
	return nil
}

func (r *ProjectCreationTurnRepository) ListBySession(ctx context.Context, sessionID string, pagination repository.Pagination) (*repository.PagedResult[*entity.ProjectCreationTurn], error) {
	ctx, span := tracer.Start(ctx, "postgres.ProjectCreationTurnRepository.ListBySession")
	defer span.End()

	db := getDB(ctx, r.client.db)
	query := db.Model(&entity.ProjectCreationTurn{}).Where("session_id = ?", sessionID)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to count project creation turns: %w", err)
	}

	var turns []*entity.ProjectCreationTurn
	if err := query.Order("created_at ASC").
		Offset(pagination.Offset()).
		Limit(pagination.Limit()).
		Find(&turns).Error; err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to list project creation turns: %w", err)
	}

	return repository.NewPagedResult(turns, total, pagination), nil
}

