package context

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"z-novel-ai-api/internal/domain/entity"
)

type KVCache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
}

const DefaultRollingConversationContextTTL = 30 * 24 * time.Hour

type RollingContextManager struct {
	cache KVCache
	ttl   time.Duration
}

func NewRollingContextManager(cache KVCache) *RollingContextManager {
	return &RollingContextManager{
		cache: cache,
		ttl:   DefaultRollingConversationContextTTL,
	}
}

func (m *RollingContextManager) SnapshotAndAppendUserPrompt(ctx context.Context, tenantID, projectID, sessionID string, task entity.ConversationTask, userPrompt string) (summary string, recentUserTurns string, updateErr error) {
	if m == nil || m.cache == nil || strings.TrimSpace(string(task)) == "" {
		return "", "", nil
	}

	key := rollingContextKey(tenantID, projectID, sessionID, task)

	var rolling RollingConversationContext
	if b, err := m.cache.Get(ctx, key); err == nil && len(b) > 0 {
		_ = json.Unmarshal(b, &rolling)
	}

	summary, recentUserTurns = rolling.SnapshotForPrompt()
	rolling.AppendUserPrompt(strings.TrimSpace(userPrompt))
	updateErr = m.cache.Set(ctx, key, &rolling, m.ttl)
	return summary, recentUserTurns, updateErr
}

func rollingContextKey(tenantID, projectID, sessionID string, task entity.ConversationTask) string {
	return fmt.Sprintf("ctx:%s:%s:%s:%s:rolling", tenantID, projectID, sessionID, task)
}
