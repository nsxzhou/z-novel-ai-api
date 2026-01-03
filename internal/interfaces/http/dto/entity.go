// Package dto 提供 HTTP 层数据传输对象
package dto

import (
	"time"

	"z-novel-ai-api/internal/domain/entity"
)

// CreateEntityRequest 创建实体请求
type CreateEntityRequest struct {
	Name        string               `json:"name" binding:"required,max=255"`
	Aliases     []string             `json:"aliases,omitempty"`
	Type        string               `json:"type" binding:"required"` // character, location, item, organization, concept
	Description string               `json:"description" binding:"max=10000"`
	Attributes  *EntityAttributesDTO `json:"attributes,omitempty"`
	Metadata    map[string]string    `json:"metadata,omitempty"`
	Importance  string               `json:"importance,omitempty"` // protagonist, major, minor, background
}

// EntityAttributesDTO 实体属性 DTO
type EntityAttributesDTO struct {
	Age         int      `json:"age,omitempty"`
	Gender      string   `json:"gender,omitempty"`
	Occupation  string   `json:"occupation,omitempty"`
	Personality string   `json:"personality,omitempty"`
	Abilities   []string `json:"abilities,omitempty"`
	Background  string   `json:"background,omitempty"`
}

// UpdateEntityRequest 更新实体请求
type UpdateEntityRequest struct {
	Name        *string              `json:"name,omitempty" binding:"omitempty,max=255"`
	Aliases     []string             `json:"aliases,omitempty"`
	Description *string              `json:"description,omitempty" binding:"omitempty,max=10000"`
	Attributes  *EntityAttributesDTO `json:"attributes,omitempty"`
	Metadata    map[string]string    `json:"metadata,omitempty"`
	Importance  *string              `json:"importance,omitempty"`
}

// UpdateEntityStateRequest 更新实体状态请求
type UpdateEntityStateRequest struct {
	CurrentState     string            `json:"current_state" binding:"required,max=5000"`
	AttributeChanges map[string]string `json:"attribute_changes,omitempty"`
	ChapterID        string            `json:"chapter_id,omitempty"`
	StoryTime        int64             `json:"story_time,omitempty"`
}

// EntityResponse 实体响应
type EntityResponse struct {
	ID           string               `json:"id"`
	ProjectID    string               `json:"project_id"`
	Name         string               `json:"name"`
	Aliases      []string             `json:"aliases,omitempty"`
	Type         string               `json:"type"`
	Description  string               `json:"description,omitempty"`
	Attributes   *EntityAttributesDTO `json:"attributes,omitempty"`
	Metadata     map[string]string    `json:"metadata,omitempty"`
	CurrentState string               `json:"current_state,omitempty"`
	Importance   string               `json:"importance,omitempty"`
	CreatedAt    time.Time            `json:"created_at"`
	UpdatedAt    time.Time            `json:"updated_at"`
}

// EntityListResponse 实体列表响应
type EntityListResponse struct {
	Entities []*EntityResponse `json:"entities"`
}

// EntityRelationsResponse 实体关系响应
type EntityRelationsResponse struct {
	EntityID  string              `json:"entity_id"`
	Relations []*RelationResponse `json:"relations"`
}

