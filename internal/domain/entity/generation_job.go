// Package entity 定义领域实体
package entity

import (
	"encoding/json"
	"time"
)

// JobType 任务类型
type JobType string

const (
	JobTypeChapterGen    JobType = "chapter_gen"
	JobTypeSummary       JobType = "summary"
	JobTypeEntityExtract JobType = "entity_extract"
	JobTypeEmbeddingGen  JobType = "embedding_gen"
	JobTypeIndexRebuild  JobType = "index_rebuild"
)

// JobStatus 任务状态
type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
	JobStatusCancelled JobStatus = "cancelled"
)

// GenerationJob 生成任务
type GenerationJob struct {
	ID             string          `json:"id"`
	TenantID       string          `json:"tenant_id"`
	ProjectID      string          `json:"project_id"`
	ChapterID      string          `json:"chapter_id,omitempty"`
	JobType        JobType         `json:"job_type"`
	Status         JobStatus       `json:"status"`
	Priority       int             `json:"priority"`
	InputParams    json.RawMessage `json:"input_params"`
	OutputResult   json.RawMessage `json:"output_result,omitempty"`
	ErrorMessage   string          `json:"error_message,omitempty"`
	LLMProvider    string          `json:"llm_provider,omitempty"`
	LLMModel       string          `json:"llm_model,omitempty"`
	TokensPrompt   int             `json:"tokens_prompt,omitempty"`
	TokensComplete int             `json:"tokens_completion,omitempty"`
	DurationMs     int             `json:"duration_ms,omitempty"`
	RetryCount     int             `json:"retry_count"`
	Progress       int             `json:"progress"` // 任务进度 (0-100)
	IdempotencyKey string          `json:"idempotency_key,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
	StartedAt      *time.Time      `json:"started_at,omitempty"`
	CompletedAt    *time.Time      `json:"completed_at,omitempty"`
}

// NewGenerationJob 创建新任务
func NewGenerationJob(tenantID, projectID string, jobType JobType, inputParams json.RawMessage) *GenerationJob {
	return &GenerationJob{
		TenantID:    tenantID,
		ProjectID:   projectID,
		JobType:     jobType,
		Status:      JobStatusPending,
		Priority:    5,
		InputParams: inputParams,
		RetryCount:  0,
		CreatedAt:   time.Now(),
	}
}

// Start 开始执行任务
func (j *GenerationJob) Start() {
	now := time.Now()
	j.Status = JobStatusRunning
	j.StartedAt = &now
}

// Complete 完成任务
func (j *GenerationJob) Complete(result json.RawMessage) {
	now := time.Now()
	j.Status = JobStatusCompleted
	j.OutputResult = result
	j.CompletedAt = &now
	if j.StartedAt != nil {
		j.DurationMs = int(now.Sub(*j.StartedAt).Milliseconds())
	}
}

// Fail 任务失败
func (j *GenerationJob) Fail(errMsg string) {
	now := time.Now()
	j.Status = JobStatusFailed
	j.ErrorMessage = errMsg
	j.CompletedAt = &now
	if j.StartedAt != nil {
		j.DurationMs = int(now.Sub(*j.StartedAt).Milliseconds())
	}
}

// Retry 重试任务
func (j *GenerationJob) Retry() {
	j.RetryCount++
	j.Status = JobStatusPending
	j.StartedAt = nil
	j.CompletedAt = nil
	j.ErrorMessage = ""
}

// CanRetry 检查是否可以重试
func (j *GenerationJob) CanRetry(maxRetries int) bool {
	return j.RetryCount < maxRetries && j.Status == JobStatusFailed
}

// SetLLMMetrics 设置 LLM 使用指标
func (j *GenerationJob) SetLLMMetrics(provider, model string, promptTokens, completionTokens int) {
	j.LLMProvider = provider
	j.LLMModel = model
	j.TokensPrompt = promptTokens
	j.TokensComplete = completionTokens
}

// UpdateProgress 更新任务进度
func (j *GenerationJob) UpdateProgress(progress int) {
	if progress < 0 {
		progress = 0
	}
	if progress > 100 {
		progress = 100
	}
	j.Progress = progress
}
