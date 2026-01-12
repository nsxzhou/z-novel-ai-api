package service

import "context"

// LLMUsageInput 表示一次 LLM 调用的可计费与可观测数据。
// 说明：该结构位于 domain/service，作为跨层的稳定契约（port），避免基础设施层依赖应用层实现。
type LLMUsageInput struct {
	TenantID string

	Workflow string
	Provider string
	Model    string

	PromptTokens     int
	CompletionTokens int
	DurationMs       int
}

// LLMUsageRecorder 负责记录 LLM 使用量（扣费 + 流水落库等）。
// 约定：该接口的实现应尽量“best-effort”，不应阻塞主业务流程。
type LLMUsageRecorder interface {
	Record(ctx context.Context, in LLMUsageInput) error
}
