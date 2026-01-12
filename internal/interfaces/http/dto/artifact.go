// Package dto 提供 HTTP 层数据传输对象
package dto

import (
	"encoding/json"
	"time"

	storyartifact "z-novel-ai-api/internal/application/story/artifact"
	"z-novel-ai-api/internal/domain/entity"
)

type ArtifactResponse struct {
	ID              string  `json:"id"`
	ProjectID       string  `json:"project_id"`
	Type            string  `json:"type"`
	ActiveVersionID *string `json:"active_version_id,omitempty"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
}

func ToArtifactResponse(a *entity.ProjectArtifact) *ArtifactResponse {
	if a == nil {
		return nil
	}
	return &ArtifactResponse{
		ID:              a.ID,
		ProjectID:       a.ProjectID,
		Type:            string(a.Type),
		ActiveVersionID: a.ActiveVersionID,
		CreatedAt:       a.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:       a.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

type ArtifactListResponse struct {
	Artifacts []*ArtifactResponse `json:"artifacts"`
}

type ArtifactVersionResponse struct {
	ID              string          `json:"id"`
	ArtifactID      string          `json:"artifact_id"`
	VersionNo       int             `json:"version_no"`
	BranchKey       string          `json:"branch_key"`
	ParentVersionID *string         `json:"parent_version_id,omitempty"`
	Content         json.RawMessage `json:"content"`
	CreatedBy       *string         `json:"created_by,omitempty"`
	SourceJobID     *string         `json:"source_job_id,omitempty"`
	CreatedAt       string          `json:"created_at"`
}

func ToArtifactVersionResponse(v *entity.ArtifactVersion) *ArtifactVersionResponse {
	if v == nil {
		return nil
	}
	return &ArtifactVersionResponse{
		ID:              v.ID,
		ArtifactID:      v.ArtifactID,
		VersionNo:       v.VersionNo,
		BranchKey:       v.BranchKey,
		ParentVersionID: v.ParentVersionID,
		Content:         v.Content,
		CreatedBy:       v.CreatedBy,
		SourceJobID:     v.SourceJobID,
		CreatedAt:       v.CreatedAt.UTC().Format(time.RFC3339),
	}
}

type ArtifactVersionListResponse struct {
	Versions []*ArtifactVersionResponse `json:"versions"`
}

type ArtifactRollbackRequest struct {
	VersionID string `json:"version_id" binding:"required"`
}

type ArtifactRollbackResponse struct {
	Artifact *ArtifactResponse        `json:"artifact"`
	Version  *ArtifactVersionResponse `json:"version"`
}

type ArtifactBranchHeadResponse struct {
	BranchKey   string  `json:"branch_key"`
	HeadVersion string  `json:"head_version_id"`
	HeadNo      int     `json:"head_version_no"`
	CreatedAt   string  `json:"created_at"`
	CreatedBy   *string `json:"created_by,omitempty"`
	SourceJobID *string `json:"source_job_id,omitempty"`
	IsActive    bool    `json:"is_active"`
}

type ArtifactBranchListResponse struct {
	Branches []*ArtifactBranchHeadResponse `json:"branches"`
}

type ArtifactCompareResponse struct {
	ArtifactID string                             `json:"artifact_id"`
	Type       string                             `json:"type"`
	From       *ArtifactVersionResponse           `json:"from"`
	To         *ArtifactVersionResponse           `json:"to"`
	Diff       *storyartifact.ArtifactCompareDiff `json:"diff,omitempty"`
}
