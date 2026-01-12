package chain

import (
	"context"
	"fmt"
	"strings"
	"sync"

	openaiopts "github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	llmctx "z-novel-ai-api/internal/domain/service"
	wfmodel "z-novel-ai-api/internal/workflow/model"
	wfnode "z-novel-ai-api/internal/workflow/node"
	workflowport "z-novel-ai-api/internal/workflow/port"
	workflowprompt "z-novel-ai-api/internal/workflow/prompt"
	"z-novel-ai-api/pkg/logger"
)

type FoundationChain struct {
	factory workflowport.ChatModelFactory

	chainOnce sync.Once
	chain     compose.Runnable[*wfmodel.FoundationGenerateInput, *schema.Message]
	chainErr  error
}

func NewFoundationChain(factory workflowport.ChatModelFactory) *FoundationChain {
	return &FoundationChain{factory: factory}
}

func (c *FoundationChain) Invoke(ctx context.Context, in *wfmodel.FoundationGenerateInput) (*schema.Message, error) {
	if c == nil || c.factory == nil {
		return nil, fmt.Errorf("llm factory not configured")
	}
	if in == nil {
		return nil, fmt.Errorf("input is nil")
	}

	chain, err := c.getChain()
	if err != nil {
		return nil, err
	}
	return chain.Invoke(ctx, in)
}

// Stream 返回 Eino StreamReader；调用方负责 Close()。
// 约定：流可能在最后返回一个 Content 为空但包含 Usage 的消息，用于 Token 统计。
func (c *FoundationChain) Stream(ctx context.Context, in *wfmodel.FoundationGenerateInput) (*schema.StreamReader[*schema.Message], error) {
	if c == nil || c.factory == nil {
		return nil, fmt.Errorf("llm factory not configured")
	}
	if in == nil {
		return nil, fmt.Errorf("input is nil")
	}

	ctx = llmctx.WithWorkflowProvider(ctx, "foundation_stream", strings.TrimSpace(in.Provider))
	chatModel, err := c.factory.Get(ctx, strings.TrimSpace(in.Provider))
	if err != nil {
		return nil, err
	}

	msgs, err := formatFoundationMessages(ctx, in)
	if err != nil {
		return nil, err
	}

	reader, err := chatModel.Stream(ctx, msgs, buildFoundationModelOptions(in, true)...)
	if err != nil && wfnode.IsResponseFormatUnsupportedError(err) {
		if reader != nil {
			reader.Close()
		}
		logger.Warn(ctx, "llm json_schema not supported for stream, fallback to prompt-only",
			"provider", strings.TrimSpace(in.Provider),
			"model", pickModel(in),
			"error", err.Error(),
		)
		return chatModel.Stream(ctx, msgs, buildFoundationModelOptions(in, false)...)
	}
	return reader, err
}

type foundationChainState struct {
	In       *wfmodel.FoundationGenerateInput
	Messages []*schema.Message
	OutMsg   *schema.Message
}

func (c *FoundationChain) getChain() (compose.Runnable[*wfmodel.FoundationGenerateInput, *schema.Message], error) {
	c.chainOnce.Do(func() {
		c.chain, c.chainErr = c.buildChain(context.Background())
	})
	return c.chain, c.chainErr
}

func (c *FoundationChain) buildChain(ctx context.Context) (compose.Runnable[*wfmodel.FoundationGenerateInput, *schema.Message], error) {
	chain := compose.NewChain[*wfmodel.FoundationGenerateInput, *schema.Message]()

	chain.AppendLambda(
		compose.InvokableLambda(func(_ context.Context, in *wfmodel.FoundationGenerateInput) (*foundationChainState, error) {
			if in == nil {
				return nil, fmt.Errorf("input is nil")
			}
			return &foundationChainState{In: in}, nil
		}),
		compose.WithNodeName("foundation.init"),
	)

	chain.AppendLambda(
		compose.InvokableLambda(func(ctx context.Context, st *foundationChainState) (*foundationChainState, error) {
			if st == nil || st.In == nil {
				return nil, fmt.Errorf("state is nil")
			}
			msgs, err := formatFoundationMessages(ctx, st.In)
			if err != nil {
				return nil, err
			}
			st.Messages = msgs
			return st, nil
		}),
		compose.WithNodeName("foundation.template"),
	)

	chain.AppendLambda(
		compose.InvokableLambda(func(ctx context.Context, st *foundationChainState) (*foundationChainState, error) {
			if st == nil || st.In == nil {
				return nil, fmt.Errorf("state is nil")
			}
			if c.factory == nil {
				return nil, fmt.Errorf("llm factory not configured")
			}

			ctx = llmctx.WithWorkflowProvider(ctx, "foundation_generate", strings.TrimSpace(st.In.Provider))
			chatModel, err := c.factory.Get(ctx, strings.TrimSpace(st.In.Provider))
			if err != nil {
				return nil, err
			}

			outMsg, err := chatModel.Generate(ctx, st.Messages, buildFoundationModelOptions(st.In, true)...)
			if err != nil && wfnode.IsResponseFormatUnsupportedError(err) {
				logger.Warn(ctx, "llm json_schema not supported, fallback to prompt-only",
					"provider", strings.TrimSpace(st.In.Provider),
					"model", pickModel(st.In),
					"error", err.Error(),
				)
				outMsg, err = chatModel.Generate(ctx, st.Messages, buildFoundationModelOptions(st.In, false)...)
			}
			if err != nil {
				return nil, err
			}
			if outMsg == nil {
				return nil, fmt.Errorf("empty llm response")
			}
			st.OutMsg = outMsg
			return st, nil
		}),
		compose.WithNodeName("foundation.llm"),
	)

	chain.AppendLambda(
		compose.InvokableLambda(func(_ context.Context, st *foundationChainState) (*schema.Message, error) {
			if st == nil || st.OutMsg == nil {
				return nil, fmt.Errorf("state is nil")
			}
			return st.OutMsg, nil
		}),
		compose.WithNodeName("foundation.finalize"),
	)

	return chain.Compile(ctx)
}

