// Package postgres 提供 PostgreSQL Repository 实现
package postgres

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"z-novel-ai-api/internal/domain/entity"
	"z-novel-ai-api/internal/domain/repository"
)

// EventRepository 事件仓储实现
type EventRepository struct {
	client *Client
}

// NewEventRepository 创建事件仓储
func NewEventRepository(client *Client) *EventRepository {
	return &EventRepository{client: client}
}

// Create 创建事件
func (r *EventRepository) Create(ctx context.Context, event *entity.Event) error {
	ctx, span := tracer.Start(ctx, "postgres.EventRepository.Create")
	defer span.End()

	db := getDB(ctx, r.client.db)
	if err := db.Create(event).Error; err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to create event: %w", err)
	}
	return nil
}

// GetByID 根据 ID 获取事件
func (r *EventRepository) GetByID(ctx context.Context, id string) (*entity.Event, error) {
	ctx, span := tracer.Start(ctx, "postgres.EventRepository.GetByID")
	defer span.End()

	db := getDB(ctx, r.client.db)
	var event entity.Event
	if err := db.First(&event, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		span.RecordError(err)
		return nil, fmt.Errorf("failed to get event: %w", err)
	}
	return &event, nil
}

// Update 更新事件
func (r *EventRepository) Update(ctx context.Context, event *entity.Event) error {
	ctx, span := tracer.Start(ctx, "postgres.EventRepository.Update")
	defer span.End()

	db := getDB(ctx, r.client.db)
	if err := db.Save(event).Error; err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to update event: %w", err)
	}
	return nil
}

// Delete 删除事件
func (r *EventRepository) Delete(ctx context.Context, id string) error {
	ctx, span := tracer.Start(ctx, "postgres.EventRepository.Delete")
	defer span.End()

	db := getDB(ctx, r.client.db)
	if err := db.Delete(&entity.Event{}, "id = ?", id).Error; err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to delete event: %w", err)
	}
	return nil
}

// ListByProject 获取项目的事件列表
func (r *EventRepository) ListByProject(ctx context.Context, projectID string, filter *repository.EventFilter, pagination repository.Pagination) (*repository.PagedResult[*entity.Event], error) {
	ctx, span := tracer.Start(ctx, "postgres.EventRepository.ListByProject")
	defer span.End()

	db := getDB(ctx, r.client.db)
	query := db.Model(&entity.Event{}).Where("project_id = ?", projectID)

	// 应用过滤条件
	if filter != nil {
		if filter.ChapterID != "" {
			query = query.Where("chapter_id = ?", filter.ChapterID)
		}
		if filter.EventType != "" {
			query = query.Where("event_type = ?", filter.EventType)
		}
		if filter.Importance != "" {
			query = query.Where("importance = ?", filter.Importance)
		}
		if filter.TimeStart > 0 {
			query = query.Where("story_time_start >= ?", filter.TimeStart)
		}
		if filter.TimeEnd > 0 {
			query = query.Where("story_time_end <= ?", filter.TimeEnd)
		}
	}

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to count events: %w", err)
	}

	// 获取列表
	var events []*entity.Event
	if err := query.Order("story_time_start ASC").
		Offset(pagination.Offset()).
		Limit(pagination.Limit()).
		Find(&events).Error; err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to list events: %w", err)
	}

	return repository.NewPagedResult(events, total, pagination), nil
}

// ListByChapter 获取章节的事件列表
func (r *EventRepository) ListByChapter(ctx context.Context, chapterID string) ([]*entity.Event, error) {
	ctx, span := tracer.Start(ctx, "postgres.EventRepository.ListByChapter")
	defer span.End()

	db := getDB(ctx, r.client.db)
	var events []*entity.Event

	if err := db.Where("chapter_id = ?", chapterID).
		Order("story_time_start ASC").
		Find(&events).Error; err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to list events by chapter: %w", err)
	}

	return events, nil
}

// GetByTimeRange 根据时间范围获取事件
func (r *EventRepository) GetByTimeRange(ctx context.Context, projectID string, startTime, endTime int64) ([]*entity.Event, error) {
	ctx, span := tracer.Start(ctx, "postgres.EventRepository.GetByTimeRange")
	defer span.End()

	db := getDB(ctx, r.client.db)
	var events []*entity.Event

	query := db.Where("project_id = ?", projectID)
	if startTime > 0 {
		query = query.Where("story_time_end >= ? OR story_time_end = 0", startTime)
	}
	if endTime > 0 {
		query = query.Where("story_time_start <= ?", endTime)
	}

	if err := query.Order("story_time_start ASC").Find(&events).Error; err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to get events by time range: %w", err)
	}

	return events, nil
}

// GetByEntity 获取涉及实体的事件
func (r *EventRepository) GetByEntity(ctx context.Context, entityID string, pagination repository.Pagination) (*repository.PagedResult[*entity.Event], error) {
	ctx, span := tracer.Start(ctx, "postgres.EventRepository.GetByEntity")
	defer span.End()

	db := getDB(ctx, r.client.db)
	query := db.Model(&entity.Event{}).Where("involved_entities @> ?", fmt.Sprintf(`["%s"]`, entityID))

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to count events by entity: %w", err)
	}

	// 获取列表
	var events []*entity.Event
	if err := query.Order("story_time_start ASC").
		Offset(pagination.Offset()).
		Limit(pagination.Limit()).
		Find(&events).Error; err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to get events by entity: %w", err)
	}

	return repository.NewPagedResult(events, total, pagination), nil
}

// UpdateVectorID 更新向量 ID
func (r *EventRepository) UpdateVectorID(ctx context.Context, id, vectorID string) error {
	ctx, span := tracer.Start(ctx, "postgres.EventRepository.UpdateVectorID")
	defer span.End()

	db := getDB(ctx, r.client.db)
	if err := db.Model(&entity.Event{}).Where("id = ?", id).Update("vector_id", vectorID).Error; err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to update vector id: %w", err)
	}
	return nil
}

// GetTimeline 获取时间轴
func (r *EventRepository) GetTimeline(ctx context.Context, projectID string, limit int) ([]*entity.Event, error) {
	ctx, span := tracer.Start(ctx, "postgres.EventRepository.GetTimeline")
	defer span.End()

	db := getDB(ctx, r.client.db)
	var events []*entity.Event

	if err := db.Where("project_id = ?", projectID).
		Order("story_time_start ASC").
		Limit(limit).
		Find(&events).Error; err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to get timeline: %w", err)
	}

	return events, nil
}

// SearchByTags 根据标签搜索事件
func (r *EventRepository) SearchByTags(ctx context.Context, projectID string, tags []string, limit int) ([]*entity.Event, error) {
	ctx, span := tracer.Start(ctx, "postgres.EventRepository.SearchByTags")
	defer span.End()

	db := getDB(ctx, r.client.db)
	var events []*entity.Event

	query := db.Where("project_id = ?", projectID)
	for _, tag := range tags {
		query = query.Where("tags @> ?", fmt.Sprintf(`["%s"]`, tag))
	}

	if err := query.Order("story_time_start ASC").Limit(limit).Find(&events).Error; err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to search events by tags: %w", err)
	}

	return events, nil
}
