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

	"z-novel-ai-api/internal/domain/entity"
	"z-novel-ai-api/internal/domain/repository"
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
// newChatModelCallbackHandler 创建 AI 大模型调用的回调处理器
func newChatModelCallbackHandler(tenantRepo repository.TenantRepository, llmRepo repository.LLMUsageEventRepository, tenantCtxMgr repository.TenantContextManager) *cbtemplate.ModelCallbackHandler {
	return &cbtemplate.ModelCallbackHandler{
		// OnStart ... (保持追踪逻辑不变)
		OnStart: func(ctx context.Context, info *einocb.RunInfo, input *model.CallbackInput) context.Context {
			ctx = context.WithValue(ctx, startTimeKey{}, time.Now())

			workflow := WorkflowFromContext(ctx)
			provider := ProviderFromContext(ctx)
			modelName := modelNameFromInput(input)

			attrs := []attribute.KeyValue{
				attribute.String("eino.workflow", workflow),
				attribute.String("llm.provider", provider),
				attribute.String("llm.model", modelName),
			}
			if info != nil {
				attrs = append(attrs,
					attribute.String("eino.node_name", info.Name),
					attribute.String("eino.type", info.Type),
				)
			}

			ctx, _ = otel.Tracer("eino").Start(ctx, "llm.generate", trace.WithAttributes(attrs...))
			return ctx
		},

		// OnEnd ... (增加扣费和持久化逻辑)
		OnEnd: func(ctx context.Context, info *einocb.RunInfo, output *model.CallbackOutput) context.Context {
			workflow := WorkflowFromContext(ctx)
			provider := ProviderFromContext(ctx)
			modelName := modelNameFromOutput(output)

			// 1. 指标上报
			metrics.LLMCallTotal.WithLabelValues(workflow, provider, modelName, "success").Inc()
			if d := elapsedSeconds(ctx); d > 0 {
				metrics.LLMCallDuration.WithLabelValues(workflow, provider, modelName).Observe(d)
			}

			if output != nil && output.TokenUsage != nil {
				promptTokens := output.TokenUsage.PromptTokens
				completionTokens := output.TokenUsage.CompletionTokens

				metrics.LLMTokensUsed.WithLabelValues(workflow, provider, modelName, "prompt").Add(float64(promptTokens))
				metrics.LLMTokensUsed.WithLabelValues(workflow, provider, modelName, "completion").Add(float64(completionTokens))

				// 2. 自动化扣费与流水记录 (如果有 Repo 注入)
				if tenantRepo != nil && llmRepo != nil && tenantCtxMgr != nil {
					if postgresCtxMgr, ok := tenantCtxMgr.(interface {
						GetCurrentTenant(ctx context.Context) (string, error)
					}); ok {
						tenantID, _ := postgresCtxMgr.GetCurrentTenant(ctx)
						if tenantID != "" {
							totalTokens := int64(promptTokens + completionTokens)

							// 扣余额
							_ = tenantRepo.DeductBalance(ctx, tenantID, totalTokens)

							// 记流水
							_ = llmRepo.Create(ctx, &entity.LLMUsageEvent{
								TenantID:         tenantID,
								Provider:         provider,
								Model:            modelName,
								Workflow:         workflow,
								TokensPrompt:     promptTokens,
								TokensCompletion: completionTokens,
								DurationMs:       int(elapsedSeconds(ctx) * 1000),
							})
						}
					}
				}
			}

			// 3. 追踪 Span 结束
			span := trace.SpanFromContext(ctx)
			if span != nil {
				if output != nil && output.TokenUsage != nil {
					span.SetAttributes(
						attribute.Int("llm.prompt_tokens", output.TokenUsage.PromptTokens),
						attribute.Int("llm.completion_tokens", output.TokenUsage.CompletionTokens),
					)
				}
				span.End()
			}
			return ctx
		},

		OnError: func(ctx context.Context, info *einocb.RunInfo, err error) context.Context {
			workflow := WorkflowFromContext(ctx)
			provider := ProviderFromContext(ctx)
			modelName := ""
			if info != nil {
				modelName = info.Type
			}

			metrics.LLMCallTotal.WithLabelValues(workflow, provider, modelName, "error").Inc()
			if d := elapsedSeconds(ctx); d > 0 {
				metrics.LLMCallDuration.WithLabelValues(workflow, provider, modelName).Observe(d)
			}

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
