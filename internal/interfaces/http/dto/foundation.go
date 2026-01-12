// Package dto 提供 HTTP 层数据传输对象
package dto

import (
	storyfoundation "z-novel-ai-api/internal/application/story/foundation"
	storymodel "z-novel-ai-api/internal/application/story/model"
	wfmodel "z-novel-ai-api/internal/workflow/model"
)

// FoundationTextAttachment 设定集生成的文本附件
type FoundationTextAttachment struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

// FoundationGenerateRequest 设定集生成请求（同步预览 / SSE / 异步 Job 共用）
type FoundationGenerateRequest struct {
	Prompt      string                     `json:"prompt" binding:"required"`
	Attachments []FoundationTextAttachment `json:"attachments,omitempty"`

	Provider    string   `json:"provider,omitempty"`
	Model       string   `json:"model,omitempty"`
	Temperature *float32 `json:"temperature,omitempty"`
	MaxTokens   *int     `json:"max_tokens,omitempty"`
}

// ToStoryInput 转换为应用层输入结构
func (r *FoundationGenerateRequest) ToStoryInput(projectTitle, projectDescription string, provider, model string) *wfmodel.FoundationGenerateInput {
	attachments := make([]wfmodel.TextAttachment, 0, len(r.Attachments))
	for i := range r.Attachments {
		a := r.Attachments[i]
		attachments = append(attachments, wfmodel.TextAttachment{
			Name:    a.Name,
			Content: a.Content,
		})
	}

	return &wfmodel.FoundationGenerateInput{
		ProjectTitle:       projectTitle,
		ProjectDescription: projectDescription,
		Prompt:             r.Prompt,
		Attachments:        attachments,
		Provider:           provider,
		Model:              model,
		Temperature:        r.Temperature,
		MaxTokens:          r.MaxTokens,
	}
}

// FoundationUsageResponse LLM 使用量信息
type FoundationUsageResponse struct {
	Provider         string  `json:"provider,omitempty"`
	Model            string  `json:"model,omitempty"`
	PromptTokens     int     `json:"prompt_tokens,omitempty"`
	CompletionTokens int     `json:"completion_tokens,omitempty"`
	Temperature      float64 `json:"temperature,omitempty"`
	DurationMs       int     `json:"duration_ms,omitempty"`
	GeneratedAt      string  `json:"generated_at,omitempty"`
}

// FoundationPreviewResponse 同步预览响应
type FoundationPreviewResponse struct {
	JobID string                     `json:"job_id,omitempty"`
	Plan  *storymodel.FoundationPlan `json:"plan"`
	Usage *FoundationUsageResponse   `json:"usage,omitempty"`
}

// FoundationApplyRequest 应用 Plan（落库）请求
// 约定：job_id 与 plan 二选一；优先使用 job_id（减少 payload 传输）。
type FoundationApplyRequest struct {
	JobID string                     `json:"job_id,omitempty"`
	Plan  *storymodel.FoundationPlan `json:"plan,omitempty"`
}

// FoundationApplyResponse 应用 Plan（落库）响应
type FoundationApplyResponse struct {
	ProjectID string                                 `json:"project_id"`
	Result    *storyfoundation.FoundationApplyResult `json:"result"`
}
