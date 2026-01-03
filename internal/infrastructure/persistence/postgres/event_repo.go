// Package postgres 提供 PostgreSQL Repository 实现
package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/lib/pq"

	"z-novel-ai-api/internal/domain/entity"
	"z-novel-ai-api/internal/domain/repository"
)

// EventRepository 事件仓储实现
type EventRepository struct {
	client *Client
}

// NewEventRepository 创建事件仓储
func NewEventRepository(client *Client) *EventRepository {
	return &EventRepository{client: client}
}

// Create 创建事件
func (r *EventRepository) Create(ctx context.Context, event *entity.Event) error {
	ctx, span := tracer.Start(ctx, "postgres.EventRepository.Create")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	query := `
		INSERT INTO events (id, project_id, chapter_id, story_time_start, story_time_end, event_type, 
			summary, description, involved_entities, location_id, importance, tags, vector_id, created_at)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW())
		RETURNING id, created_at
	`

	var chapterID, locationID, vectorID sql.NullString
	if event.ChapterID != "" {
		chapterID = sql.NullString{String: event.ChapterID, Valid: true}
	}
	if event.LocationID != "" {
		locationID = sql.NullString{String: event.LocationID, Valid: true}
	}
	if event.VectorID != "" {
		vectorID = sql.NullString{String: event.VectorID, Valid: true}
	}

	err := q.QueryRowContext(ctx, query,
		event.ProjectID, chapterID, event.StoryTimeStart, event.StoryTimeEnd, event.EventType,
		event.Summary, event.Description, pq.Array(event.InvolvedEntities), locationID,
		event.Importance, pq.Array(event.Tags), vectorID,
	).Scan(&event.ID, &event.CreatedAt)

	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to create event: %w", err)
	}

	return nil
}

// GetByID 根据 ID 获取事件
func (r *EventRepository) GetByID(ctx context.Context, id string) (*entity.Event, error) {
	ctx, span := tracer.Start(ctx, "postgres.EventRepository.GetByID")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	query := `
		SELECT id, project_id, chapter_id, story_time_start, story_time_end, event_type, 
			summary, description, involved_entities, location_id, importance, tags, vector_id, created_at
		FROM events
		WHERE id = $1
	`

	return r.scanEvent(q.QueryRowContext(ctx, query, id))
}

// Update 更新事件
func (r *EventRepository) Update(ctx context.Context, event *entity.Event) error {
	ctx, span := tracer.Start(ctx, "postgres.EventRepository.Update")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	query := `
		UPDATE events
		SET story_time_start = $1, story_time_end = $2, event_type = $3, summary = $4, 
			description = $5, involved_entities = $6, importance = $7, tags = $8
		WHERE id = $9
	`

	_, err := q.ExecContext(ctx, query,
		event.StoryTimeStart, event.StoryTimeEnd, event.EventType, event.Summary,
		event.Description, pq.Array(event.InvolvedEntities), event.Importance, pq.Array(event.Tags), event.ID,
	)

	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to update event: %w", err)
	}

	return nil
}

// Delete 删除事件
func (r *EventRepository) Delete(ctx context.Context, id string) error {
	ctx, span := tracer.Start(ctx, "postgres.EventRepository.Delete")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	query := `DELETE FROM events WHERE id = $1`
	_, err := q.ExecContext(ctx, query, id)

	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to delete event: %w", err)
	}

	return nil
}