var defaultPromptRegistry = workflowprompt.NewRegistry()

func formatFoundationMessages(ctx context.Context, in *wfmodel.FoundationGenerateInput) ([]*schema.Message, error) {
	tpl, err := defaultPromptRegistry.ChatTemplate(workflowprompt.PromptFoundationPlanV1)
	if err != nil {
		return nil, err
	}
	vars := map[string]any{
		"project_title":       strings.TrimSpace(in.ProjectTitle),
		"project_description": strings.TrimSpace(in.ProjectDescription),
		"prompt":              strings.TrimSpace(in.Prompt),
		"attachments_block":   wfnode.BuildAttachmentsBlock(in.Attachments),
	}
	return tpl.Format(ctx, vars)
}

func buildFoundationModelOptions(in *wfmodel.FoundationGenerateInput, enableSchema bool) []model.Option {
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
		m := strings.TrimSpace(in.Model)
		opts = append(opts, model.WithModel(m))
	}

	if enableSchema {
		opts = append(opts, openaiopts.WithExtraFields(map[string]any{
			"response_format": map[string]any{
				"type": "json_schema",
				"json_schema": map[string]any{
					"name":   "foundation_plan",
					"strict": false,
					"schema": foundationJSONSchema(),
				},
			},
		}))
	}

	return opts
}

func pickModel(in *wfmodel.FoundationGenerateInput) string {
	if in == nil {
		return ""
	}
	if strings.TrimSpace(in.Model) != "" {
		return strings.TrimSpace(in.Model)
	}
	return ""
}

func foundationJSONSchema() map[string]any {
	// 说明：此处 schema 以“最小可用”为目标，避免过度约束导致模型输出失败。
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []any{"version", "project", "entities", "relations", "volumes"},
		"properties": map[string]any{
			"version": map[string]any{"type": "integer"},
			"project": map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required":             []any{"world_settings"},
				"properties": map[string]any{
					"genre":             map[string]any{"type": "string"},
					"target_word_count": map[string]any{"type": "integer"},
					"writing_style":     map[string]any{"type": "string"},
					"pov":               map[string]any{"type": "string"},
					"temperature":       map[string]any{"type": "number"},
					"world_bible":       map[string]any{"type": "string"},
					"world_settings": map[string]any{
						"type":                 "object",
						"additionalProperties": true,
						"properties": map[string]any{
							"key_settings": map[string]any{"type": "string"},
						},
					},
				},
			},
			"entities": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type":                 "object",
					"additionalProperties": false,
					"required":             []any{"key", "name", "type"},
					"properties": map[string]any{
						"key":         map[string]any{"type": "string"},
						"name":        map[string]any{"type": "string"},
						"type":        map[string]any{"type": "string"},
						"importance":  map[string]any{"type": "string"},
						"description": map[string]any{"type": "string"},
						"aliases":     map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
						"attributes": map[string]any{
							"type":                 "object",
							"additionalProperties": true,
							"properties": map[string]any{
								"note": map[string]any{"type": "string"},
							},
						},
						"current_state": map[string]any{"type": "string"},
					},
				},
			},
			"relations": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type":                 "object",
					"additionalProperties": false,
					"required":             []any{"source_key", "target_key", "relation_type"},
					"properties": map[string]any{
						"source_key":    map[string]any{"type": "string"},
						"target_key":    map[string]any{"type": "string"},
						"relation_type": map[string]any{"type": "string"},
						"strength":      map[string]any{"type": "number"},
						"description":   map[string]any{"type": "string"},
						"attributes": map[string]any{
							"type":                 "object",
							"additionalProperties": true,
							"properties": map[string]any{
								"note": map[string]any{"type": "string"},
							},
						},
					},
				},
			},
			"volumes": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type":                 "object",
					"additionalProperties": false,
					"required":             []any{"key", "title", "chapters"},
					"properties": map[string]any{
						"key":     map[string]any{"type": "string"},
						"title":   map[string]any{"type": "string"},
						"summary": map[string]any{"type": "string"},
						"chapters": map[string]any{
							"type": "array",
							"items": map[string]any{
								"type":                 "object",
								"additionalProperties": false,
								"required":             []any{"key", "title", "outline"},
								"properties": map[string]any{
									"key":               map[string]any{"type": "string"},
									"title":             map[string]any{"type": "string"},
									"outline":           map[string]any{"type": "string"},
									"target_word_count": map[string]any{"type": "integer"},
									"story_time_start":  map[string]any{"type": "integer"},
								},
							},
						},
					},
				},
			},
		},
	}
}
