package quota

import (
	"context"
	"fmt"
	"strings"

	"z-novel-ai-api/internal/domain/entity"
	"z-novel-ai-api/internal/domain/repository"
	"z-novel-ai-api/internal/domain/service"
)

type LLMUsageRecorder struct {
	tenantRepo repository.TenantRepository
	usageRepo  repository.LLMUsageEventRepository
}

func NewLLMUsageRecorder(tenantRepo repository.TenantRepository, usageRepo repository.LLMUsageEventRepository) *LLMUsageRecorder {
	return &LLMUsageRecorder{
		tenantRepo: tenantRepo,
		usageRepo:  usageRepo,
	}
}

func (r *LLMUsageRecorder) Record(ctx context.Context, in service.LLMUsageInput) error {
	if r == nil || r.tenantRepo == nil || r.usageRepo == nil {
		return nil
	}

	tenantID := strings.TrimSpace(in.TenantID)
	if tenantID == "" {
		return nil
	}
	if in.PromptTokens < 0 || in.CompletionTokens < 0 {
		return fmt.Errorf("invalid token usage")
	}

	totalTokens := int64(in.PromptTokens + in.CompletionTokens)
	if totalTokens > 0 {
		_ = r.tenantRepo.DeductBalance(ctx, tenantID, totalTokens)
	}

	evt := &entity.LLMUsageEvent{
		TenantID:         tenantID,
		Provider:         strings.TrimSpace(in.Provider),
		Model:            strings.TrimSpace(in.Model),
		Workflow:         strings.TrimSpace(in.Workflow),
		TokensPrompt:     in.PromptTokens,
		TokensCompletion: in.CompletionTokens,
		DurationMs:       in.DurationMs,
	}
	_ = r.usageRepo.Create(ctx, evt)
	return nil
}
