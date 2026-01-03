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

// RelationRepository 关系仓储实现
type RelationRepository struct {
	client *Client
}

// NewRelationRepository 创建关系仓储
func NewRelationRepository(client *Client) *RelationRepository {
	return &RelationRepository{client: client}
}

// Create 创建关系
func (r *RelationRepository) Create(ctx context.Context, relation *entity.Relation) error {
	ctx, span := tracer.Start(ctx, "postgres.RelationRepository.Create")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	attributesJSON, _ := json.Marshal(relation.Attributes)

	query := `
		INSERT INTO relations (id, project_id, source_entity_id, target_entity_id, relation_type, 
			strength, description, attributes, first_chapter_id, last_chapter_id, created_at, updated_at)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`

	var firstChapter, lastChapter sql.NullString
	if relation.FirstChapterID != "" {
		firstChapter = sql.NullString{String: relation.FirstChapterID, Valid: true}
	}
	if relation.LastChapterID != "" {
		lastChapter = sql.NullString{String: relation.LastChapterID, Valid: true}
	}

	err := q.QueryRowContext(ctx, query,
		relation.ProjectID, relation.SourceEntityID, relation.TargetEntityID, relation.RelationType,
		relation.Strength, relation.Description, attributesJSON, firstChapter, lastChapter,
	).Scan(&relation.ID, &relation.CreatedAt, &relation.UpdatedAt)

	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to create relation: %w", err)
	}

	return nil
}

// GetByID 根据 ID 获取关系
func (r *RelationRepository) GetByID(ctx context.Context, id string) (*entity.Relation, error) {
	ctx, span := tracer.Start(ctx, "postgres.RelationRepository.GetByID")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	query := `
		SELECT id, project_id, source_entity_id, target_entity_id, relation_type, 
			strength, description, attributes, first_chapter_id, last_chapter_id, created_at, updated_at
		FROM relations
		WHERE id = $1
	`

	return r.scanRelation(q.QueryRowContext(ctx, query, id))
}

// Update 更新关系
func (r *RelationRepository) Update(ctx context.Context, relation *entity.Relation) error {
	ctx, span := tracer.Start(ctx, "postgres.RelationRepository.Update")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	attributesJSON, _ := json.Marshal(relation.Attributes)

	query := `
		UPDATE relations
		SET relation_type = $1, strength = $2, description = $3, attributes = $4, updated_at = NOW()
		WHERE id = $5
		RETURNING updated_at
	`

	err := q.QueryRowContext(ctx, query,
		relation.RelationType, relation.Strength, relation.Description, attributesJSON, relation.ID,
	).Scan(&relation.UpdatedAt)

	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to update relation: %w", err)
	}

	return nil
}

// Delete 删除关系
func (r *RelationRepository) Delete(ctx context.Context, id string) error {
	ctx, span := tracer.Start(ctx, "postgres.RelationRepository.Delete")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	query := `DELETE FROM relations WHERE id = $1`
	_, err := q.ExecContext(ctx, query, id)

	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to delete relation: %w", err)
	}

	return nil
}

