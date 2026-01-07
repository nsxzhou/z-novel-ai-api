// Package repository 定义数据访问层接口
package repository

import (
	"context"

	"z-novel-ai-api/internal/domain/entity"
)

type ConversationSessionRepository interface {
	Create(ctx context.Context, session *entity.ConversationSession) error
	GetByID(ctx context.Context, id string) (*entity.ConversationSession, error)
	GetByIDForUpdate(ctx context.Context, id string) (*entity.ConversationSession, error)
	Update(ctx context.Context, session *entity.ConversationSession) error
	ListByProject(ctx context.Context, projectID string, pagination Pagination) (*PagedResult[*entity.ConversationSession], error)
}

type ConversationTurnRepository interface {
	Create(ctx context.Context, turn *entity.ConversationTurn) error
	ListBySession(ctx context.Context, sessionID string, pagination Pagination) (*PagedResult[*entity.ConversationTurn], error)
}
