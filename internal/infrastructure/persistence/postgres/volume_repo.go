// Package postgres 提供 PostgreSQL Repository 实现
package postgres

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"

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

	db := getDB(ctx, r.client.db)
	if err := db.Create(volume).Error; err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to create volume: %w", err)
	}
	return nil
}

// GetByID 根据 ID 获取卷
func (r *VolumeRepository) GetByID(ctx context.Context, id string) (*entity.Volume, error) {
	ctx, span := tracer.Start(ctx, "postgres.VolumeRepository.GetByID")
	defer span.End()

	db := getDB(ctx, r.client.db)
	var volume entity.Volume
	if err := db.First(&volume, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		span.RecordError(err)
		return nil, fmt.Errorf("failed to get volume: %w", err)
	}
	return &volume, nil
}

// Update 更新卷
func (r *VolumeRepository) Update(ctx context.Context, volume *entity.Volume) error {
	ctx, span := tracer.Start(ctx, "postgres.VolumeRepository.Update")
	defer span.End()

	db := getDB(ctx, r.client.db)
	if err := db.Save(volume).Error; err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to update volume: %w", err)
	}
	return nil
}

// Delete 删除卷
func (r *VolumeRepository) Delete(ctx context.Context, id string) error {
	ctx, span := tracer.Start(ctx, "postgres.VolumeRepository.Delete")
	defer span.End()

	db := getDB(ctx, r.client.db)
	if err := db.Delete(&entity.Volume{}, "id = ?", id).Error; err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to delete volume: %w", err)
	}
	return nil
}

// ListByProject 获取项目的卷列表
func (r *VolumeRepository) ListByProject(ctx context.Context, projectID string) ([]*entity.Volume, error) {
	ctx, span := tracer.Start(ctx, "postgres.VolumeRepository.ListByProject")
	defer span.End()

	db := getDB(ctx, r.client.db)
	var volumes []*entity.Volume

	if err := db.Where("project_id = ?", projectID).
		Order("seq_num ASC").
		Find(&volumes).Error; err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to list volumes: %w", err)
	}

	return volumes, nil
}

// GetByProjectAndSeq 根据项目 ID 和序号获取卷
func (r *VolumeRepository) GetByProjectAndSeq(ctx context.Context, projectID string, seqNum int) (*entity.Volume, error) {
	ctx, span := tracer.Start(ctx, "postgres.VolumeRepository.GetByProjectAndSeq")
	defer span.End()

	db := getDB(ctx, r.client.db)
	var volume entity.Volume
	if err := db.First(&volume, "project_id = ? AND seq_num = ?", projectID, seqNum).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		span.RecordError(err)
		return nil, fmt.Errorf("failed to get volume by project and seq: %w", err)
	}
	return &volume, nil
}

// GetByAIKey 根据 AIKey 获取卷（用于 AI 生成对象的稳定映射）
func (r *VolumeRepository) GetByAIKey(ctx context.Context, projectID, aiKey string) (*entity.Volume, error) {
	ctx, span := tracer.Start(ctx, "postgres.VolumeRepository.GetByAIKey")
	defer span.End()

	db := getDB(ctx, r.client.db)
	var volume entity.Volume
	if err := db.First(&volume, "project_id = ? AND ai_key = ?", projectID, aiKey).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		span.RecordError(err)
		return nil, fmt.Errorf("failed to get volume by ai_key: %w", err)
	}
	return &volume, nil
}

// UpdateWordCount 更新卷字数
func (r *VolumeRepository) UpdateWordCount(ctx context.Context, id string, wordCount int) error {
	ctx, span := tracer.Start(ctx, "postgres.VolumeRepository.UpdateWordCount")
	defer span.End()

	db := getDB(ctx, r.client.db)
	if err := db.Model(&entity.Volume{}).Where("id = ?", id).Update("word_count", wordCount).Error; err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to update volume word count: %w", err)
	}
	return nil
}

// ReorderVolumes 重新排序卷
func (r *VolumeRepository) ReorderVolumes(ctx context.Context, projectID string, volumeIDs []string) error {
	ctx, span := tracer.Start(ctx, "postgres.VolumeRepository.ReorderVolumes")
	defer span.End()

	db := getDB(ctx, r.client.db)

	var existing []*entity.Volume
	if err := db.Model(&entity.Volume{}).
		Select("id").
		Where("project_id = ?", projectID).
		Order("seq_num ASC").
		Find(&existing).Error; err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to load volumes for reorder: %w", err)
	}

	seen := make(map[string]struct{}, len(existing))
	order := make([]string, 0, len(existing))
	for i := range volumeIDs {
		id := strings.TrimSpace(volumeIDs[i])
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		order = append(order, id)
	}
	for i := range existing {
		id := existing[i].ID
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		order = append(order, id)
	}

	if len(order) == 0 {
		return nil
	}

	const offset = 1000000
	if err := db.Model(&entity.Volume{}).
		Where("project_id = ?", projectID).
		Update("seq_num", gorm.Expr("seq_num + ?", offset)).Error; err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to bump seq_num for reorder: %w", err)
	}

	for i := range order {
		id := order[i]
		if err := db.Model(&entity.Volume{}).
			Where("id = ? AND project_id = ?", id, projectID).
			Update("seq_num", i+1).Error; err != nil {
			span.RecordError(err)
			return fmt.Errorf("failed to reorder volumes: %w", err)
		}
	}

	return nil
}

// GetNextSeqNum 获取下一个序号
func (r *VolumeRepository) GetNextSeqNum(ctx context.Context, projectID string) (int, error) {
	ctx, span := tracer.Start(ctx, "postgres.VolumeRepository.GetNextSeqNum")
	defer span.End()

	db := getDB(ctx, r.client.db)
	var maxSeq *int
	err := db.Model(&entity.Volume{}).
		Where("project_id = ?", projectID).
		Select("MAX(seq_num)").
		Scan(&maxSeq).Error

	if err != nil {
		span.RecordError(err)
		return 0, fmt.Errorf("failed to get max seq num: %w", err)
	}

	if maxSeq == nil {
		return 1, nil
	}
	return *maxSeq + 1, nil
}
