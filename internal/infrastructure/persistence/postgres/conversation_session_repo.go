// Package postgres 提供 PostgreSQL Repository 实现
package postgres

import (
	"context"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"z-novel-ai-api/internal/domain/entity"
	"z-novel-ai-api/internal/domain/repository"
)

type ConversationSessionRepository struct {
	client *Client
}

func NewConversationSessionRepository(client *Client) *ConversationSessionRepository {
	return &ConversationSessionRepository{client: client}
}

func (r *ConversationSessionRepository) Create(ctx context.Context, session *entity.ConversationSession) error {
	ctx, span := tracer.Start(ctx, "postgres.ConversationSessionRepository.Create")
	defer span.End()

	db := getDB(ctx, r.client.db)
	if err := db.Create(session).Error; err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to create conversation session: %w", err)
	}
	return nil
}

func (r *ConversationSessionRepository) GetByID(ctx context.Context, id string) (*entity.ConversationSession, error) {
	ctx, span := tracer.Start(ctx, "postgres.ConversationSessionRepository.GetByID")
	defer span.End()

	db := getDB(ctx, r.client.db)
	var session entity.ConversationSession
	if err := db.First(&session, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		span.RecordError(err)
		return nil, fmt.Errorf("failed to get conversation session: %w", err)
	}
	return &session, nil
}

func (r *ConversationSessionRepository) GetByIDForUpdate(ctx context.Context, id string) (*entity.ConversationSession, error) {
	ctx, span := tracer.Start(ctx, "postgres.ConversationSessionRepository.GetByIDForUpdate")
	defer span.End()

	db := getDB(ctx, r.client.db).Clauses(clause.Locking{Strength: "UPDATE"})
	var session entity.ConversationSession
	if err := db.First(&session, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		span.RecordError(err)
		return nil, fmt.Errorf("failed to get conversation session for update: %w", err)
	}
	return &session, nil
}

func (r *ConversationSessionRepository) Update(ctx context.Context, session *entity.ConversationSession) error {
	ctx, span := tracer.Start(ctx, "postgres.ConversationSessionRepository.Update")
	defer span.End()

	db := getDB(ctx, r.client.db)
	if err := db.Save(session).Error; err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to update conversation session: %w", err)
	}
	return nil
}

func (r *ConversationSessionRepository) ListByProject(ctx context.Context, projectID string, pagination repository.Pagination) (*repository.PagedResult[*entity.ConversationSession], error) {
	ctx, span := tracer.Start(ctx, "postgres.ConversationSessionRepository.ListByProject")
	defer span.End()

	db := getDB(ctx, r.client.db)
	query := db.Model(&entity.ConversationSession{}).Where("project_id = ?", projectID)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to count conversation sessions: %w", err)
	}

	var sessions []*entity.ConversationSession
	if err := query.Order("created_at DESC").
		Offset(pagination.Offset()).
		Limit(pagination.Limit()).
		Find(&sessions).Error; err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to list conversation sessions: %w", err)
	}

	return repository.NewPagedResult(sessions, total, pagination), nil
}
