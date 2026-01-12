// Package model 提供 story 应用层的稳定 DTO/结构定义（可被多个子包复用）。
package model

import "z-novel-ai-api/internal/domain/entity"

// FoundationPlan 小说“设定集”生成结果（世界观 + 角色/组织 + 大纲）
// 约定：该结构只表达“要落库的结构化结果”，不直接承载对话历史。
type FoundationPlan struct {
	Version int `json:"version"`

	Project   ProjectPlan    `json:"project"`
	Entities  []EntityPlan   `json:"entities"`
	Relations []RelationPlan `json:"relations"`
	Volumes   []VolumePlan   `json:"volumes"`
}

type ProjectPlan struct {
	Genre           string               `json:"genre,omitempty"`
	TargetWordCount int                  `json:"target_word_count,omitempty"`
	WritingStyle    string               `json:"writing_style,omitempty"`
	POV             string               `json:"pov,omitempty"`
	Temperature     float64              `json:"temperature,omitempty"`
	WorldSettings   entity.WorldSettings `json:"world_settings"`
	WorldBible      string               `json:"world_bible,omitempty"`
}

type EntityPlan struct {
	Key          string                   `json:"key"`
	Name         string                   `json:"name"`
	Type         entity.StoryEntityType   `json:"type"`
	Importance   entity.EntityImportance  `json:"importance,omitempty"`
	Description  string                   `json:"description,omitempty"`
	Aliases      []string                 `json:"aliases,omitempty"`
	Attributes   *entity.EntityAttributes `json:"attributes,omitempty"`
	CurrentState string                   `json:"current_state,omitempty"`
}

type RelationPlan struct {
	SourceKey    string                     `json:"source_key"`
	TargetKey    string                     `json:"target_key"`
	RelationType entity.RelationType        `json:"relation_type"`
	Strength     float64                    `json:"strength,omitempty"`
	Description  string                     `json:"description,omitempty"`
	Attributes   *entity.RelationAttributes `json:"attributes,omitempty"`
}

type VolumePlan struct {
	Key      string        `json:"key"`
	Title    string        `json:"title"`
	Summary  string        `json:"summary,omitempty"`
	Chapters []ChapterPlan `json:"chapters"`
}

type ChapterPlan struct {
	Key             string `json:"key"`
	Title           string `json:"title"`
	Outline         string `json:"outline"`
	TargetWordCount int    `json:"target_word_count,omitempty"`
	StoryTimeStart  int64  `json:"story_time_start,omitempty"`
}
