// Package quota 提供租户配额相关能力
package quota

import (
	"context"
	"fmt"
	"time"

	"z-novel-ai-api/internal/domain/entity"
	"z-novel-ai-api/internal/domain/repository"
)

// TokenQuotaExceededError 表示租户 Token 日配额已耗尽
type TokenQuotaExceededError struct {
	TenantID string
	Max      int64
	Used     int64
}

func (e TokenQuotaExceededError) Error() string {
	return fmt.Sprintf("token quota exceeded: tenant=%s used=%d max=%d", e.TenantID, e.Used, e.Max)
}

// TokenQuotaChecker 用于检查租户 Token 日配额
type TokenQuotaChecker struct {
	jobRepo repository.JobRepository
	llmRepo repository.LLMUsageEventRepository
	now     func() time.Time
}

func NewTokenQuotaChecker(jobRepo repository.JobRepository, llmRepo repository.LLMUsageEventRepository) *TokenQuotaChecker {
	return &TokenQuotaChecker{
		jobRepo: jobRepo,
		llmRepo: llmRepo,
		now:     time.Now,
	}
}

// CheckDailyTokens 检查租户是否还有当日 Token 配额。
// 返回：used/max（便于客户端展示），以及是否超过配额的 error。
func (c *TokenQuotaChecker) CheckDailyTokens(ctx context.Context, tenantID string, quota *entity.TenantQuota) (used int64, max int64, err error) {
	if quota == nil || quota.MaxTokensPerDay <= 0 {
		return 0, 0, nil
	}

	now := c.now().UTC()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)

	jobUsed, err := c.jobRepo.GetTokenUsage(ctx, tenantID, start, end)
	if err != nil {
		return 0, quota.MaxTokensPerDay, err
	}
	used = jobUsed
	if c.llmRepo != nil {
		llmUsed, llmErr := c.llmRepo.GetTokenUsage(ctx, tenantID, start, end)
		if llmErr != nil {
			return 0, quota.MaxTokensPerDay, llmErr
		}
		used += llmUsed
	}
	if used >= quota.MaxTokensPerDay {
		return used, quota.MaxTokensPerDay, TokenQuotaExceededError{
			TenantID: tenantID,
			Max:      quota.MaxTokensPerDay,
			Used:     used,
		}
	}
	return used, quota.MaxTokensPerDay, nil
}
