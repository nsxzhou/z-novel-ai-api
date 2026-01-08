package story

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	openaiopts "github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"z-novel-ai-api/internal/domain/entity"
	"z-novel-ai-api/internal/infrastructure/llm"
	einoobs "z-novel-ai-api/internal/observability/eino"
	workflowprompt "z-novel-ai-api/internal/workflow/prompt"
	"z-novel-ai-api/pkg/logger"
)

const DefaultMaxToolRounds = 4

type ArtifactGenerateInput struct {
	ProjectTitle       string
	ProjectDescription string

	Type entity.ArtifactType

	Prompt      string
	Attachments []TextAttachment

	CurrentWorldview   json.RawMessage
	CurrentCharacters  json.RawMessage
	CurrentOutline     json.RawMessage
	CurrentArtifactRaw json.RawMessage

	Provider string
	Model    string

	Temperature *float32
	MaxTokens   *int
}

type ArtifactGenerateOutput struct {
	Type    entity.ArtifactType
	Content json.RawMessage
	Raw     string
	Meta    LLMUsageMeta
}

type ArtifactGenerator struct {
	factory *llm.EinoFactory

	graphOnce sync.Once
	graph     compose.Runnable[*ArtifactGenerateInput, *ArtifactGenerateOutput]
	graphErr  error

	toolsNodeOnce sync.Once
	toolsNode     *compose.ToolsNode
	toolsNodeErr  error
}

func NewArtifactGenerator(factory *llm.EinoFactory) *ArtifactGenerator {
	return &ArtifactGenerator{factory: factory}
}

func (g *ArtifactGenerator) Generate(ctx context.Context, in *ArtifactGenerateInput) (*ArtifactGenerateOutput, error) {
	if g == nil || g.factory == nil {
		return nil, fmt.Errorf("llm factory not configured")
	}
	if in == nil {
		return nil, fmt.Errorf("input is nil")
	}

	graph, err := g.getGraph()
	if err != nil {
		return nil, err
	}
	return graph.Invoke(ctx, in, compose.WithRuntimeMaxSteps(20))
}

func formatArtifactMessages(ctx context.Context, in *ArtifactGenerateInput) ([]*schema.Message, error) {
	tpl, err := defaultPromptRegistry.ChatTemplate(workflowprompt.PromptArtifactV1)
	if err != nil {
		return nil, err
	}

	currentHint := ""
	if len(in.CurrentArtifactRaw) > 0 {
		currentHint = "当前任务对应构件已存在；更新时请先调用 `artifact_get_active` 获取当前 JSON，并保持已有 key 不变（仅新增对象创建新 key）。"
	}

	vars := map[string]any{
		"project_title":       strings.TrimSpace(in.ProjectTitle),
		"project_description": strings.TrimSpace(in.ProjectDescription),
		"artifact_type":       strings.TrimSpace(string(in.Type)),
		"prompt":              strings.TrimSpace(in.Prompt),
		"attachments_block":   buildAttachmentsBlock(in.Attachments),
		"current_hint":        currentHint,
	}
	return tpl.Format(ctx, vars)
}

type artifactReActState struct {
	In            *ArtifactGenerateInput
	BaseModel     model.BaseChatModel
	ChatModel     model.BaseChatModel
	Messages      []*schema.Message
	LastAssistant *schema.Message

	Tools         []einotool.BaseTool
	ToolInfos     []*schema.ToolInfo
	ToolRounds    int
	MaxToolRounds int
}

func (g *ArtifactGenerator) getGraph() (compose.Runnable[*ArtifactGenerateInput, *ArtifactGenerateOutput], error) {
	g.graphOnce.Do(func() {
		g.graph, g.graphErr = g.buildGraph(context.Background())
	})
	return g.graph, g.graphErr
}

