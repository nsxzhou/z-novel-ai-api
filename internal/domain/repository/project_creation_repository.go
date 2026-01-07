// Package repository 定义数据访问层接口
package repository

import (
	"context"

	"z-novel-ai-api/internal/domain/entity"
)

type ProjectCreationSessionRepository interface {
	Create(ctx context.Context, session *entity.ProjectCreationSession) error
	GetByID(ctx context.Context, id string) (*entity.ProjectCreationSession, error)
	GetByIDForUpdate(ctx context.Context, id string) (*entity.ProjectCreationSession, error)
	Update(ctx context.Context, session *entity.ProjectCreationSession) error
}

type ProjectCreationTurnRepository interface {
	Create(ctx context.Context, turn *entity.ProjectCreationTurn) error
	ListBySession(ctx context.Context, sessionID string, pagination Pagination) (*PagedResult[*entity.ProjectCreationTurn], error)
}

