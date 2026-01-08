// Package quota 提供租户配额相关能力
package quota

import (
	"context"
	"fmt"

	"z-novel-ai-api/internal/domain/repository"
)

// TokenQuotaExceededError 表示租户 Token 余额不足
type TokenBalanceExceededError struct {
	TenantID string
	Balance  int64
	Required int64
}

func (e TokenBalanceExceededError) Error() string {
	return fmt.Sprintf("token balance insufficient: tenant=%s balance=%d required=%d", e.TenantID, e.Balance, e.Required)
}

// TokenQuotaChecker 用于检查租户 Token 余额
type TokenQuotaChecker struct {
	tenantRepo repository.TenantRepository
}

func NewTokenQuotaChecker(tenantRepo repository.TenantRepository) *TokenQuotaChecker {
	return &TokenQuotaChecker{
		tenantRepo: tenantRepo,
	}
}

// CheckBalance 检查租户余额是否充足
func (c *TokenQuotaChecker) CheckBalance(ctx context.Context, tenantID string, required int64) (balance int64, err error) {
	tenant, err := c.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return 0, fmt.Errorf("failed to get tenant for balance check: %w", err)
	}
	if tenant == nil {
		return 0, fmt.Errorf("tenant not found: %s", tenantID)
	}

	if !tenant.HasSufficientBalance(required) {
		return tenant.TokenBalance, TokenBalanceExceededError{
			TenantID: tenantID,
			Balance:  tenant.TokenBalance,
			Required: required,
		}
	}

	return tenant.TokenBalance, nil
}
