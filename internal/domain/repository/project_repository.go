// Package repository 定义数据访问层接口
package repository

import (
	"context"

	"z-novel-ai-api/internal/domain/entity"
)

// ProjectFilter 项目过滤条件
type ProjectFilter struct {
	OwnerID string
	Genre   string
	Status  entity.ProjectStatus
}

// ProjectRepository 项目仓储接口
type ProjectRepository interface {
	// Create 创建项目
	Create(ctx context.Context, project *entity.Project) error

	// GetByID 根据 ID 获取项目
	GetByID(ctx context.Context, id string) (*entity.Project, error)

	// Update 更新项目
	Update(ctx context.Context, project *entity.Project) error

	// Delete 删除项目
	Delete(ctx context.Context, id string) error

	// List 获取项目列表
	List(ctx context.Context, filter *ProjectFilter, pagination Pagination) (*PagedResult[*entity.Project], error)

	// ListByOwner 获取用户项目列表
	ListByOwner(ctx context.Context, ownerID string, pagination Pagination) (*PagedResult[*entity.Project], error)

	// UpdateStatus 更新项目状态
	UpdateStatus(ctx context.Context, id string, status entity.ProjectStatus) error

	// UpdateWordCount 更新字数统计
	UpdateWordCount(ctx context.Context, id string, wordCount int) error

	// GetStats 获取项目统计信息
	GetStats(ctx context.Context, id string) (*ProjectStats, error)
}

// ProjectStats 项目统计信息
type ProjectStats struct {
	TotalChapters  int   `json:"total_chapters"`
	TotalVolumes   int   `json:"total_volumes"`
	TotalEntities  int   `json:"total_entities"`
	TotalWordCount int64 `json:"total_word_count"`
}
