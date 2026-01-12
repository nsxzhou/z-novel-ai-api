package model

import (
	"encoding/json"

	"z-novel-ai-api/internal/domain/entity"
)

type ArtifactConflictSeverity string

const (
	ArtifactConflictSeverityHigh   ArtifactConflictSeverity = "high"
	ArtifactConflictSeverityMedium ArtifactConflictSeverity = "medium"
	ArtifactConflictSeverityLow    ArtifactConflictSeverity = "low"
)

type ArtifactConflict struct {
	Severity    ArtifactConflictSeverity `json:"severity"`
	Message     string                   `json:"message"`
	ExistingRef string                   `json:"existing_ref,omitempty"`
	NewRef      string                   `json:"new_ref,omitempty"`
	Suggestion  string                   `json:"suggestion,omitempty"`
}

type ArtifactConflictScanInput struct {
	ProjectTitle       string
	ProjectDescription string
	ProjectGenre       string

	Type entity.ArtifactType

	CurrentWorldview  json.RawMessage
	CurrentCharacters json.RawMessage
	CurrentOutline    json.RawMessage
	CurrentArtifact   json.RawMessage
	NewArtifact       json.RawMessage

	Provider string
	Model    string

	Temperature *float32
	MaxTokens   *int
}

type ArtifactConflictScanOutput struct {
	Conflicts []ArtifactConflict
	Raw       string
	Meta      LLMUsageMeta
}