// ListByProject 获取项目事件列表
func (r *EventRepository) ListByProject(ctx context.Context, projectID string, filter *repository.EventFilter, pagination repository.Pagination) (*repository.PagedResult[*entity.Event], error) {
	ctx, span := tracer.Start(ctx, "postgres.EventRepository.ListByProject")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	// 构建查询条件
	whereClause := "project_id = $1"
	args := []interface{}{projectID}
	argIdx := 2

	if filter != nil {
		if filter.EventType != "" {
			whereClause += fmt.Sprintf(" AND event_type = $%d", argIdx)
			args = append(args, filter.EventType)
			argIdx++
		}
		if filter.Importance != "" {
			whereClause += fmt.Sprintf(" AND importance = $%d", argIdx)
			args = append(args, filter.Importance)
			argIdx++
		}
		if filter.ChapterID != "" {
			whereClause += fmt.Sprintf(" AND chapter_id = $%d", argIdx)
			args = append(args, filter.ChapterID)
			argIdx++
		}
		if filter.TimeStart > 0 {
			whereClause += fmt.Sprintf(" AND story_time_start >= $%d", argIdx)
			args = append(args, filter.TimeStart)
			argIdx++
		}
		if filter.TimeEnd > 0 {
			whereClause += fmt.Sprintf(" AND story_time_end <= $%d", argIdx)
			args = append(args, filter.TimeEnd)
			argIdx++
		}
	}

	// 获取总数
	var total int64
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM events WHERE %s`, whereClause)
	if err := q.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to count events: %w", err)
	}

	// 获取列表
	query := fmt.Sprintf(`
		SELECT id, project_id, chapter_id, story_time_start, story_time_end, event_type, 
			summary, description, involved_entities, location_id, importance, tags, vector_id, created_at
		FROM events
		WHERE %s
		ORDER BY story_time_start ASC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)

	args = append(args, pagination.Limit(), pagination.Offset())

	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to list events: %w", err)
	}
	defer rows.Close()

	var events []*entity.Event
	for rows.Next() {
		evt, err := r.scanEventFromRows(rows)
		if err != nil {
			span.RecordError(err)
			return nil, err
		}
		events = append(events, evt)
	}

	return repository.NewPagedResult(events, total, pagination), nil
}

// ListByChapter 获取章节事件列表
func (r *EventRepository) ListByChapter(ctx context.Context, chapterID string) ([]*entity.Event, error) {
	ctx, span := tracer.Start(ctx, "postgres.EventRepository.ListByChapter")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	query := `
		SELECT id, project_id, chapter_id, story_time_start, story_time_end, event_type, 
			summary, description, involved_entities, location_id, importance, tags, vector_id, created_at
		FROM events
		WHERE chapter_id = $1
		ORDER BY story_time_start ASC
	`

	return r.queryEvents(ctx, q, query, chapterID)
}

// GetByTimeRange 根据时间范围获取事件
func (r *EventRepository) GetByTimeRange(ctx context.Context, projectID string, startTime, endTime int64) ([]*entity.Event, error) {
	ctx, span := tracer.Start(ctx, "postgres.EventRepository.GetByTimeRange")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	query := `
		SELECT id, project_id, chapter_id, story_time_start, story_time_end, event_type, 
			summary, description, involved_entities, location_id, importance, tags, vector_id, created_at
		FROM events
		WHERE project_id = $1 AND story_time_start >= $2 AND story_time_end <= $3
		ORDER BY story_time_start ASC
	`

	return r.queryEvents(ctx, q, query, projectID, startTime, endTime)
}

