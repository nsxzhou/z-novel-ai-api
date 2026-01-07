// Package repository 定义数据访问层接口
package repository

import (
	"context"

	"z-novel-ai-api/internal/domain/entity"
)

// VolumeRepository 卷仓储接口
type VolumeRepository interface {
	// Create 创建卷
	Create(ctx context.Context, volume *entity.Volume) error

	// GetByID 根据 ID 获取卷
	GetByID(ctx context.Context, id string) (*entity.Volume, error)

	// Update 更新卷
	Update(ctx context.Context, volume *entity.Volume) error

	// Delete 删除卷
	Delete(ctx context.Context, id string) error

	// ListByProject 获取项目卷列表（按序号排序）
	ListByProject(ctx context.Context, projectID string) ([]*entity.Volume, error)

	// GetByProjectAndSeq 根据项目和序号获取卷
	GetByProjectAndSeq(ctx context.Context, projectID string, seqNum int) (*entity.Volume, error)

	// GetByAIKey 根据 AIKey 获取卷（用于 AI 生成对象的稳定映射）
	GetByAIKey(ctx context.Context, projectID, aiKey string) (*entity.Volume, error)

	// UpdateWordCount 更新字数统计
	UpdateWordCount(ctx context.Context, id string, wordCount int) error

	// ReorderVolumes 重新排序卷
	ReorderVolumes(ctx context.Context, projectID string, volumeIDs []string) error

	// GetNextSeqNum 获取下一个序号
	GetNextSeqNum(ctx context.Context, projectID string) (int, error)
}
