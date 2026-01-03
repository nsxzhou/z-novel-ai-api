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

// ChapterRepository 章节仓储实现
type ChapterRepository struct {
	client *Client
}

// NewChapterRepository 创建章节仓储
func NewChapterRepository(client *Client) *ChapterRepository {
	return &ChapterRepository{client: client}
}

// Create 创建章节
func (r *ChapterRepository) Create(ctx context.Context, chapter *entity.Chapter) error {
	ctx, span := tracer.Start(ctx, "postgres.ChapterRepository.Create")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	metadataJSON, _ := json.Marshal(chapter.GenerationMetadata)

	query := `
		INSERT INTO chapters (id, project_id, volume_id, seq_num, title, outline, content_text, 
			summary, notes, story_time_start, story_time_end, word_count, status, 
			generation_metadata, version, created_at, updated_at)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`

	var volumeID sql.NullString
	if chapter.VolumeID != "" {
		volumeID = sql.NullString{String: chapter.VolumeID, Valid: true}
	}

	err := q.QueryRowContext(ctx, query,
		chapter.ProjectID, volumeID, chapter.SeqNum, chapter.Title, chapter.Outline,
		chapter.ContentText, chapter.Summary, chapter.Notes,
		chapter.StoryTimeStart, chapter.StoryTimeEnd, chapter.WordCount, chapter.Status,
		metadataJSON, chapter.Version,
	).Scan(&chapter.ID, &chapter.CreatedAt, &chapter.UpdatedAt)

	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to create chapter: %w", err)
	}

	return nil
}

// GetByID 根据 ID 获取章节
func (r *ChapterRepository) GetByID(ctx context.Context, id string) (*entity.Chapter, error) {
	ctx, span := tracer.Start(ctx, "postgres.ChapterRepository.GetByID")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	query := `
		SELECT id, project_id, volume_id, seq_num, title, outline, content_text, 
			summary, notes, story_time_start, story_time_end, word_count, status, 
			generation_metadata, version, created_at, updated_at
		FROM chapters
		WHERE id = $1
	`

	return r.scanChapter(ctx, q.QueryRowContext(ctx, query, id))
}

// Update 更新章节
func (r *ChapterRepository) Update(ctx context.Context, chapter *entity.Chapter) error {
	ctx, span := tracer.Start(ctx, "postgres.ChapterRepository.Update")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	metadataJSON, _ := json.Marshal(chapter.GenerationMetadata)

	query := `
		UPDATE chapters
		SET title = $1, outline = $2, content_text = $3, summary = $4, notes = $5,
			story_time_start = $6, story_time_end = $7, word_count = $8, status = $9,
			generation_metadata = $10, version = version + 1, updated_at = NOW()
		WHERE id = $11
		RETURNING version, updated_at
	`

	err := q.QueryRowContext(ctx, query,
		chapter.Title, chapter.Outline, chapter.ContentText, chapter.Summary, chapter.Notes,
		chapter.StoryTimeStart, chapter.StoryTimeEnd, chapter.WordCount, chapter.Status,
		metadataJSON, chapter.ID,
	).Scan(&chapter.Version, &chapter.UpdatedAt)

	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to update chapter: %w", err)
	}

	return nil
}

// Delete 删除章节
func (r *ChapterRepository) Delete(ctx context.Context, id string) error {
	ctx, span := tracer.Start(ctx, "postgres.ChapterRepository.Delete")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	query := `DELETE FROM chapters WHERE id = $1`
	_, err := q.ExecContext(ctx, query, id)

	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to delete chapter: %w", err)
	}

	return nil
}

