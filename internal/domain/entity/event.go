// Package entity 定义领域实体
package entity

import (
	"time"
)

// EventType 事件类型
type EventType string

const (
	EventTypePlot        EventType = "plot"
	EventTypeDialogue    EventType = "dialogue"
	EventTypeAction      EventType = "action"
	EventTypeDescription EventType = "description"
)

// EventImportance 事件重要性
type EventImportance string

const (
	EventImportanceCritical EventImportance = "critical"
	EventImportanceMajor    EventImportance = "major"
	EventImportanceNormal   EventImportance = "normal"
	EventImportanceMinor    EventImportance = "minor"
)

// Event 故事事件（时间轴节点）
type Event struct {
	ID               string          `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ProjectID        string          `json:"project_id" gorm:"type:uuid;index;not null"`
	ChapterID        string          `json:"chapter_id,omitempty" gorm:"type:uuid;index"`
	StoryTimeStart   int64           `json:"story_time_start" gorm:"index"`
	StoryTimeEnd     int64           `json:"story_time_end,omitempty"`
	EventType        EventType       `json:"event_type,omitempty" gorm:"type:varchar(50)"`
	Summary          string          `json:"summary" gorm:"type:text;not null"`
	Description      string          `json:"description,omitempty" gorm:"type:text"`
	InvolvedEntities StringSlice     `json:"involved_entities,omitempty" gorm:"type:jsonb"`
	LocationID       string          `json:"location_id,omitempty" gorm:"type:uuid"`
	Importance       EventImportance `json:"importance" gorm:"type:varchar(50);default:'normal'"`
	Tags             StringSlice     `json:"tags,omitempty" gorm:"type:jsonb"`
	VectorID         string          `json:"vector_id,omitempty" gorm:"type:varchar(255)"`
	CreatedAt        time.Time       `json:"created_at" gorm:"autoCreateTime"`
}

// TableName 指定表名
func (Event) TableName() string {
	return "events"
}

// NewEvent 创建新事件
func NewEvent(projectID string, storyTimeStart int64, summary string) *Event {
	return &Event{
		ProjectID:        projectID,
		StoryTimeStart:   storyTimeStart,
		Summary:          summary,
		InvolvedEntities: StringSlice{},
		Tags:             StringSlice{},
		Importance:       EventImportanceNormal,
		CreatedAt:        time.Now(),
	}
}

// AddInvolvedEntity 添加涉及的实体
func (e *Event) AddInvolvedEntity(entityID string) {
	for _, id := range e.InvolvedEntities {
		if id == entityID {
			return
		}
	}
	e.InvolvedEntities = append(e.InvolvedEntities, entityID)
}

// AddTag 添加标签
func (e *Event) AddTag(tag string) {
	for _, t := range e.Tags {
		if t == tag {
			return
		}
	}
	e.Tags = append(e.Tags, tag)
}

// SetTimeRange 设置时间范围
func (e *Event) SetTimeRange(start, end int64) {
	e.StoryTimeStart = start
	e.StoryTimeEnd = end
}
