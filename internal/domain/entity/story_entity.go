// Package entity 定义领域实体
package entity

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// StoryEntityType 实体类型
type StoryEntityType string

const (
	EntityTypeCharacter    StoryEntityType = "character"
	EntityTypeItem         StoryEntityType = "item"
	EntityTypeLocation     StoryEntityType = "location"
	EntityTypeOrganization StoryEntityType = "organization"
	EntityTypeConcept      StoryEntityType = "concept"
)

// EntityImportance 实体重要性
type EntityImportance string

const (
	ImportanceProtagonist EntityImportance = "protagonist"
	ImportanceMajor       EntityImportance = "major"
	ImportanceSecondary   EntityImportance = "secondary"
	ImportanceMinor       EntityImportance = "minor"
)

// EntityAttributes 实体属性
type EntityAttributes struct {
	Age         int      `json:"age,omitempty"`
	Gender      string   `json:"gender,omitempty"`
	Occupation  string   `json:"occupation,omitempty"`
	Personality string   `json:"personality,omitempty"`
	Abilities   []string `json:"abilities,omitempty"`
	Background  string   `json:"background,omitempty"`
}

// StringSlice 用于 GORM JSON 序列化的字符串切片
type StringSlice []string

// Value 实现 driver.Valuer 接口
func (s StringSlice) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	return json.Marshal(s)
}

// Scan 实现 sql.Scanner 接口
func (s *StringSlice) Scan(value interface{}) error {
	if value == nil {
		*s = nil
		return nil
	}
	return json.Unmarshal(value.([]byte), s)
}

// StringMap 用于 GORM JSON 序列化的字符串映射
type StringMap map[string]string

// Value 实现 driver.Valuer 接口
func (m StringMap) Value() (driver.Value, error) {
	if m == nil {
		return nil, nil
	}
	return json.Marshal(m)
}

// Scan 实现 sql.Scanner 接口
func (m *StringMap) Scan(value interface{}) error {
	if value == nil {
		*m = nil
		return nil
	}
	return json.Unmarshal(value.([]byte), m)
}

// StoryEntity 故事实体（角色/物品/地点等）
type StoryEntity struct {
	ID                   string            `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ProjectID            string            `json:"project_id" gorm:"type:uuid;index;not null"`
	AIKey                string            `json:"ai_key,omitempty" gorm:"column:ai_key;type:varchar(128);index"`
	Name                 string            `json:"name" gorm:"type:varchar(255);not null"`
	Aliases              StringSlice       `json:"aliases,omitempty" gorm:"type:jsonb"`
	Type                 StoryEntityType   `json:"type" gorm:"type:varchar(50);not null"`
	Description          string            `json:"description,omitempty" gorm:"type:text"`
	Attributes           *EntityAttributes `json:"attributes,omitempty" gorm:"type:jsonb;serializer:json"`
	Metadata             StringMap         `json:"metadata,omitempty" gorm:"type:jsonb"`
	CurrentState         string            `json:"current_state,omitempty" gorm:"type:text"`
	FirstAppearChapterID string            `json:"first_appear_chapter_id,omitempty" gorm:"type:uuid"`
	LastAppearChapterID  string            `json:"last_appear_chapter_id,omitempty" gorm:"type:uuid"`
	AppearCount          int               `json:"appear_count" gorm:"default:0"`
	Importance           EntityImportance  `json:"importance" gorm:"type:varchar(50);default:'secondary'"`
	VectorID             string            `json:"vector_id,omitempty" gorm:"type:varchar(255)"`
	CreatedAt            time.Time         `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt            time.Time         `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName 指定表名
func (StoryEntity) TableName() string {
	return "entities"
}

// NewStoryEntity 创建新实体
func NewStoryEntity(projectID, name string, entityType StoryEntityType, importance EntityImportance) *StoryEntity {
	now := time.Now()
	imp := importance
	if imp == "" {
		imp = ImportanceSecondary
	}
	return &StoryEntity{
		ProjectID:   projectID,
		Name:        name,
		Type:        entityType,
		Aliases:     StringSlice{},
		Attributes:  &EntityAttributes{Abilities: []string{}},
		Metadata:    make(StringMap),
		AppearCount: 0,
		Importance:  imp,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// AddAlias 添加别名
func (e *StoryEntity) AddAlias(alias string) {
	for _, a := range e.Aliases {
		if a == alias {
			return
		}
	}
	e.Aliases = append(e.Aliases, alias)
	e.UpdatedAt = time.Now()
}

// UpdateState 更新当前状态
func (e *StoryEntity) UpdateState(state, chapterID string, storyTime int64) {
	e.CurrentState = state
	if chapterID != "" {
		e.LastAppearChapterID = chapterID
	}
	e.UpdatedAt = time.Now()
	// storyTime 可用于记录状态变更的故事时间，当前仅更新 UpdatedAt
}

// RecordAppearance 记录出场
func (e *StoryEntity) RecordAppearance(chapterID string) {
	if e.FirstAppearChapterID == "" {
		e.FirstAppearChapterID = chapterID
	}
	e.LastAppearChapterID = chapterID
	e.AppearCount++
	e.UpdatedAt = time.Now()
}
