# Eino 框架 Callbacks 机制详解

> 文档基于 `internal/observability/eino/` 目录下的代码实现编写
>
> 最后更新：2026-01-07

## 目录

- [一、Callbacks 机制概述](#一callbacks机制概述)
- [二、核心代码实现分析](#二核心代码实现分析)
- [三、Metrics 指标体系](#三metrics指标体系)
- [四、OpenTelemetry 集成](#四opentelemetry集成)
- [五、不同类型 Callbacks 对比](#五不同类型callbacks对比)
- [六、在工作流中的应用实践](#六在工作流中的应用实践)
- [七、最佳实践总结](#七最佳实践总结)

---

## 一、Callbacks 机制概述

### 1.1 核心概念

Callbacks 本质上是一种**事件驱动的钩子机制**，它允许开发者在 AI 组件（如 ChatModel、Tool）的执行过程中的关键节点注入自定义逻辑。这些节点包括：

| 阶段        | 说明           | 典型用途                                       |
| ----------- | -------------- | ---------------------------------------------- |
| **OnStart** | 组件执行前     | 记录开始时间、初始化追踪上下文、收集输入元数据 |
| **OnEnd**   | 组件成功执行后 | 计算耗时、记录输出结果、采集 Token 使用情况    |
| **OnError** | 组件执行异常时 | 捕获错误信息、记录失败状态、结束追踪 Span      |

这种设计遵循了**横切关注点分离**的原则，将可观测性逻辑与业务逻辑解耦，使得开发者无需在每个业务调用点重复编写监控代码。

### 1.2 架构层次

Eino 的 Callbacks 机制采用**分层架构**：

```
┌─────────────────────────────────────────────────────────────┐
│                    Global Handlers                          │
│         (einocallbacks.AppendGlobalHandlers)                │
├─────────────────────────────────────────────────────────────┤
│                    Component Handlers                       │
│     ModelCallbackHandler / ToolCallbackHandler              │
│         (cbtemplate.NewHandlerHelper)                       │
├─────────────────────────────────────────────────────────────┤
│                   Runtime Execution                         │
│         LLM.Call(ctx, input) / Tool.Invoke(ctx, input)      │
└─────────────────────────────────────────────────────────────┘
```

**各层职责说明**：

1. **全局层**：通过 `AppendGlobalHandlers` 注册，对整个进程生效
2. **组件层**：针对不同组件类型（ChatModel、Tool）提供专门的处理器
3. **运行时**：Eino 内部组件在执行时会自动触发对应回调

### 1.3 在项目中的应用位置

根据项目设计文档 `docs/22-Eino设定生成工作流重构设计.md`，Callbacks 机制是 **Phase 1** 的落地内容之一，用于实现：

> 统一 metrics/tracing/logs，增强可观测性

**相关文件**：

| 文件                                     | 职责                          |
| ---------------------------------------- | ----------------------------- |
| `internal/observability/eino/init.go`    | 全局 Callbacks 初始化入口     |
| `internal/observability/eino/handler.go` | ChatModel/Tool 回调处理器实现 |
| `internal/observability/eino/context.go` | Context 传递工具函数          |
| `pkg/metrics/metrics.go`                 | Prometheus 指标定义           |

---

## 二、核心代码实现分析

### 2.1 全局初始化（init.go）

`init.go` 是 Callbacks 机制的入口文件：

```go
package eino

import (
    "sync"

    einocallbacks "github.com/cloudwego/eino/callbacks"
    cbtemplate "github.com/cloudwego/eino/utils/callbacks"
)

var initOnce sync.Once

// Init 注册 Eino 全局 callbacks（进程级一次）。
func Init() {
    initOnce.Do(func() {
        handler := cbtemplate.NewHandlerHelper().
            ChatModel(newChatModelCallbackHandler()).
            Tool(newToolCallbackHandler()).
            Handler()

        einocallbacks.AppendGlobalHandlers(handler)
    })
}
```

**技术要点解析**：

1. **`sync.Once`**：确保初始化函数在整个进程生命周期内只执行一次，这是进程级全局 Handler 的典型模式

2. **`NewHandlerHelper()` 构建器模式**：

   ```go
   handler := cbtemplate.NewHandlerHelper().
       ChatModel(newChatModelCallbackHandler()).  // 注册ChatModel回调
       Tool(newToolCallbackHandler()).            // 注册Tool回调
       Handler()                                  // 生成最终处理器
   ```

3. **`AppendGlobalHandlers(handler)`**：将构建好的处理器注册为全局处理器，之后所有通过 Eino 组件执行的 LLM 调用和 Tool 调用都会自动触发这些回调

### 2.2 ChatModel 回调处理器（handler.go）

`handler.go` 实现了 ChatModel 的回调逻辑：

```go
func newChatModelCallbackHandler() *cbtemplate.ModelCallbackHandler {
    return &cbtemplate.ModelCallbackHandler{
        OnStart: func(ctx context.Context, info *einocb.RunInfo, input *model.CallbackInput) context.Context {
            // 1. 记录开始时间到context
            ctx = context.WithValue(ctx, startTimeKey{}, time.Now())

            // 2. 从context提取workflow和provider信息
            workflow := WorkflowFromContext(ctx)
            provider := ProviderFromContext(ctx)
            modelName := modelNameFromInput(input)

            // 3. 构建OpenTelemetry属性
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

            // 4. 启动追踪Span
            ctx, _ = otel.Tracer("eino").Start(ctx, "llm.generate",
                trace.WithAttributes(attrs...))
            return ctx
        },
        OnEnd: func(ctx context.Context, info *einocb.RunInfo, output *model.CallbackOutput) context.Context {
            workflow := WorkflowFromContext(ctx)
            provider := ProviderFromContext(ctx)
            modelName := modelNameFromOutput(output)

            // 5. 记录Prometheus指标
            metrics.LLMCallTotal.WithLabelValues(workflow, provider, modelName, "success").Inc()
            metrics.LLMCallDuration.WithLabelValues(workflow, provider, modelName).Observe(d)
            metrics.LLMTokensUsed.WithLabelValues(workflow, provider, modelName, "prompt").Add(float64(output.TokenUsage.PromptTokens))
            metrics.LLMTokensUsed.WithLabelValues(workflow, provider, modelName, "completion").Add(float64(output.TokenUsage.CompletionTokens))

            // 6. 结束追踪Span并记录Token使用
            span := trace.SpanFromContext(ctx)
            if span != nil {
                span.SetAttributes(
                    attribute.Int("llm.prompt_tokens", output.TokenUsage.PromptTokens),
                    attribute.Int("llm.completion_tokens", output.TokenUsage.CompletionTokens),
                )
                span.End()
            }
            return ctx
        },
        OnError: func(ctx context.Context, info *einocb.RunInfo, err error) context.Context {
            // 7. 记录错误指标和状态
            metrics.LLMCallTotal.WithLabelValues(workflow, provider, modelName, "error").Inc()
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
```

**执行流程图**：

```
LLM.Generate() 被调用
        │
        ▼
┌───────────────────┐
│    OnStart        │ ──► 记录开始时间
│                   │ ──► 提取context中的workflow/provider
│                   │ ──► 构建Span属性
│                   │ ──► 启动OpenTelemetry Span
└───────────────────┘
        │
        ▼
   LLM实际执行
        │
        ▼
    ┌────┴────┐
    │         │
    ▼         ▼
  成功       失败
    │         │
    ▼         ▼
┌───────┐ ┌───────┐
│OnEnd  │ │OnError│
└───────┘ └───────┘
    │         │
    └───┬─────┘
        ▼
   记录metrics/结束Span
```

### 2.3 Tool 回调处理器

Tool 回调处理器与 ChatModel 类似，但监控对象是工具调用：

```go
func newToolCallbackHandler() *cbtemplate.ToolCallbackHandler {
    return &cbtemplate.ToolCallbackHandler{
        OnStart: func(ctx context.Context, info *einocb.RunInfo, input *tool.CallbackInput) context.Context {
            ctx = context.WithValue(ctx, startTimeKey{}, time.Now())
            workflow := WorkflowFromContext(ctx)
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
        OnEnd: func(ctx context.Context, info *einocb.RunInfo, output *tool.CallbackOutput) context.Context {
            metrics.ToolCallTotal.WithLabelValues(workflow, toolName, "success").Inc()
            metrics.ToolCallDuration.WithLabelValues(workflow, toolName).Observe(d)
            span := trace.SpanFromContext(ctx)
            if span != nil {
                span.End()
            }
            return ctx
        },
        OnError: func(ctx context.Context, info *einocb.RunInfo, err error) context.Context {
            metrics.ToolCallTotal.WithLabelValues(workflow, toolName, "error").Inc()
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
```

### 2.4 Context 传递机制（context.go）

`context.go` 定义了跨请求传递业务上下文的工具函数：

```go
type ctxKey string

const (
    ctxKeyWorkflow ctxKey = "eino_workflow"
    ctxKeyProvider ctxKey = "eino_provider"
)

// WithWorkflow 将workflow信息注入context
func WithWorkflow(ctx context.Context, workflow string) context.Context {
    if ctx == nil {
        return nil
    }
    w := strings.TrimSpace(workflow)
    if w == "" {
        return ctx
    }
    return context.WithValue(ctx, ctxKeyWorkflow, w)
}

// WithProvider 将provider信息注入context
func WithProvider(ctx context.Context, provider string) context.Context {
    if ctx == nil {
        return nil
    }
    p := strings.TrimSpace(provider)
    if p == "" {
        return ctx
    }
    return context.WithValue(ctx, ctxKeyProvider, p)
}

// 组合注入
func WithWorkflowProvider(ctx context.Context, workflow, provider string) context.Context {
    return WithProvider(WithWorkflow(ctx, workflow), provider)
}

// 从context提取workflow
func WorkflowFromContext(ctx context.Context) string {
    if ctx == nil {
        return "unknown"
    }
    v := ctx.Value(ctxKeyWorkflow)
    s, ok := v.(string)
    if !ok || strings.TrimSpace(s) == "" {
        return "unknown"
    }
    return strings.TrimSpace(s)
}

// 从context提取provider
func ProviderFromContext(ctx context.Context) string {
    if ctx == nil {
        return "unknown"
    }
    v := ctx.Value(ctxKeyProvider)
    s, ok := v.(string)
    if !ok || strings.TrimSpace(s) == "" {
        return "unknown"
    }
    return strings.TrimSpace(s)
}
```

**设计特点**：

1. **类型安全**：使用自定义类型 `ctxKey` 避免 key 冲突
2. **防御性检查**：对 nil context、空值进行健壮处理
3. **便捷组合**：`WithWorkflowProvider` 提供一次性注入多个值的能力
4. **默认值**：`unknown` 作为兜底值，避免 metrics 标签出现空值

---

## 三、Metrics 指标体系

项目定义了一套完整的 Prometheus 指标体系，详见 `pkg/metrics/metrics.go`：

### 3.1 LLM 相关指标

```go
// 调用次数：统计成功/失败次数
LLMCallTotal = promauto.NewCounterVec(
    prometheus.CounterOpts{
        Namespace: namespace,  // z_novel
        Subsystem: "llm",
        Name:      "call_total",
        Help:      "Total number of LLM calls",
    },
    []string{"workflow", "provider", "model", "status"},  // 标签
)

// 调用耗时：Histogram用于计算P50/P95/P99
LLMCallDuration = promauto.NewHistogramVec(
    prometheus.HistogramOpts{
        Namespace: namespace,
        Subsystem: "llm",
        Name:      "call_duration_seconds",
        Help:      "LLM call duration in seconds",
        Buckets:   []float64{1, 5, 10, 30, 60, 120},
    },
    []string{"workflow", "provider", "model"},
)

// Token使用：分别统计prompt和completion
LLMTokensUsed = promauto.NewCounterVec(
    prometheus.CounterOpts{
        Namespace: namespace,
        Subsystem: "llm",
        Name:      "tokens_used_total",
        Help:      "Total tokens used for LLM calls",
    },
    []string{"workflow", "provider", "model", "type"},
)
```

### 3.2 Tool 相关指标

```go
// Tool调用耗时
ToolCallDuration = promauto.NewHistogramVec(
    prometheus.HistogramOpts{
        Namespace: namespace,
        Subsystem: "tool",
        Name:      "call_duration_seconds",
        Help:      "Tool call duration in seconds",
        Buckets:   []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
    },
    []string{"workflow", "tool"},
)

// Tool调用次数
ToolCallTotal = promauto.NewCounterVec(
    prometheus.CounterOpts{
        Namespace: namespace,
        Subsystem: "tool",
        Name:      "call_total",
        Help:      "Total number of tool calls",
    },
    []string{"workflow", "tool", "status"},
)
```

### 3.3 指标标签设计

指标采用多维度标签设计，支持灵活的聚合查询：

| 标签     | 说明             | 示例值                                       |
| -------- | ---------------- | -------------------------------------------- |
| workflow | 区分不同业务场景 | `foundation`, `artifact`, `project_creation` |
| provider | 区分 LLM 提供商  | `openai`, `anthropic`                        |
| model    | 区分具体模型     | `gpt-4`, `claude-3`                          |
| status   | 区分成功/失败    | `success`, `error`                           |
| type     | 区分 token 类型  | `prompt`, `completion`                       |
| tool     | 工具名称         | `artifact_get_active`, `artifact_search`     |

### 3.4 典型查询场景

```promql
# 查询特定workflow的LLM调用失败率
sum(rate(llm_call_total{workflow="foundation",status="error"}[5m]))
  / sum(rate(llm_call_total{workflow="foundation"}[5m]))

# 查询P95延迟
histogram_quantile(0.95,
  sum(rate(llm_call_duration_bucket{workflow="artifact",model="gpt-4"}[5m])) by (le))

# 查询Token使用趋势
sum(rate(llm_tokens_used_total{provider="openai",type="prompt"}[5m]))

# 查询最耗时的Tool
topk(10, sum(rate(tool_call_duration_sum[5m])) by (tool))
```

---

## 四、OpenTelemetry 集成

### 4.1 链路追踪配置

在 OnStart 阶段创建 Span：

```go
ctx, _ = otel.Tracer("eino").Start(ctx, "llm.generate",
    trace.WithAttributes(
        attribute.String("eino.workflow", workflow),
        attribute.String("llm.provider", provider),
        attribute.String("llm.model", modelName),
    ),
)
```

### 4.2 Span 属性记录

```go
// 在OnEnd中记录Token使用
span.SetAttributes(
    attribute.Int("llm.prompt_tokens", output.TokenUsage.PromptTokens),
    attribute.Int("llm.completion_tokens", output.TokenUsage.CompletionTokens),
)

// 在OnError中记录错误状态
span.RecordError(err)
span.SetStatus(codes.Error, err.Error())
```

### 4.3 分布式追踪效果

通过 Eino 的全局 Callbacks 机制，每个 LLM 调用和 Tool 调用都会自动生成对应的 Span，形成完整的调用链路：

```
LLM.generate (foundation)
├── Span: llm.generate
│   ├── Attributes: workflow=foundation, provider=openai, model=gpt-4
│   └── Events: [token_usage: prompt=100, completion=50]
│
└── Tool.invoke (artifact_get_active)
    ├── Span: tool.invoke
    ├── Attributes: tool.name=artifact_get_active, workflow=foundation
    └── Events: [duration: 0.05s]
```

### 4.4 Span 生命周期管理

| 阶段    | 操作                 | 说明                          |
| ------- | -------------------- | ----------------------------- |
| OnStart | `Tracer.Start()`     | 创建新的 Span 并设为当前 Span |
| OnEnd   | `span.End()`         | 结束 Span，发送到追踪后端     |
| OnError | `span.RecordError()` | 记录错误事件到 Span           |
| OnError | `span.SetStatus()`   | 设置 Span 状态为 Error        |

---

## 五、不同类型 Callbacks 对比

### 5.1 按组件类型分类

| 组件类型  | 回调处理器           | 监控重点 | 典型指标                       |
| --------- | -------------------- | -------- | ------------------------------ |
| ChatModel | ModelCallbackHandler | LLM 调用 | token 使用、响应时间、模型名称 |
| Tool      | ToolCallbackHandler  | 工具调用 | 调用次数、执行耗时、工具名称   |
| Retriever | (未在本项目实现)     | 向量检索 | 召回数量、检索耗时             |
| Embedder  | (未在本项目实现)     | 向量生成 | embedding 耗时、token 使用     |

### 5.2 按生命周期阶段分类

| 阶段    | 触发时机       | 适用场景                                 |
| ------- | -------------- | ---------------------------------------- |
| OnStart | 组件执行前     | 初始化追踪、记录开始时间、收集输入元数据 |
| OnEnd   | 组件成功执行后 | 记录结果、计算耗时、采集输出指标         |
| OnError | 组件执行失败时 | 记录错误、更新失败指标、标记错误状态     |

### 5.3 适用场景分析

**ChatModel 回调适用场景**：

1. **成本监控**：Token 使用统计，用于成本核算和预算控制
2. **性能优化**：识别慢查询，优化提示词或切换模型
3. **质量分析**：对比不同模型/Provider 的输出质量
4. **异常告警**：失败率超过阈值时触发告警

**Tool 回调适用场景**：

1. **工具效能分析**：统计各工具调用频率和耗时
2. **工具优化**：识别低效工具，考虑合并或优化
3. **调用链路分析**：理解 Graph 执行过程中的工具调用模式
4. **资源规划**：根据工具调用量规划资源容量

### 5.4 注册方式对比

```go
// ChatModel回调注册
handler := cbtemplate.NewHandlerHelper().
    ChatModel(newChatModelCallbackHandler()).
    Handler()

// Tool回调注册
handler := cbtemplate.NewHandlerHelper().
    Tool(newToolCallbackHandler()).
    Handler()

// 多组件组合注册
handler := cbtemplate.NewHandlerHelper().
    ChatModel(newChatModelCallbackHandler()).
    Tool(newToolCallbackHandler()).
    Retriever(newRetrieverCallbackHandler()).  // 如有需要
    Embedder(newEmbedderCallbackHandler()).    // 如有需要
    Handler()
```

---

## 六、在工作流中的应用实践

### 6.1 EinoFactory 集成

`internal/infrastructure/llm/eino_factory.go` 是项目的 LLM 客户端工厂：

```go
// 使用Eino的OpenAI适配器
chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
    APIKey:      providerCfg.APIKey,
    BaseURL:     providerCfg.BaseURL,
    Model:       providerCfg.Model,
    MaxTokens:   &providerCfg.MaxTokens,
    Temperature: ptrFloat32(float32(providerCfg.Temperature)),
    Timeout:     providerCfg.Timeout,
})
```

### 6.2 Context 注入模式

在业务调用时，需要先注入 workflow 和 provider 信息：

```go
// 设定生成场景
ctx := eino.WithWorkflowProvider(context.Background(),
    "foundation",  // workflow
    "openai",      // provider
)

// 调用LLM（自动触发Callbacks）
result, err := chatModel.Generate(ctx, messages)
```

### 6.3 完整调用链路

```
API层
  │
  ▼ WithWorkflowProvider(ctx, "foundation", "openai")
  │
应用层：foundation_generator.go
  │
  ▼ 调用 EinoFactory.Get(ctx, "")
  │
基础设施层：eino_factory.go (openai.NewChatModel)
  │
  ▼ chatModel.Generate(ctx, messages)
  │
Eino框架内部
  │
  ▼ 触发全局Callbacks
  │
  ├── OnStart: 创建Span，记录开始时间
  ├── OnEnd: 记录metrics，结束Span
  └── OnError: 记录错误指标
```

### 6.4 实际指标采集示例

当用户执行一次 Foundation 设定生成时：

**1. 请求进入**：API 层注入 `workflow=foundation, provider=openai`

**2. LLM 调用**：

- **OnStart**：记录开始时间`t0`，创建 Span
- **LLM 执行**：耗时 2.3 秒，消耗 `prompt_tokens=150`, `completion_tokens=80`
- **OnEnd**：记录指标
  ```
  llm_call_total{workflow=foundation,provider=openai,model=gpt-4,status=success} +1
  llm_call_duration{workflow=foundation,provider=openai,model=gpt-4} += 2.3
  llm_tokens_used{workflow=foundation,provider=openai,model=gpt-4,type=prompt} += 150
  llm_tokens_used{workflow=foundation,provider=openai,model=gpt-4,type=completion} += 80
  ```
- **结束 Span**：记录 token 使用到 trace attributes

**3. Tool 调用**（如需要）：

- **OnStart**：记录开始时间`t1`，创建 Span
- **Tool 执行**：耗时 0.05 秒
- **OnEnd**：记录指标
  ```
  tool_call_total{workflow=foundation,tool=artifact_get_active,status=success} +1
  tool_call_duration{workflow=foundation,tool=artifact_get_active} += 0.05
  ```
- **结束 Span**

### 6.5 在不同场景中的使用

**Foundation 场景**：

```go
ctx := eino.WithWorkflowProvider(ctx, "foundation", providerName)
foundation, err := f.chatModel.Generate(ctx, messages)
```

**Artifact 场景**（含 ToolCalling）：

```go
ctx := eino.WithWorkflowProvider(ctx, "artifact", providerName)
result, err := graph.Run(ctx)  // Graph内自动触发LLM和Tool回调
```

**ProjectCreation 场景**：

```go
ctx := eino.WithWorkflowProvider(ctx, "project_creation", providerName)
project, err := chain.Run(ctx)
```

---

## 七、最佳实践总结

### 7.1 设计模式

| 模式         | 应用                 | 说明                     |
| ------------ | -------------------- | ------------------------ |
| 单例初始化   | `sync.Once`          | 确保全局回调只初始化一次 |
| 构建器模式   | `NewHandlerHelper()` | 链式组合不同组件处理器   |
| Context 传递 | 自定义 Context Key   | 传递业务元数据           |
| 防御性编程   | nil/空值检查         | 避免 panic               |

### 7.2 性能注意事项

1. **避免重量级操作**：回调中不要执行耗时操作
2. **谨慎记录数据**：不记录 prompt 原文，只记录长度/Hash
3. **Span 管理**：确保每个 Span 都被正确结束
4. **指标基数控制**：避免高基数标签导致 Prometheus 性能问题

### 7.3 可扩展性

**新增组件类型**：

```go
handler := cbtemplate.NewHandlerHelper().
    ChatModel(newChatModelCallbackHandler()).
    Tool(newToolCallbackHandler()).
    Retriever(newRetrieverCallbackHandler()).  // 新增
    Embedder(newEmbedderCallbackHandler()).    // 新增
    Handler()
```

**新增指标**：

```go
// 在 pkg/metrics/metrics.go 中定义
var NewMetric = promauto.NewCounterVec(
    prometheus.CounterOpts{
        Namespace: namespace,
        Subsystem: "new_subsystem",
        Name:      "metric_name",
        Help:      "Help text",
    },
    []string{"label1", "label2"},
)

// 在 handler.go 中记录
metrics.NewMetric.WithLabelValues(label1, label2).Inc()
```

**新增追踪属性**：

```go
attrs := []attribute.KeyValue{
    attribute.String("eino.workflow", workflow),
    attribute.String("new.attribute", value),  // 新增
}
```

### 7.4 与项目架构的结合

```
┌─────────────────────────────────────────────────────────────────┐
│                     可观测性层 (observability)                   │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                   eino callbacks                          │   │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐   │   │
│  │  │ ModelCallback│  │ToolCallback  │  │(Retriever)...│   │   │
│  │  └──────────────┘  └──────────────┘  └──────────────┘   │   │
│  └─────────────────────────────────────────────────────────┘   │
│                              │                                  │
│                              ▼                                  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                   metrics (Prometheus)                   │   │
│  │  tracer (OpenTelemetry)                                  │   │
│  └─────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                     工作流层 (workflow)                          │
│  ┌────────────┐  ┌────────────┐  ┌────────────────────────┐   │
│  │ ChatTemplate│  │   Chain    │  │ Graph + ToolCalling    │   │
│  └────────────┘  └────────────┘  └────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                     基础设施层 (infrastructure)                   │
│  ┌────────────┐  ┌────────────┐  ┌────────────────────────┐   │
│  │EinoFactory │  │EinoEmbedder│  │(其他Eino组件)           │   │
│  └────────────┘  └────────────┘  └────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

Callbacks 机制作为可观测性层的核心，通过标准化的事件钩子，将底层的 LLM 调用和 Tool 调用与上层的 Metrics 采集、链路追踪有机串联，实现了"零侵入"的可观测性增强。

### 7.5 初始化位置

根据设计文档，EinoCallbacks 在以下入口初始化：

```go
// cmd/api-gateway/main.go
func main() {
    // 初始化Eino全局callbacks
    eino.Init()
    // ... 其他初始化
}

// cmd/job-worker/main.go
func main() {
    // 初始化Eino全局callbacks
    eino.Init()
    // ... 其他初始化
}
```

---

## 附录

### A. 相关文件索引

| 文件路径                                      | 说明                      |
| --------------------------------------------- | ------------------------- |
| `internal/observability/eino/init.go`         | 全局 Callbacks 初始化     |
| `internal/observability/eino/handler.go`      | ChatModel/Tool 回调处理器 |
| `internal/observability/eino/context.go`      | Context 传递工具函数      |
| `pkg/metrics/metrics.go`                      | Prometheus 指标定义       |
| `internal/infrastructure/llm/eino_factory.go` | Eino LLM 工厂             |
| `docs/22-Eino设定生成工作流重构设计.md`       | Eino 架构设计文档         |

### B. 监控指标清单

| 指标名称                             | 类型      | 描述             |
| ------------------------------------ | --------- | ---------------- |
| `z_novel_llm_call_total`             | Counter   | LLM 调用总次数   |
| `z_novel_llm_call_duration_seconds`  | Histogram | LLM 调用耗时     |
| `z_novel_llm_tokens_used_total`      | Counter   | LLM Token 使用量 |
| `z_novel_tool_call_total`            | Counter   | Tool 调用总次数  |
| `z_novel_tool_call_duration_seconds` | Histogram | Tool 调用耗时    |

### C. 参考资源

- [Eino 官方文档](https://eino.ai/)
- [Eino GitHub](https://github.com/cloudwego/eino)
- [OpenTelemetry Go SDK](https://pkg.go.dev/go.opentelemetry.io/otel)
- [Prometheus Go Client](https://pkg.go.dev/github.com/prometheus/client_golang/prometheus)
