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

	appretrieval "z-novel-ai-api/internal/application/retrieval"
	"z-novel-ai-api/internal/domain/entity"
	einoobs "z-novel-ai-api/internal/observability/eino"
	workflowprompt "z-novel-ai-api/internal/workflow/prompt"
	"z-novel-ai-api/pkg/logger"
)

const DefaultMaxToolRounds = 4
const DefaultMaxRepairRounds = 2

type artifactOutputMode string

const (
	artifactOutputModeFull      artifactOutputMode = "full"
	artifactOutputModeJSONPatch artifactOutputMode = "json_patch"
)

type ArtifactGenerateInput struct {
	TenantID  string
	ProjectID string

	ProjectTitle       string
	ProjectDescription string

	Type entity.ArtifactType

	Prompt      string
	Attachments []TextAttachment

	ConversationSummary string
	RecentUserTurns     string

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
	Type     entity.ArtifactType
	Content  json.RawMessage
	Raw      string
	ModelRaw string
	Mode     string
	Meta     LLMUsageMeta
}

type ArtifactGenerator struct {
	factory         ChatModelFactory
	retrievalEngine *appretrieval.Engine

	graphOnce sync.Once
	graph     compose.Runnable[*ArtifactGenerateInput, *ArtifactGenerateOutput]
	graphErr  error

	toolsNodeOnce sync.Once
	toolsNode     *compose.ToolsNode
	toolsNodeErr  error
}

func NewArtifactGenerator(factory ChatModelFactory, retrievalEngine *appretrieval.Engine) *ArtifactGenerator {
	return &ArtifactGenerator{factory: factory, retrievalEngine: retrievalEngine}
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
	tpl, err := defaultPromptRegistry.ChatTemplate(workflowprompt.PromptArtifactV2)
	if err != nil {
		return nil, err
	}

	currentHint := ""
	if len(in.CurrentArtifactRaw) > 0 {
		currentHint = "当前任务对应构件已存在；更新时请先调用 `artifact_get_active` 获取当前 JSON，并保持已有 key 不变（仅新增对象创建新 key）。"
	}

	vars := map[string]any{
		"project_title":        strings.TrimSpace(in.ProjectTitle),
		"project_description":  strings.TrimSpace(in.ProjectDescription),
		"artifact_type":        strings.TrimSpace(string(in.Type)),
		"conversation_summary": strings.TrimSpace(in.ConversationSummary),
		"recent_user_turns":    strings.TrimSpace(in.RecentUserTurns),
		"prompt":               strings.TrimSpace(in.Prompt),
		"attachments_block":    buildAttachmentsBlock(in.Attachments),
		"current_hint":         currentHint,
	}
	return tpl.Format(ctx, vars)
}

func formatArtifactPatchMessages(ctx context.Context, in *ArtifactGenerateInput) ([]*schema.Message, error) {
	tpl, err := defaultPromptRegistry.ChatTemplate(workflowprompt.PromptArtifactPatchV1)
	if err != nil {
		return nil, err
	}

	current := strings.TrimSpace(string(in.CurrentArtifactRaw))
	if current == "" {
		current = "{}"
	}
	current = truncateByRunes(current, 60000)

	allowedOps := strings.Join(artifactJSONPatchAllowedOps(), ", ")
	allowedPaths := strings.Join(artifactJSONPatchAllowedPaths(in.Type), ", ")

	vars := map[string]any{
		"project_title":         strings.TrimSpace(in.ProjectTitle),
		"project_description":   strings.TrimSpace(in.ProjectDescription),
		"artifact_type":         strings.TrimSpace(string(in.Type)),
		"current_artifact_json": current,
		"conversation_summary":  strings.TrimSpace(in.ConversationSummary),
		"recent_user_turns":     strings.TrimSpace(in.RecentUserTurns),
		"prompt":                strings.TrimSpace(in.Prompt),
		"attachments_block":     buildAttachmentsBlock(in.Attachments),
		"allowed_ops":           allowedOps,
		"allowed_paths":         allowedPaths,
	}
	return tpl.Format(ctx, vars)
}

func cloneMessages(msgs []*schema.Message) []*schema.Message {
	if len(msgs) == 0 {
		return nil
	}
	out := make([]*schema.Message, len(msgs))
	copy(out, msgs)
	return out
}

