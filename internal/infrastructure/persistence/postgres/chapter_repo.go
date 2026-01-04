// Package postgres 提供 PostgreSQL Repository 实现
package postgres

import (
	"context"
	"fmt"

	"gorm.io/gorm"

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

	db := getDB(ctx, r.client.db)
	if err := db.Create(chapter).Error; err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to create chapter: %w", err)
	}
	return nil
}

// GetByID 根据 ID 获取章节
func (r *ChapterRepository) GetByID(ctx context.Context, id string) (*entity.Chapter, error) {
	ctx, span := tracer.Start(ctx, "postgres.ChapterRepository.GetByID")
	defer span.End()

	db := getDB(ctx, r.client.db)
	var chapter entity.Chapter
	if err := db.First(&chapter, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		span.RecordError(err)
		return nil, fmt.Errorf("failed to get chapter: %w", err)
	}
	return &chapter, nil
}

// Update 更新章节
func (r *ChapterRepository) Update(ctx context.Context, chapter *entity.Chapter) error {
	ctx, span := tracer.Start(ctx, "postgres.ChapterRepository.Update")
	defer span.End()

	db := getDB(ctx, r.client.db)
	if err := db.Save(chapter).Error; err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to update chapter: %w", err)
	}
	return nil
}

// Delete 删除章节
func (r *ChapterRepository) Delete(ctx context.Context, id string) error {
	ctx, span := tracer.Start(ctx, "postgres.ChapterRepository.Delete")
	defer span.End()

	db := getDB(ctx, r.client.db)
	if err := db.Delete(&entity.Chapter{}, "id = ?", id).Error; err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to delete chapter: %w", err)
	}
	return nil
}

// ListByProject 获取项目的章节列表
func (r *ChapterRepository) ListByProject(ctx context.Context, projectID string, filter *repository.ChapterFilter, pagination repository.Pagination) (*repository.PagedResult[*entity.Chapter], error) {
	ctx, span := tracer.Start(ctx, "postgres.ChapterRepository.ListByProject")
	defer span.End()

	db := getDB(ctx, r.client.db)
	query := db.Model(&entity.Chapter{}).Where("project_id = ?", projectID)

	// 应用过滤条件
	if filter != nil {
		if filter.VolumeID != "" {
			query = query.Where("volume_id = ?", filter.VolumeID)
		}
		if filter.Status != "" {
			query = query.Where("status = ?", filter.Status)
		}
	}

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to count chapters: %w", err)
	}

	// 获取列表
	var chapters []*entity.Chapter
	if err := query.Order("seq_num ASC").
		Offset(pagination.Offset()).
		Limit(pagination.Limit()).
		Find(&chapters).Error; err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to list chapters: %w", err)
	}

	return repository.NewPagedResult(chapters, total, pagination), nil
}

// ListByVolume 获取卷的章节列表
func (r *ChapterRepository) ListByVolume(ctx context.Context, volumeID string) ([]*entity.Chapter, error) {
	ctx, span := tracer.Start(ctx, "postgres.ChapterRepository.ListByVolume")
	defer span.End()

	db := getDB(ctx, r.client.db)
	var chapters []*entity.Chapter

	if err := db.Where("volume_id = ?", volumeID).
		Order("seq_num ASC").
		Find(&chapters).Error; err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to list chapters by volume: %w", err)
	}

	return chapters, nil
}

// GetByProjectAndSeq 根据项目、卷和序号获取章节
func (r *ChapterRepository) GetByProjectAndSeq(ctx context.Context, projectID string, volumeID string, seqNum int) (*entity.Chapter, error) {
	ctx, span := tracer.Start(ctx, "postgres.ChapterRepository.GetByProjectAndSeq")
	defer span.End()

	db := getDB(ctx, r.client.db)
	var chapter entity.Chapter

	query := db.Where("project_id = ? AND seq_num = ?", projectID, seqNum)
	if volumeID != "" {
		query = query.Where("volume_id = ?", volumeID)
	}

	if err := query.First(&chapter).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		span.RecordError(err)
		return nil, fmt.Errorf("failed to get chapter by project and seq: %w", err)
	}
	return &chapter, nil
}

// UpdateContent 更新章节内容
func (r *ChapterRepository) UpdateContent(ctx context.Context, id, content, summary string) error {
	ctx, span := tracer.Start(ctx, "postgres.ChapterRepository.UpdateContent")
	defer span.End()

	db := getDB(ctx, r.client.db)
	wordCount := len([]rune(content))
	if err := db.Model(&entity.Chapter{}).Where("id = ?", id).Updates(map[string]interface{}{
		"content_text": content,
		"summary":      summary,
		"word_count":   wordCount,
	}).Error; err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to update chapter content: %w", err)
	}
	return nil
}

// UpdateStatus 更新章节状态
func (r *ChapterRepository) UpdateStatus(ctx context.Context, id string, status entity.ChapterStatus) error {
	ctx, span := tracer.Start(ctx, "postgres.ChapterRepository.UpdateStatus")
	defer span.End()

	db := getDB(ctx, r.client.db)
	if err := db.Model(&entity.Chapter{}).Where("id = ?", id).Update("status", status).Error; err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to update chapter status: %w", err)
	}
	return nil
}

// GetNextSeqNum 获取下一个序号
func (r *ChapterRepository) GetNextSeqNum(ctx context.Context, projectID, volumeID string) (int, error) {
	ctx, span := tracer.Start(ctx, "postgres.ChapterRepository.GetNextSeqNum")
	defer span.End()

	db := getDB(ctx, r.client.db)
	var maxSeq *int

	query := db.Model(&entity.Chapter{}).Where("project_id = ?", projectID)
	if volumeID != "" {
		query = query.Where("volume_id = ?", volumeID)
	}

	err := query.Select("MAX(seq_num)").Scan(&maxSeq).Error

	if err != nil {
		span.RecordError(err)
		return 0, fmt.Errorf("failed to get max seq num: %w", err)
	}

	if maxSeq == nil {
		return 1, nil
	}
	return *maxSeq + 1, nil
}

// GetByStoryTimeRange 根据故事时间范围获取章节
func (r *ChapterRepository) GetByStoryTimeRange(ctx context.Context, projectID string, startTime, endTime int64) ([]*entity.Chapter, error) {
	ctx, span := tracer.Start(ctx, "postgres.ChapterRepository.GetByStoryTimeRange")
	defer span.End()

	db := getDB(ctx, r.client.db)
	var chapters []*entity.Chapter

	query := db.Where("project_id = ?", projectID)
	if startTime > 0 {
		query = query.Where("story_time_end >= ?", startTime)
	}
	if endTime > 0 {
		query = query.Where("story_time_start <= ?", endTime)
	}

	if err := query.Order("seq_num ASC").Find(&chapters).Error; err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to get chapters by time range: %w", err)
	}

	return chapters, nil
}

// GetRecent 获取最近章节
func (r *ChapterRepository) GetRecent(ctx context.Context, projectID string, limit int) ([]*entity.Chapter, error) {
	ctx, span := tracer.Start(ctx, "postgres.ChapterRepository.GetRecent")
	defer span.End()

	db := getDB(ctx, r.client.db)
	var chapters []*entity.Chapter

	if err := db.Where("project_id = ?", projectID).
		Order("seq_num DESC").
		Limit(limit).
		Find(&chapters).Error; err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to get recent chapters: %w", err)
	}

	return chapters, nil
}
