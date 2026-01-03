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

// JobRepository 生成任务仓储实现
type JobRepository struct {
	client *Client
}

// NewJobRepository 创建生成任务仓储
func NewJobRepository(client *Client) *JobRepository {
	return &JobRepository{client: client}
}

// Create 创建任务
func (r *JobRepository) Create(ctx context.Context, job *entity.GenerationJob) error {
	ctx, span := tracer.Start(ctx, "postgres.JobRepository.Create")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	query := `
		INSERT INTO generation_jobs (id, tenant_id, project_id, chapter_id, job_type, status, priority, 
			input_params, output_result, error_message, llm_provider, llm_model, tokens_prompt, tokens_completion, 
			duration_ms, retry_count, idempotency_key, created_at, started_at, completed_at)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, NOW(), $17, $18)
		RETURNING id, created_at
	`

	var chapterID sql.NullString
	if job.ChapterID != "" {
		chapterID = sql.NullString{String: job.ChapterID, Valid: true}
	}

	err := q.QueryRowContext(ctx, query,
		job.TenantID, job.ProjectID, chapterID, job.JobType, job.Status, job.Priority,
		job.InputParams, job.OutputResult, job.ErrorMessage, job.LLMProvider, job.LLMModel,
		job.TokensPrompt, job.TokensComplete, job.DurationMs, job.RetryCount, job.IdempotencyKey,
		job.StartedAt, job.CompletedAt,
	).Scan(&job.ID, &job.CreatedAt)

	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to create job: %w", err)
	}

	return nil
}

// GetByID 根据 ID 获取任务
func (r *JobRepository) GetByID(ctx context.Context, id string) (*entity.GenerationJob, error) {
	ctx, span := tracer.Start(ctx, "postgres.JobRepository.GetByID")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	query := `
		SELECT id, tenant_id, project_id, chapter_id, job_type, status, priority, 
			input_params, output_result, error_message, llm_provider, llm_model, tokens_prompt, tokens_completion, 
			duration_ms, retry_count, idempotency_key, created_at, started_at, completed_at
		FROM generation_jobs
		WHERE id = $1
	`

	return r.scanJob(q.QueryRowContext(ctx, query, id))
}

// Update 更新任务
func (r *JobRepository) Update(ctx context.Context, job *entity.GenerationJob) error {
	ctx, span := tracer.Start(ctx, "postgres.JobRepository.Update")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	query := `
		UPDATE generation_jobs
		SET status = $1, output_result = $2, error_message = $3, llm_provider = $4, llm_model = $5, 
			tokens_prompt = $6, tokens_completion = $7, duration_ms = $8, retry_count = $9, 
			started_at = $10, completed_at = $11
		WHERE id = $12
	`

	_, err := q.ExecContext(ctx, query,
		job.Status, job.OutputResult, job.ErrorMessage, job.LLMProvider, job.LLMModel,
		job.TokensPrompt, job.TokensComplete, job.DurationMs, job.RetryCount,
		job.StartedAt, job.CompletedAt, job.ID,
	)

	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to update job: %w", err)
	}

	return nil
}

// Delete 删除任务
func (r *JobRepository) Delete(ctx context.Context, id string) error {
	ctx, span := tracer.Start(ctx, "postgres.JobRepository.Delete")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	query := `DELETE FROM generation_jobs WHERE id = $1`
	_, err := q.ExecContext(ctx, query, id)

	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to delete job: %w", err)
	}

	return nil
}

// ListByProject 获取项目任务列表
func (r *JobRepository) ListByProject(ctx context.Context, projectID string, filter *repository.JobFilter, pagination repository.Pagination) (*repository.PagedResult[*entity.GenerationJob], error) {
	ctx, span := tracer.Start(ctx, "postgres.JobRepository.ListByProject")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	// 构建查询条件
	whereClause := "project_id = $1"
	args := []interface{}{projectID}
	argIdx := 2

	if filter != nil {
		if filter.JobType != "" {
			whereClause += fmt.Sprintf(" AND job_type = $%d", argIdx)
			args = append(args, filter.JobType)
			argIdx++
		}
		if filter.Status != "" {
			whereClause += fmt.Sprintf(" AND status = $%d", argIdx)
			args = append(args, filter.Status)
			argIdx++
		}
		if filter.ChapterID != "" {
			whereClause += fmt.Sprintf(" AND chapter_id = $%d", argIdx)
			args = append(args, filter.ChapterID)
			argIdx++
		}
	}

	// 获取总数
	var total int64
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM generation_jobs WHERE %s`, whereClause)
	if err := q.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to count jobs: %w", err)
	}

	// 获取列表
	query := fmt.Sprintf(`
		SELECT id, tenant_id, project_id, chapter_id, job_type, status, priority, 
			input_params, output_result, error_message, llm_provider, llm_model, tokens_prompt, tokens_completion, 
			duration_ms, retry_count, idempotency_key, created_at, started_at, completed_at
		FROM generation_jobs
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)

	args = append(args, pagination.Limit(), pagination.Offset())

	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to list jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*entity.GenerationJob
	for rows.Next() {
		job, err := r.scanJobFromRows(rows)
		if err != nil {
			span.RecordError(err)
			return nil, err
		}
		jobs = append(jobs, job)
	}

	return repository.NewPagedResult(jobs, total, pagination), nil
}