// getToolsNode 获取（懒加载）Eino 标准工具执行节点
// ToolsNode 是 Eino 框架提供的一个预置组件，专门用于解析 LLM 返回的 tool_calls，
// 并自动调用对应的工具函数。
func (g *ArtifactGenerator) getToolsNode() (*compose.ToolsNode, error) {
	// 使用 sync.Once 确保全局只初始化一次 ToolsNode 实例
	g.toolsNodeOnce.Do(func() {
		// 创建一个新的 ToolsNode
		g.toolsNode, g.toolsNodeErr = compose.NewToolNode(context.Background(), &compose.ToolsNodeConfig{
			// Tools 在这里设为 nil，因为具体的工具列表是动态的（根据请求不同而不同），
			// 我们会在 invoke 时通过 compose.WithToolList(...) 传入具体的工具集。
			Tools: nil,

			// 设为 true 表示按顺序执行多个工具调用，这通常更安全，避免并发写入或依赖问题。
			ExecuteSequentially: true,

			// 自定义未知工具处理器
			// 当 LLM 产生幻觉，调用了一个不在列表中的工具时，返回一个友好的 JSON 错误提示，
			// 而不是直接抛出 panic 或让流程崩溃。这样 LLM 可以在下一轮对话中看到错误并自我修正。
			UnknownToolsHandler: func(_ context.Context, name, _ string) (string, error) {
				b, _ := json.Marshal(map[string]any{
					"error": fmt.Sprintf("unknown tool: %s", strings.TrimSpace(name)),
				})
				return string(b), nil
			},
		})
	})
	return g.toolsNode, g.toolsNodeErr
}

