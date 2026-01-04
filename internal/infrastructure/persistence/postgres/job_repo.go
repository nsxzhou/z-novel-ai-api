// Package postgres 提供 PostgreSQL Repository 实现
package postgres

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"z-novel-ai-api/internal/domain/entity"
	"z-novel-ai-api/internal/domain/repository"
)

// JobRepository 任务仓储实现
type JobRepository struct {
	client *Client
}

// NewJobRepository 创建任务仓储
func NewJobRepository(client *Client) *JobRepository {
	return &JobRepository{client: client}
}

// Create 创建任务
func (r *JobRepository) Create(ctx context.Context, job *entity.GenerationJob) error {
	ctx, span := tracer.Start(ctx, "postgres.JobRepository.Create")
	defer span.End()

	db := getDB(ctx, r.client.db)
	if err := db.Create(job).Error; err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to create job: %w", err)
	}
	return nil
}

// GetByID 根据 ID 获取任务
func (r *JobRepository) GetByID(ctx context.Context, id string) (*entity.GenerationJob, error) {
	ctx, span := tracer.Start(ctx, "postgres.JobRepository.GetByID")
	defer span.End()

	db := getDB(ctx, r.client.db)
	var job entity.GenerationJob
	if err := db.First(&job, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		span.RecordError(err)
		return nil, fmt.Errorf("failed to get job: %w", err)
	}
	return &job, nil
}

// Update 更新任务
func (r *JobRepository) Update(ctx context.Context, job *entity.GenerationJob) error {
	ctx, span := tracer.Start(ctx, "postgres.JobRepository.Update")
	defer span.End()

	db := getDB(ctx, r.client.db)
	if err := db.Save(job).Error; err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to update job: %w", err)
	}
	return nil
}

// Delete 删除任务
func (r *JobRepository) Delete(ctx context.Context, id string) error {
	ctx, span := tracer.Start(ctx, "postgres.JobRepository.Delete")
	defer span.End()

	db := getDB(ctx, r.client.db)
	if err := db.Delete(&entity.GenerationJob{}, "id = ?", id).Error; err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to delete job: %w", err)
	}
	return nil
}

// ListByProject 获取项目任务列表
func (r *JobRepository) ListByProject(ctx context.Context, projectID string, filter *repository.JobFilter, pagination repository.Pagination) (*repository.PagedResult[*entity.GenerationJob], error) {
	ctx, span := tracer.Start(ctx, "postgres.JobRepository.ListByProject")
	defer span.End()

	db := getDB(ctx, r.client.db)
	query := db.Model(&entity.GenerationJob{}).Where("project_id = ?", projectID)

	// 应用过滤条件
	if filter != nil {
		if filter.ChapterID != "" {
			query = query.Where("chapter_id = ?", filter.ChapterID)
		}
		if filter.JobType != "" {
			query = query.Where("job_type = ?", filter.JobType)
		}
		if filter.Status != "" {
			query = query.Where("status = ?", filter.Status)
		}
	}

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to count jobs: %w", err)
	}

	// 获取列表
	var jobs []*entity.GenerationJob
	if err := query.Order("created_at DESC").
		Offset(pagination.Offset()).
		Limit(pagination.Limit()).
		Find(&jobs).Error; err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to list jobs: %w", err)
	}

	return repository.NewPagedResult(jobs, total, pagination), nil
}

// GetByIdempotencyKey 根据幂等键获取任务
func (r *JobRepository) GetByIdempotencyKey(ctx context.Context, key string) (*entity.GenerationJob, error) {
	ctx, span := tracer.Start(ctx, "postgres.JobRepository.GetByIdempotencyKey")
	defer span.End()

	db := getDB(ctx, r.client.db)
	var job entity.GenerationJob
	if err := db.First(&job, "idempotency_key = ?", key).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		span.RecordError(err)
		return nil, fmt.Errorf("failed to get job by idempotency key: %w", err)
	}
	return &job, nil
}

// UpdateStatus 更新任务状态
func (r *JobRepository) UpdateStatus(ctx context.Context, id string, status entity.JobStatus) error {
	ctx, span := tracer.Start(ctx, "postgres.JobRepository.UpdateStatus")
	defer span.End()

	db := getDB(ctx, r.client.db)
	if err := db.Model(&entity.GenerationJob{}).Where("id = ?", id).Update("status", status).Error; err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to update job status: %w", err)
	}
	return nil
}

// MarkRunning 标记任务为运行中
func (r *JobRepository) MarkRunning(ctx context.Context, id string) error {
	ctx, span := tracer.Start(ctx, "postgres.JobRepository.MarkRunning")
	defer span.End()

	db := getDB(ctx, r.client.db)
	now := time.Now()
	if err := db.Model(&entity.GenerationJob{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":     entity.JobStatusRunning,
		"started_at": now,
	}).Error; err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to mark job running: %w", err)
	}
	return nil
}

