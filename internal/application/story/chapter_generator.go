package story

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	"z-novel-ai-api/internal/infrastructure/llm"
	einoobs "z-novel-ai-api/internal/observability/eino"
	workflowprompt "z-novel-ai-api/internal/workflow/prompt"
)

type ChapterGenerateInput struct {
	ProjectTitle       string
	ProjectDescription string

	ChapterTitle   string
	ChapterOutline string

	RetrievedContext string

	TargetWordCount int
	WritingStyle    string
	POV             string

	Provider string
	Model    string

	Temperature *float32
	MaxTokens   *int
}

type ChapterGenerateOutput struct {
	Content string
	Meta    LLMUsageMeta
}

type ChapterGenerator struct {
	factory *llm.EinoFactory
}

func NewChapterGenerator(factory *llm.EinoFactory) *ChapterGenerator {
	return &ChapterGenerator{factory: factory}
}

func (g *ChapterGenerator) Generate(ctx context.Context, in *ChapterGenerateInput) (*ChapterGenerateOutput, error) {
	if g == nil || g.factory == nil {
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

	ctx = einoobs.WithWorkflowProvider(ctx, "chapter_generate", strings.TrimSpace(in.Provider))

	chatModel, err := g.factory.Get(ctx, strings.TrimSpace(in.Provider))
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

	meta := LLMUsageMeta{
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

	return &ChapterGenerateOutput{
		Content: content,
		Meta:    meta,
	}, nil
}

// Stream 返回 Eino StreamReader；调用方负责 Close()。
// 约定：流可能在最后返回一个 Content 为空但包含 Usage 的消息，用于 Token 统计。
func (g *ChapterGenerator) Stream(ctx context.Context, in *ChapterGenerateInput) (*schema.StreamReader[*schema.Message], error) {
	if g == nil || g.factory == nil {
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

	ctx = einoobs.WithWorkflowProvider(ctx, "chapter_stream", strings.TrimSpace(in.Provider))

	chatModel, err := g.factory.Get(ctx, strings.TrimSpace(in.Provider))
	if err != nil {
		return nil, err
	}

	msgs, err := formatChapterMessages(ctx, in)
	if err != nil {
		return nil, err
	}
	return chatModel.Stream(ctx, msgs, buildChapterModelOptions(in)...)
}

func formatChapterMessages(ctx context.Context, in *ChapterGenerateInput) ([]*schema.Message, error) {
	tpl, err := defaultPromptRegistry.ChatTemplate(workflowprompt.PromptChapterGenV1)
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

func buildChapterModelOptions(in *ChapterGenerateInput) []model.Option {
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
