// Package postgres 提供 PostgreSQL Repository 实现
package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/lib/pq"

	"z-novel-ai-api/internal/domain/entity"
	"z-novel-ai-api/internal/domain/repository"
)

// EntityRepository 实体仓储实现
type EntityRepository struct {
	client *Client
}

// NewEntityRepository 创建实体仓储
func NewEntityRepository(client *Client) *EntityRepository {
	return &EntityRepository{client: client}
}

// Create 创建实体
func (r *EntityRepository) Create(ctx context.Context, storyEntity *entity.StoryEntity) error {
	ctx, span := tracer.Start(ctx, "postgres.EntityRepository.Create")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	attributesJSON, _ := json.Marshal(storyEntity.Attributes)
	metadataJSON, _ := json.Marshal(storyEntity.Metadata)

	query := `
		INSERT INTO entities (id, project_id, name, aliases, type, description, attributes, metadata, current_state, 
			first_appear_chapter_id, last_appear_chapter_id, appear_count, importance, vector_id, created_at, updated_at)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`

	var firstAppear, lastAppear, vectorID sql.NullString
	if storyEntity.FirstAppearChapterID != "" {
		firstAppear = sql.NullString{String: storyEntity.FirstAppearChapterID, Valid: true}
	}
	if storyEntity.LastAppearChapterID != "" {
		lastAppear = sql.NullString{String: storyEntity.LastAppearChapterID, Valid: true}
	}
	if storyEntity.VectorID != "" {
		vectorID = sql.NullString{String: storyEntity.VectorID, Valid: true}
	}

	err := q.QueryRowContext(ctx, query,
		storyEntity.ProjectID, storyEntity.Name, pq.Array(storyEntity.Aliases), storyEntity.Type,
		storyEntity.Description, attributesJSON, metadataJSON, storyEntity.CurrentState,
		firstAppear, lastAppear, storyEntity.AppearCount, storyEntity.Importance, vectorID,
	).Scan(&storyEntity.ID, &storyEntity.CreatedAt, &storyEntity.UpdatedAt)

	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to create entity: %w", err)
	}

	return nil
}

// GetByID 根据 ID 获取实体
func (r *EntityRepository) GetByID(ctx context.Context, id string) (*entity.StoryEntity, error) {
	ctx, span := tracer.Start(ctx, "postgres.EntityRepository.GetByID")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	query := `
		SELECT id, project_id, name, aliases, type, description, attributes, metadata, current_state, 
			first_appear_chapter_id, last_appear_chapter_id, appear_count, importance, vector_id, created_at, updated_at
		FROM entities
		WHERE id = $1
	`

	return r.scanEntity(q.QueryRowContext(ctx, query, id))
}

// Update 更新实体
func (r *EntityRepository) Update(ctx context.Context, storyEntity *entity.StoryEntity) error {
	ctx, span := tracer.Start(ctx, "postgres.EntityRepository.Update")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	attributesJSON, _ := json.Marshal(storyEntity.Attributes)
	metadataJSON, _ := json.Marshal(storyEntity.Metadata)

	query := `
		UPDATE entities
		SET name = $1, aliases = $2, description = $3, attributes = $4, metadata = $5, current_state = $6, importance = $7, updated_at = NOW()
		WHERE id = $8
		RETURNING updated_at
	`

	err := q.QueryRowContext(ctx, query,
		storyEntity.Name, pq.Array(storyEntity.Aliases), storyEntity.Description,
		attributesJSON, metadataJSON, storyEntity.CurrentState, storyEntity.Importance, storyEntity.ID,
	).Scan(&storyEntity.UpdatedAt)

	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to update entity: %w", err)
	}

	return nil
}

// Delete 删除实体
func (r *EntityRepository) Delete(ctx context.Context, id string) error {
	ctx, span := tracer.Start(ctx, "postgres.EntityRepository.Delete")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	query := `DELETE FROM entities WHERE id = $1`
	_, err := q.ExecContext(ctx, query, id)

	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to delete entity: %w", err)
	}

	return nil
}