// buildGraph 构建 Eino 处理图（ReAct 循环）：Init -> Model <-> Tools -> Finalize
// 该图负责执行复杂的生成任务，支持模型根据需要调用工具（如检索、查询），
// 并在多轮交互后最终生成符合格式要求的 Artifact（设定集/正文等）。
func (g *ArtifactGenerator) buildGraph(ctx context.Context) (compose.Runnable[*ArtifactGenerateInput, *ArtifactGenerateOutput], error) {
	graph := compose.NewGraph[*ArtifactGenerateInput, *ArtifactGenerateOutput]()

	// 预加载 Tools 节点（Eino 提供的标准工具执行组件）
	toolsNode, err := g.getToolsNode()
	if err != nil {
		return nil, err
	}

	// ---------------------------------------------------------------------
	// 1. Init: 初始化状态与工具集
	// ---------------------------------------------------------------------
	// 作用：
	//    1. 格式化 Prompt 消息。
	//    2. 初始化可用的工具列表 (Tool Set)，如搜索、查询项目简报等。
	//    3. 绑定工具到 ChatModel：如果模型支持工具调用 (Function Calling)，将工具信息注入模型配置。
	//    4. 创建 artifactReActState 状态对象，作为图在节点间传递的上下文。
	if err := graph.AddLambdaNode("init", compose.InvokableLambda(func(ctx context.Context, in *ArtifactGenerateInput) (*artifactReActState, error) {
		if in == nil {
			return nil, fmt.Errorf("input is nil")
		}
		if g == nil || g.factory == nil {
			return nil, fmt.Errorf("llm factory not configured")
		}

		msgs, err := formatArtifactMessages(ctx, in)
		if err != nil {
			return nil, err
		}

		ctx = einoobs.WithWorkflowProvider(ctx, "artifact_generate", in.Provider)
		baseModel, err := g.factory.Get(ctx, in.Provider)
		if err != nil {
			return nil, err
		}

		// 定义该任务可用的工具列表
		tools := []einotool.BaseTool{
			newArtifactGetActiveTool(in), // 获取当前正在编辑的 Artifact 内容
			newArtifactSearchTool(in),    // 语义搜索（RAG）
			newProjectGetBriefTool(in),   // 获取项目摘要信息
		}

		// 提取工具元数据 (Schema)
		toolInfos := make([]*schema.ToolInfo, 0, len(tools))
		for i := range tools {
			info, err := tools[i].Info(ctx)
			if err != nil {
				return nil, err
			}
			toolInfos = append(toolInfos, info)
		}

		// 如果模型支持，绑定工具信息
		chatModel := baseModel
		if tcm, ok := baseModel.(model.ToolCallingChatModel); ok {
			withTools, err := tcm.WithTools(toolInfos)
			if err == nil && withTools != nil {
				chatModel = withTools
			}
		}

		return &artifactReActState{
			In:            in,
			BaseModel:     baseModel,
			ChatModel:     chatModel,
			Messages:      msgs,
			Tools:         tools,
			ToolInfos:     toolInfos,
			MaxToolRounds: DefaultMaxToolRounds, // 防止死循环的最大轮数限制
		}, nil
	}), compose.WithNodeName("artifact.init")); err != nil {
		return nil, err
	}

	// ---------------------------------------------------------------------
	// 2. Model: 模型推理节点
	// ---------------------------------------------------------------------
	// 作用：执行 LLM 调用。
	// 核心逻辑与降级策略：
	//    1. 优先尝试：使用带工具绑定 (WithTools) 且要求 JSON Schema (Structured Output) 的配置调用模型。
	//    2. 降级策略 A (工具不支持)：如果 Provider 报错不支持工具，回退到基础模型 (BaseModel) 重试。
	//    3. 降级策略 B (Schema 不支持)：如果 Provider 报错不支持 JSON Schema，回退到普通 Prompt 模式重试。
	// 输出：更新状态中的 Messages 列表（追加 Assistant 的回复）。
	if err := graph.AddLambdaNode("model", compose.InvokableLambda(func(ctx context.Context, st *artifactReActState) (*artifactReActState, error) {
		if st == nil || st.In == nil || st.ChatModel == nil {
			return nil, fmt.Errorf("state is nil")
		}
		ctx = einoobs.WithWorkflowProvider(ctx, "artifact_generate", st.In.Provider)

		// 尝试生成
		outMsg, err := st.ChatModel.Generate(ctx, st.Messages, buildArtifactModelOptions(st.In, true)...)

		// 降级 A: 如果模型不支持工具调用，回退到不带工具的基础模型
		if err != nil && isToolsUnsupportedError(err) && st.BaseModel != nil && st.ChatModel != st.BaseModel {
			logger.Warn(ctx, "llm tools not supported, fallback to no-tools",
				"provider", st.In.Provider,
				"model", pickArtifactModel(st.In),
				"artifact_type", string(st.In.Type),
				"error", err.Error(),
			)
			st.ChatModel = st.BaseModel
			outMsg, err = st.ChatModel.Generate(ctx, st.Messages, buildArtifactModelOptions(st.In, true)...)
		}

		// 降级 B: 如果模型不支持 JSON Schema，回退到普通模式
		if err != nil && isResponseFormatUnsupportedError(err) {
			logger.Warn(ctx, "llm json_schema not supported, fallback to prompt-only",
				"provider", st.In.Provider,
				"model", pickArtifactModel(st.In),
				"artifact_type", string(st.In.Type),
				"error", err.Error(),
			)
			outMsg, err = st.ChatModel.Generate(ctx, st.Messages, buildArtifactModelOptions(st.In, false)...)
		}
		if err != nil {
			return nil, err
		}
		if outMsg == nil {
			return nil, fmt.Errorf("empty llm response")
		}

		st.LastAssistant = outMsg
		st.Messages = append(st.Messages, outMsg)
		return st, nil
	}), compose.WithNodeName("artifact.model")); err != nil {
		return nil, err
	}

	// ---------------------------------------------------------------------
	// 3. Tools: 工具执行节点
	// ---------------------------------------------------------------------
	// 作用：当 LLM 决定调用工具时（返回 ToolCalls），执行该节点。
	// 逻辑：
	//    1. 使用 Eino 标准的 ToolsNode 来解析并执行工具调用。
	//    2. 将工具执行结果 (ToolMessage) 追加到 Messages 列表中。
	//    3. 增加轮数计数器 (ToolRounds) 以防止无限循环。
	if err := graph.AddLambdaNode("tools", compose.InvokableLambda(func(ctx context.Context, st *artifactReActState) (*artifactReActState, error) {
		if st == nil || st.LastAssistant == nil {
			return nil, fmt.Errorf("state is nil")
		}
		if len(st.LastAssistant.ToolCalls) == 0 {
			return st, nil
		}
		if st.ToolRounds >= st.MaxToolRounds {
			return nil, fmt.Errorf("too many tool rounds")
		}

		ctx = einoobs.WithWorkflowProvider(ctx, "artifact_generate", st.In.Provider)
		outMsgs, err := toolsNode.Invoke(ctx, st.LastAssistant, compose.WithToolList(st.Tools...))
		if err != nil {
			return nil, err
		}
		st.Messages = append(st.Messages, outMsgs...)
		st.ToolRounds++
		return st, nil
	}), compose.WithNodeName("artifact.tools")); err != nil {
		return nil, err
	}

	// ---------------------------------------------------------------------
	// 4. Finalize: 结果处理节点
	// ---------------------------------------------------------------------
	// 作用：当 LLM 不再调用工具，而是返回最终文本内容时，执行此节点。
	// 逻辑：
	//    1. 提取 JSON 内容。
	//    2. 校验并规范化生成的 Artifact 内容 (normalizeAndValidateArtifact)。
	//    3. 封装最终输出，包含元数据。
	if err := graph.AddLambdaNode("finalize", compose.InvokableLambda(func(ctx context.Context, st *artifactReActState) (*ArtifactGenerateOutput, error) {
		if st == nil || st.In == nil || st.LastAssistant == nil {
			return nil, fmt.Errorf("state is nil")
		}

		rawJSON := extractJSONObject(st.LastAssistant.Content)
		if strings.TrimSpace(rawJSON) == "" {
			return nil, fmt.Errorf("empty artifact output")
		}

		content, err := normalizeAndValidateArtifact(st.In.Type, rawJSON)
		if err != nil {
			return nil, err
		}

		meta := LLMUsageMeta{
			Provider:    st.In.Provider,
			Model:       pickArtifactModel(st.In),
			GeneratedAt: time.Now().UTC(),
		}
		if st.In.Temperature != nil {
			meta.Temperature = float64(*st.In.Temperature)
		}
		if st.LastAssistant.ResponseMeta != nil && st.LastAssistant.ResponseMeta.Usage != nil {
			meta.PromptTokens = st.LastAssistant.ResponseMeta.Usage.PromptTokens
			meta.CompletionTokens = st.LastAssistant.ResponseMeta.Usage.CompletionTokens
		}

		return &ArtifactGenerateOutput{
			Type:    st.In.Type,
			Content: content,
			Raw:     rawJSON,
			Meta:    meta,
		}, nil
	}), compose.WithNodeName("artifact.finalize")); err != nil {
		return nil, err
	}

	// ---------------------------------------------------------------------
	// 5. Edges & Branches: 定义图的流转逻辑
	// ---------------------------------------------------------------------
	// 流程：
	//   START -> init -> model
	//                     ↓
	//                   <分支判断>
	//                  /        \
	//         (有 ToolCalls)    (无 ToolCalls)
	//               ↓              ↓
	//             tools         finalize -> END
	//               ↓
	//             model (循环回模型)
	if err := graph.AddEdge(compose.START, "init"); err != nil {
		return nil, err
	}
	if err := graph.AddEdge("init", "model"); err != nil {
		return nil, err
	}

	branch := func(ctx context.Context, st *artifactReActState) (string, error) {
		if st == nil || st.LastAssistant == nil {
			return "finalize", nil
		}
		// 如果 LLM 想要调用工具，且未超过最大轮数 -> 进入 tools 节点
		if len(st.LastAssistant.ToolCalls) > 0 {
			if st.ToolRounds >= st.MaxToolRounds {
				return "", fmt.Errorf("too many tool rounds")
			}
			return "tools", nil
		}
		// 否则 -> 进入 finalize 节点结束
		return "finalize", nil
	}
	if err := graph.AddBranch("model", compose.NewGraphBranch(branch, map[string]bool{"tools": true, "finalize": true})); err != nil {
		return nil, err
	}
	// 工具执行完后，必须跳回模型节点，让模型看到工具结果并继续生成
	if err := graph.AddEdge("tools", "model"); err != nil {
		return nil, err
	}
	if err := graph.AddEdge("finalize", compose.END); err != nil {
		return nil, err
	}

	return graph.Compile(ctx, compose.WithGraphName("artifact_generate_graph"))
}

