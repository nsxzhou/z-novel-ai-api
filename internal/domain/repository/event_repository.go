// Package repository 定义数据访问层接口
package repository

import (
	"context"

	"z-novel-ai-api/internal/domain/entity"
)

// EventFilter 事件过滤条件
type EventFilter struct {
	EventType  entity.EventType
	Importance entity.EventImportance
	ChapterID  string
	LocationID string
	Tags       []string
	TimeStart  int64
	TimeEnd    int64
}

// EventRepository 事件仓储接口
type EventRepository interface {
	// Create 创建事件
	Create(ctx context.Context, event *entity.Event) error

	// GetByID 根据 ID 获取事件
	GetByID(ctx context.Context, id string) (*entity.Event, error)

	// Update 更新事件
	Update(ctx context.Context, event *entity.Event) error

	// Delete 删除事件
	Delete(ctx context.Context, id string) error

	// ListByProject 获取项目事件列表
	ListByProject(ctx context.Context, projectID string, filter *EventFilter, pagination Pagination) (*PagedResult[*entity.Event], error)

	// ListByChapter 获取章节事件列表
	ListByChapter(ctx context.Context, chapterID string) ([]*entity.Event, error)

	// GetByTimeRange 根据时间范围获取事件
	GetByTimeRange(ctx context.Context, projectID string, startTime, endTime int64) ([]*entity.Event, error)

	// GetByEntity 获取涉及实体的事件
	GetByEntity(ctx context.Context, entityID string, pagination Pagination) (*PagedResult[*entity.Event], error)

	// UpdateVectorID 更新向量 ID
	UpdateVectorID(ctx context.Context, id, vectorID string) error

	// GetTimeline 获取时间轴
	GetTimeline(ctx context.Context, projectID string, limit int) ([]*entity.Event, error)

	// SearchByTags 根据标签搜索事件
	SearchByTags(ctx context.Context, projectID string, tags []string, limit int) ([]*entity.Event, error)
}