// GetByIdempotencyKey 根据幂等键获取任务
func (r *JobRepository) GetByIdempotencyKey(ctx context.Context, key string) (*entity.GenerationJob, error) {
	ctx, span := tracer.Start(ctx, "postgres.JobRepository.GetByIdempotencyKey")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	query := `
		SELECT id, tenant_id, project_id, chapter_id, job_type, status, priority, 
			input_params, output_result, error_message, llm_provider, llm_model, tokens_prompt, tokens_completion, 
			duration_ms, retry_count, idempotency_key, created_at, started_at, completed_at
		FROM generation_jobs
		WHERE idempotency_key = $1
	`

	return r.scanJob(q.QueryRowContext(ctx, query, key))
}

// UpdateStatus 更新任务状态
func (r *JobRepository) UpdateStatus(ctx context.Context, id string, status entity.JobStatus) error {
	ctx, span := tracer.Start(ctx, "postgres.JobRepository.UpdateStatus")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	query := `UPDATE generation_jobs SET status = $1 WHERE id = $2`
	_, err := q.ExecContext(ctx, query, status, id)

	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to update job status: %w", err)
	}

	return nil
}

// GetPendingJobs 获取待处理任务
func (r *JobRepository) GetPendingJobs(ctx context.Context, limit int) ([]*entity.GenerationJob, error) {
	ctx, span := tracer.Start(ctx, "postgres.JobRepository.GetPendingJobs")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	query := `
		SELECT id, tenant_id, project_id, chapter_id, job_type, status, priority, 
			input_params, output_result, error_message, llm_provider, llm_model, tokens_prompt, tokens_completion, 
			duration_ms, retry_count, idempotency_key, created_at, started_at, completed_at
		FROM generation_jobs
		WHERE status = 'pending'
		ORDER BY priority DESC, created_at ASC
		LIMIT $1
	`

	return r.queryJobs(ctx, q, query, limit)
}

// GetRunningJobs 获取运行中任务
func (r *JobRepository) GetRunningJobs(ctx context.Context) ([]*entity.GenerationJob, error) {
	ctx, span := tracer.Start(ctx, "postgres.JobRepository.GetRunningJobs")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	query := `
		SELECT id, tenant_id, project_id, chapter_id, job_type, status, priority, 
			input_params, output_result, error_message, llm_provider, llm_model, tokens_prompt, tokens_completion, 
			duration_ms, retry_count, idempotency_key, created_at, started_at, completed_at
		FROM generation_jobs
		WHERE status = 'running'
		ORDER BY started_at ASC
	`

	return r.queryJobs(ctx, q, query)
}

// GetFailedJobs 获取失败任务（可重试）
func (r *JobRepository) GetFailedJobs(ctx context.Context, maxRetries int, limit int) ([]*entity.GenerationJob, error) {
	ctx, span := tracer.Start(ctx, "postgres.JobRepository.GetFailedJobs")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	query := `
		SELECT id, tenant_id, project_id, chapter_id, job_type, status, priority, 
			input_params, output_result, error_message, llm_provider, llm_model, tokens_prompt, tokens_completion, 
			duration_ms, retry_count, idempotency_key, created_at, started_at, completed_at
		FROM generation_jobs
		WHERE status = 'failed' AND retry_count < $1
		ORDER BY created_at ASC
		LIMIT $2
	`

	return r.queryJobs(ctx, q, query, maxRetries, limit)
}

// IncrementRetryCount 增加重试次数
func (r *JobRepository) IncrementRetryCount(ctx context.Context, id string) error {
	ctx, span := tracer.Start(ctx, "postgres.JobRepository.IncrementRetryCount")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	query := `UPDATE generation_jobs SET retry_count = retry_count + 1, status = 'pending' WHERE id = $1`
	_, err := q.ExecContext(ctx, query, id)

	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to increment retry count: %w", err)
	}

	return nil
}

