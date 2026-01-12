package callback

import (
	"context"
	"time"

	einocb "github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	cbtemplate "github.com/cloudwego/eino/utils/callbacks"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"z-novel-ai-api/internal/domain/service"
	"z-novel-ai-api/pkg/metrics"
)

type startTimeKey struct{}

type TenantIDGetter interface {
	GetCurrentTenant(ctx context.Context) (string, error)
}

func newChatModelCallbackHandler(usageRecorder service.LLMUsageRecorder, tenantIDGetter TenantIDGetter) *cbtemplate.ModelCallbackHandler {
	return &cbtemplate.ModelCallbackHandler{
		OnStart: func(ctx context.Context, info *einocb.RunInfo, input *model.CallbackInput) context.Context {
			ctx = context.WithValue(ctx, startTimeKey{}, time.Now())

			workflow := service.WorkflowFromContext(ctx)
			provider := service.ProviderFromContext(ctx)
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

		OnEnd: func(ctx context.Context, _ *einocb.RunInfo, output *model.CallbackOutput) context.Context {
			workflow := service.WorkflowFromContext(ctx)
			provider := service.ProviderFromContext(ctx)
			modelName := modelNameFromOutput(output)

			metrics.LLMCallTotal.WithLabelValues(workflow, provider, modelName, "success").Inc()
			if d := elapsedSeconds(ctx); d > 0 {
				metrics.LLMCallDuration.WithLabelValues(workflow, provider, modelName).Observe(d)
			}

			if output != nil && output.TokenUsage != nil {
				promptTokens := output.TokenUsage.PromptTokens
				completionTokens := output.TokenUsage.CompletionTokens

				metrics.LLMTokensUsed.WithLabelValues(workflow, provider, modelName, "prompt").Add(float64(promptTokens))
				metrics.LLMTokensUsed.WithLabelValues(workflow, provider, modelName, "completion").Add(float64(completionTokens))

				// 扣费/流水：从 callbacks 中解耦到应用层（quota），这里仅做 best-effort 调用。
				if usageRecorder != nil && tenantIDGetter != nil {
					tenantID, _ := tenantIDGetter.GetCurrentTenant(ctx)
					_ = usageRecorder.Record(ctx, service.LLMUsageInput{
						TenantID:         tenantID,
						Workflow:         workflow,
						Provider:         provider,
						Model:            modelName,
						PromptTokens:     promptTokens,
						CompletionTokens: completionTokens,
						DurationMs:       int(elapsedSeconds(ctx) * 1000),
					})
				}
			}

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
			workflow := service.WorkflowFromContext(ctx)
			provider := service.ProviderFromContext(ctx)
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

func newToolCallbackHandler() *cbtemplate.ToolCallbackHandler {
	return &cbtemplate.ToolCallbackHandler{
		OnStart: func(ctx context.Context, info *einocb.RunInfo, _ *tool.CallbackInput) context.Context {
			ctx = context.WithValue(ctx, startTimeKey{}, time.Now())

			workflow := service.WorkflowFromContext(ctx)
			toolName := ""
			if info != nil {
				toolName = info.Type
			}

			ctx, _ = otel.Tracer("eino").Start(ctx, "tool.invoke",
				trace.WithAttributes(
					attribute.String("eino.workflow", workflow),
					attribute.String("tool.name", toolName),
				),
			)
			return ctx
		},

		OnEnd: func(ctx context.Context, info *einocb.RunInfo, _ *tool.CallbackOutput) context.Context {
			workflow := service.WorkflowFromContext(ctx)
			toolName := ""
			if info != nil {
				toolName = info.Type
			}

			metrics.ToolCallTotal.WithLabelValues(workflow, toolName, "success").Inc()
			if d := elapsedSeconds(ctx); d > 0 {
				metrics.ToolCallDuration.WithLabelValues(workflow, toolName).Observe(d)
			}

			span := trace.SpanFromContext(ctx)
			if span != nil {
				span.End()
			}
			return ctx
		},

		OnError: func(ctx context.Context, info *einocb.RunInfo, err error) context.Context {
			workflow := service.WorkflowFromContext(ctx)
			toolName := ""
			if info != nil {
				toolName = info.Type
			}

			metrics.ToolCallTotal.WithLabelValues(workflow, toolName, "error").Inc()
			if d := elapsedSeconds(ctx); d > 0 {
				metrics.ToolCallDuration.WithLabelValues(workflow, toolName).Observe(d)
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

func elapsedSeconds(ctx context.Context) float64 {
	v := ctx.Value(startTimeKey{})
	start, ok := v.(time.Time)
	if !ok || start.IsZero() {
		return 0
	}
	return time.Since(start).Seconds()
}

func modelNameFromInput(in *model.CallbackInput) string {
	if in == nil || in.Config == nil {
		return ""
	}
	return in.Config.Model
}

func modelNameFromOutput(out *model.CallbackOutput) string {
	if out == nil || out.Config == nil {
		return ""
	}
	return out.Config.Model
}