func buildArtifactModelOptions(in *ArtifactGenerateInput, enableSchema bool) []model.Option {
	opts := make([]model.Option, 0, 4)

	if in.Temperature != nil {
		opts = append(opts, model.WithTemperature(*in.Temperature))
	}
	if in.MaxTokens != nil {
		opts = append(opts, model.WithMaxTokens(*in.MaxTokens))
	}
	if strings.TrimSpace(in.Model) != "" {
		opts = append(opts, model.WithModel(strings.TrimSpace(in.Model)))
	}

	if enableSchema {
		schemaObj := artifactJSONSchema(in.Type)
		if schemaObj != nil {
			opts = append(opts, openaiopts.WithExtraFields(map[string]any{
				"response_format": map[string]any{
					"type": "json_schema",
					"json_schema": map[string]any{
						"name":   fmt.Sprintf("artifact_%s", in.Type),
						"strict": false,
						"schema": schemaObj,
					},
				},
			}))
		}
	}

	return opts
}

func pickArtifactModel(in *ArtifactGenerateInput) string {
	if in == nil {
		return ""
	}
	if strings.TrimSpace(in.Model) != "" {
		return strings.TrimSpace(in.Model)
	}
	return ""
}

func artifactJSONSchema(t entity.ArtifactType) map[string]any {
	switch t {
	case entity.ArtifactTypeNovelFoundation:
		return novelFoundationJSONSchema()
	case entity.ArtifactTypeWorldview:
		return worldviewJSONSchema()
	case entity.ArtifactTypeCharacters:
		return charactersJSONSchema()
	case entity.ArtifactTypeOutline:
		return outlineJSONSchema()
	default:
		return nil
	}
}

func novelFoundationJSONSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []any{"title", "description", "genre"},
		"properties": map[string]any{
			"title":       map[string]any{"type": "string"},
			"description": map[string]any{"type": "string"},
			"genre":       map[string]any{"type": "string"},
		},
	}
}

func worldviewJSONSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required": []any{
			"genre", "target_word_count", "writing_style", "pov", "temperature",
			"world_bible", "world_settings",
		},
		"properties": map[string]any{
			"genre":             map[string]any{"type": "string"},
			"target_word_count": map[string]any{"type": "integer"},
			"writing_style":     map[string]any{"type": "string"},
			"pov":               map[string]any{"type": "string"},
			"temperature":       map[string]any{"type": "number"},
			"world_bible":       map[string]any{"type": "string"},
			"world_settings": map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required":             []any{"time_system", "calendar", "locations"},
				"properties": map[string]any{
					"time_system": map[string]any{"type": "string"},
					"calendar":    map[string]any{"type": "string"},
					"locations":   map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
				},
			},
		},
	}
}

func charactersJSONSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []any{"entities", "relations"},
		"properties": map[string]any{
			"entities": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type":                 "object",
					"additionalProperties": false,
					"required": []any{
						"key", "name", "type", "importance", "description",
						"aliases", "attributes", "current_state",
					},
					"properties": map[string]any{
						"key":  map[string]any{"type": "string"},
						"name": map[string]any{"type": "string"},
						"type": map[string]any{
							"type": "string",
							"enum": []any{"character", "item", "location", "organization", "concept"},
						},
						"importance": map[string]any{
							"type": "string",
							"enum": []any{"protagonist", "major", "secondary", "minor"},
						},
						"description": map[string]any{"type": "string"},
						"aliases":     map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
						"attributes": map[string]any{
							"type":                 "object",
							"additionalProperties": false,
							"required":             []any{"age", "gender", "occupation", "personality", "abilities", "background"},
							"properties": map[string]any{
								"age":         map[string]any{"type": "integer"},
								"gender":      map[string]any{"type": "string"},
								"occupation":  map[string]any{"type": "string"},
								"personality": map[string]any{"type": "string"},
								"abilities":   map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
								"background":  map[string]any{"type": "string"},
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
					"required": []any{
						"source_key", "target_key", "relation_type",
						"strength", "description", "attributes",
					},
					"properties": map[string]any{
						"source_key": map[string]any{"type": "string"},
						"target_key": map[string]any{"type": "string"},
						"relation_type": map[string]any{
							"type": "string",
							"enum": []any{"friend", "enemy", "family", "lover", "subordinate", "mentor", "rival", "ally"},
						},
						"strength":    map[string]any{"type": "number"},
						"description": map[string]any{"type": "string"},
						"attributes": map[string]any{
							"type":                 "object",
							"additionalProperties": false,
							"required":             []any{"since", "origin", "development"},
							"properties": map[string]any{
								"since":       map[string]any{"type": "string"},
								"origin":      map[string]any{"type": "string"},
								"development": map[string]any{"type": "string"},
							},
						},
					},
				},
			},
		},
	}
}

func outlineJSONSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []any{"volumes"},
		"properties": map[string]any{
			"volumes": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type":                 "object",
					"additionalProperties": false,
					"required":             []any{"key", "title", "summary", "chapters"},
					"properties": map[string]any{
						"key":     map[string]any{"type": "string"},
						"title":   map[string]any{"type": "string"},
						"summary": map[string]any{"type": "string"},
						"chapters": map[string]any{
							"type": "array",
							"items": map[string]any{
								"type":                 "object",
								"additionalProperties": false,
								"required":             []any{"key", "title", "outline", "target_word_count", "story_time_start"},
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

func isToolsUnsupportedError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "unknown parameter") && strings.Contains(msg, "tools"):
		return true
	case strings.Contains(msg, "unknown parameter") && strings.Contains(msg, "tool"):
		return true
	case strings.Contains(msg, "tools") && strings.Contains(msg, "not supported"):
		return true
	case strings.Contains(msg, "tool") && strings.Contains(msg, "not supported"):
		return true
	default:
		return false
	}
}