// UpdateProgress 更新任务进度
func (r *JobRepository) UpdateProgress(ctx context.Context, id string, progress int) error {
	ctx, span := tracer.Start(ctx, "postgres.JobRepository.UpdateProgress")
	defer span.End()

	db := getDB(ctx, r.client.db)
	if err := db.Model(&entity.GenerationJob{}).Where("id = ?", id).Update("progress", progress).Error; err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to update job progress: %w", err)
	}
	return nil
}

// GetPendingJobs 获取待处理任务
func (r *JobRepository) GetPendingJobs(ctx context.Context, limit int) ([]*entity.GenerationJob, error) {
	ctx, span := tracer.Start(ctx, "postgres.JobRepository.GetPendingJobs")
	defer span.End()

	db := getDB(ctx, r.client.db)
	var jobs []*entity.GenerationJob

	if err := db.Where("status = ?", entity.JobStatusPending).
		Order("priority DESC, created_at ASC").
		Limit(limit).
		Find(&jobs).Error; err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to get pending jobs: %w", err)
	}

	return jobs, nil
}

// GetRunningJobs 获取运行中任务
func (r *JobRepository) GetRunningJobs(ctx context.Context) ([]*entity.GenerationJob, error) {
	ctx, span := tracer.Start(ctx, "postgres.JobRepository.GetRunningJobs")
	defer span.End()

	db := getDB(ctx, r.client.db)
	var jobs []*entity.GenerationJob

	if err := db.Where("status = ?", entity.JobStatusRunning).
		Order("started_at ASC").
		Find(&jobs).Error; err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to get running jobs: %w", err)
	}

	return jobs, nil
}

// GetFailedJobs 获取失败任务（可重试）
func (r *JobRepository) GetFailedJobs(ctx context.Context, maxRetries int, limit int) ([]*entity.GenerationJob, error) {
	ctx, span := tracer.Start(ctx, "postgres.JobRepository.GetFailedJobs")
	defer span.End()

	db := getDB(ctx, r.client.db)
	var jobs []*entity.GenerationJob

	if err := db.Where("status = ? AND retry_count < ?", entity.JobStatusFailed, maxRetries).
		Order("created_at ASC").
		Limit(limit).
		Find(&jobs).Error; err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to get failed jobs: %w", err)
	}

	return jobs, nil
}

// IncrementRetryCount 增加重试次数
func (r *JobRepository) IncrementRetryCount(ctx context.Context, id string) error {
	ctx, span := tracer.Start(ctx, "postgres.JobRepository.IncrementRetryCount")
	defer span.End()

	db := getDB(ctx, r.client.db)
	if err := db.Model(&entity.GenerationJob{}).Where("id = ?", id).
		Update("retry_count", gorm.Expr("retry_count + 1")).Error; err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to increment retry count: %w", err)
	}
	return nil
}

// SetResult 设置任务结果
func (r *JobRepository) SetResult(ctx context.Context, id string, result []byte, errMsg string) error {
	ctx, span := tracer.Start(ctx, "postgres.JobRepository.SetResult")
	defer span.End()

	db := getDB(ctx, r.client.db)
	now := time.Now()
	updates := map[string]interface{}{
		"completed_at": now,
	}
	if result != nil {
		updates["output_result"] = result
		updates["status"] = entity.JobStatusCompleted
	}
	if errMsg != "" {
		updates["error_message"] = errMsg
		updates["status"] = entity.JobStatusFailed
	}

	if err := db.Model(&entity.GenerationJob{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to set job result: %w", err)
	}
	return nil
}

// GetJobStats 获取任务统计信息
func (r *JobRepository) GetJobStats(ctx context.Context, projectID string) (*repository.JobStats, error) {
	ctx, span := tracer.Start(ctx, "postgres.JobRepository.GetJobStats")
	defer span.End()

	db := getDB(ctx, r.client.db)
	var stats repository.JobStats

	// 基础查询
	baseQuery := db.Model(&entity.GenerationJob{}).Where("project_id = ?", projectID)

	// 总任务数
	baseQuery.Count(&stats.TotalJobs)

	// 按状态统计
	db.Model(&entity.GenerationJob{}).Where("project_id = ? AND status = ?", projectID, entity.JobStatusPending).Count(&stats.PendingJobs)
	db.Model(&entity.GenerationJob{}).Where("project_id = ? AND status = ?", projectID, entity.JobStatusRunning).Count(&stats.RunningJobs)
	db.Model(&entity.GenerationJob{}).Where("project_id = ? AND status = ?", projectID, entity.JobStatusCompleted).Count(&stats.CompletedJobs)
	db.Model(&entity.GenerationJob{}).Where("project_id = ? AND status = ?", projectID, entity.JobStatusFailed).Count(&stats.FailedJobs)

	// Token 使用统计
	var tokensUsed *int64
	db.Model(&entity.GenerationJob{}).Where("project_id = ?", projectID).Select("SUM(tokens_prompt + tokens_completion)").Scan(&tokensUsed)
	if tokensUsed != nil {
		stats.TotalTokensUsed = *tokensUsed
	}

	return &stats, nil
}
