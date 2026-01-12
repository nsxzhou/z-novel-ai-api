package model

import (
	"encoding/json"

	"z-novel-ai-api/internal/domain/entity"
)

const DefaultMaxToolRounds = 4
const DefaultMaxRepairRounds = 2

type ArtifactGenerateInput struct {
	TenantID  string
	ProjectID string

	ProjectTitle       string
	ProjectDescription string

	Type entity.ArtifactType

	Prompt      string
	Attachments []TextAttachment

	ConversationSummary string
	RecentUserTurns     string

	CurrentWorldview   json.RawMessage
	CurrentCharacters  json.RawMessage
	CurrentOutline     json.RawMessage
	CurrentArtifactRaw json.RawMessage

	Provider string
	Model    string

	Temperature *float32
	MaxTokens   *int
}

type ArtifactGenerateOutput struct {
	Type     entity.ArtifactType
	Content  json.RawMessage
	Raw      string
	ModelRaw string
	Mode     string
	Meta     LLMUsageMeta
}
