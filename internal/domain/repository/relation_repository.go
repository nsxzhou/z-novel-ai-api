// Package repository 定义数据访问层接口
package repository

import (
	"context"

	"z-novel-ai-api/internal/domain/entity"
)

// RelationFilter 关系过滤条件
type RelationFilter struct {
	RelationType entity.RelationType
	MinStrength  float64
}

// RelationRepository 关系仓储接口
type RelationRepository interface {
	// Create 创建关系
	Create(ctx context.Context, relation *entity.Relation) error

	// GetByID 根据 ID 获取关系
	GetByID(ctx context.Context, id string) (*entity.Relation, error)

	// Update 更新关系
	Update(ctx context.Context, relation *entity.Relation) error

	// Delete 删除关系
	Delete(ctx context.Context, id string) error

	// ListByProject 获取项目关系列表
	ListByProject(ctx context.Context, projectID string, filter *RelationFilter, pagination Pagination) (*PagedResult[*entity.Relation], error)

	// GetByEntities 根据实体对获取关系
	GetByEntities(ctx context.Context, projectID, sourceID, targetID string) (*entity.Relation, error)

	// ListBySourceEntity 获取源实体的关系列表
	ListBySourceEntity(ctx context.Context, entityID string) ([]*entity.Relation, error)

	// ListByTargetEntity 获取目标实体的关系列表
	ListByTargetEntity(ctx context.Context, entityID string) ([]*entity.Relation, error)

	// ListByEntity 获取实体的所有关系（包括源和目标）
	ListByEntity(ctx context.Context, entityID string) ([]*entity.Relation, error)

	// UpdateStrength 更新关系强度
	UpdateStrength(ctx context.Context, id string, strength float64) error

	// DeleteByEntity 删除实体相关的所有关系
	DeleteByEntity(ctx context.Context, entityID string) error

	// GetRelationGraph 获取关系图谱
	GetRelationGraph(ctx context.Context, projectID string, entityIDs []string) ([]*entity.Relation, error)
}
