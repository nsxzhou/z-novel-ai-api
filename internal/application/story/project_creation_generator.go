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
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	einoobs "z-novel-ai-api/internal/observability/eino"
	workflowprompt "z-novel-ai-api/internal/workflow/prompt"
	"z-novel-ai-api/pkg/logger"
)

// ProjectCreationGenerateInput 定义了项目创建生成器的输入参数
type ProjectCreationGenerateInput struct {
	// Stage 当前会话所处的阶段 (discover, narrow, draft, confirm)
	Stage string
	// Draft 当前累积的项目草稿数据 (JSON)，包含 title, description, genre 等
	Draft json.RawMessage

	// Prompt 用户最新的输入消息
	Prompt      string
	Attachments []TextAttachment

	// Provider/Model 指定使用的大模型
	Provider string
	Model    string

	Temperature *float32
	MaxTokens   *int
}

// ProjectCreationGenerateOutput 定义了项目创建生成器的输出结果
type ProjectCreationGenerateOutput struct {
	// AssistantMessage AI 回复给用户的文本消息
	AssistantMessage string
	// NextStage AI 决定的下一阶段
	NextStage string
	// Draft 更新后的项目草稿数据
	Draft json.RawMessage

	// Action AI 建议执行的动作 (none, propose_creation, create_project)
	Action               string
	RequiresConfirmation bool
	// ProposedProject 当 Action 为 create_project 时，包含最终确定的项目信息
	ProposedProject *ProjectCreationProjectDraft
	// Meta Token 使用量等元数据
	Meta LLMUsageMeta
}

// ProjectCreationProjectDraft 项目草稿的数据结构
type ProjectCreationProjectDraft struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Genre       string `json:"genre,omitempty"`
}

// projectCreationLLMEnvelope 用于解析 LLM 返回的 JSON 结构的信封
type projectCreationLLMEnvelope struct {
	AssistantMessage     string                       `json:"assistant_message"`
	NextStage            string                       `json:"stage"`
	Draft                json.RawMessage              `json:"draft"`
	Action               string                       `json:"action"`
	RequiresConfirmation bool                         `json:"requires_confirmation"`
	Project              *ProjectCreationProjectDraft `json:"project,omitempty"`
}

// ProjectCreationGenerator 实现了基于对话的项目创建逻辑。
// 它使用 Eino 编排一个 Chain，负责维护对话状态机，引导用户完善项目构思。
type ProjectCreationGenerator struct {
	factory ChatModelFactory

	chainOnce sync.Once
	//应用层（你的 Go 代码逻辑）与生成器组件之间的输入输出协议
	chain    compose.Runnable[*ProjectCreationGenerateInput, *ProjectCreationGenerateOutput]
	chainErr error
}

func NewProjectCreationGenerator(factory ChatModelFactory) *ProjectCreationGenerator {
	return &ProjectCreationGenerator{factory: factory}
}

// Generate 执行生成流程：Prompt 渲染 -> LLM 调用 (Structured Output) -> 结果解析
func (g *ProjectCreationGenerator) Generate(ctx context.Context, in *ProjectCreationGenerateInput) (*ProjectCreationGenerateOutput, error) {
	if g == nil || g.factory == nil {
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

// formatProjectCreationMessages 加载 Prompt 模板并填充变量
func formatProjectCreationMessages(ctx context.Context, in *ProjectCreationGenerateInput) ([]*schema.Message, error) {
	tpl, err := defaultPromptRegistry.ChatTemplate(workflowprompt.PromptProjectCreationV1)
	if err != nil {
		return nil, err
	}
	draft := "{}"
	if len(in.Draft) > 0 {
		draft = strings.TrimSpace(string(in.Draft))
		if draft == "" {
			draft = "{}"
		}
	}
	vars := map[string]any{
		"stage":             strings.TrimSpace(in.Stage),
		"draft_json":        draft,
		"prompt":            strings.TrimSpace(in.Prompt),
		"attachments_block": buildAttachmentsBlock(in.Attachments),
	}
	return tpl.Format(ctx, vars)
}

// buildProjectCreationModelOptions 构建 LLM 调用选项，强制启用 JSON Schema (Structured Outputs)
func buildProjectCreationModelOptions(in *ProjectCreationGenerateInput, enableSchema bool) []model.Option {
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

	// 强制要求模型返回符合 Schema 的 JSON 格式
	if enableSchema {
		opts = append(opts, openaiopts.WithExtraFields(map[string]any{
			"response_format": map[string]any{
				"type": "json_schema",
				"json_schema": map[string]any{
					"name":   "project_creation",
					"strict": false,
					"schema": projectCreationJSONSchema(),
				},
			},
		}))
	}

	return opts
}

// projectCreationJSONSchema 定义了期望 LLM 返回的 JSON 结构 Schema
func projectCreationJSONSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []any{"assistant_message", "stage", "draft", "action", "requires_confirmation"},
		"properties": map[string]any{
			"assistant_message": map[string]any{"type": "string"},
			"stage": map[string]any{
				"type": "string",
				"enum": []any{"discover", "narrow", "draft", "confirm"},
			},
			"draft": map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required":             []any{"title", "description", "genre"},
				"properties": map[string]any{
					"title":       map[string]any{"type": "string"},
					"description": map[string]any{"type": "string"},
					"genre":       map[string]any{"type": "string"},
				},
			},
			"action": map[string]any{
				"type": "string",
				"enum": []any{"none", "propose_creation", "create_project"},
			},
			"requires_confirmation": map[string]any{"type": "boolean"},
			"project": map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required":             []any{"title", "description"},
				"properties": map[string]any{
					"title":       map[string]any{"type": "string"},
					"description": map[string]any{"type": "string"},
					"genre":       map[string]any{"type": "string"},
				},
			},
		},
	}
}

