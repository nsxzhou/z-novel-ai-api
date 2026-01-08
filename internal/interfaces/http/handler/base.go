package handler

import (
	"context"
	"fmt"
	"strings"

	"z-novel-ai-api/internal/application/quota"
	"z-novel-ai-api/internal/config"
	"z-novel-ai-api/internal/domain/entity"
	"z-novel-ai-api/internal/domain/repository"
)

// resolveProviderModel 解析 LLM Provider 和 Model
func resolveProviderModel(cfg *config.Config, provider, model string) (string, string, error) {
	if cfg == nil {
		return "", "", fmt.Errorf("server config not configured")
	}

	p := strings.TrimSpace(provider)
	if p == "" {
		p = strings.TrimSpace(cfg.LLM.DefaultProvider)
	}
	if p == "" {
		return "", "", fmt.Errorf("llm provider not specified")
	}
	if len(p) > 32 {
		return "", "", fmt.Errorf("llm provider too long")
	}

	providerCfg, ok := cfg.LLM.Providers[p]
	if !ok {
		return "", "", fmt.Errorf("llm provider not found: %s", p)
	}

	m := strings.TrimSpace(model)
	if m == "" {
		m = strings.TrimSpace(providerCfg.Model)
	}
	if len(m) > 64 {
		return "", "", fmt.Errorf("llm model too long")
	}
	return p, m, nil
}

// precheckQuota 检查余额是否足以进行至少一次基础调用
func precheckQuota(ctx context.Context, quotaChecker *quota.TokenQuotaChecker, tenant *entity.Tenant) error {
	if quotaChecker == nil {
		return nil
	}
	// 预查时至少需要 1000 token 余额以允许开始任务
	_, err := quotaChecker.CheckBalance(ctx, tenant.ID, 1000)
	return err
}

// withTenantTx 在租户事务中执行
func withTenantTx(ctx context.Context, txMgr repository.Transactor, tenantCtx repository.TenantContextManager, tenantID string, fn func(context.Context) error) error {
	if txMgr == nil || tenantCtx == nil {
		return fmt.Errorf("transaction dependencies not configured")
	}
	return txMgr.WithTransaction(ctx, func(txCtx context.Context) error {
		if err := tenantCtx.SetTenant(txCtx, tenantID); err != nil {
			return err
		}
		return fn(txCtx)
	})
}
