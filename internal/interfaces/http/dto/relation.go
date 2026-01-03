// Package dto 提供 HTTP 层数据传输对象
package dto

import (
	"time"

	"z-novel-ai-api/internal/domain/entity"

	"github.com/google/uuid"
)

// RelationListResponse 关系列表响应
type RelationListResponse struct {
	Items []*RelationResponse `json:"items"`
}

// CreateRelationRequest 创建关系请求
type CreateRelationRequest struct {
	SourceEntityID string                     `json:"source_entity_id" binding:"required,uuid"`
	TargetEntityID string                     `json:"target_entity_id" binding:"required,uuid"`
	RelationType   entity.RelationType        `json:"relation_type" binding:"required"`
	Strength       float64                    `json:"strength" binding:"omitempty,gte=0,lte=1"`
	Description    string                     `json:"description" binding:"omitempty,max=2000"`
	Attributes     *entity.RelationAttributes `json:"attributes" binding:"omitempty"`
}

// ToRelationEntity 转换为关系实体
func (r *CreateRelationRequest) ToRelationEntity(projectID string) *entity.Relation {
	strength := r.Strength
	if strength == 0 {
		strength = 0.5 // 默认强度
	}

	now := time.Now()
	return &entity.Relation{
		ID:             uuid.New().String(),
		ProjectID:      projectID,
		SourceEntityID: r.SourceEntityID,
		TargetEntityID: r.TargetEntityID,
		RelationType:   r.RelationType,
		Strength:       strength,
		Description:    r.Description,
		Attributes:     r.Attributes,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// UpdateRelationRequest 更新关系请求
type UpdateRelationRequest struct {
	RelationType *entity.RelationType       `json:"relation_type" binding:"omitempty"`
	Strength     *float64                   `json:"strength" binding:"omitempty,gte=0,lte=1"`
	Description  *string                    `json:"description" binding:"omitempty,max=2000"`
	Attributes   *entity.RelationAttributes `json:"attributes" binding:"omitempty"`
}

// ApplyToRelation 应用更新到关系实体
func (r *UpdateRelationRequest) ApplyToRelation(rel *entity.Relation) {
	if r.RelationType != nil {
		rel.RelationType = *r.RelationType
	}
	if r.Strength != nil {
		rel.Strength = *r.Strength
	}
	if r.Description != nil {
		rel.Description = *r.Description
	}
	if r.Attributes != nil {
		rel.Attributes = r.Attributes
	}
	rel.UpdatedAt = time.Now()
}

// ToRelationListResponse 实体列表转换为响应
func ToRelationListResponse(relations []*entity.Relation) *RelationListResponse {
	items := make([]*RelationResponse, len(relations))
	for i, r := range relations {
		items[i] = ToRelationResponse(r)
	}
	return &RelationListResponse{Items: items}
}
