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
	// ListVersions 列出版本；branchKey 为空表示不过滤
	ListVersions(ctx context.Context, artifactID string, branchKey string, pagination Pagination) (*PagedResult[*entity.ArtifactVersion], error)
	// GetLatestVersionByBranch 获取指定分支的最新版本；不存在返回 nil
	GetLatestVersionByBranch(ctx context.Context, artifactID string, branchKey string) (*entity.ArtifactVersion, error)
	// ListBranchHeads 返回每个分支的最新版本（按 branch_key 聚合）
	ListBranchHeads(ctx context.Context, artifactID string) ([]*entity.ArtifactVersion, error)

	// SetActiveVersion 设置激活版本
	SetActiveVersion(ctx context.Context, artifactID, versionID string) error
}