// ListByProject 获取项目实体列表
func (r *EntityRepository) ListByProject(ctx context.Context, projectID string, filter *repository.EntityFilter, pagination repository.Pagination) (*repository.PagedResult[*entity.StoryEntity], error) {
	ctx, span := tracer.Start(ctx, "postgres.EntityRepository.ListByProject")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	// 构建查询条件
	whereClause := "project_id = $1"
	args := []interface{}{projectID}
	argIdx := 2

	if filter != nil {
		if filter.Type != "" {
			whereClause += fmt.Sprintf(" AND type = $%d", argIdx)
			args = append(args, filter.Type)
			argIdx++
		}
		if filter.Importance != "" {
			whereClause += fmt.Sprintf(" AND importance = $%d", argIdx)
			args = append(args, filter.Importance)
			argIdx++
		}
		if filter.Name != "" {
			whereClause += fmt.Sprintf(" AND name ILIKE $%d", argIdx)
			args = append(args, "%"+filter.Name+"%")
			argIdx++
		}
	}

	// 获取总数
	var total int64
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM entities WHERE %s`, whereClause)
	if err := q.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to count entities: %w", err)
	}

	// 获取列表
	query := fmt.Sprintf(`
		SELECT id, project_id, name, aliases, type, description, attributes, metadata, current_state, 
			first_appear_chapter_id, last_appear_chapter_id, appear_count, importance, vector_id, created_at, updated_at
		FROM entities
		WHERE %s
		ORDER BY importance DESC, appear_count DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)

	args = append(args, pagination.Limit(), pagination.Offset())

	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to list entities: %w", err)
	}
	defer rows.Close()

	var entities []*entity.StoryEntity
	for rows.Next() {
		e, err := r.scanEntityFromRows(rows)
		if err != nil {
			span.RecordError(err)
			return nil, err
		}
		entities = append(entities, e)
	}

	return repository.NewPagedResult(entities, total, pagination), nil
}

// GetByName 根据名称获取实体
func (r *EntityRepository) GetByName(ctx context.Context, projectID, name string) (*entity.StoryEntity, error) {
	ctx, span := tracer.Start(ctx, "postgres.EntityRepository.GetByName")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	query := `
		SELECT id, project_id, name, aliases, type, description, attributes, metadata, current_state, 
			first_appear_chapter_id, last_appear_chapter_id, appear_count, importance, vector_id, created_at, updated_at
		FROM entities
		WHERE project_id = $1 AND name = $2
	`

	return r.scanEntity(q.QueryRowContext(ctx, query, projectID, name))
}

// SearchByName 搜索实体名称（支持别名）
func (r *EntityRepository) SearchByName(ctx context.Context, projectID, queryStr string, limit int) ([]*entity.StoryEntity, error) {
	ctx, span := tracer.Start(ctx, "postgres.EntityRepository.SearchByName")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	query := `
		SELECT id, project_id, name, aliases, type, description, attributes, metadata, current_state, 
			first_appear_chapter_id, last_appear_chapter_id, appear_count, importance, vector_id, created_at, updated_at
		FROM entities
		WHERE project_id = $1 AND (name ILIKE $2 OR $2 = ANY(aliases))
		ORDER BY importance DESC
		LIMIT $3
	`

	rows, err := q.QueryContext(ctx, query, projectID, "%"+queryStr+"%", limit)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to search entities: %w", err)
	}
	defer rows.Close()

	var entities []*entity.StoryEntity
	for rows.Next() {
		e, err := r.scanEntityFromRows(rows)
		if err != nil {
			span.RecordError(err)
			return nil, err
		}
		entities = append(entities, e)
	}

	return entities, nil
}

// UpdateState 更新实体状态
func (r *EntityRepository) UpdateState(ctx context.Context, id, state string) error {
	ctx, span := tracer.Start(ctx, "postgres.EntityRepository.UpdateState")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	query := `UPDATE entities SET current_state = $1, updated_at = NOW() WHERE id = $2`
	_, err := q.ExecContext(ctx, query, state, id)

	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to update entity state: %w", err)
	}

	return nil
}

// UpdateVectorID 更新向量 ID
func (r *EntityRepository) UpdateVectorID(ctx context.Context, id, vectorID string) error {
	ctx, span := tracer.Start(ctx, "postgres.EntityRepository.UpdateVectorID")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	query := `UPDATE entities SET vector_id = $1, updated_at = NOW() WHERE id = $2`
	_, err := q.ExecContext(ctx, query, vectorID, id)

	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to update vector id: %w", err)
	}

	return nil
}

// RecordAppearance 记录出场
func (r *EntityRepository) RecordAppearance(ctx context.Context, id, chapterID string) error {
	ctx, span := tracer.Start(ctx, "postgres.EntityRepository.RecordAppearance")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	query := `
		UPDATE entities 
		SET first_appear_chapter_id = COALESCE(first_appear_chapter_id, $1),
			last_appear_chapter_id = $1,
			appear_count = appear_count + 1,
			updated_at = NOW()
		WHERE id = $2
	`
	_, err := q.ExecContext(ctx, query, chapterID, id)

	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to record appearance: %w", err)
	}

	return nil
}

