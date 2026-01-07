// Package postgres 提供 PostgreSQL Repository 实现
package postgres

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"z-novel-ai-api/internal/domain/entity"
	"z-novel-ai-api/internal/domain/repository"
)

// EntityRepository 实体仓储实现
type EntityRepository struct {
	client *Client
}

// NewEntityRepository 创建实体仓储
func NewEntityRepository(client *Client) *EntityRepository {
	return &EntityRepository{client: client}
}

// Create 创建实体
func (r *EntityRepository) Create(ctx context.Context, ent *entity.StoryEntity) error {
	ctx, span := tracer.Start(ctx, "postgres.EntityRepository.Create")
	defer span.End()

	db := getDB(ctx, r.client.db)
	if err := db.Create(ent).Error; err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to create entity: %w", err)
	}
	return nil
}

// GetByID 根据 ID 获取实体
func (r *EntityRepository) GetByID(ctx context.Context, id string) (*entity.StoryEntity, error) {
	ctx, span := tracer.Start(ctx, "postgres.EntityRepository.GetByID")
	defer span.End()

	db := getDB(ctx, r.client.db)
	var ent entity.StoryEntity
	if err := db.First(&ent, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		span.RecordError(err)
		return nil, fmt.Errorf("failed to get entity: %w", err)
	}
	return &ent, nil
}

// Update 更新实体
func (r *EntityRepository) Update(ctx context.Context, ent *entity.StoryEntity) error {
	ctx, span := tracer.Start(ctx, "postgres.EntityRepository.Update")
	defer span.End()

	db := getDB(ctx, r.client.db)
	if err := db.Save(ent).Error; err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to update entity: %w", err)
	}
	return nil
}

// Delete 删除实体
func (r *EntityRepository) Delete(ctx context.Context, id string) error {
	ctx, span := tracer.Start(ctx, "postgres.EntityRepository.Delete")
	defer span.End()

	db := getDB(ctx, r.client.db)
	if err := db.Delete(&entity.StoryEntity{}, "id = ?", id).Error; err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to delete entity: %w", err)
	}
	return nil
}

// ListByProject 获取项目的实体列表
func (r *EntityRepository) ListByProject(ctx context.Context, projectID string, filter *repository.EntityFilter, pagination repository.Pagination) (*repository.PagedResult[*entity.StoryEntity], error) {
	ctx, span := tracer.Start(ctx, "postgres.EntityRepository.ListByProject")
	defer span.End()

	db := getDB(ctx, r.client.db)
	query := db.Model(&entity.StoryEntity{}).Where("project_id = ?", projectID)

	// 应用过滤条件
	if filter != nil {
		if filter.Type != "" {
			query = query.Where("type = ?", filter.Type)
		}
		if filter.Importance != "" {
			query = query.Where("importance = ?", filter.Importance)
		}
		if filter.Name != "" {
			query = query.Where("name ILIKE ?", "%"+filter.Name+"%")
		}
	}

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to count entities: %w", err)
	}

	// 获取列表
	var entities []*entity.StoryEntity
	if err := query.Order("importance ASC, created_at DESC").
		Offset(pagination.Offset()).
		Limit(pagination.Limit()).
		Find(&entities).Error; err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to list entities: %w", err)
	}

	return repository.NewPagedResult(entities, total, pagination), nil
}

// GetByAIKey 根据 AIKey 获取实体（用于 AI 生成对象的稳定映射）
func (r *EntityRepository) GetByAIKey(ctx context.Context, projectID, aiKey string) (*entity.StoryEntity, error) {
	ctx, span := tracer.Start(ctx, "postgres.EntityRepository.GetByAIKey")
	defer span.End()

	db := getDB(ctx, r.client.db)
	var ent entity.StoryEntity
	if err := db.First(&ent, "project_id = ? AND ai_key = ?", projectID, aiKey).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		span.RecordError(err)
		return nil, fmt.Errorf("failed to get entity by ai_key: %w", err)
	}
	return &ent, nil
}

