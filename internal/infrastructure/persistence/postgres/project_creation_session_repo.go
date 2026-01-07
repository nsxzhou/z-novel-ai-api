// Package postgres 提供 PostgreSQL Repository 实现
package postgres

import (
	"context"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"z-novel-ai-api/internal/domain/entity"
)

type ProjectCreationSessionRepository struct {
	client *Client
}

func NewProjectCreationSessionRepository(client *Client) *ProjectCreationSessionRepository {
	return &ProjectCreationSessionRepository{client: client}
}

func (r *ProjectCreationSessionRepository) Create(ctx context.Context, session *entity.ProjectCreationSession) error {
	ctx, span := tracer.Start(ctx, "postgres.ProjectCreationSessionRepository.Create")
	defer span.End()

	db := getDB(ctx, r.client.db)
	if err := db.Create(session).Error; err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to create project creation session: %w", err)
	}
	return nil
}

func (r *ProjectCreationSessionRepository) GetByID(ctx context.Context, id string) (*entity.ProjectCreationSession, error) {
	ctx, span := tracer.Start(ctx, "postgres.ProjectCreationSessionRepository.GetByID")
	defer span.End()

	db := getDB(ctx, r.client.db)
	var session entity.ProjectCreationSession
	if err := db.First(&session, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		span.RecordError(err)
		return nil, fmt.Errorf("failed to get project creation session: %w", err)
	}
	return &session, nil
}

func (r *ProjectCreationSessionRepository) GetByIDForUpdate(ctx context.Context, id string) (*entity.ProjectCreationSession, error) {
	ctx, span := tracer.Start(ctx, "postgres.ProjectCreationSessionRepository.GetByIDForUpdate")
	defer span.End()

	db := getDB(ctx, r.client.db).Clauses(clause.Locking{Strength: "UPDATE"})
	var session entity.ProjectCreationSession
	if err := db.First(&session, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		span.RecordError(err)
		return nil, fmt.Errorf("failed to get project creation session for update: %w", err)
	}
	return &session, nil
}

func (r *ProjectCreationSessionRepository) Update(ctx context.Context, session *entity.ProjectCreationSession) error {
	ctx, span := tracer.Start(ctx, "postgres.ProjectCreationSessionRepository.Update")
	defer span.End()

	db := getDB(ctx, r.client.db)
	if err := db.Save(session).Error; err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to update project creation session: %w", err)
	}
	return nil
}