// GetByType 根据类型获取实体列表
func (r *EntityRepository) GetByType(ctx context.Context, projectID string, entityType entity.StoryEntityType) ([]*entity.StoryEntity, error) {
	ctx, span := tracer.Start(ctx, "postgres.EntityRepository.GetByType")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	query := `
		SELECT id, project_id, name, aliases, type, description, attributes, metadata, current_state, 
			first_appear_chapter_id, last_appear_chapter_id, appear_count, importance, vector_id, created_at, updated_at
		FROM entities
		WHERE project_id = $1 AND type = $2
		ORDER BY importance DESC, appear_count DESC
	`

	rows, err := q.QueryContext(ctx, query, projectID, entityType)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to get entities by type: %w", err)
	}
	defer rows.Close()

	var entities []*entity.StoryEntity
	for rows.Next() {
		e, err := r.scanEntityFromRows(rows)
		if err != nil {
			span.RecordError(err)
			return nil, err
		}
		entities = append(entities, e)
	}

	return entities, nil
}

// GetProtagonists 获取主角列表
func (r *EntityRepository) GetProtagonists(ctx context.Context, projectID string) ([]*entity.StoryEntity, error) {
	ctx, span := tracer.Start(ctx, "postgres.EntityRepository.GetProtagonists")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	query := `
		SELECT id, project_id, name, aliases, type, description, attributes, metadata, current_state, 
			first_appear_chapter_id, last_appear_chapter_id, appear_count, importance, vector_id, created_at, updated_at
		FROM entities
		WHERE project_id = $1 AND type = 'character' AND importance = 'protagonist'
		ORDER BY appear_count DESC
	`

	rows, err := q.QueryContext(ctx, query, projectID)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to get protagonists: %w", err)
	}
	defer rows.Close()

	var entities []*entity.StoryEntity
	for rows.Next() {
		e, err := r.scanEntityFromRows(rows)
		if err != nil {
			span.RecordError(err)
			return nil, err
		}
		entities = append(entities, e)
	}

	return entities, nil
}

// scanEntity 扫描单行实体数据
func (r *EntityRepository) scanEntity(row *sql.Row) (*entity.StoryEntity, error) {
	var e entity.StoryEntity
	var aliases pq.StringArray
	var firstAppear, lastAppear, vectorID sql.NullString
	var attributesJSON, metadataJSON []byte

	err := row.Scan(
		&e.ID, &e.ProjectID, &e.Name, &aliases, &e.Type, &e.Description, &attributesJSON, &metadataJSON, &e.CurrentState,
		&firstAppear, &lastAppear, &e.AppearCount, &e.Importance, &vectorID, &e.CreatedAt, &e.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan entity: %w", err)
	}

	e.Aliases = aliases
	if firstAppear.Valid {
		e.FirstAppearChapterID = firstAppear.String
	}
	if lastAppear.Valid {
		e.LastAppearChapterID = lastAppear.String
	}
	if vectorID.Valid {
		e.VectorID = vectorID.String
	}
	json.Unmarshal(attributesJSON, &e.Attributes)
	json.Unmarshal(metadataJSON, &e.Metadata)

	return &e, nil
}

// scanEntityFromRows 从多行结果扫描
func (r *EntityRepository) scanEntityFromRows(rows *sql.Rows) (*entity.StoryEntity, error) {
	var e entity.StoryEntity
	var aliases pq.StringArray
	var firstAppear, lastAppear, vectorID sql.NullString
	var attributesJSON, metadataJSON []byte

	err := rows.Scan(
		&e.ID, &e.ProjectID, &e.Name, &aliases, &e.Type, &e.Description, &attributesJSON, &metadataJSON, &e.CurrentState,
		&firstAppear, &lastAppear, &e.AppearCount, &e.Importance, &vectorID, &e.CreatedAt, &e.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to scan entity row: %w", err)
	}

	e.Aliases = aliases
	if firstAppear.Valid {
		e.FirstAppearChapterID = firstAppear.String
	}
	if lastAppear.Valid {
		e.LastAppearChapterID = lastAppear.String
	}
	if vectorID.Valid {
		e.VectorID = vectorID.String
	}
	json.Unmarshal(attributesJSON, &e.Attributes)
	json.Unmarshal(metadataJSON, &e.Metadata)

	return &e, nil
}
