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

// EntityStateRepository 实体状态历史仓储实现
type EntityStateRepository struct {
	client *Client
}

// NewEntityStateRepository 创建实体状态历史仓储
func NewEntityStateRepository(client *Client) *EntityStateRepository {
	return &EntityStateRepository{client: client}
}

// Create 创建状态记录
func (r *EntityStateRepository) Create(ctx context.Context, state *entity.EntityState) error {
	ctx, span := tracer.Start(ctx, "postgres.EntityStateRepository.Create")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	var chapterID sql.NullString
	if state.ChapterID != "" {
		chapterID = sql.NullString{String: state.ChapterID, Valid: true}
	}

	var storyTime sql.NullInt64
	if state.StoryTime != 0 {
		storyTime = sql.NullInt64{Int64: state.StoryTime, Valid: true}
	}

	var attrChangesJSON []byte
	if len(state.AttributeChanges) > 0 {
		attrChangesJSON, _ = json.Marshal(state.AttributeChanges)
	}

	var eventSummary sql.NullString
	if state.EventSummary != "" {
		eventSummary = sql.NullString{String: state.EventSummary, Valid: true}
	}

	query := `
		INSERT INTO entity_states (id, entity_id, chapter_id, story_time, state_description, attribute_changes, event_summary, created_at)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, NOW())
		RETURNING id, created_at
	`

	if err := q.QueryRowContext(ctx, query,
		state.EntityID,
		chapterID,
		storyTime,
		state.StateDescription,
		attrChangesJSON,
		eventSummary,
	).Scan(&state.ID, &state.CreatedAt); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to create entity state: %w", err)
	}

	return nil
}

// GetByID 根据 ID 获取状态记录
func (r *EntityStateRepository) GetByID(ctx context.Context, id string) (*entity.EntityState, error) {
	ctx, span := tracer.Start(ctx, "postgres.EntityStateRepository.GetByID")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	query := `
		SELECT id, entity_id, chapter_id, story_time, state_description, attribute_changes, event_summary, created_at
		FROM entity_states
		WHERE id = $1
	`

	return r.scanEntityState(q.QueryRowContext(ctx, query, id))
}

