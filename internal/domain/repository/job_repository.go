// Package repository 定义数据访问层接口
package repository

import (
	"context"
	"time"

	"z-novel-ai-api/internal/domain/entity"
)

// JobFilter 任务过滤条件
type JobFilter struct {
	JobType   entity.JobType
	Status    entity.JobStatus
	ChapterID *string
}

// JobRepository 生成任务仓储接口
type JobRepository interface {
	// Create 创建任务
	Create(ctx context.Context, job *entity.GenerationJob) error

	// GetByID 根据 ID 获取任务
	GetByID(ctx context.Context, id string) (*entity.GenerationJob, error)

	// Update 更新任务
	Update(ctx context.Context, job *entity.GenerationJob) error

	// Delete 删除任务
	Delete(ctx context.Context, id string) error

	// ListByProject 获取项目任务列表
	ListByProject(ctx context.Context, projectID string, filter *JobFilter, pagination Pagination) (*PagedResult[*entity.GenerationJob], error)

	// GetByIdempotencyKey 根据幂等键获取任务
	GetByIdempotencyKey(ctx context.Context, key string) (*entity.GenerationJob, error)

	// UpdateStatus 更新任务状态
	UpdateStatus(ctx context.Context, id string, status entity.JobStatus) error

	// UpdateProgress 更新任务进度（0-100）
	UpdateProgress(ctx context.Context, id string, progress int) error

	// GetPendingJobs 获取待处理任务
	GetPendingJobs(ctx context.Context, limit int) ([]*entity.GenerationJob, error)

	// GetRunningJobs 获取运行中任务
	GetRunningJobs(ctx context.Context) ([]*entity.GenerationJob, error)

	// GetFailedJobs 获取失败任务（可重试）
	GetFailedJobs(ctx context.Context, maxRetries int, limit int) ([]*entity.GenerationJob, error)

	// GetJobStats 获取任务统计信息
	GetJobStats(ctx context.Context, projectID string) (*JobStats, error)

	// GetTokenUsage 获取租户在指定时间范围内的 Token 使用量（prompt + completion）
	GetTokenUsage(ctx context.Context, tenantID string, startInclusive, endExclusive time.Time) (int64, error)
}

// JobStats 任务统计信息
type JobStats struct {
	TotalJobs       int64 `json:"total_jobs"`
	PendingJobs     int64 `json:"pending_jobs"`
	RunningJobs     int64 `json:"running_jobs"`
	CompletedJobs   int64 `json:"completed_jobs"`
	FailedJobs      int64 `json:"failed_jobs"`
	TotalTokensUsed int64 `json:"total_tokens_used"`
}
