package eino

import (
	"context"
	"fmt"
	"time"

	einocb "github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	cbtemplate "github.com/cloudwego/eino/utils/callbacks"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"z-novel-ai-api/pkg/metrics"
)

// startTimeKey 用于在 Context 中存储调用开始时间
// 这样可以在 OnEnd/OnError 时计算总耗时
type startTimeKey struct{}

// newChatModelCallbackHandler 创建 AI 大模型调用的回调处理器
//
// 这个处理器会在每次 AI 模型生成内容时触发，记录：
//   - 调用次数（成功/失败）
//   - 耗时
//   - Token 消耗
//   - 分布式追踪信息
//
// 返回值会注册到全局回调链中，监控所有 AI 模型调用
func newChatModelCallbackHandler() *cbtemplate.ModelCallbackHandler {
	return &cbtemplate.ModelCallbackHandler{
		// OnStart 在 AI 模型开始生成时被调用
		// 记录开始时间、收集元信息、启动分布式追踪
		OnStart: func(ctx context.Context, info *einocb.RunInfo, input *model.CallbackInput) context.Context {
			// 记录开始时间，用于后续计算耗时
			ctx = context.WithValue(ctx, startTimeKey{}, time.Now())

			// 从上下文中获取工作流名称和模型提供商
			// 这些信息由 WithWorkflow/WithProvider 预先设置
			workflow := WorkflowFromContext(ctx)
			provider := ProviderFromContext(ctx)
			modelName := modelNameFromInput(input)

			// 构建 OpenTelemetry 追踪属性
			attrs := []attribute.KeyValue{
				attribute.String("eino.workflow", workflow), // 工作流名称
				attribute.String("llm.provider", provider),  // 模型提供商
				attribute.String("llm.model", modelName),    // 模型名称
			}
			// 如果有节点信息，也添加到属性中
			if info != nil {
				attrs = append(attrs,
					attribute.String("eino.node_name", info.Name), // 节点名称
					attribute.String("eino.type", info.Type),      // 组件类型
				)
			}

			// 启动分布式追踪，创建一个新的 Span
			// Span 会记录这次 AI 调用的完整链路
			ctx, _ = otel.Tracer("eino").Start(ctx, "llm.generate", trace.WithAttributes(attrs...))
			return ctx
		},

		// OnEnd 在 AI 模型完成生成时被调用
		// 记录成功状态、Token 消耗、耗时，更新追踪信息
		OnEnd: func(ctx context.Context, info *einocb.RunInfo, output *model.CallbackOutput) context.Context {
			workflow := WorkflowFromContext(ctx)
			provider := ProviderFromContext(ctx)
			modelName := modelNameFromOutput(output)

			// 上报调用次数指标（成功）
			metrics.LLMCallTotal.WithLabelValues(workflow, provider, modelName, "success").Inc()

			// 计算耗时并上报
			if d := elapsedSeconds(ctx); d > 0 {
				metrics.LLMCallDuration.WithLabelValues(workflow, provider, modelName).Observe(d)
			}

			// 如果有 Token 使用情况，上报 Token 消耗
			// Prompt Tokens：输入消耗的 Token
			// Completion Tokens：输出消耗的 Token
			if output != nil && output.TokenUsage != nil {
				metrics.LLMTokensUsed.WithLabelValues(workflow, provider, modelName, "prompt").Add(float64(output.TokenUsage.PromptTokens))
				metrics.LLMTokensUsed.WithLabelValues(workflow, provider, modelName, "completion").Add(float64(output.TokenUsage.CompletionTokens))
			}

			// 更新分布式追踪的 Span 信息
			span := trace.SpanFromContext(ctx)
			if span != nil {
				// 将 Token 使用情况添加到追踪信息中
				if output != nil && output.TokenUsage != nil {
					span.SetAttributes(
						attribute.Int("llm.prompt_tokens", output.TokenUsage.PromptTokens),
						attribute.Int("llm.completion_tokens", output.TokenUsage.CompletionTokens),
					)
				}
				// 结束这个 Span
				span.End()
			}
			return ctx
		},

		// OnError 在 AI 模型调用出错时被调用
		// 记录失败状态、错误信息、耗时
		OnError: func(ctx context.Context, info *einocb.RunInfo, err error) context.Context {
			workflow := WorkflowFromContext(ctx)
			provider := ProviderFromContext(ctx)
			// 发生错误时，从 info.Type 获取模型名称
			modelName := ""
			if info != nil {
				modelName = info.Type
			}

			// 上报调用次数指标（失败）
			metrics.LLMCallTotal.WithLabelValues(workflow, provider, modelName, "error").Inc()

			// 计算耗时并上报
			if d := elapsedSeconds(ctx); d > 0 {
				metrics.LLMCallDuration.WithLabelValues(workflow, provider, modelName).Observe(d)
			}

			// 更新分布式追踪的 Span 信息
			span := trace.SpanFromContext(ctx)
			if span != nil {
				// 记录错误
				span.RecordError(err)
				// 设置 Span 状态为错误
				span.SetStatus(codes.Error, err.Error())
				// 结束这个 Span
				span.End()
			}
			return ctx
		},
	}
}

