# 22 - Eino 设定生成工作流重构设计（Chain / Graph / ToolCalling / ChatTemplate / Callback）

更新时间：2026-01-07

> 目标：在不破坏现有 HTTP API 与数据结构的前提下，将“设定生成”从手写 prompt + 手写流程，升级为可组合、可观测、可扩展的 Eino 编排体系。

---

## 1. 背景与现状

当前项目仅使用 Eino 的 OpenAI 适配器（`eino-ext/components/model/openai`）作为 LLM 调用封装，业务侧仍以“手写 prompt + 手写解析/校验/降级”为主：

- LLM 工厂：`internal/infrastructure/llm/eino_factory.go`
- 设定生成（Foundation / Artifact / ProjectCreation）：
  - `internal/application/story/foundation_generator.go`
  - `internal/application/story/artifact_generator.go`
  - `internal/application/story/project_creation_generator.go`

现状的主要痛点：

1) Prompt 分散，缺少统一版本管理与复用。
2) 分支/回路（降级、校验失败重试、工具调用）靠业务代码 if/for，难以维护扩展。
3) 无 ToolCalling：上下文只能“全量塞入”，项目越大 token 成本越高。
4) 可观测性缺少组件级标准指标：无法按 node/model/tool 分析耗时与 token。

---

## 2. 设计目标与边界（KISS / YAGNI）

### 2.1 目标（本次要落地）

1) **ChatTemplate 统一 Prompt 管理**：集中化、可版本化（go:embed）。
2) **Chain 重构主路径**：Prompt → LLM → Parse → Validate → Normalize。
3) **Graph 承载复杂流程**：分支、回路、工具调用（ReAct）等。
4) **ToolCalling 自主检索**：按需获取当前设定（世界观/角色/大纲/当前构件）。
5) **Eino Callback 增强可观测性**：统一 metrics/tracing/logs。
6) **项目孵化确定性确认门控**：服务端基于用户输入与阶段做强约束，避免模型幻觉触发创建。

### 2.2 非目标（明确不做）

- 不改变现有 HTTP API 与 DB schema。
- 不一次性接入完整 RAG（检索服务当前仍为占位实现）；工具调用先基于“系统已拿到的设定快照”。

---

## 3. 总体架构

### 3.1 分层与代码组织

- Prompt 管理：`internal/workflow/prompt`（go:embed）
- 工作流编排（一期落地以“兼容现有入口”为优先）：
  - 目前 Chain/Graph 实现内嵌在 `internal/application/story/*_generator.go`，对外 API 不变
  - 后续可再抽离到 `internal/workflow/story/setting` 以强化分层（非阻塞）
- 可观测性：`internal/observability/eino`（全局 callbacks 初始化）

### 3.2 基础设施与兼容性

- `EinoFactory` 统一提供 `model.BaseChatModel`，可按需断言为 `model.ToolCallingChatModel`（支持 ToolCalling）。
- 现有 handlers/job-worker 不改对外行为；内部将调用 workflow 层实现。

---

## 4. Prompt 管理：ChatTemplate + go:embed

### 4.1 方案

- 所有 prompt 以“PromptID + 版本号”命名，例如：
  - `foundation_plan_v1`
  - `artifact_v1`
  - `project_creation_v1`
- Prompt 文件以 `templates/*.txt` 存储，通过 `go:embed` 内嵌进二进制。
- `PromptRegistry` 提供 `ChatTemplate(id)`，返回 `prompt.FromMessages(schema.FString, ...)`。

### 4.2 好处

- Prompt 统一入口，便于审计、回滚与复用。
- 版本化可控，避免线上 prompt 漂移。

---

## 5. Chain：主路径流水线

### 5.1 统一流水线（Foundation / ProjectCreation）

以 Foundation 为例（ProjectCreation 同理）：

1) BuildVars：将输入组装为模板变量（附件块、项目字段等）
2) ChatTemplate.Format：生成 messages
3) LLM.Generate：调用模型（优先 json_schema；不支持则降级 prompt-only）
4) Parse：抽取 JSON 对象
5) Validate / Normalize：强校验后输出标准 JSON + meta

### 5.2 原则

- 单一职责：每个节点只做一件事。
- DRY：降级策略、meta 提取复用为通用函数/节点。

---

## 6. Graph：复杂分支与回路（Artifact：ToolCalling ReAct）

### 6.1 ReAct 回路（工具调用）

Artifact 迭代中引入 ToolCalling，采用 ReAct 样式回路：

```mermaid
flowchart TD
  start([Input]) --> init[InitState + Template]
  init --> model[LLM.Generate (ToolCalling)]
  model -->|has tool_calls| tools[ToolsNode.Invoke]
  tools --> model
  model -->|no tool_calls| finalize[Parse + Validate + Normalize]
  finalize --> end([Output])
```

