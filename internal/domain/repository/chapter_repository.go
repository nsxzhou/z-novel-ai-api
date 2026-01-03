// Package repository 定义数据访问层接口
package repository

import (
	"context"

	"z-novel-ai-api/internal/domain/entity"
)

// ChapterFilter 章节过滤条件
type ChapterFilter struct {
	VolumeID string
	Status   entity.ChapterStatus
}

// ChapterRepository 章节仓储接口
type ChapterRepository interface {
	// Create 创建章节
	Create(ctx context.Context, chapter *entity.Chapter) error

	// GetByID 根据 ID 获取章节
	GetByID(ctx context.Context, id string) (*entity.Chapter, error)

	// Update 更新章节
	Update(ctx context.Context, chapter *entity.Chapter) error

	// Delete 删除章节
	Delete(ctx context.Context, id string) error

	// ListByProject 获取项目章节列表
	ListByProject(ctx context.Context, projectID string, filter *ChapterFilter, pagination Pagination) (*PagedResult[*entity.Chapter], error)

	// ListByVolume 获取卷章节列表（按序号排序）
	ListByVolume(ctx context.Context, volumeID string) ([]*entity.Chapter, error)

	// GetByProjectAndSeq 根据项目和序号获取章节
	GetByProjectAndSeq(ctx context.Context, projectID string, volumeID string, seqNum int) (*entity.Chapter, error)

	// UpdateContent 更新章节内容
	UpdateContent(ctx context.Context, id, content, summary string) error

	// UpdateStatus 更新章节状态
	UpdateStatus(ctx context.Context, id string, status entity.ChapterStatus) error

	// GetNextSeqNum 获取下一个序号
	GetNextSeqNum(ctx context.Context, projectID, volumeID string) (int, error)

	// GetByStoryTimeRange 根据故事时间范围获取章节
	GetByStoryTimeRange(ctx context.Context, projectID string, startTime, endTime int64) ([]*entity.Chapter, error)

	// GetRecent 获取最近章节
	GetRecent(ctx context.Context, projectID string, limit int) ([]*entity.Chapter, error)
}
