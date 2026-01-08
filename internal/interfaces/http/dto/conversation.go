// Package dto 提供 HTTP 层数据传输对象
package dto

import (
	"encoding/json"
	"time"

	"z-novel-ai-api/internal/domain/entity"
)

type CreateSessionRequest struct {
	Task string `json:"task,omitempty"`
}

type SessionResponse struct {
	ID          string `json:"id"`
	ProjectID   string `json:"project_id"`
	CurrentTask string `json:"current_task"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

func ToSessionResponse(s *entity.ConversationSession) *SessionResponse {
	if s == nil {
		return nil
	}
	return &SessionResponse{
		ID:          s.ID,
		ProjectID:   s.ProjectID,
		CurrentTask: string(s.CurrentTask),
		CreatedAt:   s.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:   s.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

type SessionListResponse struct {
	Sessions []*SessionResponse `json:"sessions"`
}

type TurnResponse struct {
	ID        string          `json:"id"`
	Role      string          `json:"role"`
	Task      string          `json:"task"`
	Content   string          `json:"content"`
	Metadata  json.RawMessage `json:"metadata,omitempty"`
	CreatedAt string          `json:"created_at"`
}

func ToTurnResponse(t *entity.ConversationTurn) *TurnResponse {
	if t == nil {
		return nil
	}
	return &TurnResponse{
		ID:        t.ID,
		Role:      string(t.Role),
		Task:      string(t.Task),
		Content:   t.Content,
		Metadata:  t.Metadata,
		CreatedAt: t.CreatedAt.UTC().Format(time.RFC3339),
	}
}

type TurnListResponse struct {
	Turns []*TurnResponse `json:"turns"`
}

type SendMessageRequest struct {
	Task string `json:"task,omitempty"`
	ConversationMessageRequest
}

type ArtifactSnapshotResponse struct {
	ArtifactID string          `json:"artifact_id"`
	Type       string          `json:"type"`
	VersionID  string          `json:"version_id"`
	VersionNo  int             `json:"version_no"`
	Content    json.RawMessage `json:"content"`
}

type SendMessageResponse struct {
	Session          *SessionResponse          `json:"session"`
	UserTurnID       string                    `json:"user_turn_id"`
	AssistantTurnID  string                    `json:"assistant_turn_id"`
	AssistantMessage string                    `json:"assistant_message"`
	JobID            string                    `json:"job_id"`
	ArtifactSnapshot *ArtifactSnapshotResponse `json:"artifact_snapshot,omitempty"`
	Usage            *FoundationUsageResponse  `json:"usage,omitempty"`
}
