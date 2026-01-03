// Package entity 定义领域实体
package entity

import (
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

// StoryEntity 故事实体（角色/物品/地点等）
type StoryEntity struct {
	ID                   string            `json:"id"`
	ProjectID            string            `json:"project_id"`
	Name                 string            `json:"name"`
	Aliases              []string          `json:"aliases,omitempty"`
	Type                 StoryEntityType   `json:"type"`
	Description          string            `json:"description,omitempty"`
	Attributes           *EntityAttributes `json:"attributes,omitempty"`
	Metadata             map[string]string `json:"metadata,omitempty"` // 扩展属性
	CurrentState         string            `json:"current_state,omitempty"`
	FirstAppearChapterID string            `json:"first_appear_chapter_id,omitempty"`
	LastAppearChapterID  string            `json:"last_appear_chapter_id,omitempty"`
	AppearCount          int               `json:"appear_count"`
	Importance           EntityImportance  `json:"importance"`
	VectorID             string            `json:"vector_id,omitempty"`
	CreatedAt            time.Time         `json:"created_at"`
	UpdatedAt            time.Time         `json:"updated_at"`
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
		Aliases:     []string{},
		Attributes:  &EntityAttributes{Abilities: []string{}},
		Metadata:    make(map[string]string),
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
