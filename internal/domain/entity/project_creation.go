// Package entity 定义领域实体
package entity

import (
	"encoding/json"
	"time"
)

type ProjectCreationStage string

const (
	ProjectCreationStageDiscover ProjectCreationStage = "discover"
	ProjectCreationStageNarrow   ProjectCreationStage = "narrow"
	ProjectCreationStageDraft    ProjectCreationStage = "draft"
	ProjectCreationStageConfirm  ProjectCreationStage = "confirm"
)

type ProjectCreationStatus string

const (
	ProjectCreationStatusActive    ProjectCreationStatus = "active"
	ProjectCreationStatusCompleted ProjectCreationStatus = "completed"
	ProjectCreationStatusCancelled ProjectCreationStatus = "cancelled"
)

type ProjectCreationSession struct {
	ID                      string                `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TenantID                string                `json:"tenant_id" gorm:"type:uuid;index;not null"`
	UserID                  *string               `json:"user_id,omitempty" gorm:"type:uuid;index"`
	Stage                   ProjectCreationStage  `json:"stage" gorm:"type:varchar(32);not null;default:'discover'"`
	Status                  ProjectCreationStatus `json:"status" gorm:"type:varchar(32);not null;default:'active'"`
	Draft                   json.RawMessage       `json:"draft" gorm:"type:jsonb;not null"`
	CreatedProjectID        *string               `json:"created_project_id,omitempty" gorm:"type:uuid"`
	CreatedProjectSessionID *string               `json:"created_project_session_id,omitempty" gorm:"type:uuid"`
	CreatedAt               time.Time             `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt               time.Time             `json:"updated_at" gorm:"autoUpdateTime"`
}

func (ProjectCreationSession) TableName() string {
	return "project_creation_sessions"
}

func NewProjectCreationSession(tenantID, userID string) *ProjectCreationSession {
	now := time.Now()
	uid := userID
	return &ProjectCreationSession{
		TenantID:  tenantID,
		UserID:    &uid,
		Stage:     ProjectCreationStageDiscover,
		Status:    ProjectCreationStatusActive,
		Draft:     json.RawMessage("{}"),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

type ProjectCreationTurn struct {
	ID        string          `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	SessionID string          `json:"session_id" gorm:"type:uuid;index;not null"`
	Role      Role            `json:"role" gorm:"type:varchar(16);not null"`
	Content   string          `json:"content" gorm:"type:text;not null"`
	Metadata  json.RawMessage `json:"metadata,omitempty" gorm:"type:jsonb"`
	CreatedAt time.Time       `json:"created_at" gorm:"autoCreateTime"`
}

func (ProjectCreationTurn) TableName() string {
	return "project_creation_turns"
}

func NewProjectCreationTurn(sessionID string, role Role, content string, metadata json.RawMessage) *ProjectCreationTurn {
	return &ProjectCreationTurn{
		SessionID: sessionID,
		Role:      role,
		Content:   content,
		Metadata:  metadata,
		CreatedAt: time.Now(),
	}
}
