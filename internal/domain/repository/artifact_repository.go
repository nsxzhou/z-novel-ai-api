// Package repository 定义数据访问层接口
package repository

import (
	"context"

	"z-novel-ai-api/internal/domain/entity"
)

type ArtifactRepository interface {
	// EnsureArtifact 获取或创建构件（按 project_id+type 唯一）
	EnsureArtifact(ctx context.Context, tenantID, projectID string, artifactType entity.ArtifactType) (*entity.ProjectArtifact, error)
	GetArtifactByID(ctx context.Context, id string) (*entity.ProjectArtifact, error)
	ListArtifactsByProject(ctx context.Context, projectID string) ([]*entity.ProjectArtifact, error)

	// CreateVersion 创建新版本（要求 version_no 单调递增）
	CreateVersion(ctx context.Context, version *entity.ArtifactVersion) error
	GetLatestVersionNo(ctx context.Context, artifactID string) (int, error)
	GetVersionByID(ctx context.Context, id string) (*entity.ArtifactVersion, error)
	ListVersions(ctx context.Context, artifactID string, pagination Pagination) (*PagedResult[*entity.ArtifactVersion], error)

	// SetActiveVersion 设置激活版本
	SetActiveVersion(ctx context.Context, artifactID, versionID string) error
}
