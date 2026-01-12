package chain

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	llmctx "z-novel-ai-api/internal/domain/service"
	wfmodel "z-novel-ai-api/internal/workflow/model"
	workflowport "z-novel-ai-api/internal/workflow/port"
	workflowprompt "z-novel-ai-api/internal/workflow/prompt"
)

type ChapterChain struct {
	factory workflowport.ChatModelFactory
}

func NewChapterChain(factory workflowport.ChatModelFactory) *ChapterChain {
	return &ChapterChain{factory: factory}
}

func (c *ChapterChain) Invoke(ctx context.Context, in *wfmodel.ChapterGenerateInput) (*schema.Message, error) {
	if c == nil || c.factory == nil {
		return nil, fmt.Errorf("llm factory not configured")
	}
	if in == nil {
		return nil, fmt.Errorf("input is nil")
	}
	if strings.TrimSpace(in.Provider) == "" {
		return nil, fmt.Errorf("provider is required")
	}
	if strings.TrimSpace(in.ChapterOutline) == "" {
		return nil, fmt.Errorf("chapter outline is required")
	}
	if in.TargetWordCount <= 0 {
		return nil, fmt.Errorf("target_word_count is required")
	}

	ctx = llmctx.WithWorkflowProvider(ctx, "chapter_generate", strings.TrimSpace(in.Provider))
	chatModel, err := c.factory.Get(ctx, strings.TrimSpace(in.Provider))
	if err != nil {
		return nil, err
	}

	msgs, err := formatChapterMessages(ctx, in)
	if err != nil {
		return nil, err
	}

	outMsg, err := chatModel.Generate(ctx, msgs, buildChapterModelOptions(in)...)
	if err != nil {
		return nil, err
	}
	if outMsg == nil {
		return nil, fmt.Errorf("empty llm response")
	}
	return outMsg, nil
}

// Stream 返回 Eino StreamReader；调用方负责 Close()。
// 约定：流可能在最后返回一个 Content 为空但包含 Usage 的消息，用于 Token 统计。
func (c *ChapterChain) Stream(ctx context.Context, in *wfmodel.ChapterGenerateInput) (*schema.StreamReader[*schema.Message], error) {
	if c == nil || c.factory == nil {
		return nil, fmt.Errorf("llm factory not configured")
	}
	if in == nil {
		return nil, fmt.Errorf("input is nil")
	}
	if strings.TrimSpace(in.Provider) == "" {
		return nil, fmt.Errorf("provider is required")
	}
	if strings.TrimSpace(in.ChapterOutline) == "" {
		return nil, fmt.Errorf("chapter outline is required")
	}
	if in.TargetWordCount <= 0 {
		return nil, fmt.Errorf("target_word_count is required")
	}

	ctx = llmctx.WithWorkflowProvider(ctx, "chapter_stream", strings.TrimSpace(in.Provider))
	chatModel, err := c.factory.Get(ctx, strings.TrimSpace(in.Provider))
	if err != nil {
		return nil, err
	}

	msgs, err := formatChapterMessages(ctx, in)
	if err != nil {
		return nil, err
	}
	return chatModel.Stream(ctx, msgs, buildChapterModelOptions(in)...)
}

var chapterPromptRegistry = workflowprompt.NewRegistry()

func formatChapterMessages(ctx context.Context, in *wfmodel.ChapterGenerateInput) ([]*schema.Message, error) {
	tpl, err := chapterPromptRegistry.ChatTemplate(workflowprompt.PromptChapterGenV1)
	if err != nil {
		return nil, err
	}
	vars := map[string]any{
		"project_title":       strings.TrimSpace(in.ProjectTitle),
		"project_description": strings.TrimSpace(in.ProjectDescription),
		"writing_style":       strings.TrimSpace(in.WritingStyle),
		"pov":                 strings.TrimSpace(in.POV),
		"target_word_count":   in.TargetWordCount,
		"chapter_title":       strings.TrimSpace(in.ChapterTitle),
		"chapter_outline":     strings.TrimSpace(in.ChapterOutline),
		"retrieved_context":   strings.TrimSpace(in.RetrievedContext),
	}
	return tpl.Format(ctx, vars)
}

func buildChapterModelOptions(in *wfmodel.ChapterGenerateInput) []model.Option {
	opts := make([]model.Option, 0, 4)
	if in == nil {
		return opts
	}
	if in.Temperature != nil {
		opts = append(opts, model.WithTemperature(*in.Temperature))
	}
	if in.MaxTokens != nil {
		opts = append(opts, model.WithMaxTokens(*in.MaxTokens))
	}
	if strings.TrimSpace(in.Model) != "" {
		opts = append(opts, model.WithModel(strings.TrimSpace(in.Model)))
	}
	return opts
}
