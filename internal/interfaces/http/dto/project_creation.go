// Package dto 提供 HTTP 层数据传输对象
package dto

import (
	"encoding/json"
	"time"

	"z-novel-ai-api/internal/domain/entity"
)

type CreateProjectCreationSessionRequest struct {
	Prompt string `json:"prompt,omitempty"`
	ConversationMessageRequest
}

type ProjectCreationSessionResponse struct {
	ID                      string          `json:"id"`
	Stage                   string          `json:"stage"`
	Status                  string          `json:"status"`
	Draft                   json.RawMessage `json:"draft"`
	CreatedProjectID        *string         `json:"created_project_id,omitempty"`
	CreatedProjectSessionID *string         `json:"created_project_session_id,omitempty"`
	CreatedAt               string          `json:"created_at"`
	UpdatedAt               string          `json:"updated_at"`
}

func ToProjectCreationSessionResponse(s *entity.ProjectCreationSession) *ProjectCreationSessionResponse {
	if s == nil {
		return nil
	}
	return &ProjectCreationSessionResponse{
		ID:                      s.ID,
		Stage:                   string(s.Stage),
		Status:                  string(s.Status),
		Draft:                   s.Draft,
		CreatedProjectID:        s.CreatedProjectID,
		CreatedProjectSessionID: s.CreatedProjectSessionID,
		CreatedAt:               s.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:               s.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

type ProjectCreationTurnResponse struct {
	ID        string          `json:"id"`
	Role      string          `json:"role"`
	Content   string          `json:"content"`
	Metadata  json.RawMessage `json:"metadata,omitempty"`
	CreatedAt string          `json:"created_at"`
}

func ToProjectCreationTurnResponse(t *entity.ProjectCreationTurn) *ProjectCreationTurnResponse {
	if t == nil {
		return nil
	}
	return &ProjectCreationTurnResponse{
		ID:        t.ID,
		Role:      string(t.Role),
		Content:   t.Content,
		Metadata:  t.Metadata,
		CreatedAt: t.CreatedAt.UTC().Format(time.RFC3339),
	}
}

type ProjectCreationTurnListResponse struct {
	Turns []*ProjectCreationTurnResponse `json:"turns"`
}

type SendProjectCreationMessageRequest = ConversationMessageRequest

type SendProjectCreationMessageResponse struct {
	Session          *ProjectCreationSessionResponse `json:"session"`
	UserTurnID       string                          `json:"user_turn_id"`
	AssistantTurnID  string                          `json:"assistant_turn_id"`
	AssistantMessage string                          `json:"assistant_message"`
	ProjectID        *string                         `json:"project_id,omitempty"`
	ProjectSessionID *string                         `json:"project_session_id,omitempty"`
	Usage            *FoundationUsageResponse        `json:"usage,omitempty"`
}
