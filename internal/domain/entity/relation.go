// Package entity 定义领域实体
package entity

import (
	"time"
)

// RelationType 关系类型
type RelationType string

const (
	RelationTypeFriend      RelationType = "friend"
	RelationTypeEnemy       RelationType = "enemy"
	RelationTypeFamily      RelationType = "family"
	RelationTypeLover       RelationType = "lover"
	RelationTypeSubordinate RelationType = "subordinate"
	RelationTypeMentor      RelationType = "mentor"
	RelationTypeRival       RelationType = "rival"
	RelationTypeAlly        RelationType = "ally"
)

// RelationAttributes 关系属性
type RelationAttributes struct {
	Since       string `json:"since,omitempty"`
	Origin      string `json:"origin,omitempty"`
	Development string `json:"development,omitempty"`
}

// Relation 实体间关系
type Relation struct {
	ID             string              `json:"id"`
	ProjectID      string              `json:"project_id"`
	SourceEntityID string              `json:"source_entity_id"`
	TargetEntityID string              `json:"target_entity_id"`
	RelationType   RelationType        `json:"relation_type"`
	Strength       float64             `json:"strength"`
	Description    string              `json:"description,omitempty"`
	Attributes     *RelationAttributes `json:"attributes,omitempty"`
	FirstChapterID string              `json:"first_chapter_id,omitempty"`
	LastChapterID  string              `json:"last_chapter_id,omitempty"`
	CreatedAt      time.Time           `json:"created_at"`
	UpdatedAt      time.Time           `json:"updated_at"`
}

// NewRelation 创建新关系
func NewRelation(projectID, sourceID, targetID string, relType RelationType) *Relation {
	now := time.Now()
	return &Relation{
		ProjectID:      projectID,
		SourceEntityID: sourceID,
		TargetEntityID: targetID,
		RelationType:   relType,
		Strength:       0.5,
		Attributes:     &RelationAttributes{},
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// UpdateStrength 更新关系强度 (0-1)
func (r *Relation) UpdateStrength(strength float64) {
	if strength < 0 {
		strength = 0
	} else if strength > 1 {
		strength = 1
	}
	r.Strength = strength
	r.UpdatedAt = time.Now()
}

// RecordAppearance 记录关系出现的章节
func (r *Relation) RecordAppearance(chapterID string) {
	if r.FirstChapterID == "" {
		r.FirstChapterID = chapterID
	}
	r.LastChapterID = chapterID
	r.UpdatedAt = time.Now()
}
