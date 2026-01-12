package projectcreation

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"z-novel-ai-api/internal/application/story/storyutil"
	workflowchain "z-novel-ai-api/internal/workflow/chain"
	wfmodel "z-novel-ai-api/internal/workflow/model"
	workflowport "z-novel-ai-api/internal/workflow/port"
	"z-novel-ai-api/pkg/logger"
)

// projectCreationLLMEnvelope 用于解析 LLM 返回的 JSON 结构的信封
type projectCreationLLMEnvelope struct {
	AssistantMessage     string                               `json:"assistant_message"`
	NextStage            string                               `json:"stage"`
	Draft                json.RawMessage                      `json:"draft"`
	Action               string                               `json:"action"`
	RequiresConfirmation bool                                 `json:"requires_confirmation"`
	Project              *wfmodel.ProjectCreationProjectDraft `json:"project,omitempty"`
}

type ProjectCreationGenerator struct {
	chain *workflowchain.ProjectCreationChain
}

func NewProjectCreationGenerator(factory workflowport.ChatModelFactory) *ProjectCreationGenerator {
	return &ProjectCreationGenerator{
		chain: workflowchain.NewProjectCreationChain(factory),
	}
}

// Generate 执行生成流程：Prompt 渲染 -> LLM 调用 (Structured Output) -> 结果解析
func (g *ProjectCreationGenerator) Generate(ctx context.Context, in *wfmodel.ProjectCreationGenerateInput) (*wfmodel.ProjectCreationGenerateOutput, error) {
	if g == nil || g.chain == nil {
		return nil, fmt.Errorf("project creation workflow not configured")
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

	raw := storyutil.ExtractJSONObject(outMsg.Content)
	if strings.TrimSpace(raw) == "" {
		return nil, fmt.Errorf("empty project creation output")
	}

	var env projectCreationLLMEnvelope
	if err := json.Unmarshal([]byte(raw), &env); err != nil {
		logger.Error(ctx, "failed to unmarshal project creation output", err, "raw", raw)
		return nil, fmt.Errorf("invalid project creation output: %w", err)
	}

	meta := wfmodel.LLMUsageMeta{Provider: strings.TrimSpace(in.Provider), Model: strings.TrimSpace(in.Model), GeneratedAt: time.Now().UTC()}
	if in.Temperature != nil {
		meta.Temperature = float64(*in.Temperature)
	}
	if outMsg.ResponseMeta != nil && outMsg.ResponseMeta.Usage != nil {
		meta.PromptTokens = outMsg.ResponseMeta.Usage.PromptTokens
		meta.CompletionTokens = outMsg.ResponseMeta.Usage.CompletionTokens
	}

	nextStage := strings.TrimSpace(env.NextStage)
	if nextStage == "" {
		nextStage = strings.TrimSpace(in.Stage)
		if nextStage == "" {
			nextStage = "discover"
		}
	}

	return &wfmodel.ProjectCreationGenerateOutput{
		AssistantMessage:     strings.TrimSpace(env.AssistantMessage),
		NextStage:            nextStage,
		Draft:                env.Draft,
		Action:               strings.TrimSpace(env.Action),
		RequiresConfirmation: env.RequiresConfirmation,
		ProposedProject:      env.Project,
		Meta:                 meta,
	}, nil
}