### 6.2 工具集（一期，默认开启）

- `artifact_get_active(type)`：返回指定类型的当前激活设定 JSON（来自系统已有快照）
- `artifact_search(query, type?, top_k?)`：在设定 JSON 中做关键词检索（便于精准定位）
- `project_get_brief()`：返回项目标题/简介/当前任务类型等摘要信息

约束：

- 工具只读，不接收 tenant/project 参数（由服务端上下文注入）。
- 设定最大步数/最大工具轮次，避免死循环与成本失控。
- 对不支持 ToolCalling 的 provider 自动降级到“无工具”的一次性生成。

---

## 7. Callback：可观测性增强

### 7.1 初始化方式

在各进程启动时（api-gateway / job-worker）注册全局回调：

- `callbacks.AppendGlobalHandlers(...)`

### 7.2 指标建议

- `llm_requests_total{workflow,provider,model,success}`
- `llm_latency_ms_bucket{workflow,provider,model}`
- `llm_tokens_prompt_total{provider,model}`
- `llm_tokens_completion_total{provider,model}`
- `tool_calls_total{tool,success}`
- `tool_latency_ms_bucket{tool}`

约束：

- 不记录 prompt 原文；只记录长度/hash、token、耗时、错误码等元信息。

---

## 8. 项目孵化：确定性确认门控

问题：模型可能在未确认时输出 `create_project`，导致误创建。

方案：服务端增加确定性门控：

1) 必须处于 `confirm` 阶段（会话状态机约束）
2) 必须从用户输入中检测到明确确认意图（否定词优先拦截）
3) 否则即使模型输出 `create_project` 也不执行创建，改为继续要求用户确认

---

## 9. 实施路线（按风险递进）

### Phase 1（低风险：立即收益）

- ✅ PromptRegistry + ChatTemplate 落地（go:embed）
- ✅ Foundation / ProjectCreation：Chain 重构 Generate 主路径
- ✅ 全局 Callback 初始化与基础 metrics
- ✅ ProjectCreation 确定性确认门控

### Phase 2（中风险：质量提升）

- ✅ Artifact：Graph + ToolCalling ReAct 回路（不支持 tools 的 provider 自动降级）
- ✅ 工具一期：get_active / search / get_brief

### Phase 3（高收益：规模化）

- ⏳ 校验失败修复回路（Validate → Repair → Re-run）
- ⏳ 逐步把“全量上下文塞入”替换为“按需工具获取 + 自动摘要”
- ⏳ 检索服务（RAG）落地后，将 `artifact_search` 升级为向量召回 + 结构化片段返回

---

## 10. 验收标准

- 兼容：现有 API 行为与字段不变（除项目孵化误创建被拦截属于安全修复）。
- 可观测：能看到 LLM 调用次数、耗时、token、工具调用次数与耗时。
- 可维护：Prompt 统一管理，可明确定位到 PromptID 与版本。

---

## 11. 当前落地状态（2026-01-07）

### 11.1 已完成

- ✅ 设计文档：本文件。
- ✅ PromptRegistry（go:embed）：`internal/workflow/prompt/registry.go` + `internal/workflow/prompt/templates/*.txt`
- ✅ Foundation（Chain）：`internal/application/story/foundation_generator.go`
- ✅ ProjectCreation（Chain）：`internal/application/story/project_creation_generator.go`
- ✅ Artifact（Graph + ToolCalling ReAct）：`internal/application/story/artifact_generator.go`
- ✅ 工具一期（只读）：`internal/application/story/artifact_tools.go`
- ✅ Eino 可观测性（全局 callbacks）：`internal/observability/eino/*`
  - 初始化入口：`cmd/api-gateway/main.go`、`cmd/job-worker/main.go`
- ✅ ProjectCreation 确定性确认门控：`internal/interfaces/http/handler/project_creation.go`
- ✅ `EinoFactory` 输出类型升级为 `model.BaseChatModel`（支持 ToolCalling）：`internal/infrastructure/llm/eino_factory.go`

### 11.2 待完成（下一步）

- ⏳ Artifact “校验失败修复回路”（Graph 内 Validate → Repair → Re-run），用于减少不合规 JSON 直接失败。
- ⏳ “上下文自动摘要”（长会话压缩），与 ToolCalling 结合降低 token 成本。
- ⏳ `artifact_search` 从“字符串包含”演进为“结构化定位 + RAG 召回”（取决于检索服务落地）。
- ⏳ 进一步分层：将 Chain/Graph 从 `internal/application/story` 抽离到 `internal/workflow/story/setting`（不影响功能，但提升结构清晰度）。

### 11.3 本地验证

在沙箱环境下建议显式指定 `GOCACHE` 到工作区：

```bash
GOCACHE="$(pwd)/.gocache" go test ./...
```