// RelationResponse 关系响应
type RelationResponse struct {
	ID             string    `json:"id"`
	SourceEntityID string    `json:"source_entity_id"`
	TargetEntityID string    `json:"target_entity_id"`
	RelationType   string    `json:"relation_type"`
	Strength       float64   `json:"strength"`
	Description    string    `json:"description,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// ToEntityResponse 将领域实体转换为响应 DTO
func ToEntityResponse(e *entity.StoryEntity) *EntityResponse {
	if e == nil {
		return nil
	}

	resp := &EntityResponse{
		ID:           e.ID,
		ProjectID:    e.ProjectID,
		Name:         e.Name,
		Aliases:      e.Aliases,
		Type:         string(e.Type),
		Description:  e.Description,
		Metadata:     e.Metadata,
		CurrentState: e.CurrentState,
		Importance:   string(e.Importance),
		CreatedAt:    e.CreatedAt,
		UpdatedAt:    e.UpdatedAt,
	}

	if e.Attributes != nil {
		resp.Attributes = &EntityAttributesDTO{
			Age:         e.Attributes.Age,
			Gender:      e.Attributes.Gender,
			Occupation:  e.Attributes.Occupation,
			Personality: e.Attributes.Personality,
			Abilities:   e.Attributes.Abilities,
			Background:  e.Attributes.Background,
		}
	}

	return resp
}

// ToEntityListResponse 将领域实体列表转换为响应 DTO
func ToEntityListResponse(entities []*entity.StoryEntity) *EntityListResponse {
	resp := &EntityListResponse{
		Entities: make([]*EntityResponse, 0, len(entities)),
	}

	for _, e := range entities {
		resp.Entities = append(resp.Entities, ToEntityResponse(e))
	}

	return resp
}

// ToRelationResponse 将关系实体转换为响应 DTO
func ToRelationResponse(r *entity.Relation) *RelationResponse {
	if r == nil {
		return nil
	}

	return &RelationResponse{
		ID:             r.ID,
		SourceEntityID: r.SourceEntityID,
		TargetEntityID: r.TargetEntityID,
		RelationType:   string(r.RelationType),
		Strength:       r.Strength,
		Description:    r.Description,
		CreatedAt:      r.CreatedAt,
		UpdatedAt:      r.UpdatedAt,
	}
}

// ToEntityRelationsResponse 构建实体关系响应
func ToEntityRelationsResponse(entityID string, relations []*entity.Relation) *EntityRelationsResponse {
	resp := &EntityRelationsResponse{
		EntityID:  entityID,
		Relations: make([]*RelationResponse, 0, len(relations)),
	}

	for _, r := range relations {
		resp.Relations = append(resp.Relations, ToRelationResponse(r))
	}

	return resp
}

// ToStoryEntity 将请求 DTO 转换为领域实体
func (r *CreateEntityRequest) ToStoryEntity(projectID string) *entity.StoryEntity {
	e := entity.NewStoryEntity(
		projectID,
		r.Name,
		entity.StoryEntityType(r.Type),
		entity.EntityImportance(r.Importance),
	)
	e.Description = r.Description
	e.Aliases = r.Aliases
	e.Metadata = r.Metadata

	if r.Attributes != nil {
		e.Attributes = &entity.EntityAttributes{
			Age:         r.Attributes.Age,
			Gender:      r.Attributes.Gender,
			Occupation:  r.Attributes.Occupation,
			Personality: r.Attributes.Personality,
			Abilities:   r.Attributes.Abilities,
			Background:  r.Attributes.Background,
		}
	}
	return e
}

// ApplyToEntity 将更新请求应用到实体
func (r *UpdateEntityRequest) ApplyToEntity(e *entity.StoryEntity) {
	if r.Name != nil {
		e.Name = *r.Name
	}
	if r.Aliases != nil {
		e.Aliases = r.Aliases
	}
	if r.Description != nil {
		e.Description = *r.Description
	}
	if r.Metadata != nil {
		e.Metadata = r.Metadata
	}
	if r.Attributes != nil {
		if e.Attributes == nil {
			e.Attributes = &entity.EntityAttributes{}
		}
		if r.Attributes.Age > 0 {
			e.Attributes.Age = r.Attributes.Age
		}
		if r.Attributes.Gender != "" {
			e.Attributes.Gender = r.Attributes.Gender
		}
		if r.Attributes.Occupation != "" {
			e.Attributes.Occupation = r.Attributes.Occupation
		}
		if r.Attributes.Personality != "" {
			e.Attributes.Personality = r.Attributes.Personality
		}
		if r.Attributes.Abilities != nil {
			e.Attributes.Abilities = r.Attributes.Abilities
		}
		if r.Attributes.Background != "" {
			e.Attributes.Background = r.Attributes.Background
		}
	}
	if r.Importance != nil {
		e.Importance = entity.EntityImportance(*r.Importance)
	}

	e.UpdatedAt = time.Now()
}
