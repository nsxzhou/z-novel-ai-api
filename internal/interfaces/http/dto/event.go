// Package dto 提供 HTTP 层数据传输对象
package dto

import (
	"time"

	"z-novel-ai-api/internal/domain/entity"

	"github.com/google/uuid"
)

// EventResponse 事件响应
type EventResponse struct {
	ID               string                 `json:"id"`
	ProjectID        string                 `json:"project_id"`
	ChapterID        string                 `json:"chapter_id,omitempty"`
	StoryTimeStart   int64                  `json:"story_time_start"`
	StoryTimeEnd     int64                  `json:"story_time_end,omitempty"`
	EventType        entity.EventType       `json:"event_type,omitempty"`
	Summary          string                 `json:"summary"`
	Description      string                 `json:"description,omitempty"`
	InvolvedEntities []string               `json:"involved_entities,omitempty"`
	LocationID       string                 `json:"location_id,omitempty"`
	Importance       entity.EventImportance `json:"importance"`
	Tags             []string               `json:"tags,omitempty"`
	CreatedAt        time.Time              `json:"created_at"`
}

// EventListResponse 事件列表响应
type EventListResponse struct {
	Items []*EventResponse `json:"items"`
}

// CreateEventRequest 创建事件请求
type CreateEventRequest struct {
	ChapterID        string                 `json:"chapter_id" binding:"omitempty,uuid"`
	StoryTimeStart   int64                  `json:"story_time_start" binding:"required"`
	StoryTimeEnd     int64                  `json:"story_time_end" binding:"omitempty"`
	EventType        entity.EventType       `json:"event_type" binding:"omitempty"`
	Summary          string                 `json:"summary" binding:"required,max=500"`
	Description      string                 `json:"description" binding:"omitempty,max=5000"`
	InvolvedEntities []string               `json:"involved_entities" binding:"omitempty"`
	LocationID       string                 `json:"location_id" binding:"omitempty,uuid"`
	Importance       entity.EventImportance `json:"importance" binding:"omitempty"`
	Tags             []string               `json:"tags" binding:"omitempty"`
}

// ToEventEntity 转换为事件实体
func (r *CreateEventRequest) ToEventEntity(projectID string) *entity.Event {
	importance := r.Importance
	if importance == "" {
		importance = entity.EventImportanceNormal
	}

	involvedEntities := r.InvolvedEntities
	if involvedEntities == nil {
		involvedEntities = []string{}
	}

	tags := r.Tags
	if tags == nil {
		tags = []string{}
	}

	return &entity.Event{
		ID:               uuid.New().String(),
		ProjectID:        projectID,
		ChapterID:        r.ChapterID,
		StoryTimeStart:   r.StoryTimeStart,
		StoryTimeEnd:     r.StoryTimeEnd,
		EventType:        r.EventType,
		Summary:          r.Summary,
		Description:      r.Description,
		InvolvedEntities: involvedEntities,
		LocationID:       r.LocationID,
		Importance:       importance,
		Tags:             tags,
		CreatedAt:        time.Now(),
	}
}

// UpdateEventRequest 更新事件请求
type UpdateEventRequest struct {
	ChapterID        *string                 `json:"chapter_id" binding:"omitempty"`
	StoryTimeStart   *int64                  `json:"story_time_start" binding:"omitempty"`
	StoryTimeEnd     *int64                  `json:"story_time_end" binding:"omitempty"`
	EventType        *entity.EventType       `json:"event_type" binding:"omitempty"`
	Summary          *string                 `json:"summary" binding:"omitempty,max=500"`
	Description      *string                 `json:"description" binding:"omitempty,max=5000"`
	InvolvedEntities []string                `json:"involved_entities" binding:"omitempty"`
	LocationID       *string                 `json:"location_id" binding:"omitempty"`
	Importance       *entity.EventImportance `json:"importance" binding:"omitempty"`
	Tags             []string                `json:"tags" binding:"omitempty"`
}

// ApplyToEvent 应用更新到事件实体
func (r *UpdateEventRequest) ApplyToEvent(e *entity.Event) {
	if r.ChapterID != nil {
		e.ChapterID = *r.ChapterID
	}
	if r.StoryTimeStart != nil {
		e.StoryTimeStart = *r.StoryTimeStart
	}
	if r.StoryTimeEnd != nil {
		e.StoryTimeEnd = *r.StoryTimeEnd
	}
	if r.EventType != nil {
		e.EventType = *r.EventType
	}
	if r.Summary != nil {
		e.Summary = *r.Summary
	}
	if r.Description != nil {
		e.Description = *r.Description
	}
	if r.InvolvedEntities != nil {
		e.InvolvedEntities = r.InvolvedEntities
	}
	if r.LocationID != nil {
		e.LocationID = *r.LocationID
	}
	if r.Importance != nil {
		e.Importance = *r.Importance
	}
	if r.Tags != nil {
		e.Tags = r.Tags
	}
}

// ToEventResponse 实体转换为响应
func ToEventResponse(e *entity.Event) *EventResponse {
	if e == nil {
		return nil
	}
	return &EventResponse{
		ID:               e.ID,
		ProjectID:        e.ProjectID,
		ChapterID:        e.ChapterID,
		StoryTimeStart:   e.StoryTimeStart,
		StoryTimeEnd:     e.StoryTimeEnd,
		EventType:        e.EventType,
		Summary:          e.Summary,
		Description:      e.Description,
		InvolvedEntities: e.InvolvedEntities,
		LocationID:       e.LocationID,
		Importance:       e.Importance,
		Tags:             e.Tags,
		CreatedAt:        e.CreatedAt,
	}
}

// ToEventListResponse 实体列表转换为响应
func ToEventListResponse(events []*entity.Event) *EventListResponse {
	items := make([]*EventResponse, len(events))
	for i, e := range events {
		items[i] = ToEventResponse(e)
	}
	return &EventListResponse{Items: items}
}