// SetResult 设置任务结果
func (r *JobRepository) SetResult(ctx context.Context, id string, result []byte, errMsg string) error {
	ctx, span := tracer.Start(ctx, "postgres.JobRepository.SetResult")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	var status string
	if errMsg != "" {
		status = "failed"
	} else {
		status = "completed"
	}

	query := `
		UPDATE generation_jobs 
		SET status = $1, output_result = $2, error_message = $3, completed_at = NOW()
		WHERE id = $4
	`
	_, err := q.ExecContext(ctx, query, status, result, errMsg, id)

	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to set result: %w", err)
	}

	return nil
}

// GetJobStats 获取任务统计信息
func (r *JobRepository) GetJobStats(ctx context.Context, projectID string) (*repository.JobStats, error) {
	ctx, span := tracer.Start(ctx, "postgres.JobRepository.GetJobStats")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	query := `
		SELECT 
			COUNT(*) as total_jobs,
			COUNT(*) FILTER (WHERE status = 'pending') as pending_jobs,
			COUNT(*) FILTER (WHERE status = 'running') as running_jobs,
			COUNT(*) FILTER (WHERE status = 'completed') as completed_jobs,
			COUNT(*) FILTER (WHERE status = 'failed') as failed_jobs,
			COALESCE(SUM(tokens_prompt + tokens_completion), 0) as total_tokens_used
		FROM generation_jobs
		WHERE project_id = $1
	`

	var stats repository.JobStats
	err := q.QueryRowContext(ctx, query, projectID).Scan(
		&stats.TotalJobs, &stats.PendingJobs, &stats.RunningJobs,
		&stats.CompletedJobs, &stats.FailedJobs, &stats.TotalTokensUsed,
	)

	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to get job stats: %w", err)
	}

	return &stats, nil
}

// queryJobs 通用查询任务
func (r *JobRepository) queryJobs(ctx context.Context, q Querier, query string, args ...interface{}) ([]*entity.GenerationJob, error) {
	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*entity.GenerationJob
	for rows.Next() {
		job, err := r.scanJobFromRows(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}

	return jobs, nil
}

// scanJob 扫描单行任务数据
func (r *JobRepository) scanJob(row *sql.Row) (*entity.GenerationJob, error) {
	var job entity.GenerationJob
	var chapterID, idempotencyKey sql.NullString
	var inputParams, outputResult json.RawMessage
	var startedAt, completedAt sql.NullTime

	err := row.Scan(
		&job.ID, &job.TenantID, &job.ProjectID, &chapterID, &job.JobType, &job.Status, &job.Priority,
		&inputParams, &outputResult, &job.ErrorMessage, &job.LLMProvider, &job.LLMModel,
		&job.TokensPrompt, &job.TokensComplete, &job.DurationMs, &job.RetryCount, &idempotencyKey,
		&job.CreatedAt, &startedAt, &completedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan job: %w", err)
	}

	if chapterID.Valid {
		job.ChapterID = chapterID.String
	}
	if idempotencyKey.Valid {
		job.IdempotencyKey = idempotencyKey.String
	}
	if startedAt.Valid {
		job.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		job.CompletedAt = &completedAt.Time
	}
	job.InputParams = inputParams
	job.OutputResult = outputResult

	return &job, nil
}

// scanJobFromRows 从多行结果扫描
func (r *JobRepository) scanJobFromRows(rows *sql.Rows) (*entity.GenerationJob, error) {
	var job entity.GenerationJob
	var chapterID, idempotencyKey sql.NullString
	var inputParams, outputResult json.RawMessage
	var startedAt, completedAt sql.NullTime

	err := rows.Scan(
		&job.ID, &job.TenantID, &job.ProjectID, &chapterID, &job.JobType, &job.Status, &job.Priority,
		&inputParams, &outputResult, &job.ErrorMessage, &job.LLMProvider, &job.LLMModel,
		&job.TokensPrompt, &job.TokensComplete, &job.DurationMs, &job.RetryCount, &idempotencyKey,
		&job.CreatedAt, &startedAt, &completedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to scan job row: %w", err)
	}

	if chapterID.Valid {
		job.ChapterID = chapterID.String
	}
	if idempotencyKey.Valid {
		job.IdempotencyKey = idempotencyKey.String
	}
	if startedAt.Valid {
		job.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		job.CompletedAt = &completedAt.Time
	}
	job.InputParams = inputParams
	job.OutputResult = outputResult

	return &job, nil
}
