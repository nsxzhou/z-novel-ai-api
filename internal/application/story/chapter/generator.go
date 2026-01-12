package chapter

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/schema"

	workflowchain "z-novel-ai-api/internal/workflow/chain"
	wfmodel "z-novel-ai-api/internal/workflow/model"
	workflowport "z-novel-ai-api/internal/workflow/port"
)

type ChapterGenerator struct {
	chain *workflowchain.ChapterChain
}

func NewChapterGenerator(factory workflowport.ChatModelFactory) *ChapterGenerator {
	return &ChapterGenerator{
		chain: workflowchain.NewChapterChain(factory),
	}
}

func (g *ChapterGenerator) Generate(ctx context.Context, in *wfmodel.ChapterGenerateInput) (*wfmodel.ChapterGenerateOutput, error) {
	if g == nil || g.chain == nil {
		return nil, fmt.Errorf("chapter workflow not configured")
	}
	if in == nil {
		return nil, fmt.Errorf("input is nil")
	}

	outMsg, err := g.chain.Invoke(ctx, in)
	if err != nil {
		return nil, err
	}
	if outMsg == nil {
		return nil, fmt.Errorf("empty llm response")
	}

	meta := wfmodel.LLMUsageMeta{
		Provider:    strings.TrimSpace(in.Provider),
		Model:       strings.TrimSpace(in.Model),
		GeneratedAt: time.Now().UTC(),
	}
	if in.Temperature != nil {
		meta.Temperature = float64(*in.Temperature)
	}
	if outMsg.ResponseMeta != nil && outMsg.ResponseMeta.Usage != nil {
		meta.PromptTokens = outMsg.ResponseMeta.Usage.PromptTokens
		meta.CompletionTokens = outMsg.ResponseMeta.Usage.CompletionTokens
	}

	content := strings.TrimSpace(outMsg.Content)
	if content == "" {
		return nil, fmt.Errorf("empty chapter content")
	}

	return &wfmodel.ChapterGenerateOutput{
		Content: content,
		Meta:    meta,
	}, nil
}

// Stream 返回 Eino StreamReader；调用方负责 Close()。
// 约定：流可能在最后返回一个 Content 为空但包含 Usage 的消息，用于 Token 统计。
func (g *ChapterGenerator) Stream(ctx context.Context, in *wfmodel.ChapterGenerateInput) (*schema.StreamReader[*schema.Message], error) {
	if g == nil || g.chain == nil {
		return nil, fmt.Errorf("chapter workflow not configured")
	}
	if in == nil {
		return nil, fmt.Errorf("input is nil")
	}
	return g.chain.Stream(ctx, in)
}
