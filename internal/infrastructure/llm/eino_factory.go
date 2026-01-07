package llm

import (
	"context"
	"fmt"
	"sync"

	"z-novel-ai-api/internal/config"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
)

// EinoFactory 管理多个 Eino ChatModel 客户端实例
type EinoFactory struct {
	config *config.LLMConfig
	models map[string]model.BaseChatModel
	mu     sync.RWMutex
}

// NewEinoFactory 创建 Eino LLM 工厂
func NewEinoFactory(cfg *config.Config) *EinoFactory {
	return &EinoFactory{
		config: &cfg.LLM,
		models: make(map[string]model.BaseChatModel),
	}
}

// Get 获取指定名称的 ChatModel，如果未指定则返回默认客户端
func (f *EinoFactory) Get(ctx context.Context, name string) (model.BaseChatModel, error) {
	if name == "" {
		name = f.config.DefaultProvider
	}

	f.mu.RLock()
	m, ok := f.models[name]
	f.mu.RUnlock()
	if ok {
		return m, nil
	}

	// 惰性加载
	f.mu.Lock()
	defer f.mu.Unlock()

	// 再次检查防止竞态
	if m, ok = f.models[name]; ok {
		return m, nil
	}

	providerCfg, ok := f.config.Providers[name]
	if !ok {
		return nil, fmt.Errorf("provider %s not found in LLM config", name)
	}

	// 使用 Eino 的 OpenAI 适配器
	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey:      providerCfg.APIKey,
		BaseURL:     providerCfg.BaseURL,
		Model:       providerCfg.Model,
		MaxTokens:   &providerCfg.MaxTokens,
		Temperature: ptrFloat32(float32(providerCfg.Temperature)),
		Timeout:     providerCfg.Timeout,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create eino chat model for %s: %w", name, err)
	}

	f.models[name] = chatModel
	return chatModel, nil
}

// Default 返回默认 ChatModel
func (f *EinoFactory) Default(ctx context.Context) (model.BaseChatModel, error) {
	return f.Get(ctx, "")
}

func ptrFloat32(f float32) *float32 {
	return &f
}
