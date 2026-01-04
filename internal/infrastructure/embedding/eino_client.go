package embedding

import (
	"context"
	"fmt"

	"z-novel-ai-api/internal/config"

	"github.com/cloudwego/eino-ext/components/embedding/openai"
	"github.com/cloudwego/eino/components/embedding"
)

// NewEinoEmbedder 创建基于 Eino 的 Embedder
func NewEinoEmbedder(ctx context.Context, cfg *config.EmbeddingConfig) (embedding.Embedder, error) {
	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("embedding endpoint is required")
	}

	// 使用 Eino 的 OpenAI 适配器
	embedder, err := openai.NewEmbedder(ctx, &openai.EmbeddingConfig{
		APIKey:  cfg.APIKey,
		BaseURL: cfg.Endpoint,
		Model:   cfg.Model,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create eino embedder: %w", err)
	}

	return embedder, nil
}