// ListByProject 获取项目关系列表
func (r *RelationRepository) ListByProject(ctx context.Context, projectID string, filter *repository.RelationFilter, pagination repository.Pagination) (*repository.PagedResult[*entity.Relation], error) {
	ctx, span := tracer.Start(ctx, "postgres.RelationRepository.ListByProject")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	// 构建查询条件
	whereClause := "project_id = $1"
	args := []interface{}{projectID}
	argIdx := 2

	if filter != nil {
		if filter.RelationType != "" {
			whereClause += fmt.Sprintf(" AND relation_type = $%d", argIdx)
			args = append(args, filter.RelationType)
			argIdx++
		}
		if filter.MinStrength > 0 {
			whereClause += fmt.Sprintf(" AND strength >= $%d", argIdx)
			args = append(args, filter.MinStrength)
			argIdx++
		}
	}

	// 获取总数
	var total int64
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM relations WHERE %s`, whereClause)
	if err := q.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to count relations: %w", err)
	}

	// 获取列表
	query := fmt.Sprintf(`
		SELECT id, project_id, source_entity_id, target_entity_id, relation_type, 
			strength, description, attributes, first_chapter_id, last_chapter_id, created_at, updated_at
		FROM relations
		WHERE %s
		ORDER BY strength DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)

	args = append(args, pagination.Limit(), pagination.Offset())

	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to list relations: %w", err)
	}
	defer rows.Close()

	var relations []*entity.Relation
	for rows.Next() {
		rel, err := r.scanRelationFromRows(rows)
		if err != nil {
			span.RecordError(err)
			return nil, err
		}
		relations = append(relations, rel)
	}

	return repository.NewPagedResult(relations, total, pagination), nil
}

// GetByEntities 根据实体对获取关系
func (r *RelationRepository) GetByEntities(ctx context.Context, projectID, sourceID, targetID string) (*entity.Relation, error) {
	ctx, span := tracer.Start(ctx, "postgres.RelationRepository.GetByEntities")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	query := `
		SELECT id, project_id, source_entity_id, target_entity_id, relation_type, 
			strength, description, attributes, first_chapter_id, last_chapter_id, created_at, updated_at
		FROM relations
		WHERE project_id = $1 AND source_entity_id = $2 AND target_entity_id = $3
	`

	return r.scanRelation(q.QueryRowContext(ctx, query, projectID, sourceID, targetID))
}

// ListBySourceEntity 获取源实体的关系列表
func (r *RelationRepository) ListBySourceEntity(ctx context.Context, entityID string) ([]*entity.Relation, error) {
	ctx, span := tracer.Start(ctx, "postgres.RelationRepository.ListBySourceEntity")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	query := `
		SELECT id, project_id, source_entity_id, target_entity_id, relation_type, 
			strength, description, attributes, first_chapter_id, last_chapter_id, created_at, updated_at
		FROM relations
		WHERE source_entity_id = $1
		ORDER BY strength DESC
	`

	return r.queryRelations(ctx, q, query, entityID)
}

// ListByTargetEntity 获取目标实体的关系列表
func (r *RelationRepository) ListByTargetEntity(ctx context.Context, entityID string) ([]*entity.Relation, error) {
	ctx, span := tracer.Start(ctx, "postgres.RelationRepository.ListByTargetEntity")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	query := `
		SELECT id, project_id, source_entity_id, target_entity_id, relation_type, 
			strength, description, attributes, first_chapter_id, last_chapter_id, created_at, updated_at
		FROM relations
		WHERE target_entity_id = $1
		ORDER BY strength DESC
	`

	return r.queryRelations(ctx, q, query, entityID)
}

// ListByEntity 获取实体的所有关系（包括源和目标）
func (r *RelationRepository) ListByEntity(ctx context.Context, entityID string) ([]*entity.Relation, error) {
	ctx, span := tracer.Start(ctx, "postgres.RelationRepository.ListByEntity")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	query := `
		SELECT id, project_id, source_entity_id, target_entity_id, relation_type, 
			strength, description, attributes, first_chapter_id, last_chapter_id, created_at, updated_at
		FROM relations
		WHERE source_entity_id = $1 OR target_entity_id = $1
		ORDER BY strength DESC
	`

	return r.queryRelations(ctx, q, query, entityID)
}

// UpdateStrength 更新关系强度
func (r *RelationRepository) UpdateStrength(ctx context.Context, id string, strength float64) error {
	ctx, span := tracer.Start(ctx, "postgres.RelationRepository.UpdateStrength")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	query := `UPDATE relations SET strength = $1, updated_at = NOW() WHERE id = $2`
	_, err := q.ExecContext(ctx, query, strength, id)

	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to update relation strength: %w", err)
	}

	return nil
}

