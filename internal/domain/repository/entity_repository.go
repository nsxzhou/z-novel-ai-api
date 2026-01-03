// Package repository 定义数据访问层接口
package repository

import (
	"context"

	"z-novel-ai-api/internal/domain/entity"
)

// EntityFilter 实体过滤条件
type EntityFilter struct {
	Type       entity.StoryEntityType
	Importance entity.EntityImportance
	Name       string
}

// EntityRepository 实体仓储接口
type EntityRepository interface {
	// Create 创建实体
	Create(ctx context.Context, storyEntity *entity.StoryEntity) error

	// GetByID 根据 ID 获取实体
	GetByID(ctx context.Context, id string) (*entity.StoryEntity, error)

	// Update 更新实体
	Update(ctx context.Context, storyEntity *entity.StoryEntity) error

	// Delete 删除实体
	Delete(ctx context.Context, id string) error

	// ListByProject 获取项目实体列表
	ListByProject(ctx context.Context, projectID string, filter *EntityFilter, pagination Pagination) (*PagedResult[*entity.StoryEntity], error)

	// GetByName 根据名称获取实体
	GetByName(ctx context.Context, projectID, name string) (*entity.StoryEntity, error)

	// SearchByName 搜索实体名称（支持别名）
	SearchByName(ctx context.Context, projectID, query string, limit int) ([]*entity.StoryEntity, error)

	// UpdateState 更新实体状态
	UpdateState(ctx context.Context, id, state string) error

	// UpdateVectorID 更新向量 ID
	UpdateVectorID(ctx context.Context, id, vectorID string) error

	// RecordAppearance 记录出场
	RecordAppearance(ctx context.Context, id, chapterID string) error

	// GetByType 根据类型获取实体列表
	GetByType(ctx context.Context, projectID string, entityType entity.StoryEntityType) ([]*entity.StoryEntity, error)

	// GetProtagonists 获取主角列表
	GetProtagonists(ctx context.Context, projectID string) ([]*entity.StoryEntity, error)
}

// EntityStateRepository 实体状态历史仓储接口
type EntityStateRepository interface {
	// Create 创建状态记录
	Create(ctx context.Context, state *entity.EntityState) error

	// GetByID 根据 ID 获取状态记录
	GetByID(ctx context.Context, id string) (*entity.EntityState, error)

	// ListByEntity 获取实体状态历史
	ListByEntity(ctx context.Context, entityID string, pagination Pagination) (*PagedResult[*entity.EntityState], error)

	// GetByChapter 获取章节中的状态变更
	GetByChapter(ctx context.Context, chapterID string) ([]*entity.EntityState, error)

	// GetStateAtTime 获取指定时间点的状态
	GetStateAtTime(ctx context.Context, entityID string, storyTime int64) (*entity.EntityState, error)

	// GetLatestState 获取最新状态
	GetLatestState(ctx context.Context, entityID string) (*entity.EntityState, error)

	// DeleteByEntity 删除实体所有状态记录
	DeleteByEntity(ctx context.Context, entityID string) error
}