type projectCreationChainState struct {
	In       *ProjectCreationGenerateInput
	Messages []*schema.Message
	OutMsg   *schema.Message
}

func (g *ProjectCreationGenerator) getChain() (compose.Runnable[*ProjectCreationGenerateInput, *ProjectCreationGenerateOutput], error) {
	g.chainOnce.Do(func() {
		g.chain, g.chainErr = g.buildChain(context.Background())
	})
	return g.chain, g.chainErr
}

// buildChain 构建 Eino 处理链：Init -> Template -> LLM -> Finalize
// 该链负责协调从接收输入到最终生成结构化项目草稿的全过程。
func (g *ProjectCreationGenerator) buildChain(ctx context.Context) (compose.Runnable[*ProjectCreationGenerateInput, *ProjectCreationGenerateOutput], error) {
	chain := compose.NewChain[*ProjectCreationGenerateInput, *ProjectCreationGenerateOutput]()

	// ---------------------------------------------------------------------
	// 1. Init: 初始化状态节点
	// ---------------------------------------------------------------------
	// 作用：将外部传入的 Input 封装进 chain 内部的状态对象 (projectCreationChainState)。
	// 目的：为后续节点提供统一的状态上下文，方便在链中传递数据。
	chain.AppendLambda(
		compose.InvokableLambda(func(ctx context.Context, in *ProjectCreationGenerateInput) (*projectCreationChainState, error) {
			if in == nil {
				return nil, fmt.Errorf("input is nil")
			}
			return &projectCreationChainState{In: in}, nil
		}),
		compose.WithNodeName("project_creation.init"),
	)

	// ---------------------------------------------------------------------
	// 2. Template: Prompt 渲染节点
	// ---------------------------------------------------------------------
	// 作用：加载 Prompt 模板并填充变量。
	// 逻辑：调用 formatProjectCreationMessages，将 Stage, Draft, UserPrompt 等数据
	//      转换为 LLM 可理解的消息列表 ([]*schema.Message)。
	chain.AppendLambda(
		compose.InvokableLambda(func(ctx context.Context, st *projectCreationChainState) (*projectCreationChainState, error) {
			if st == nil || st.In == nil {
				return nil, fmt.Errorf("state is nil")
			}
			msgs, err := formatProjectCreationMessages(ctx, st.In)
			if err != nil {
				return nil, err
			}
			st.Messages = msgs
			return st, nil
		}),
		compose.WithNodeName("project_creation.template"),
	)

	// ---------------------------------------------------------------------
	// 3. LLM: 模型推理节点
	// ---------------------------------------------------------------------
	// 作用：调用底层大模型生成回复。
	// 特性：
	//    1. 可观测性：注入 Workflow 和 Provider 信息，方便追踪调用链路。
	//    2. 鲁棒性/兼容性策略（Fallback）：
	//       - 优先尝试 Structured Output (JSON Schema) 模式，以获得最严格的格式保证。
	//       - 如果模型提供商返回“不支持响应格式”的错误，自动降级为普通 JSON 模式重试。
	chain.AppendLambda(
		compose.InvokableLambda(func(ctx context.Context, st *projectCreationChainState) (*projectCreationChainState, error) {
			if st == nil || st.In == nil {
				return nil, fmt.Errorf("state is nil")
			}
			if g == nil || g.factory == nil {
				return nil, fmt.Errorf("llm factory not configured")
			}

			// 注入可观测性上下文 (用于 Tracing/Metrics)
			ctx = einoobs.WithWorkflowProvider(ctx, "project_creation_generate", st.In.Provider)

			chatModel, err := g.factory.Get(ctx, st.In.Provider)
			if err != nil {
				return nil, err
			}

			// 策略：优先尝试 Structured Output (JSON Schema) 以确保格式稳定
			outMsg, err := chatModel.Generate(ctx, st.Messages, buildProjectCreationModelOptions(st.In, true)...)
			// 降级：如果模型不支持 Structured Output，降级为普通 JSON 模式重试
			// 这确保了对不支持高级特性的旧模型或特定 Provider 的兼容性
			if err != nil && isResponseFormatUnsupportedError(err) {
				outMsg, err = chatModel.Generate(ctx, st.Messages, buildProjectCreationModelOptions(st.In, false)...)
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
		compose.WithNodeName("project_creation.llm"),
	)

	// ---------------------------------------------------------------------
	// 4. Finalize: 结果解析与封装节点
	// ---------------------------------------------------------------------
	// 作用：将 LLM 的原始文本回复转换为强类型的 Go 结构体。
	// 逻辑：
	//    1. 提取 JSON：处理可能存在的 Markdown 代码块 (```json ... ```)。
	//    2. 反序列化：解析为中间结构体 projectCreationLLMEnvelope。
	//    3. 元数据收集：记录 Token 使用量 (Usage) 和生成参数。
	//    4. 默认值兜底：确保 Stage 等关键字段有值。
	chain.AppendLambda(
		compose.InvokableLambda(func(ctx context.Context, st *projectCreationChainState) (*ProjectCreationGenerateOutput, error) {
			if st == nil || st.In == nil || st.OutMsg == nil {
				return nil, fmt.Errorf("state is nil")
			}

			// 提取并清理 JSON 字符串
			raw := extractJSONObject(st.OutMsg.Content)
			if strings.TrimSpace(raw) == "" {
				return nil, fmt.Errorf("empty project creation output")
			}

			// 反序列化为结构化对象 (信封模式)
			var env projectCreationLLMEnvelope
			if err := json.Unmarshal([]byte(raw), &env); err != nil {
				logger.Error(ctx, "failed to unmarshal project creation output", err,
					"raw", raw,
				)
				return nil, fmt.Errorf("invalid project creation output: %w", err)
			}

			// 收集元数据 (Token Usage, Latency 等信息对于计费和监控至关重要)
			meta := LLMUsageMeta{Provider: st.In.Provider, Model: strings.TrimSpace(st.In.Model), GeneratedAt: time.Now().UTC()}
			if st.In.Temperature != nil {
				meta.Temperature = float64(*st.In.Temperature)
			}
			if st.OutMsg.ResponseMeta != nil && st.OutMsg.ResponseMeta.Usage != nil {
				meta.PromptTokens = st.OutMsg.ResponseMeta.Usage.PromptTokens
				meta.CompletionTokens = st.OutMsg.ResponseMeta.Usage.CompletionTokens
			}

			// 状态机流转保护：如果 AI 未返回阶段，保持当前阶段或重置为 discover
			nextStage := strings.TrimSpace(env.NextStage)
			if nextStage == "" {
				nextStage = strings.TrimSpace(st.In.Stage)
				if nextStage == "" {
					nextStage = "discover"
				}
			}

			return &ProjectCreationGenerateOutput{
				AssistantMessage:     strings.TrimSpace(env.AssistantMessage),
				NextStage:            nextStage,
				Draft:                env.Draft,
				Action:               strings.TrimSpace(env.Action),
				RequiresConfirmation: env.RequiresConfirmation,
				ProposedProject:      env.Project,
				Meta:                 meta,
			}, nil
		}),
		compose.WithNodeName("project_creation.finalize"),
	)

	return chain.Compile(ctx, compose.WithGraphName("project_creation_generate_chain"))
}