// DeleteByEntity 删除实体相关的所有关系
func (r *RelationRepository) DeleteByEntity(ctx context.Context, entityID string) error {
	ctx, span := tracer.Start(ctx, "postgres.RelationRepository.DeleteByEntity")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	query := `DELETE FROM relations WHERE source_entity_id = $1 OR target_entity_id = $1`
	_, err := q.ExecContext(ctx, query, entityID)

	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to delete relations by entity: %w", err)
	}

	return nil
}

// GetRelationGraph 获取关系图谱
func (r *RelationRepository) GetRelationGraph(ctx context.Context, projectID string, entityIDs []string) ([]*entity.Relation, error) {
	ctx, span := tracer.Start(ctx, "postgres.RelationRepository.GetRelationGraph")
	defer span.End()

	if len(entityIDs) == 0 {
		return nil, nil
	}

	q := getQuerier(ctx, r.client.db)

	// 使用 ANY 来查询所有相关实体的关系
	query := `
		SELECT id, project_id, source_entity_id, target_entity_id, relation_type, 
			strength, description, attributes, first_chapter_id, last_chapter_id, created_at, updated_at
		FROM relations
		WHERE project_id = $1 AND (source_entity_id = ANY($2) OR target_entity_id = ANY($2))
		ORDER BY strength DESC
	`

	rows, err := q.QueryContext(ctx, query, projectID, entityIDs)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to get relation graph: %w", err)
	}
	defer rows.Close()

	var relations []*entity.Relation
	for rows.Next() {
		rel, err := r.scanRelationFromRows(rows)
		if err != nil {
			span.RecordError(err)
			return nil, err
		}
		relations = append(relations, rel)
	}

	return relations, nil
}

// queryRelations 通用查询关系
func (r *RelationRepository) queryRelations(ctx context.Context, q Querier, query string, args ...interface{}) ([]*entity.Relation, error) {
	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query relations: %w", err)
	}
	defer rows.Close()

	var relations []*entity.Relation
	for rows.Next() {
		rel, err := r.scanRelationFromRows(rows)
		if err != nil {
			return nil, err
		}
		relations = append(relations, rel)
	}

	return relations, nil
}

// scanRelation 扫描单行关系数据
func (r *RelationRepository) scanRelation(row *sql.Row) (*entity.Relation, error) {
	var rel entity.Relation
	var firstChapter, lastChapter sql.NullString
	var attributesJSON []byte

	err := row.Scan(
		&rel.ID, &rel.ProjectID, &rel.SourceEntityID, &rel.TargetEntityID, &rel.RelationType,
		&rel.Strength, &rel.Description, &attributesJSON, &firstChapter, &lastChapter,
		&rel.CreatedAt, &rel.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan relation: %w", err)
	}

	if firstChapter.Valid {
		rel.FirstChapterID = firstChapter.String
	}
	if lastChapter.Valid {
		rel.LastChapterID = lastChapter.String
	}
	json.Unmarshal(attributesJSON, &rel.Attributes)

	return &rel, nil
}

// scanRelationFromRows 从多行结果扫描
func (r *RelationRepository) scanRelationFromRows(rows *sql.Rows) (*entity.Relation, error) {
	var rel entity.Relation
	var firstChapter, lastChapter sql.NullString
	var attributesJSON []byte

	err := rows.Scan(
		&rel.ID, &rel.ProjectID, &rel.SourceEntityID, &rel.TargetEntityID, &rel.RelationType,
		&rel.Strength, &rel.Description, &attributesJSON, &firstChapter, &lastChapter,
		&rel.CreatedAt, &rel.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to scan relation row: %w", err)
	}

	if firstChapter.Valid {
		rel.FirstChapterID = firstChapter.String
	}
	if lastChapter.Valid {
		rel.LastChapterID = lastChapter.String
	}
	json.Unmarshal(attributesJSON, &rel.Attributes)

	return &rel, nil
}