type artifactReActState struct {
	In            *ArtifactGenerateInput
	BaseModel     model.BaseChatModel
	ChatModel     model.BaseChatModel
	Messages      []*schema.Message
	LastAssistant *schema.Message
	FullMessages  []*schema.Message
	PatchMessages []*schema.Message

	Tools         []einotool.BaseTool
	ToolInfos     []*schema.ToolInfo
	ToolRounds    int
	MaxToolRounds int

	Mode         artifactOutputMode
	FallbackUsed bool

	ValidatedContent json.RawMessage
	LastRawJSON      string
	ValidateErr      error

	RepairRounds    int
	MaxRepairRounds int
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

		fullMsgs, err := formatArtifactMessages(ctx, in)
		if err != nil {
			return nil, err
		}

		mode := artifactOutputModeFull
		var patchMsgs []*schema.Message
		if isArtifactJSONPatchEnabled(in) {
			patchMsgs, err = formatArtifactPatchMessages(ctx, in)
			if err != nil {
				return nil, err
			}
			mode = artifactOutputModeJSONPatch
		}

		msgs := cloneMessages(fullMsgs)
		if mode == artifactOutputModeJSONPatch && len(patchMsgs) > 0 {
			msgs = cloneMessages(patchMsgs)
		}

		ctx = einoobs.WithWorkflowProvider(ctx, "artifact_generate", in.Provider)
		baseModel, err := g.factory.Get(ctx, in.Provider)
		if err != nil {
			return nil, err
		}

		// 定义该任务可用的工具列表
		tools := []einotool.BaseTool{
			newArtifactGetActiveTool(in),                 // 获取当前正在编辑的 Artifact 内容
			newArtifactSearchTool(g.retrievalEngine, in), // 语义搜索（RAG）
			newProjectGetBriefTool(in),                   // 获取项目摘要信息
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
			In:              in,
			BaseModel:       baseModel,
			ChatModel:       chatModel,
			Messages:        msgs,
			FullMessages:    fullMsgs,
			PatchMessages:   patchMsgs,
			Tools:           tools,
			ToolInfos:       toolInfos,
			MaxToolRounds:   DefaultMaxToolRounds, // 防止死循环的最大轮数限制
			MaxRepairRounds: DefaultMaxRepairRounds,
			Mode:            mode,
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
		outMsg, err := st.ChatModel.Generate(ctx, st.Messages, buildArtifactModelOptions(st.In, true, st.Mode)...)

		// 降级 A: 如果模型不支持工具调用，回退到不带工具的基础模型
		if err != nil && isToolsUnsupportedError(err) && st.BaseModel != nil && st.ChatModel != st.BaseModel {
			logger.Warn(ctx, "llm tools not supported, fallback to no-tools",
				"provider", st.In.Provider,
				"model", pickArtifactModel(st.In),
				"artifact_type", string(st.In.Type),
				"error", err.Error(),
			)
			st.ChatModel = st.BaseModel
			outMsg, err = st.ChatModel.Generate(ctx, st.Messages, buildArtifactModelOptions(st.In, true, st.Mode)...)
		}

		// 降级 B: 如果模型不支持 JSON Schema，回退到普通模式
		if err != nil && isResponseFormatUnsupportedError(err) {
			logger.Warn(ctx, "llm json_schema not supported, fallback to prompt-only",
				"provider", st.In.Provider,
				"model", pickArtifactModel(st.In),
				"artifact_type", string(st.In.Type),
				"error", err.Error(),
			)
			outMsg, err = st.ChatModel.Generate(ctx, st.Messages, buildArtifactModelOptions(st.In, false, st.Mode)...)
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
	// 4. Validate: 解析与校验节点
	// ---------------------------------------------------------------------
	// 作用：当 LLM 不再调用工具，而是返回最终文本内容时，执行此节点。
	// 逻辑：
	//    1. 提取 JSON 内容。
	//    2. 校验并规范化生成的 Artifact 内容 (normalizeAndValidateArtifact)。
	//    3. 将结果写入状态，供 Repair / Finalize 使用。
	if err := graph.AddLambdaNode("validate", compose.InvokableLambda(func(ctx context.Context, st *artifactReActState) (*artifactReActState, error) {
		if st == nil || st.In == nil || st.LastAssistant == nil {
			return nil, fmt.Errorf("state is nil")
		}

		st.ValidateErr = nil
		st.ValidatedContent = nil
		st.LastRawJSON = ""

		rawJSON := strings.TrimSpace(extractJSONObject(st.LastAssistant.Content))
		st.LastRawJSON = rawJSON
		if rawJSON == "" {
			st.ValidateErr = fmt.Errorf("empty artifact output")
			return st, nil
		}

		switch st.Mode {
		case artifactOutputModeJSONPatch:
			patched, err := applyArtifactJSONPatch(st.In.Type, st.In.CurrentArtifactRaw, rawJSON)
			if err != nil {
				st.ValidateErr = err
				return st, nil
			}
			content, err := normalizeAndValidateArtifact(st.In.Type, strings.TrimSpace(string(patched)))
			if err != nil {
				st.ValidateErr = err
				return st, nil
			}
			st.ValidatedContent = content

		default:
			content, err := normalizeAndValidateArtifact(st.In.Type, rawJSON)
			if err != nil {
				st.ValidateErr = err
				return st, nil
			}
			st.ValidatedContent = content
		}
		return st, nil
	}), compose.WithNodeName("artifact.validate")); err != nil {
		return nil, err
	}

	// ---------------------------------------------------------------------
	// 5. Repair: 校验失败修复节点（Validate → Repair → Re-run）
	// ---------------------------------------------------------------------
	// 作用：当解析/校验失败时，向 Messages 追加修复指令并回到 model 重试。
	// 约束：最多修复 MaxRepairRounds 次，避免死循环与成本失控。
	if err := graph.AddLambdaNode("repair", compose.InvokableLambda(func(ctx context.Context, st *artifactReActState) (*artifactReActState, error) {
		if st == nil || st.In == nil || st.LastAssistant == nil {
			return nil, fmt.Errorf("state is nil")
		}
		if st.ValidateErr == nil {
			return st, nil
		}
		if st.RepairRounds >= st.MaxRepairRounds {
			return nil, st.ValidateErr
		}

		repairMsg := buildArtifactRepairMessage(st.Mode, st.In.Type, st.ValidateErr, st.LastRawJSON)
		st.Messages = append(st.Messages, schema.UserMessage(repairMsg))
		st.RepairRounds++
		return st, nil
	}), compose.WithNodeName("artifact.repair")); err != nil {
		return nil, err
	}

	// ---------------------------------------------------------------------
	// 6. Finalize: 结果封装节点
	// ---------------------------------------------------------------------
	if err := graph.AddLambdaNode("finalize", compose.InvokableLambda(func(ctx context.Context, st *artifactReActState) (*ArtifactGenerateOutput, error) {
		if st == nil || st.In == nil || st.LastAssistant == nil {
			return nil, fmt.Errorf("state is nil")
		}
		if st.ValidateErr != nil {
			return nil, st.ValidateErr
		}
		if len(st.ValidatedContent) == 0 {
			return nil, fmt.Errorf("empty validated content")
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

		modelRaw := strings.TrimSpace(st.LastRawJSON)
		raw := modelRaw
		if st.Mode == artifactOutputModeJSONPatch {
			raw = strings.TrimSpace(string(st.ValidatedContent))
		}
		if strings.TrimSpace(raw) == "" {
			raw = strings.TrimSpace(string(st.ValidatedContent))
		}
		if strings.TrimSpace(modelRaw) == "" {
			modelRaw = raw
		}

		return &ArtifactGenerateOutput{
			Type:     st.In.Type,
			Content:  st.ValidatedContent,
			Raw:      raw,
			ModelRaw: modelRaw,
			Mode:     string(st.Mode),
			Meta:     meta,
		}, nil
	}), compose.WithNodeName("artifact.finalize")); err != nil {
		return nil, err
	}

	// ---------------------------------------------------------------------
	// 7. Edges & Branches: 定义图的流转逻辑
	// ---------------------------------------------------------------------
	// 流程：
	//   START -> init -> model
	//                     ↓
	//                   <分支判断>
	//                  /        \
	//         (有 ToolCalls)    (无 ToolCalls)
	//               ↓              ↓
	//             tools         validate -> <repair?> -> finalize -> END
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
			return "validate", nil
		}
		// 如果 LLM 想要调用工具，且未超过最大轮数 -> 进入 tools 节点
		if len(st.LastAssistant.ToolCalls) > 0 {
			if st.ToolRounds >= st.MaxToolRounds {
				return "", fmt.Errorf("too many tool rounds")
			}
			return "tools", nil
		}
		// 否则 -> 进入 validate 节点
		return "validate", nil
	}
	if err := graph.AddBranch("model", compose.NewGraphBranch(branch, map[string]bool{"tools": true, "validate": true})); err != nil {
		return nil, err
	}
	// 工具执行完后，必须跳回模型节点，让模型看到工具结果并继续生成
	if err := graph.AddEdge("tools", "model"); err != nil {
		return nil, err
	}

	validateBranch := func(ctx context.Context, st *artifactReActState) (string, error) {
		if st == nil {
			return "", fmt.Errorf("state is nil")
		}
		if st.ValidateErr == nil {
			return "finalize", nil
		}
		// Patch 模式修复耗尽后，自动回退到“全量 JSON 输出”再尝试一次，避免增量模式放大失败率。
		if st.Mode == artifactOutputModeJSONPatch && !st.FallbackUsed && st.RepairRounds >= st.MaxRepairRounds && len(st.FullMessages) > 0 {
			st.FallbackUsed = true
			st.Mode = artifactOutputModeFull
			st.Messages = cloneMessages(st.FullMessages)
			st.RepairRounds = 0
			st.ValidateErr = nil
			st.ValidatedContent = nil
			st.LastRawJSON = ""
			return "model", nil
		}

		if st.RepairRounds >= st.MaxRepairRounds {
			return "", st.ValidateErr
		}
		return "repair", nil
	}
	if err := graph.AddBranch("validate", compose.NewGraphBranch(validateBranch, map[string]bool{"repair": true, "finalize": true, "model": true})); err != nil {
		return nil, err
	}
	if err := graph.AddEdge("repair", "model"); err != nil {
		return nil, err
	}
	if err := graph.AddEdge("finalize", compose.END); err != nil {
		return nil, err
	}

	return graph.Compile(ctx, compose.WithGraphName("artifact_generate_graph"))
}

func buildArtifactModelOptions(in *ArtifactGenerateInput, enableSchema bool, mode artifactOutputMode) []model.Option {
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
		schemaObj := artifactJSONSchemaForMode(in.Type, mode)
		if schemaObj != nil {
			name := fmt.Sprintf("artifact_%s", in.Type)
			if mode == artifactOutputModeJSONPatch {
				name = fmt.Sprintf("artifact_patch_%s", in.Type)
			}
			opts = append(opts, openaiopts.WithExtraFields(map[string]any{
				"response_format": map[string]any{
					"type": "json_schema",
					"json_schema": map[string]any{
						"name":   name,
						"strict": false,
						"schema": schemaObj,
					},
				},
			}))
		}
	}

	return opts
}