// newToolCallbackHandler 创建工具函数调用的回调处理器
//
// 这个处理器会在每次 AI 调用外部工具时触发，记录：
//   - 调用次数（成功/失败）
//   - 耗时
//   - 工具名称
//
// 工具包括：搜索、数据库查询、计算器等外部服务
func newToolCallbackHandler() *cbtemplate.ToolCallbackHandler {
	return &cbtemplate.ToolCallbackHandler{
		// OnStart 在工具开始执行时被调用
		// 记录开始时间，启动分布式追踪
		OnStart: func(ctx context.Context, info *einocb.RunInfo, input *tool.CallbackInput) context.Context {
			// 记录开始时间
			ctx = context.WithValue(ctx, startTimeKey{}, time.Now())

			workflow := WorkflowFromContext(ctx)
			toolName := ""
			if info != nil {
				toolName = info.Type
			}

			// 启动分布式追踪
			ctx, _ = otel.Tracer("eino").Start(ctx, "tool.invoke",
				trace.WithAttributes(
					attribute.String("eino.workflow", workflow), // 工作流名称
					attribute.String("tool.name", toolName),     // 工具名称
				),
			)
			return ctx
		},

		// OnEnd 在工具完成执行时被调用
		// 记录成功状态和耗时
		OnEnd: func(ctx context.Context, info *einocb.RunInfo, output *tool.CallbackOutput) context.Context {
			workflow := WorkflowFromContext(ctx)
			toolName := ""
			if info != nil {
				toolName = info.Type
			}

			// 上报工具调用次数（成功）
			metrics.ToolCallTotal.WithLabelValues(workflow, toolName, "success").Inc()

			// 上报耗时
			if d := elapsedSeconds(ctx); d > 0 {
				metrics.ToolCallDuration.WithLabelValues(workflow, toolName).Observe(d)
			}

			// 结束追踪 Span
			span := trace.SpanFromContext(ctx)
			if span != nil {
				span.End()
			}
			return ctx
		},

		// OnError 在工具执行出错时被调用
		// 记录失败状态、错误信息、耗时
		OnError: func(ctx context.Context, info *einocb.RunInfo, err error) context.Context {
			workflow := WorkflowFromContext(ctx)
			toolName := ""
			if info != nil {
				toolName = info.Type
			}

			// 上报工具调用次数（失败）
			metrics.ToolCallTotal.WithLabelValues(workflow, toolName, "error").Inc()

			// 上报耗时
			if d := elapsedSeconds(ctx); d > 0 {
				metrics.ToolCallDuration.WithLabelValues(workflow, toolName).Observe(d)
			}

			// 更新追踪信息
			span := trace.SpanFromContext(ctx)
			if span != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				span.End()
			}
			return ctx
		},
	}
}

// elapsedSeconds 计算从 Context 开始到当前的时间差（秒）
//
// 工作流程：
//  1. OnStart 将开始时间存入 Context
//  2. OnEnd/OnError 调用此函数计算耗时
//
// 返回值：
//
//	> 0：有效的耗时（秒）
//	0：无法获取开始时间（不应该发生）
func elapsedSeconds(ctx context.Context) float64 {
	// 从 Context 中获取开始时间
	v := ctx.Value(startTimeKey{})
	start, ok := v.(time.Time)
	// 转换失败或时间为空，返回 0
	if !ok || start.IsZero() {
		return 0
	}
	// 计算时间差
	return time.Since(start).Seconds()
}

// modelNameFromInput 从输入配置中提取模型名称
func modelNameFromInput(in *model.CallbackInput) string {
	if in == nil || in.Config == nil {
		return ""
	}
	return in.Config.Model
}

// modelNameFromOutput 从输出配置中提取模型名称
func modelNameFromOutput(out *model.CallbackOutput) string {
	if out == nil || out.Config == nil {
		return ""
	}
	return out.Config.Model
}

// fmtDurationMs 将秒转换为毫秒字符串（未使用，保留备用）
func fmtDurationMs(seconds float64) string {
	if seconds <= 0 {
		return "0"
	}
	return fmt.Sprintf("%d", int64(seconds*1000))
}
