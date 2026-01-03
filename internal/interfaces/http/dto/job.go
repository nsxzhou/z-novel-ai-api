// Package dto 提供 HTTP 层数据传输对象
package dto

import (
	"time"

	"z-novel-ai-api/internal/domain/entity"
)

// JobResponse 任务响应
type JobResponse struct {
	ID          string                 `json:"id"`
	ProjectID   string                 `json:"project_id"`
	ChapterID   string                 `json:"chapter_id,omitempty"`
	JobType     string                 `json:"job_type"`
	Status      string                 `json:"status"`
	Priority    int                    `json:"priority"`
	Payload     map[string]interface{} `json:"payload,omitempty"`
	Result      map[string]interface{} `json:"result,omitempty"`
	ErrorMsg    string                 `json:"error_msg,omitempty"`
	RetryCount  int                    `json:"retry_count"`
	Progress    int                    `json:"progress"`
	ScheduledAt time.Time              `json:"scheduled_at,omitempty"`
	StartedAt   time.Time              `json:"started_at,omitempty"`
	CompletedAt time.Time              `json:"completed_at,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// JobListResponse 任务列表响应
type JobListResponse struct {
	Jobs []*JobResponse `json:"jobs"`
}

// CancelJobResponse 取消任务响应
type CancelJobResponse struct {
	ID        string `json:"id"`
	Cancelled bool   `json:"cancelled"`
}

// ToJobResponse 将领域实体转换为响应 DTO
func ToJobResponse(j *entity.GenerationJob) *JobResponse {
	if j == nil {
		return nil
	}

	resp := &JobResponse{
		ID:         j.ID,
		ProjectID:  j.ProjectID,
		ChapterID:  j.ChapterID,
		JobType:    string(j.JobType),
		Status:     string(j.Status),
		Priority:   j.Priority,
		ErrorMsg:   j.ErrorMessage,
		RetryCount: j.RetryCount,
		Progress:   j.Progress,
		CreatedAt:  j.CreatedAt,
		UpdatedAt:  j.UpdatedAt,
	}

	if j.StartedAt != nil {
		resp.StartedAt = *j.StartedAt
	}
	if j.CompletedAt != nil {
		resp.CompletedAt = *j.CompletedAt
	}

	return resp
}

// ToJobListResponse 将领域实体列表转换为响应 DTO
func ToJobListResponse(jobs []*entity.GenerationJob) *JobListResponse {
	resp := &JobListResponse{
		Jobs: make([]*JobResponse, 0, len(jobs)),
	}

	for _, j := range jobs {
		resp.Jobs = append(resp.Jobs, ToJobResponse(j))
	}

	return resp
}