func artifactJSONSchemaForMode(t entity.ArtifactType, mode artifactOutputMode) map[string]any {
	if mode == artifactOutputModeJSONPatch {
		return artifactJSONPatchSchema(t)
	}
	return artifactJSONSchema(t)
}

func artifactJSONPatchSchema(t entity.ArtifactType) map[string]any {
	paths := artifactJSONPatchAllowedPaths(t)
	if len(paths) == 0 {
		return nil
	}

	enumPaths := make([]any, 0, len(paths))
	for i := range paths {
		enumPaths = append(enumPaths, paths[i])
	}

	return map[string]any{
		"type": "array",
		"items": map[string]any{
			"type":                 "object",
			"additionalProperties": false,
			"required":             []any{"op", "path", "value"},
			"properties": map[string]any{
				"op": map[string]any{
					"type": "string",
					"enum": []any{"add", "replace"},
				},
				"path": map[string]any{
					"type": "string",
					"enum": enumPaths,
				},
				"value": map[string]any{},
			},
		},
	}
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

func buildArtifactRepairMessage(mode artifactOutputMode, t entity.ArtifactType, err error, rawJSON string) string {
	switch mode {
	case artifactOutputModeJSONPatch:
		return buildArtifactJSONPatchRepairMessage(t, err, rawJSON)
	default:
		return buildArtifactFullJSONRepairMessage(t, err, rawJSON)
	}
}

func buildArtifactFullJSONRepairMessage(t entity.ArtifactType, err error, rawJSON string) string {
	raw := strings.TrimSpace(rawJSON)
	if raw == "" {
		raw = "{}"
	}
	raw = truncateByRunes(raw, 20000)

	errText := ""
	if err != nil {
		errText = strings.TrimSpace(err.Error())
	}

	return fmt.Sprintf(
		"上一次输出未通过服务端解析/校验，请你只做格式与字段修复，并重新输出“完整新版本 JSON”。\n\n要求：\n1) 只输出 JSON（不要 Markdown、不要代码块）。\n2) 必须可被 json.Unmarshal 解析。\n3) 保持已有 key 不变（仅新增对象时创建新 key）。\n4) 不要改变用户意图，只修复错误。\n\nartifact_type=%s\nerror=%s\n\n上一次输出（供修复）：\n%s",
		strings.TrimSpace(string(t)),
		errText,
		raw,
	)
}

func buildArtifactJSONPatchRepairMessage(t entity.ArtifactType, err error, rawJSON string) string {
	raw := strings.TrimSpace(rawJSON)
	if raw == "" {
		raw = "[]"
	}
	raw = truncateByRunes(raw, 20000)

	errText := ""
	if err != nil {
		errText = strings.TrimSpace(err.Error())
	}

	allowedPaths := strings.Join(artifactJSONPatchAllowedPaths(t), ", ")
	return fmt.Sprintf(
		"上一次 JSON Patch 未通过服务端解析/应用/校验，请你只做 patch 修复，并重新输出 JSON Patch 数组。\n\n要求：\n1) 只输出 JSON Patch 数组（不要 Markdown、不要代码块）。\n2) op 只允许 add 或 replace，且每个 op 必须包含 op/path/value。\n3) path 只允许：%s\n4) 不要改变用户意图，只修复错误。\n\nartifact_type=%s\nerror=%s\n\n上一次输出（供修复）：\n%s",
		allowedPaths,
		strings.TrimSpace(string(t)),
		errText,
		raw,
	)
}
