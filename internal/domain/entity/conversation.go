// Package entity 定义领域实体
package entity

import (
	"encoding/json"
	"time"
)

type ConversationTask string

const (
	ConversationTaskNovelFoundation ConversationTask = "novel_foundation"
	ConversationTaskWorldview       ConversationTask = "worldview"
	ConversationTaskCharacters      ConversationTask = "characters"
	ConversationTaskOutline         ConversationTask = "outline"
)

type ConversationSession struct {
	ID          string           `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TenantID    string           `json:"tenant_id" gorm:"type:uuid;index;not null"`
	ProjectID   string           `json:"project_id" gorm:"type:uuid;index;not null"`
	CurrentTask ConversationTask `json:"current_task" gorm:"type:varchar(32);not null;default:'novel_foundation'"`
	CreatedAt   time.Time        `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time        `json:"updated_at" gorm:"autoUpdateTime"`
}

func (ConversationSession) TableName() string {
	return "conversation_sessions"
}

func NewConversationSession(tenantID, projectID string, task ConversationTask) *ConversationSession {
	now := time.Now()
	if task == "" {
		task = ConversationTaskNovelFoundation
	}
	return &ConversationSession{
		TenantID:    tenantID,
		ProjectID:   projectID,
		CurrentTask: task,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

type ConversationTurn struct {
	ID        string           `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	SessionID string           `json:"session_id" gorm:"type:uuid;index;not null"`
	Role      Role             `json:"role" gorm:"type:varchar(16);not null"`
	Task      ConversationTask `json:"task" gorm:"type:varchar(32);not null"`
	Content   string           `json:"content" gorm:"type:text;not null"`
	Metadata  json.RawMessage  `json:"metadata,omitempty" gorm:"type:jsonb"`
	CreatedAt time.Time        `json:"created_at" gorm:"autoCreateTime"`
}

func (ConversationTurn) TableName() string {
	return "conversation_turns"
}

func NewConversationTurn(sessionID string, role Role, task ConversationTask, content string, metadata json.RawMessage) *ConversationTurn {
	return &ConversationTurn{
		SessionID: sessionID,
		Role:      role,
		Task:      task,
		Content:   content,
		Metadata:  metadata,
		CreatedAt: time.Now(),
	}
}