// ListByProject 获取项目章节列表
func (r *ChapterRepository) ListByProject(ctx context.Context, projectID string, filter *repository.ChapterFilter, pagination repository.Pagination) (*repository.PagedResult[*entity.Chapter], error) {
	ctx, span := tracer.Start(ctx, "postgres.ChapterRepository.ListByProject")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	// 构建查询条件
	whereClause := "project_id = $1"
	args := []interface{}{projectID}
	argIdx := 2

	if filter != nil {
		if filter.VolumeID != "" {
			whereClause += fmt.Sprintf(" AND volume_id = $%d", argIdx)
			args = append(args, filter.VolumeID)
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
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM chapters WHERE %s`, whereClause)
	if err := q.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to count chapters: %w", err)
	}

	// 获取列表
	query := fmt.Sprintf(`
		SELECT id, project_id, volume_id, seq_num, title, outline, content_text, 
			summary, notes, story_time_start, story_time_end, word_count, status, 
			generation_metadata, version, created_at, updated_at
		FROM chapters
		WHERE %s
		ORDER BY seq_num ASC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)

	args = append(args, pagination.Limit(), pagination.Offset())

	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to list chapters: %w", err)
	}
	defer rows.Close()

	var chapters []*entity.Chapter
	for rows.Next() {
		chapter, err := r.scanChapterFromRows(rows)
		if err != nil {
			span.RecordError(err)
			return nil, err
		}
		chapters = append(chapters, chapter)
	}

	return repository.NewPagedResult(chapters, total, pagination), nil
}

// ListByVolume 获取卷章节列表
func (r *ChapterRepository) ListByVolume(ctx context.Context, volumeID string) ([]*entity.Chapter, error) {
	ctx, span := tracer.Start(ctx, "postgres.ChapterRepository.ListByVolume")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	query := `
		SELECT id, project_id, volume_id, seq_num, title, outline, content_text, 
			summary, notes, story_time_start, story_time_end, word_count, status, 
			generation_metadata, version, created_at, updated_at
		FROM chapters
		WHERE volume_id = $1
		ORDER BY seq_num ASC
	`

	rows, err := q.QueryContext(ctx, query, volumeID)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to list chapters by volume: %w", err)
	}
	defer rows.Close()

	var chapters []*entity.Chapter
	for rows.Next() {
		chapter, err := r.scanChapterFromRows(rows)
		if err != nil {
			span.RecordError(err)
			return nil, err
		}
		chapters = append(chapters, chapter)
	}

	return chapters, nil
}

// GetByProjectAndSeq 根据项目和序号获取章节
func (r *ChapterRepository) GetByProjectAndSeq(ctx context.Context, projectID string, volumeID string, seqNum int) (*entity.Chapter, error) {
	ctx, span := tracer.Start(ctx, "postgres.ChapterRepository.GetByProjectAndSeq")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	query := `
		SELECT id, project_id, volume_id, seq_num, title, outline, content_text, 
			summary, notes, story_time_start, story_time_end, word_count, status, 
			generation_metadata, version, created_at, updated_at
		FROM chapters
		WHERE project_id = $1 AND volume_id = $2 AND seq_num = $3
	`

	return r.scanChapter(ctx, q.QueryRowContext(ctx, query, projectID, volumeID, seqNum))
}

// UpdateContent 更新章节内容
func (r *ChapterRepository) UpdateContent(ctx context.Context, id, content, summary string) error {
	ctx, span := tracer.Start(ctx, "postgres.ChapterRepository.UpdateContent")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	wordCount := len([]rune(content))

	query := `
		UPDATE chapters
		SET content_text = $1, summary = $2, word_count = $3, version = version + 1, updated_at = NOW()
		WHERE id = $4
	`
	_, err := q.ExecContext(ctx, query, content, summary, wordCount, id)

	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to update chapter content: %w", err)
	}

	return nil
}

// UpdateStatus 更新章节状态
func (r *ChapterRepository) UpdateStatus(ctx context.Context, id string, status entity.ChapterStatus) error {
	ctx, span := tracer.Start(ctx, "postgres.ChapterRepository.UpdateStatus")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	query := `UPDATE chapters SET status = $1, updated_at = NOW() WHERE id = $2`
	_, err := q.ExecContext(ctx, query, status, id)

	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to update chapter status: %w", err)
	}

	return nil
}

// GetNextSeqNum 获取下一个序号
func (r *ChapterRepository) GetNextSeqNum(ctx context.Context, projectID, volumeID string) (int, error) {
	ctx, span := tracer.Start(ctx, "postgres.ChapterRepository.GetNextSeqNum")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	var maxSeq sql.NullInt64
	query := `SELECT MAX(seq_num) FROM chapters WHERE project_id = $1 AND volume_id = $2`
	err := q.QueryRowContext(ctx, query, projectID, volumeID).Scan(&maxSeq)

	if err != nil {
		span.RecordError(err)
		return 0, fmt.Errorf("failed to get max seq num: %w", err)
	}

	if maxSeq.Valid {
		return int(maxSeq.Int64) + 1, nil
	}
	return 1, nil
}

