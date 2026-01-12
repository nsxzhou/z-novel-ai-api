package foundation

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/schema"

	storymodel "z-novel-ai-api/internal/application/story/model"
	workflowchain "z-novel-ai-api/internal/workflow/chain"
	wfmodel "z-novel-ai-api/internal/workflow/model"
	workflowport "z-novel-ai-api/internal/workflow/port"
)

type FoundationGenerateOutput struct {
	Plan *storymodel.FoundationPlan
	Raw  string
	Meta wfmodel.LLMUsageMeta
}

type FoundationGenerator struct {
	chain *workflowchain.FoundationChain
}

func NewFoundationGenerator(factory workflowport.ChatModelFactory) *FoundationGenerator {
	return &FoundationGenerator{
		chain: workflowchain.NewFoundationChain(factory),
	}
}

func (g *FoundationGenerator) Generate(ctx context.Context, in *wfmodel.FoundationGenerateInput) (*FoundationGenerateOutput, error) {
	if g == nil || g.chain == nil {
		return nil, fmt.Errorf("foundation workflow not configured")
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

	plan, raw, err := ParseFoundationPlan(outMsg.Content)
	if err != nil {
		return nil, err
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

	return &FoundationGenerateOutput{
		Plan: plan,
		Raw:  raw,
		Meta: meta,
	}, nil
}

// Stream 返回 Eino StreamReader；调用方负责 Close()。
// 约定：流可能在最后返回一个 Content 为空但包含 Usage 的消息，用于 Token 统计。
func (g *FoundationGenerator) Stream(ctx context.Context, in *wfmodel.FoundationGenerateInput) (*schema.StreamReader[*schema.Message], error) {
	if g == nil || g.chain == nil {
		return nil, fmt.Errorf("foundation workflow not configured")
	}
	if in == nil {
		return nil, fmt.Errorf("input is nil")
	}
	return g.chain.Stream(ctx, in)
}
