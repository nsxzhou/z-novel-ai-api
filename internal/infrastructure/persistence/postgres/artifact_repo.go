// Package postgres 提供 PostgreSQL Repository 实现
package postgres

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"z-novel-ai-api/internal/domain/entity"
	"z-novel-ai-api/internal/domain/repository"
)

type ArtifactRepository struct {
	client *Client
}

func NewArtifactRepository(client *Client) *ArtifactRepository {
	return &ArtifactRepository{client: client}
}

func (r *ArtifactRepository) EnsureArtifact(ctx context.Context, tenantID, projectID string, artifactType entity.ArtifactType) (*entity.ProjectArtifact, error) {
	ctx, span := tracer.Start(ctx, "postgres.ArtifactRepository.EnsureArtifact")
	defer span.End()

	db := getDB(ctx, r.client.db)

	var art entity.ProjectArtifact
	err := db.First(&art, "project_id = ? AND type = ?", projectID, artifactType).Error
	if err == nil {
		return &art, nil
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to get project artifact: %w", err)
	}

	created := &entity.ProjectArtifact{
		TenantID:  tenantID,
		ProjectID: projectID,
		Type:      artifactType,
	}
	if err := db.Create(created).Error; err != nil {
		// 处理并发创建：唯一约束命中时回读
		var art2 entity.ProjectArtifact
		if readErr := db.First(&art2, "project_id = ? AND type = ?", projectID, artifactType).Error; readErr == nil {
			return &art2, nil
		}
		span.RecordError(err)
		return nil, fmt.Errorf("failed to create project artifact: %w", err)
	}
	return created, nil
}

func (r *ArtifactRepository) GetArtifactByID(ctx context.Context, id string) (*entity.ProjectArtifact, error) {
	ctx, span := tracer.Start(ctx, "postgres.ArtifactRepository.GetArtifactByID")
	defer span.End()

	db := getDB(ctx, r.client.db)
	var art entity.ProjectArtifact
	if err := db.First(&art, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		span.RecordError(err)
		return nil, fmt.Errorf("failed to get project artifact: %w", err)
	}
	return &art, nil
}

func (r *ArtifactRepository) ListArtifactsByProject(ctx context.Context, projectID string) ([]*entity.ProjectArtifact, error) {
	ctx, span := tracer.Start(ctx, "postgres.ArtifactRepository.ListArtifactsByProject")
	defer span.End()

	db := getDB(ctx, r.client.db)
	var arts []*entity.ProjectArtifact
	if err := db.Where("project_id = ?", projectID).Order("created_at ASC").Find(&arts).Error; err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to list project artifacts: %w", err)
	}
	return arts, nil
}

func (r *ArtifactRepository) CreateVersion(ctx context.Context, version *entity.ArtifactVersion) error {
	ctx, span := tracer.Start(ctx, "postgres.ArtifactRepository.CreateVersion")
	defer span.End()

	db := getDB(ctx, r.client.db)
	if err := db.Create(version).Error; err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to create artifact version: %w", err)
	}
	return nil
}

func (r *ArtifactRepository) GetLatestVersionNo(ctx context.Context, artifactID string) (int, error) {
	ctx, span := tracer.Start(ctx, "postgres.ArtifactRepository.GetLatestVersionNo")
	defer span.End()

	db := getDB(ctx, r.client.db)
	var maxNo *int
	if err := db.Model(&entity.ArtifactVersion{}).
		Where("artifact_id = ?", artifactID).
		Select("MAX(version_no)").
		Scan(&maxNo).Error; err != nil {
		span.RecordError(err)
		return 0, fmt.Errorf("failed to get latest version_no: %w", err)
	}
	if maxNo == nil {
		return 0, nil
	}
	return *maxNo, nil
}

func (r *ArtifactRepository) GetVersionByID(ctx context.Context, id string) (*entity.ArtifactVersion, error) {
	ctx, span := tracer.Start(ctx, "postgres.ArtifactRepository.GetVersionByID")
	defer span.End()

	db := getDB(ctx, r.client.db)
	var v entity.ArtifactVersion
	if err := db.First(&v, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		span.RecordError(err)
		return nil, fmt.Errorf("failed to get artifact version: %w", err)
	}
	return &v, nil
}

func (r *ArtifactRepository) ListVersions(ctx context.Context, artifactID string, branchKey string, pagination repository.Pagination) (*repository.PagedResult[*entity.ArtifactVersion], error) {
	ctx, span := tracer.Start(ctx, "postgres.ArtifactRepository.ListVersions")
	defer span.End()

	db := getDB(ctx, r.client.db)
	query := db.Model(&entity.ArtifactVersion{}).Where("artifact_id = ?", artifactID)
	if branchKey != "" {
		query = query.Where("branch_key = ?", branchKey)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to count artifact versions: %w", err)
	}

	var versions []*entity.ArtifactVersion
	if err := query.Order("version_no DESC").
		Offset(pagination.Offset()).
		Limit(pagination.Limit()).
		Find(&versions).Error; err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to list artifact versions: %w", err)
	}

	return repository.NewPagedResult(versions, total, pagination), nil
}

func (r *ArtifactRepository) GetLatestVersionByBranch(ctx context.Context, artifactID string, branchKey string) (*entity.ArtifactVersion, error) {
	ctx, span := tracer.Start(ctx, "postgres.ArtifactRepository.GetLatestVersionByBranch")
	defer span.End()

	db := getDB(ctx, r.client.db)
	var v entity.ArtifactVersion
	if err := db.Where("artifact_id = ? AND branch_key = ?", artifactID, branchKey).Order("version_no DESC").First(&v).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		span.RecordError(err)
		return nil, fmt.Errorf("failed to get latest artifact version by branch: %w", err)
	}
	return &v, nil
}

func (r *ArtifactRepository) ListBranchHeads(ctx context.Context, artifactID string) ([]*entity.ArtifactVersion, error) {
	ctx, span := tracer.Start(ctx, "postgres.ArtifactRepository.ListBranchHeads")
	defer span.End()

	db := getDB(ctx, r.client.db)
	var versions []*entity.ArtifactVersion
	if err := db.Raw(`
SELECT DISTINCT ON (branch_key)
    id, artifact_id, version_no, branch_key, parent_version_id, created_by, source_job_id, created_at
FROM artifact_versions
WHERE artifact_id = ?
ORDER BY branch_key, version_no DESC;
`, artifactID).Scan(&versions).Error; err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to list artifact branch heads: %w", err)
	}
	return versions, nil
}

func (r *ArtifactRepository) SetActiveVersion(ctx context.Context, artifactID, versionID string) error {
	ctx, span := tracer.Start(ctx, "postgres.ArtifactRepository.SetActiveVersion")
	defer span.End()

	db := getDB(ctx, r.client.db)
	if err := db.Model(&entity.ProjectArtifact{}).
		Where("id = ?", artifactID).
		Update("active_version_id", versionID).Error; err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to set active version: %w", err)
	}
	return nil
}
