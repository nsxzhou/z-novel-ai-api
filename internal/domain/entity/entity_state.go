// Package entity 定义领域实体
package entity

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// AttributeChanges 用于 GORM JSON 序列化的属性变更映射
type AttributeChanges map[string]interface{}

// Value 实现 driver.Valuer 接口
func (a AttributeChanges) Value() (driver.Value, error) {
	if a == nil {
		return nil, nil
	}
	return json.Marshal(a)
}

// Scan 实现 sql.Scanner 接口
func (a *AttributeChanges) Scan(value interface{}) error {
	if value == nil {
		*a = nil
		return nil
	}
	return json.Unmarshal(value.([]byte), a)
}

// EntityState 实体状态变更历史记录
type EntityState struct {
	ID               string           `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	EntityID         string           `json:"entity_id" gorm:"type:uuid;index;not null"`
	ChapterID        string           `json:"chapter_id,omitempty" gorm:"type:uuid;index"`
	StoryTime        int64            `json:"story_time,omitempty"`
	StateDescription string           `json:"state_description" gorm:"type:text;not null"`
	AttributeChanges AttributeChanges `json:"attribute_changes,omitempty" gorm:"type:jsonb"`
	EventSummary     string           `json:"event_summary,omitempty" gorm:"type:text"`
	CreatedAt        time.Time        `json:"created_at" gorm:"autoCreateTime"`
}

// TableName 指定表名
func (EntityState) TableName() string {
	return "entity_states"
}

// NewEntityState 创建新的实体状态记录
func NewEntityState(entityID, chapterID, stateDesc string, storyTime int64) *EntityState {
	return &EntityState{
		EntityID:         entityID,
		ChapterID:        chapterID,
		StoryTime:        storyTime,
		StateDescription: stateDesc,
		AttributeChanges: make(AttributeChanges),
		CreatedAt:        time.Now(),
	}
}

// AddAttributeChange 添加属性变更记录
func (s *EntityState) AddAttributeChange(key string, value interface{}) {
	if s.AttributeChanges == nil {
		s.AttributeChanges = make(AttributeChanges)
	}
	s.AttributeChanges[key] = value
}
