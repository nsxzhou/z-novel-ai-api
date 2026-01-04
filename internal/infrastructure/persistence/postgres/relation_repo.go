// Package postgres 提供 PostgreSQL Repository 实现
package postgres

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"z-novel-ai-api/internal/domain/entity"
	"z-novel-ai-api/internal/domain/repository"
)

// RelationRepository 关系仓储实现
type RelationRepository struct {
	client *Client
}

// NewRelationRepository 创建关系仓储
func NewRelationRepository(client *Client) *RelationRepository {
	return &RelationRepository{client: client}
}

// Create 创建关系
func (r *RelationRepository) Create(ctx context.Context, relation *entity.Relation) error {
	ctx, span := tracer.Start(ctx, "postgres.RelationRepository.Create")
	defer span.End()

	db := getDB(ctx, r.client.db)
	if err := db.Create(relation).Error; err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to create relation: %w", err)
	}
	return nil
}

// GetByID 根据 ID 获取关系
func (r *RelationRepository) GetByID(ctx context.Context, id string) (*entity.Relation, error) {
	ctx, span := tracer.Start(ctx, "postgres.RelationRepository.GetByID")
	defer span.End()

	db := getDB(ctx, r.client.db)
	var relation entity.Relation
	if err := db.First(&relation, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		span.RecordError(err)
		return nil, fmt.Errorf("failed to get relation: %w", err)
	}
	return &relation, nil
}

// Update 更新关系
func (r *RelationRepository) Update(ctx context.Context, relation *entity.Relation) error {
	ctx, span := tracer.Start(ctx, "postgres.RelationRepository.Update")
	defer span.End()

	db := getDB(ctx, r.client.db)
	if err := db.Save(relation).Error; err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to update relation: %w", err)
	}
	return nil
}

// Delete 删除关系
func (r *RelationRepository) Delete(ctx context.Context, id string) error {
	ctx, span := tracer.Start(ctx, "postgres.RelationRepository.Delete")
	defer span.End()

	db := getDB(ctx, r.client.db)
	if err := db.Delete(&entity.Relation{}, "id = ?", id).Error; err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to delete relation: %w", err)
	}
	return nil
}

// ListByProject 获取项目的关系列表
func (r *RelationRepository) ListByProject(ctx context.Context, projectID string, filter *repository.RelationFilter, pagination repository.Pagination) (*repository.PagedResult[*entity.Relation], error) {
	ctx, span := tracer.Start(ctx, "postgres.RelationRepository.ListByProject")
	defer span.End()

	db := getDB(ctx, r.client.db)
	query := db.Model(&entity.Relation{}).Where("project_id = ?", projectID)

	// 应用过滤条件
	if filter != nil {
		if filter.RelationType != "" {
			query = query.Where("relation_type = ?", filter.RelationType)
		}
		if filter.MinStrength > 0 {
			query = query.Where("strength >= ?", filter.MinStrength)
		}
	}

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to count relations: %w", err)
	}

	// 获取列表
	var relations []*entity.Relation
	if err := query.Order("created_at DESC").
		Offset(pagination.Offset()).
		Limit(pagination.Limit()).
		Find(&relations).Error; err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to list relations: %w", err)
	}

	return repository.NewPagedResult(relations, total, pagination), nil
}

// GetByEntities 根据两个实体获取关系
func (r *RelationRepository) GetByEntities(ctx context.Context, projectID, sourceID, targetID string) (*entity.Relation, error) {
	ctx, span := tracer.Start(ctx, "postgres.RelationRepository.GetByEntities")
	defer span.End()

	db := getDB(ctx, r.client.db)
	var relation entity.Relation
	if err := db.First(&relation, "project_id = ? AND source_entity_id = ? AND target_entity_id = ?", projectID, sourceID, targetID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		span.RecordError(err)
		return nil, fmt.Errorf("failed to get relation by entities: %w", err)
	}
	return &relation, nil
}

// ListBySourceEntity 获取源实体的关系列表
func (r *RelationRepository) ListBySourceEntity(ctx context.Context, entityID string) ([]*entity.Relation, error) {
	ctx, span := tracer.Start(ctx, "postgres.RelationRepository.ListBySourceEntity")
	defer span.End()

	db := getDB(ctx, r.client.db)
	var relations []*entity.Relation

	if err := db.Where("source_entity_id = ?", entityID).
		Order("strength DESC").
		Find(&relations).Error; err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to list relations by source entity: %w", err)
	}

	return relations, nil
}

// ListByTargetEntity 获取目标实体的关系列表
func (r *RelationRepository) ListByTargetEntity(ctx context.Context, entityID string) ([]*entity.Relation, error) {
	ctx, span := tracer.Start(ctx, "postgres.RelationRepository.ListByTargetEntity")
	defer span.End()

	db := getDB(ctx, r.client.db)
	var relations []*entity.Relation

	if err := db.Where("target_entity_id = ?", entityID).
		Order("strength DESC").
		Find(&relations).Error; err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to list relations by target entity: %w", err)
	}

	return relations, nil
}

// ListByEntity 获取实体相关的所有关系
func (r *RelationRepository) ListByEntity(ctx context.Context, entityID string) ([]*entity.Relation, error) {
	ctx, span := tracer.Start(ctx, "postgres.RelationRepository.ListByEntity")
	defer span.End()

	db := getDB(ctx, r.client.db)
	var relations []*entity.Relation

	if err := db.Where("source_entity_id = ? OR target_entity_id = ?", entityID, entityID).
		Order("strength DESC").
		Find(&relations).Error; err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to list relations by entity: %w", err)
	}

	return relations, nil
}

// UpdateStrength 更新关系强度
func (r *RelationRepository) UpdateStrength(ctx context.Context, id string, strength float64) error {
	ctx, span := tracer.Start(ctx, "postgres.RelationRepository.UpdateStrength")
	defer span.End()

	db := getDB(ctx, r.client.db)
	if err := db.Model(&entity.Relation{}).Where("id = ?", id).Update("strength", strength).Error; err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to update relation strength: %w", err)
	}
	return nil
}

// DeleteByEntity 删除实体相关的所有关系
func (r *RelationRepository) DeleteByEntity(ctx context.Context, entityID string) error {
	ctx, span := tracer.Start(ctx, "postgres.RelationRepository.DeleteByEntity")
	defer span.End()

	db := getDB(ctx, r.client.db)
	if err := db.Delete(&entity.Relation{}, "source_entity_id = ? OR target_entity_id = ?", entityID, entityID).Error; err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to delete relations by entity: %w", err)
	}
	return nil
}

// GetRelationGraph 获取关系图谱
func (r *RelationRepository) GetRelationGraph(ctx context.Context, projectID string, entityIDs []string) ([]*entity.Relation, error) {
	ctx, span := tracer.Start(ctx, "postgres.RelationRepository.GetRelationGraph")
	defer span.End()

	db := getDB(ctx, r.client.db)
	var relations []*entity.Relation

	if err := db.Where("project_id = ? AND (source_entity_id IN ? OR target_entity_id IN ?)", projectID, entityIDs, entityIDs).
		Order("strength DESC").
		Find(&relations).Error; err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to get relation graph: %w", err)
	}

	return relations, nil
}
