// Package postgres 提供 PostgreSQL Repository 实现
package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

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

	q := getQuerier(ctx, r.client.db)

	settingsJSON, _ := json.Marshal(project.Settings)
	worldSettingsJSON, _ := json.Marshal(project.WorldSettings)

	query := `
		INSERT INTO projects (id, tenant_id, owner_id, title, description, genre, target_word_count, 
			current_word_count, settings, world_settings, status, created_at, updated_at)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`

	var ownerID sql.NullString
	if project.OwnerID != "" {
		ownerID = sql.NullString{String: project.OwnerID, Valid: true}
	}

	err := q.QueryRowContext(ctx, query,
		project.TenantID, ownerID, project.Title, project.Description, project.Genre,
		project.TargetWordCount, project.CurrentWordCount, settingsJSON, worldSettingsJSON, project.Status,
	).Scan(&project.ID, &project.CreatedAt, &project.UpdatedAt)

	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to create project: %w", err)
	}

	return nil
}

// GetByID 根据 ID 获取项目
func (r *ProjectRepository) GetByID(ctx context.Context, id string) (*entity.Project, error) {
	ctx, span := tracer.Start(ctx, "postgres.ProjectRepository.GetByID")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	query := `
		SELECT id, tenant_id, owner_id, title, description, genre, target_word_count, 
			current_word_count, settings, world_settings, status, created_at, updated_at
		FROM projects
		WHERE id = $1
	`

	var project entity.Project
	var ownerID sql.NullString
	var settingsJSON, worldSettingsJSON []byte

	err := q.QueryRowContext(ctx, query, id).Scan(
		&project.ID, &project.TenantID, &ownerID, &project.Title, &project.Description,
		&project.Genre, &project.TargetWordCount, &project.CurrentWordCount,
		&settingsJSON, &worldSettingsJSON, &project.Status,
		&project.CreatedAt, &project.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		span.RecordError(err)
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	if ownerID.Valid {
		project.OwnerID = ownerID.String
	}
	json.Unmarshal(settingsJSON, &project.Settings)
	json.Unmarshal(worldSettingsJSON, &project.WorldSettings)

	return &project, nil
}

// Update 更新项目
func (r *ProjectRepository) Update(ctx context.Context, project *entity.Project) error {
	ctx, span := tracer.Start(ctx, "postgres.ProjectRepository.Update")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	settingsJSON, _ := json.Marshal(project.Settings)
	worldSettingsJSON, _ := json.Marshal(project.WorldSettings)

	query := `
		UPDATE projects
		SET title = $1, description = $2, genre = $3, target_word_count = $4, 
			current_word_count = $5, settings = $6, world_settings = $7, status = $8, updated_at = NOW()
		WHERE id = $9
		RETURNING updated_at
	`

	err := q.QueryRowContext(ctx, query,
		project.Title, project.Description, project.Genre, project.TargetWordCount,
		project.CurrentWordCount, settingsJSON, worldSettingsJSON, project.Status, project.ID,
	).Scan(&project.UpdatedAt)

	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to update project: %w", err)
	}

	return nil
}

// Delete 删除项目
func (r *ProjectRepository) Delete(ctx context.Context, id string) error {
	ctx, span := tracer.Start(ctx, "postgres.ProjectRepository.Delete")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	query := `DELETE FROM projects WHERE id = $1`
	_, err := q.ExecContext(ctx, query, id)

	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to delete project: %w", err)
	}

	return nil
}

// List 获取项目列表
func (r *ProjectRepository) List(ctx context.Context, filter *repository.ProjectFilter, pagination repository.Pagination) (*repository.PagedResult[*entity.Project], error) {
	ctx, span := tracer.Start(ctx, "postgres.ProjectRepository.List")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	// 构建查询条件
	whereClause := "1=1"
	args := []interface{}{}
	argIdx := 1

	if filter != nil {
		if filter.OwnerID != "" {
			whereClause += fmt.Sprintf(" AND owner_id = $%d", argIdx)
			args = append(args, filter.OwnerID)
			argIdx++
		}
		if filter.Genre != "" {
			whereClause += fmt.Sprintf(" AND genre = $%d", argIdx)
			args = append(args, filter.Genre)
			argIdx++
		}
		if filter.Status != "" {
			whereClause += fmt.Sprintf(" AND status = $%d", argIdx)
			args = append(args, filter.Status)
			argIdx++
		}
	}

	// 获取总数
	var total int64
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM projects WHERE %s`, whereClause)
	if err := q.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to count projects: %w", err)
	}

	// 获取列表
	query := fmt.Sprintf(`
		SELECT id, tenant_id, owner_id, title, description, genre, target_word_count, 
			current_word_count, settings, world_settings, status, created_at, updated_at
		FROM projects
		WHERE %s
		ORDER BY updated_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)

	args = append(args, pagination.Limit(), pagination.Offset())

	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}
	defer rows.Close()

	var projects []*entity.Project
	for rows.Next() {
		var project entity.Project
		var ownerID sql.NullString
		var settingsJSON, worldSettingsJSON []byte

		if err := rows.Scan(
			&project.ID, &project.TenantID, &ownerID, &project.Title, &project.Description,
			&project.Genre, &project.TargetWordCount, &project.CurrentWordCount,
			&settingsJSON, &worldSettingsJSON, &project.Status,
			&project.CreatedAt, &project.UpdatedAt,
		); err != nil {
			span.RecordError(err)
			return nil, fmt.Errorf("failed to scan project: %w", err)
		}

		if ownerID.Valid {
			project.OwnerID = ownerID.String
		}
		json.Unmarshal(settingsJSON, &project.Settings)
		json.Unmarshal(worldSettingsJSON, &project.WorldSettings)
		projects = append(projects, &project)
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

	q := getQuerier(ctx, r.client.db)

	query := `UPDATE projects SET status = $1, updated_at = NOW() WHERE id = $2`
	_, err := q.ExecContext(ctx, query, status, id)

	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to update project status: %w", err)
	}

	return nil
}

// UpdateWordCount 更新字数统计
func (r *ProjectRepository) UpdateWordCount(ctx context.Context, id string, wordCount int) error {
	ctx, span := tracer.Start(ctx, "postgres.ProjectRepository.UpdateWordCount")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	query := `UPDATE projects SET current_word_count = $1, updated_at = NOW() WHERE id = $2`
	_, err := q.ExecContext(ctx, query, wordCount, id)

	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to update word count: %w", err)
	}

	return nil
}

// GetStats 获取项目统计信息
func (r *ProjectRepository) GetStats(ctx context.Context, id string) (*repository.ProjectStats, error) {
	ctx, span := tracer.Start(ctx, "postgres.ProjectRepository.GetStats")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	query := `
		SELECT 
			COALESCE((SELECT COUNT(*) FROM chapters WHERE project_id = $1), 0) as total_chapters,
			COALESCE((SELECT COUNT(*) FROM volumes WHERE project_id = $1), 0) as total_volumes,
			COALESCE((SELECT COUNT(*) FROM entities WHERE project_id = $1), 0) as total_entities,
			COALESCE((SELECT SUM(word_count) FROM chapters WHERE project_id = $1), 0) as total_word_count
	`

	var stats repository.ProjectStats
	err := q.QueryRowContext(ctx, query, id).Scan(
		&stats.TotalChapters, &stats.TotalVolumes, &stats.TotalEntities, &stats.TotalWordCount,
	)

	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to get project stats: %w", err)
	}

	return &stats, nil
}
