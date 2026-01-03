// Package postgres 提供 PostgreSQL Repository 实现
package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"z-novel-ai-api/internal/domain/entity"
)

// VolumeRepository 卷仓储实现
type VolumeRepository struct {
	client *Client
}

// NewVolumeRepository 创建卷仓储
func NewVolumeRepository(client *Client) *VolumeRepository {
	return &VolumeRepository{client: client}
}

// Create 创建卷
func (r *VolumeRepository) Create(ctx context.Context, volume *entity.Volume) error {
	ctx, span := tracer.Start(ctx, "postgres.VolumeRepository.Create")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	query := `
		INSERT INTO volumes (id, project_id, seq_num, title, description, summary, word_count, status, created_at, updated_at)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`

	err := q.QueryRowContext(ctx, query,
		volume.ProjectID, volume.SeqNum, volume.Title, volume.Description, volume.Summary, volume.WordCount, volume.Status,
	).Scan(&volume.ID, &volume.CreatedAt, &volume.UpdatedAt)

	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to create volume: %w", err)
	}

	return nil
}

// GetByID 根据 ID 获取卷
func (r *VolumeRepository) GetByID(ctx context.Context, id string) (*entity.Volume, error) {
	ctx, span := tracer.Start(ctx, "postgres.VolumeRepository.GetByID")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	query := `
		SELECT id, project_id, seq_num, title, description, summary, word_count, status, created_at, updated_at
		FROM volumes
		WHERE id = $1
	`

	return r.scanVolume(q.QueryRowContext(ctx, query, id))
}

// Update 更新卷
func (r *VolumeRepository) Update(ctx context.Context, volume *entity.Volume) error {
	ctx, span := tracer.Start(ctx, "postgres.VolumeRepository.Update")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	query := `
		UPDATE volumes
		SET title = $1, description = $2, summary = $3, word_count = $4, status = $5, updated_at = NOW()
		WHERE id = $6
		RETURNING updated_at
	`

	err := q.QueryRowContext(ctx, query,
		volume.Title, volume.Description, volume.Summary, volume.WordCount, volume.Status, volume.ID,
	).Scan(&volume.UpdatedAt)

	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to update volume: %w", err)
	}

	return nil
}

// Delete 删除卷
func (r *VolumeRepository) Delete(ctx context.Context, id string) error {
	ctx, span := tracer.Start(ctx, "postgres.VolumeRepository.Delete")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	query := `DELETE FROM volumes WHERE id = $1`
	_, err := q.ExecContext(ctx, query, id)

	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to delete volume: %w", err)
	}

	return nil
}

// ListByProject 获取项目卷列表
func (r *VolumeRepository) ListByProject(ctx context.Context, projectID string) ([]*entity.Volume, error) {
	ctx, span := tracer.Start(ctx, "postgres.VolumeRepository.ListByProject")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	query := `
		SELECT id, project_id, seq_num, title, description, summary, word_count, status, created_at, updated_at
		FROM volumes
		WHERE project_id = $1
		ORDER BY seq_num ASC
	`

	rows, err := q.QueryContext(ctx, query, projectID)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to list volumes: %w", err)
	}
	defer rows.Close()

	var volumes []*entity.Volume
	for rows.Next() {
		volume, err := r.scanVolumeFromRows(rows)
		if err != nil {
			span.RecordError(err)
			return nil, err
		}
		volumes = append(volumes, volume)
	}

	return volumes, nil
}

// GetByProjectAndSeq 根据项目和序号获取卷
func (r *VolumeRepository) GetByProjectAndSeq(ctx context.Context, projectID string, seqNum int) (*entity.Volume, error) {
	ctx, span := tracer.Start(ctx, "postgres.VolumeRepository.GetByProjectAndSeq")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	query := `
		SELECT id, project_id, seq_num, title, description, summary, word_count, status, created_at, updated_at
		FROM volumes
		WHERE project_id = $1 AND seq_num = $2
	`

	return r.scanVolume(q.QueryRowContext(ctx, query, projectID, seqNum))
}

// UpdateWordCount 更新字数统计
func (r *VolumeRepository) UpdateWordCount(ctx context.Context, id string, wordCount int) error {
	ctx, span := tracer.Start(ctx, "postgres.VolumeRepository.UpdateWordCount")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	query := `UPDATE volumes SET word_count = $1, updated_at = NOW() WHERE id = $2`
	_, err := q.ExecContext(ctx, query, wordCount, id)

	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to update word count: %w", err)
	}

	return nil
}

// ReorderVolumes 重新排序卷
func (r *VolumeRepository) ReorderVolumes(ctx context.Context, projectID string, volumeIDs []string) error {
	ctx, span := tracer.Start(ctx, "postgres.VolumeRepository.ReorderVolumes")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	for i, id := range volumeIDs {
		query := `UPDATE volumes SET seq_num = $1, updated_at = NOW() WHERE id = $2 AND project_id = $3`
		_, err := q.ExecContext(ctx, query, i+1, id, projectID)
		if err != nil {
			span.RecordError(err)
			return fmt.Errorf("failed to reorder volume: %w", err)
		}
	}

	return nil
}

// GetNextSeqNum 获取下一个序号
func (r *VolumeRepository) GetNextSeqNum(ctx context.Context, projectID string) (int, error) {
	ctx, span := tracer.Start(ctx, "postgres.VolumeRepository.GetNextSeqNum")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	var maxSeq sql.NullInt64
	query := `SELECT MAX(seq_num) FROM volumes WHERE project_id = $1`
	err := q.QueryRowContext(ctx, query, projectID).Scan(&maxSeq)

	if err != nil {
		span.RecordError(err)
		return 0, fmt.Errorf("failed to get max seq num: %w", err)
	}

	if maxSeq.Valid {
		return int(maxSeq.Int64) + 1, nil
	}
	return 1, nil
}

// scanVolume 扫描单行卷数据
func (r *VolumeRepository) scanVolume(row *sql.Row) (*entity.Volume, error) {
	var volume entity.Volume

	err := row.Scan(
		&volume.ID, &volume.ProjectID, &volume.SeqNum, &volume.Title,
		&volume.Description, &volume.Summary, &volume.WordCount, &volume.Status,
		&volume.CreatedAt, &volume.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan volume: %w", err)
	}

	return &volume, nil
}

// scanVolumeFromRows 从多行结果扫描
func (r *VolumeRepository) scanVolumeFromRows(rows *sql.Rows) (*entity.Volume, error) {
	var volume entity.Volume

	err := rows.Scan(
		&volume.ID, &volume.ProjectID, &volume.SeqNum, &volume.Title,
		&volume.Description, &volume.Summary, &volume.WordCount, &volume.Status,
		&volume.CreatedAt, &volume.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to scan volume row: %w", err)
	}

	return &volume, nil
}