// GetByStoryTimeRange 根据故事时间范围获取章节
func (r *ChapterRepository) GetByStoryTimeRange(ctx context.Context, projectID string, startTime, endTime int64) ([]*entity.Chapter, error) {
	ctx, span := tracer.Start(ctx, "postgres.ChapterRepository.GetByStoryTimeRange")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	query := `
		SELECT id, project_id, volume_id, seq_num, title, outline, content_text, 
			summary, notes, story_time_start, story_time_end, word_count, status, 
			generation_metadata, version, created_at, updated_at
		FROM chapters
		WHERE project_id = $1 AND story_time_start >= $2 AND story_time_end <= $3
		ORDER BY story_time_start ASC
	`

	rows, err := q.QueryContext(ctx, query, projectID, startTime, endTime)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to get chapters by time range: %w", err)
	}
	defer rows.Close()

	var chapters []*entity.Chapter
	for rows.Next() {
		chapter, err := r.scanChapterFromRows(rows)
		if err != nil {
			span.RecordError(err)
			return nil, err
		}
		chapters = append(chapters, chapter)
	}

	return chapters, nil
}

// GetRecent 获取最近章节
func (r *ChapterRepository) GetRecent(ctx context.Context, projectID string, limit int) ([]*entity.Chapter, error) {
	ctx, span := tracer.Start(ctx, "postgres.ChapterRepository.GetRecent")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	query := `
		SELECT id, project_id, volume_id, seq_num, title, outline, content_text, 
			summary, notes, story_time_start, story_time_end, word_count, status, 
			generation_metadata, version, created_at, updated_at
		FROM chapters
		WHERE project_id = $1
		ORDER BY seq_num DESC
		LIMIT $2
	`

	rows, err := q.QueryContext(ctx, query, projectID, limit)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to get recent chapters: %w", err)
	}
	defer rows.Close()

	var chapters []*entity.Chapter
	for rows.Next() {
		chapter, err := r.scanChapterFromRows(rows)
		if err != nil {
			span.RecordError(err)
			return nil, err
		}
		chapters = append(chapters, chapter)
	}

	return chapters, nil
}

// scanChapter 扫描单行数据
func (r *ChapterRepository) scanChapter(ctx context.Context, row *sql.Row) (*entity.Chapter, error) {
	var chapter entity.Chapter
	var volumeID sql.NullString
	var metadataJSON []byte

	err := row.Scan(
		&chapter.ID, &chapter.ProjectID, &volumeID, &chapter.SeqNum, &chapter.Title,
		&chapter.Outline, &chapter.ContentText, &chapter.Summary, &chapter.Notes,
		&chapter.StoryTimeStart, &chapter.StoryTimeEnd, &chapter.WordCount, &chapter.Status,
		&metadataJSON, &chapter.Version, &chapter.CreatedAt, &chapter.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan chapter: %w", err)
	}

	if volumeID.Valid {
		chapter.VolumeID = volumeID.String
	}
	json.Unmarshal(metadataJSON, &chapter.GenerationMetadata)

	return &chapter, nil
}

// scanChapterFromRows 从多行结果扫描
func (r *ChapterRepository) scanChapterFromRows(rows *sql.Rows) (*entity.Chapter, error) {
	var chapter entity.Chapter
	var volumeID sql.NullString
	var metadataJSON []byte

	err := rows.Scan(
		&chapter.ID, &chapter.ProjectID, &volumeID, &chapter.SeqNum, &chapter.Title,
		&chapter.Outline, &chapter.ContentText, &chapter.Summary, &chapter.Notes,
		&chapter.StoryTimeStart, &chapter.StoryTimeEnd, &chapter.WordCount, &chapter.Status,
		&metadataJSON, &chapter.Version, &chapter.CreatedAt, &chapter.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to scan chapter row: %w", err)
	}

	if volumeID.Valid {
		chapter.VolumeID = volumeID.String
	}
	json.Unmarshal(metadataJSON, &chapter.GenerationMetadata)

	return &chapter, nil
}
