package story

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	openaiopts "github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	einoobs "z-novel-ai-api/internal/observability/eino"
	workflowprompt "z-novel-ai-api/internal/workflow/prompt"
	"z-novel-ai-api/pkg/logger"
)

type FoundationGenerateInput struct {
	ProjectTitle       string
	ProjectDescription string

	Prompt      string
	Attachments []TextAttachment

	Provider string
	Model    string

	Temperature *float32
	MaxTokens   *int
}

type TextAttachment struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

type LLMUsageMeta struct {
	Provider         string
	Model            string
	PromptTokens     int
	CompletionTokens int
	Temperature      float64
	GeneratedAt      time.Time
}

type FoundationGenerateOutput struct {
	Plan *FoundationPlan
	Raw  string
	Meta LLMUsageMeta
}

type FoundationGenerator struct {
	factory ChatModelFactory

	chainOnce sync.Once
	chain     compose.Runnable[*FoundationGenerateInput, *FoundationGenerateOutput]
	chainErr  error
}

func NewFoundationGenerator(factory ChatModelFactory) *FoundationGenerator {
	return &FoundationGenerator{factory: factory}
}

func (g *FoundationGenerator) Generate(ctx context.Context, in *FoundationGenerateInput) (*FoundationGenerateOutput, error) {
	if g.factory == nil {
		return nil, fmt.Errorf("llm factory not configured")
	}
	if in == nil {
		return nil, fmt.Errorf("input is nil")
	}

	chain, err := g.getChain()
	if err != nil {
		return nil, err
	}
	return chain.Invoke(ctx, in)
}

// Stream 返回 Eino StreamReader；调用方负责 Close()。
// 约定：流可能在最后返回一个 Content 为空但包含 Usage 的消息，用于 Token 统计。
func (g *FoundationGenerator) Stream(ctx context.Context, in *FoundationGenerateInput) (*schema.StreamReader[*schema.Message], error) {
	if g.factory == nil {
		return nil, fmt.Errorf("llm factory not configured")
	}
	if in == nil {
		return nil, fmt.Errorf("input is nil")
	}

	ctx = einoobs.WithWorkflowProvider(ctx, "foundation_stream", in.Provider)

	chatModel, err := g.factory.Get(ctx, in.Provider)
	if err != nil {
		return nil, err
	}

	msgs, err := formatFoundationMessages(ctx, in)
	if err != nil {
		return nil, err
	}
	reader, err := chatModel.Stream(ctx, msgs, buildFoundationModelOptions(in, true)...)
	if err != nil && isResponseFormatUnsupportedError(err) {
		if reader != nil {
			reader.Close()
		}
		logger.Warn(ctx, "llm json_schema not supported for stream, fallback to prompt-only",
			"provider", in.Provider,
			"model", pickModel(in),
			"error", err.Error(),
		)
		return chatModel.Stream(ctx, msgs, buildFoundationModelOptions(in, false)...)
	}
	return reader, err
}

func formatFoundationMessages(ctx context.Context, in *FoundationGenerateInput) ([]*schema.Message, error) {
	tpl, err := defaultPromptRegistry.ChatTemplate(workflowprompt.PromptFoundationPlanV1)
	if err != nil {
		return nil, err
	}
	vars := map[string]any{
		"project_title":       strings.TrimSpace(in.ProjectTitle),
		"project_description": strings.TrimSpace(in.ProjectDescription),
		"prompt":              strings.TrimSpace(in.Prompt),
		"attachments_block":   buildAttachmentsBlock(in.Attachments),
	}
	return tpl.Format(ctx, vars)
}