// GetByEntity 获取涉及实体的事件
func (r *EventRepository) GetByEntity(ctx context.Context, entityID string, pagination repository.Pagination) (*repository.PagedResult[*entity.Event], error) {
	ctx, span := tracer.Start(ctx, "postgres.EventRepository.GetByEntity")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	// 获取总数
	var total int64
	countQuery := `SELECT COUNT(*) FROM events WHERE $1 = ANY(involved_entities)`
	if err := q.QueryRowContext(ctx, countQuery, entityID).Scan(&total); err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to count events: %w", err)
	}

	// 获取列表
	query := `
		SELECT id, project_id, chapter_id, story_time_start, story_time_end, event_type, 
			summary, description, involved_entities, location_id, importance, tags, vector_id, created_at
		FROM events
		WHERE $1 = ANY(involved_entities)
		ORDER BY story_time_start DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := q.QueryContext(ctx, query, entityID, pagination.Limit(), pagination.Offset())
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to get events by entity: %w", err)
	}
	defer rows.Close()

	var events []*entity.Event
	for rows.Next() {
		evt, err := r.scanEventFromRows(rows)
		if err != nil {
			span.RecordError(err)
			return nil, err
		}
		events = append(events, evt)
	}

	return repository.NewPagedResult(events, total, pagination), nil
}

// UpdateVectorID 更新向量 ID
func (r *EventRepository) UpdateVectorID(ctx context.Context, id, vectorID string) error {
	ctx, span := tracer.Start(ctx, "postgres.EventRepository.UpdateVectorID")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	query := `UPDATE events SET vector_id = $1 WHERE id = $2`
	_, err := q.ExecContext(ctx, query, vectorID, id)

	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to update vector id: %w", err)
	}

	return nil
}

// GetTimeline 获取时间轴
func (r *EventRepository) GetTimeline(ctx context.Context, projectID string, limit int) ([]*entity.Event, error) {
	ctx, span := tracer.Start(ctx, "postgres.EventRepository.GetTimeline")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	query := `
		SELECT id, project_id, chapter_id, story_time_start, story_time_end, event_type, 
			summary, description, involved_entities, location_id, importance, tags, vector_id, created_at
		FROM events
		WHERE project_id = $1
		ORDER BY story_time_start ASC
		LIMIT $2
	`

	return r.queryEvents(ctx, q, query, projectID, limit)
}

// SearchByTags 根据标签搜索事件
func (r *EventRepository) SearchByTags(ctx context.Context, projectID string, tags []string, limit int) ([]*entity.Event, error) {
	ctx, span := tracer.Start(ctx, "postgres.EventRepository.SearchByTags")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	query := `
		SELECT id, project_id, chapter_id, story_time_start, story_time_end, event_type, 
			summary, description, involved_entities, location_id, importance, tags, vector_id, created_at
		FROM events
		WHERE project_id = $1 AND tags && $2
		ORDER BY importance DESC, story_time_start DESC
		LIMIT $3
	`

	return r.queryEvents(ctx, q, query, projectID, pq.Array(tags), limit)
}

// queryEvents 通用查询事件
func (r *EventRepository) queryEvents(ctx context.Context, q Querier, query string, args ...interface{}) ([]*entity.Event, error) {
	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query events: %w", err)
	}
	defer rows.Close()

	var events []*entity.Event
	for rows.Next() {
		evt, err := r.scanEventFromRows(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, evt)
	}

	return events, nil
}

// scanEvent 扫描单行事件数据
func (r *EventRepository) scanEvent(row *sql.Row) (*entity.Event, error) {
	var evt entity.Event
	var chapterID, locationID, vectorID sql.NullString
	var involvedEntities, tags pq.StringArray

	err := row.Scan(
		&evt.ID, &evt.ProjectID, &chapterID, &evt.StoryTimeStart, &evt.StoryTimeEnd, &evt.EventType,
		&evt.Summary, &evt.Description, &involvedEntities, &locationID, &evt.Importance, &tags, &vectorID, &evt.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan event: %w", err)
	}

	if chapterID.Valid {
		evt.ChapterID = chapterID.String
	}
	if locationID.Valid {
		evt.LocationID = locationID.String
	}
	if vectorID.Valid {
		evt.VectorID = vectorID.String
	}
	evt.InvolvedEntities = involvedEntities
	evt.Tags = tags

	return &evt, nil
}

// scanEventFromRows 从多行结果扫描
func (r *EventRepository) scanEventFromRows(rows *sql.Rows) (*entity.Event, error) {
	var evt entity.Event
	var chapterID, locationID, vectorID sql.NullString
	var involvedEntities, tags pq.StringArray

	err := rows.Scan(
		&evt.ID, &evt.ProjectID, &chapterID, &evt.StoryTimeStart, &evt.StoryTimeEnd, &evt.EventType,
		&evt.Summary, &evt.Description, &involvedEntities, &locationID, &evt.Importance, &tags, &vectorID, &evt.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to scan event row: %w", err)
	}

	if chapterID.Valid {
		evt.ChapterID = chapterID.String
	}
	if locationID.Valid {
		evt.LocationID = locationID.String
	}
	if vectorID.Valid {
		evt.VectorID = vectorID.String
	}
	evt.InvolvedEntities = involvedEntities
	evt.Tags = tags

	return &evt, nil
}