// ListByEntity 获取实体状态历史
func (r *EntityStateRepository) ListByEntity(ctx context.Context, entityID string, pagination repository.Pagination) (*repository.PagedResult[*entity.EntityState], error) {
	ctx, span := tracer.Start(ctx, "postgres.EntityStateRepository.ListByEntity")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	var total int64
	if err := q.QueryRowContext(ctx, `SELECT COUNT(*) FROM entity_states WHERE entity_id = $1`, entityID).Scan(&total); err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to count entity states: %w", err)
	}

	query := `
		SELECT id, entity_id, chapter_id, story_time, state_description, attribute_changes, event_summary, created_at
		FROM entity_states
		WHERE entity_id = $1
		ORDER BY story_time DESC NULLS LAST, created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := q.QueryContext(ctx, query, entityID, pagination.Limit(), pagination.Offset())
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to list entity states: %w", err)
	}
	defer rows.Close()

	var states []*entity.EntityState
	for rows.Next() {
		s, err := r.scanEntityStateFromRows(rows)
		if err != nil {
			span.RecordError(err)
			return nil, err
		}
		states = append(states, s)
	}

	return repository.NewPagedResult(states, total, pagination), nil
}

// GetByChapter 获取章节中的状态变更
func (r *EntityStateRepository) GetByChapter(ctx context.Context, chapterID string) ([]*entity.EntityState, error) {
	ctx, span := tracer.Start(ctx, "postgres.EntityStateRepository.GetByChapter")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	query := `
		SELECT id, entity_id, chapter_id, story_time, state_description, attribute_changes, event_summary, created_at
		FROM entity_states
		WHERE chapter_id = $1
		ORDER BY created_at ASC
	`

	rows, err := q.QueryContext(ctx, query, chapterID)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to list entity states by chapter: %w", err)
	}
	defer rows.Close()

	var states []*entity.EntityState
	for rows.Next() {
		s, err := r.scanEntityStateFromRows(rows)
		if err != nil {
			span.RecordError(err)
			return nil, err
		}
		states = append(states, s)
	}

	return states, nil
}

// GetStateAtTime 获取指定时间点的状态
func (r *EntityStateRepository) GetStateAtTime(ctx context.Context, entityID string, storyTime int64) (*entity.EntityState, error) {
	ctx, span := tracer.Start(ctx, "postgres.EntityStateRepository.GetStateAtTime")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	query := `
		SELECT id, entity_id, chapter_id, story_time, state_description, attribute_changes, event_summary, created_at
		FROM entity_states
		WHERE entity_id = $1 AND story_time IS NOT NULL AND story_time <= $2
		ORDER BY story_time DESC, created_at DESC
		LIMIT 1
	`

	return r.scanEntityState(q.QueryRowContext(ctx, query, entityID, storyTime))
}

// GetLatestState 获取最新状态
func (r *EntityStateRepository) GetLatestState(ctx context.Context, entityID string) (*entity.EntityState, error) {
	ctx, span := tracer.Start(ctx, "postgres.EntityStateRepository.GetLatestState")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	query := `
		SELECT id, entity_id, chapter_id, story_time, state_description, attribute_changes, event_summary, created_at
		FROM entity_states
		WHERE entity_id = $1
		ORDER BY story_time DESC NULLS LAST, created_at DESC
		LIMIT 1
	`

	return r.scanEntityState(q.QueryRowContext(ctx, query, entityID))
}

// DeleteByEntity 删除实体所有状态记录
func (r *EntityStateRepository) DeleteByEntity(ctx context.Context, entityID string) error {
	ctx, span := tracer.Start(ctx, "postgres.EntityStateRepository.DeleteByEntity")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	_, err := q.ExecContext(ctx, `DELETE FROM entity_states WHERE entity_id = $1`, entityID)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to delete entity states: %w", err)
	}
	return nil
}

func (r *EntityStateRepository) scanEntityState(row *sql.Row) (*entity.EntityState, error) {
	var s entity.EntityState
	var chapterID sql.NullString
	var storyTime sql.NullInt64
	var attrJSON []byte
	var eventSummary sql.NullString

	err := row.Scan(
		&s.ID,
		&s.EntityID,
		&chapterID,
		&storyTime,
		&s.StateDescription,
		&attrJSON,
		&eventSummary,
		&s.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan entity state: %w", err)
	}

	if chapterID.Valid {
		s.ChapterID = chapterID.String
	}
	if storyTime.Valid {
		s.StoryTime = storyTime.Int64
	}
	if len(attrJSON) > 0 {
		_ = json.Unmarshal(attrJSON, &s.AttributeChanges)
	}
	if eventSummary.Valid {
		s.EventSummary = eventSummary.String
	}

	return &s, nil
}

func (r *EntityStateRepository) scanEntityStateFromRows(rows *sql.Rows) (*entity.EntityState, error) {
	var s entity.EntityState
	var chapterID sql.NullString
	var storyTime sql.NullInt64
	var attrJSON []byte
	var eventSummary sql.NullString

	if err := rows.Scan(
		&s.ID,
		&s.EntityID,
		&chapterID,
		&storyTime,
		&s.StateDescription,
		&attrJSON,
		&eventSummary,
		&s.CreatedAt,
	); err != nil {
		return nil, fmt.Errorf("failed to scan entity state row: %w", err)
	}

	if chapterID.Valid {
		s.ChapterID = chapterID.String
	}
	if storyTime.Valid {
		s.StoryTime = storyTime.Int64
	}
	if len(attrJSON) > 0 {
		_ = json.Unmarshal(attrJSON, &s.AttributeChanges)
	}
	if eventSummary.Valid {
		s.EventSummary = eventSummary.String
	}

	return &s, nil
}