func buildFoundationModelOptions(in *FoundationGenerateInput, enableSchema bool) []model.Option {
	opts := make([]model.Option, 0, 4)

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
		// 优先使用 response_format=json_schema 强约束；失败时由调用方降级为“纯 Prompt 约束”。
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

func pickModel(in *FoundationGenerateInput) string {
	if strings.TrimSpace(in.Model) != "" {
		return strings.TrimSpace(in.Model)
	}
	return ""
}

type foundationChainState struct {
	In       *FoundationGenerateInput
	Messages []*schema.Message
	OutMsg   *schema.Message
}

func (g *FoundationGenerator) getChain() (compose.Runnable[*FoundationGenerateInput, *FoundationGenerateOutput], error) {
	g.chainOnce.Do(func() {
		g.chain, g.chainErr = g.buildChain(context.Background())
	})
	return g.chain, g.chainErr
}

func (g *FoundationGenerator) buildChain(ctx context.Context) (compose.Runnable[*FoundationGenerateInput, *FoundationGenerateOutput], error) {
	chain := compose.NewChain[*FoundationGenerateInput, *FoundationGenerateOutput]()

	chain.AppendLambda(
		compose.InvokableLambda(func(ctx context.Context, in *FoundationGenerateInput) (*foundationChainState, error) {
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
			if g == nil || g.factory == nil {
				return nil, fmt.Errorf("llm factory not configured")
			}

			ctx = einoobs.WithWorkflowProvider(ctx, "foundation_generate", st.In.Provider)

			chatModel, err := g.factory.Get(ctx, st.In.Provider)
			if err != nil {
				return nil, err
			}

			outMsg, err := chatModel.Generate(ctx, st.Messages, buildFoundationModelOptions(st.In, true)...)
			if err != nil && isResponseFormatUnsupportedError(err) {
				logger.Warn(ctx, "llm json_schema not supported, fallback to prompt-only",
					"provider", st.In.Provider,
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
		compose.InvokableLambda(func(ctx context.Context, st *foundationChainState) (*FoundationGenerateOutput, error) {
			if st == nil || st.In == nil || st.OutMsg == nil {
				return nil, fmt.Errorf("state is nil")
			}
			plan, raw, err := ParseFoundationPlan(st.OutMsg.Content)
			if err != nil {
				return nil, err
			}

			meta := LLMUsageMeta{
				Provider:    st.In.Provider,
				Model:       pickModel(st.In),
				GeneratedAt: time.Now().UTC(),
			}
			if st.In.Temperature != nil {
				meta.Temperature = float64(*st.In.Temperature)
			}
			if st.OutMsg.ResponseMeta != nil && st.OutMsg.ResponseMeta.Usage != nil {
				meta.PromptTokens = st.OutMsg.ResponseMeta.Usage.PromptTokens
				meta.CompletionTokens = st.OutMsg.ResponseMeta.Usage.CompletionTokens
			}

			return &FoundationGenerateOutput{
				Plan: plan,
				Raw:  raw,
				Meta: meta,
			}, nil
		}),
		compose.WithNodeName("foundation.finalize"),
	)

	return chain.Compile(ctx, compose.WithGraphName("foundation_generate_chain"))
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

func extractJSONObject(s string) string {
	raw := strings.TrimSpace(s)
	if raw == "" {
		return raw
	}

	// 如果模型输出夹杂了其它文本，尽量截取第一个 JSON 值（对象/数组）。
	objStart := strings.Index(raw, "{")
	arrStart := strings.Index(raw, "[")
	start := -1
	end := -1
	switch {
	case objStart >= 0 && (arrStart < 0 || objStart < arrStart):
		start = objStart
		end = strings.LastIndex(raw, "}")
	case arrStart >= 0:
		start = arrStart
		end = strings.LastIndex(raw, "]")
	}
	if start >= 0 && end > start {
		raw = raw[start : end+1]
	}

	// 简单校验：确保至少能被 Decoder 消费到一个 JSON 起始。
	dec := json.NewDecoder(strings.NewReader(raw))
	dec.UseNumber()
	tok, err := dec.Token()
	if err == nil {
		if d, ok := tok.(json.Delim); ok && (d == '{' || d == '[') {
			return raw
		}
	}

	// 最后兜底：尝试读取到 EOF 为止，避免调用方误用。
	dec = json.NewDecoder(strings.NewReader(raw))
	for {
		_, e := dec.Token()
		if e != nil {
			if errors.Is(e, io.EOF) {
				break
			}
			return strings.TrimSpace(s)
		}
	}
	return raw
}

func isResponseFormatUnsupportedError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "response_format"):
		return true
	case strings.Contains(msg, "json_schema"):
		return true
	case strings.Contains(msg, "unknown parameter") && strings.Contains(msg, "response"):
		return true
	case strings.Contains(msg, "invalid") && strings.Contains(msg, "response"):
		return true
	case strings.Contains(msg, "response_schema"):
		return true
	case strings.Contains(msg, "failed to parse"):
		return true
	default:
		return false
	}
}