// SearchByName 搜索实体名称（支持别名）
func (r *EntityRepository) SearchByName(ctx context.Context, projectID, query string, limit int) ([]*entity.StoryEntity, error) {
	ctx, span := tracer.Start(ctx, "postgres.EntityRepository.SearchByName")
	defer span.End()

	db := getDB(ctx, r.client.db)
	var entities []*entity.StoryEntity

	// 搜索名称或别名
	searchPattern := "%" + query + "%"
	if err := db.Where("project_id = ? AND (name ILIKE ? OR aliases::text ILIKE ?)", projectID, searchPattern, searchPattern).
		Limit(limit).
		Find(&entities).Error; err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to search entities by name: %w", err)
	}

	return entities, nil
}

// UpdateState 更新实体状态
func (r *EntityRepository) UpdateState(ctx context.Context, id, state string) error {
	ctx, span := tracer.Start(ctx, "postgres.EntityRepository.UpdateState")
	defer span.End()

	db := getDB(ctx, r.client.db)
	if err := db.Model(&entity.StoryEntity{}).Where("id = ?", id).Update("current_state", state).Error; err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to update entity state: %w", err)
	}
	return nil
}

// UpdateVectorID 更新向量 ID
func (r *EntityRepository) UpdateVectorID(ctx context.Context, id, vectorID string) error {
	ctx, span := tracer.Start(ctx, "postgres.EntityRepository.UpdateVectorID")
	defer span.End()

	db := getDB(ctx, r.client.db)
	if err := db.Model(&entity.StoryEntity{}).Where("id = ?", id).Update("vector_id", vectorID).Error; err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to update vector id: %w", err)
	}
	return nil
}

// RecordAppearance 记录出场
func (r *EntityRepository) RecordAppearance(ctx context.Context, id, chapterID string) error {
	ctx, span := tracer.Start(ctx, "postgres.EntityRepository.RecordAppearance")
	defer span.End()

	db := getDB(ctx, r.client.db)

	// 更新出场次数和最后出场章节
	updates := map[string]interface{}{
		"appear_count":           gorm.Expr("appear_count + 1"),
		"last_appear_chapter_id": chapterID,
	}

	// 如果是首次出场，同时设置 first_appear_chapter_id
	var ent entity.StoryEntity
	db.Select("first_appear_chapter_id").First(&ent, "id = ?", id)
	if ent.FirstAppearChapterID == "" {
		updates["first_appear_chapter_id"] = chapterID
	}

	if err := db.Model(&entity.StoryEntity{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to record appearance: %w", err)
	}
	return nil
}

// GetByType 根据类型获取实体列表
func (r *EntityRepository) GetByType(ctx context.Context, projectID string, entityType entity.StoryEntityType) ([]*entity.StoryEntity, error) {
	ctx, span := tracer.Start(ctx, "postgres.EntityRepository.GetByType")
	defer span.End()

	db := getDB(ctx, r.client.db)
	var entities []*entity.StoryEntity

	if err := db.Where("project_id = ? AND type = ?", projectID, entityType).
		Order("importance ASC, created_at DESC").
		Find(&entities).Error; err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to get entities by type: %w", err)
	}

	return entities, nil
}

// GetProtagonists 获取主角列表
func (r *EntityRepository) GetProtagonists(ctx context.Context, projectID string) ([]*entity.StoryEntity, error) {
	ctx, span := tracer.Start(ctx, "postgres.EntityRepository.GetProtagonists")
	defer span.End()

	db := getDB(ctx, r.client.db)
	var entities []*entity.StoryEntity

	if err := db.Where("project_id = ? AND importance = ?", projectID, entity.ImportanceProtagonist).
		Find(&entities).Error; err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to get protagonists: %w", err)
	}

	return entities, nil
}
