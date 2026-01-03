// Package entity 定义领域实体
package entity

import (
	"time"
)

// EntityState 实体状态变更历史记录
type EntityState struct {
	ID               string                 `json:"id"`
	EntityID         string                 `json:"entity_id"`
	ChapterID        string                 `json:"chapter_id,omitempty"`
	StoryTime        int64                  `json:"story_time,omitempty"`
	StateDescription string                 `json:"state_description"`
	AttributeChanges map[string]interface{} `json:"attribute_changes,omitempty"`
	EventSummary     string                 `json:"event_summary,omitempty"`
	CreatedAt        time.Time              `json:"created_at"`
}

// NewEntityState 创建新的实体状态记录
func NewEntityState(entityID, chapterID, stateDesc string, storyTime int64) *EntityState {
	return &EntityState{
		EntityID:         entityID,
		ChapterID:        chapterID,
		StoryTime:        storyTime,
		StateDescription: stateDesc,
		AttributeChanges: make(map[string]interface{}),
		CreatedAt:        time.Now(),
	}
}

// AddAttributeChange 添加属性变更记录
func (s *EntityState) AddAttributeChange(key string, value interface{}) {
	if s.AttributeChanges == nil {
		s.AttributeChanges = make(map[string]interface{})
	}
	s.AttributeChanges[key] = value
}
