// Package entity 定义领域实体
package entity

import (
	"encoding/json"
	"fmt"
	"time"
)

type ArtifactType string

const (
	ArtifactTypeNovelFoundation ArtifactType = "novel_foundation"
	ArtifactTypeWorldview       ArtifactType = "worldview"
	ArtifactTypeCharacters      ArtifactType = "characters"
	ArtifactTypeOutline         ArtifactType = "outline"
)

type ProjectArtifact struct {
	ID              string       `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TenantID        string       `json:"tenant_id" gorm:"type:uuid;index;not null"`
	ProjectID       string       `json:"project_id" gorm:"type:uuid;index;not null"`
	Type            ArtifactType `json:"type" gorm:"type:varchar(32);not null"`
	ActiveVersionID *string      `json:"active_version_id,omitempty" gorm:"type:uuid"`
	CreatedAt       time.Time    `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt       time.Time    `json:"updated_at" gorm:"autoUpdateTime"`
}

func (ProjectArtifact) TableName() string {
	return "project_artifacts"
}

type ArtifactVersion struct {
	ID          string          `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ArtifactID  string          `json:"artifact_id" gorm:"type:uuid;index;not null"`
	VersionNo   int             `json:"version_no" gorm:"not null"`
	Content     json.RawMessage `json:"content" gorm:"type:jsonb;not null"`
	CreatedBy   *string         `json:"created_by,omitempty" gorm:"type:uuid"`
	SourceJobID *string         `json:"source_job_id,omitempty" gorm:"type:uuid"`
	CreatedAt   time.Time       `json:"created_at" gorm:"autoCreateTime"`
}

func (ArtifactVersion) TableName() string {
	return "artifact_versions"
}

func TaskToArtifactType(task ConversationTask) (ArtifactType, error) {
	switch task {
	case ConversationTaskNovelFoundation:
		return ArtifactTypeNovelFoundation, nil
	case ConversationTaskWorldview:
		return ArtifactTypeWorldview, nil
	case ConversationTaskCharacters:
		return ArtifactTypeCharacters, nil
	case ConversationTaskOutline:
		return ArtifactTypeOutline, nil
	default:
		return "", fmt.Errorf("invalid task: %s", task)
	}
}
