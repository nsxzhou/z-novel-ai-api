// Package postgres 提供 PostgreSQL Repository 实现
package postgres

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"z-novel-ai-api/internal/domain/entity"
	"z-novel-ai-api/internal/domain/repository"
)

// ProjectRepository 项目仓储实现
type ProjectRepository struct {
	client *Client
}

// NewProjectRepository 创建项目仓储
func NewProjectRepository(client *Client) *ProjectRepository {
	return &ProjectRepository{client: client}
}

// Create 创建项目
func (r *ProjectRepository) Create(ctx context.Context, project *entity.Project) error {
	ctx, span := tracer.Start(ctx, "postgres.ProjectRepository.Create")
	defer span.End()

	db := getDB(ctx, r.client.db)
	if err := db.Create(project).Error; err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to create project: %w", err)
	}
	return nil
}

// GetByID 根据 ID 获取项目
func (r *ProjectRepository) GetByID(ctx context.Context, id string) (*entity.Project, error) {
	ctx, span := tracer.Start(ctx, "postgres.ProjectRepository.GetByID")
	defer span.End()

	db := getDB(ctx, r.client.db)
	var project entity.Project
	if err := db.First(&project, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		span.RecordError(err)
		return nil, fmt.Errorf("failed to get project: %w", err)
	}
	return &project, nil
}

// Update 更新项目
func (r *ProjectRepository) Update(ctx context.Context, project *entity.Project) error {
	ctx, span := tracer.Start(ctx, "postgres.ProjectRepository.Update")
	defer span.End()

	db := getDB(ctx, r.client.db)
	if err := db.Save(project).Error; err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to update project: %w", err)
	}
	return nil
}

// Delete 删除项目
func (r *ProjectRepository) Delete(ctx context.Context, id string) error {
	ctx, span := tracer.Start(ctx, "postgres.ProjectRepository.Delete")
	defer span.End()

	db := getDB(ctx, r.client.db)
	if err := db.Delete(&entity.Project{}, "id = ?", id).Error; err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to delete project: %w", err)
	}
	return nil
}

// List 获取项目列表
func (r *ProjectRepository) List(ctx context.Context, filter *repository.ProjectFilter, pagination repository.Pagination) (*repository.PagedResult[*entity.Project], error) {
	ctx, span := tracer.Start(ctx, "postgres.ProjectRepository.List")
	defer span.End()

	db := getDB(ctx, r.client.db)
	query := db.Model(&entity.Project{})

	// 应用过滤条件
	if filter != nil {
		if filter.OwnerID != "" {
			query = query.Where("owner_id = ?", filter.OwnerID)
		}
		if filter.Genre != "" {
			query = query.Where("genre = ?", filter.Genre)
		}
		if filter.Status != "" {
			query = query.Where("status = ?", filter.Status)
		}
	}

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to count projects: %w", err)
	}

	// 获取列表
	var projects []*entity.Project
	if err := query.Order("updated_at DESC").
		Offset(pagination.Offset()).
		Limit(pagination.Limit()).
		Find(&projects).Error; err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	return repository.NewPagedResult(projects, total, pagination), nil
}

// ListByOwner 获取用户项目列表
func (r *ProjectRepository) ListByOwner(ctx context.Context, ownerID string, pagination repository.Pagination) (*repository.PagedResult[*entity.Project], error) {
	return r.List(ctx, &repository.ProjectFilter{OwnerID: ownerID}, pagination)
}

// UpdateStatus 更新项目状态
func (r *ProjectRepository) UpdateStatus(ctx context.Context, id string, status entity.ProjectStatus) error {
	ctx, span := tracer.Start(ctx, "postgres.ProjectRepository.UpdateStatus")
	defer span.End()

	db := getDB(ctx, r.client.db)
	if err := db.Model(&entity.Project{}).Where("id = ?", id).Update("status", status).Error; err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to update project status: %w", err)
	}
	return nil
}

// UpdateWordCount 更新字数统计
func (r *ProjectRepository) UpdateWordCount(ctx context.Context, id string, wordCount int) error {
	ctx, span := tracer.Start(ctx, "postgres.ProjectRepository.UpdateWordCount")
	defer span.End()

	db := getDB(ctx, r.client.db)
	if err := db.Model(&entity.Project{}).Where("id = ?", id).Update("current_word_count", wordCount).Error; err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to update word count: %w", err)
	}
	return nil
}

// GetStats 获取项目统计信息
func (r *ProjectRepository) GetStats(ctx context.Context, id string) (*repository.ProjectStats, error) {
	ctx, span := tracer.Start(ctx, "postgres.ProjectRepository.GetStats")
	defer span.End()

	db := getDB(ctx, r.client.db)
	var stats repository.ProjectStats

	// 使用原生 SQL 进行聚合查询
	err := db.Raw(`
		SELECT 
			COALESCE((SELECT COUNT(*) FROM chapters WHERE project_id = ?), 0) as total_chapters,
			COALESCE((SELECT COUNT(*) FROM volumes WHERE project_id = ?), 0) as total_volumes,
			COALESCE((SELECT COUNT(*) FROM entities WHERE project_id = ?), 0) as total_entities,
			COALESCE((SELECT SUM(word_count) FROM chapters WHERE project_id = ?), 0) as total_word_count
	`, id, id, id, id).Scan(&stats).Error

	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to get project stats: %w", err)
	}

	return &stats, nil
}
