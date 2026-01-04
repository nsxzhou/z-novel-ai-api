// Package postgres 提供 PostgreSQL Repository 实现
package postgres

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"z-novel-ai-api/internal/domain/entity"
	"z-novel-ai-api/internal/domain/repository"
)

// EntityStateRepository 实体状态仓储实现
type EntityStateRepository struct {
	client *Client
}

// NewEntityStateRepository 创建实体状态仓储
func NewEntityStateRepository(client *Client) *EntityStateRepository {
	return &EntityStateRepository{client: client}
}

// Create 创建实体状态记录
func (r *EntityStateRepository) Create(ctx context.Context, state *entity.EntityState) error {
	ctx, span := tracer.Start(ctx, "postgres.EntityStateRepository.Create")
	defer span.End()

	db := getDB(ctx, r.client.db)
	if err := db.Create(state).Error; err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to create entity state: %w", err)
	}
	return nil
}

// GetByID 根据 ID 获取实体状态
func (r *EntityStateRepository) GetByID(ctx context.Context, id string) (*entity.EntityState, error) {
	ctx, span := tracer.Start(ctx, "postgres.EntityStateRepository.GetByID")
	defer span.End()

	db := getDB(ctx, r.client.db)
	var state entity.EntityState
	if err := db.First(&state, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		span.RecordError(err)
		return nil, fmt.Errorf("failed to get entity state: %w", err)
	}
	return &state, nil
}

// ListByEntity 获取实体的状态历史
func (r *EntityStateRepository) ListByEntity(ctx context.Context, entityID string, pagination repository.Pagination) (*repository.PagedResult[*entity.EntityState], error) {
	ctx, span := tracer.Start(ctx, "postgres.EntityStateRepository.ListByEntity")
	defer span.End()

	db := getDB(ctx, r.client.db)
	query := db.Model(&entity.EntityState{}).Where("entity_id = ?", entityID)

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to count entity states: %w", err)
	}

	// 获取列表
	var states []*entity.EntityState
	if err := query.Order("story_time DESC, created_at DESC").
		Offset(pagination.Offset()).
		Limit(pagination.Limit()).
		Find(&states).Error; err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to list entity states: %w", err)
	}

	return repository.NewPagedResult(states, total, pagination), nil
}

// GetByChapter 获取章节中的状态变更
func (r *EntityStateRepository) GetByChapter(ctx context.Context, chapterID string) ([]*entity.EntityState, error) {
	ctx, span := tracer.Start(ctx, "postgres.EntityStateRepository.GetByChapter")
	defer span.End()

	db := getDB(ctx, r.client.db)
	var states []*entity.EntityState

	if err := db.Where("chapter_id = ?", chapterID).
		Order("story_time ASC, created_at ASC").
		Find(&states).Error; err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to get entity states by chapter: %w", err)
	}

	return states, nil
}

// GetStateAtTime 获取指定时间点的状态
func (r *EntityStateRepository) GetStateAtTime(ctx context.Context, entityID string, storyTime int64) (*entity.EntityState, error) {
	ctx, span := tracer.Start(ctx, "postgres.EntityStateRepository.GetStateAtTime")
	defer span.End()

	db := getDB(ctx, r.client.db)
	var state entity.EntityState
	if err := db.Where("entity_id = ? AND story_time <= ?", entityID, storyTime).
		Order("story_time DESC").
		First(&state).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		span.RecordError(err)
		return nil, fmt.Errorf("failed to get entity state at time: %w", err)
	}
	return &state, nil
}

// GetLatestState 获取最新状态
func (r *EntityStateRepository) GetLatestState(ctx context.Context, entityID string) (*entity.EntityState, error) {
	ctx, span := tracer.Start(ctx, "postgres.EntityStateRepository.GetLatestState")
	defer span.End()

	db := getDB(ctx, r.client.db)
	var state entity.EntityState
	if err := db.Where("entity_id = ?", entityID).
		Order("story_time DESC, created_at DESC").
		First(&state).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		span.RecordError(err)
		return nil, fmt.Errorf("failed to get latest entity state: %w", err)
	}
	return &state, nil
}

// DeleteByEntity 删除实体的所有状态记录
func (r *EntityStateRepository) DeleteByEntity(ctx context.Context, entityID string) error {
	ctx, span := tracer.Start(ctx, "postgres.EntityStateRepository.DeleteByEntity")
	defer span.End()

	db := getDB(ctx, r.client.db)
	if err := db.Delete(&entity.EntityState{}, "entity_id = ?", entityID).Error; err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to delete entity states by entity: %w", err)
	}
	return nil
}
