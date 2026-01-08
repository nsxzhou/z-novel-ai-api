# z-novel-ai-api（后端）— 当前状态与目录结构

更新时间：2026-01-08

本文档面向"维护/二次开发"，目标是让你在 1 分钟内明确：

1. 默认能跑什么（CRUD / 核心能力）
2. 代码入口在哪（服务入口 / 路由 / 数据层）
3. 仓库结构是什么（目录树）

---

## 1. 已完成的核心功能

### 1.1 租户、认证与用户体系

本任务已完整实现租户 (Tenant)、认证 (Auth) 和用户 (User) 模块的核心功能，建立了稳健的多租户隔离基础和精细化的权限管理机制。

- **多租户架构:** 基于 PostgreSQL RLS (Row Level Security) 实现强数据隔离。
- **认证机制:** 采用 JWT 双 Token 方案，Refresh Token 通过 `HttpOnly` Cookie 传递。
- **RBAC 权限控制:** 实现了静态的 RBAC0 模型，支持角色到权限的映射及显式的读写权限分离控制。
- **安全设计:** 注册流程默认关闭，需在租户设置中显式开启。所有认证和业务请求均需明确提供 `tenant_id`。
- **主要入口:**
  - Handlers: `internal/interfaces/http/handler/auth.go`, `user.go`, `tenant.go`
  - RLS 中间件: `internal/interfaces/http/middleware/db_transaction.go`
  - RBAC 中间件: `internal/interfaces/http/middleware/rbac.go`

### 1.2 对话驱动小说创作（完整闭环）

已实现"从模糊想法到完整小说设定"的全链路 AI 辅助创作能力。

#### 1.2.1 项目孵化（Project Creation）— **新增**

通过 4 阶段对话引导（discover → narrow → draft → confirm），帮助用户将模糊想法转化为正式项目。

- **主要入口:**
  - HTTP Handler: `internal/interfaces/http/handler/project_creation.go`
  - Generator: `internal/application/story/project_creation_generator.go`
  - Entity: `internal/domain/entity/project_creation.go`
- **HTTP API:**
  - `POST /v1/project-creation-sessions`：创建孵化会话
  - `POST /v1/project-creation-sessions/:sid/messages`：发送对话指令
  - `GET /v1/project-creation-sessions/:sid`：获取会话状态
  - `GET /v1/project-creation-sessions/:sid/turns`：获取对话轮次
- **核心特性:**
  - 满足条件后自动创建 Project
  - 自动关联新的长期会话（project_session_id）
  - 状态机控制流程，防止 AI 幻觉触发误操作

#### 1.2.2 设定迭代（Artifact Flow）

支持在已有项目上通过长期对话反复打磨设定（世界观/角色/大纲）。

- **主要入口:**
  - HTTP Handler: `internal/interfaces/http/handler/conversation.go`、`internal/interfaces/http/handler/artifact.go`
  - Generator: `internal/application/story/artifact_generator.go`
- **HTTP API:**
  - `POST /v1/projects/:pid/sessions`：创建长期会话
  - `POST /v1/projects/:pid/sessions/:sid/messages`：发送任务指令
  - `GET /v1/projects/:pid/artifacts`：构件列表
  - `GET /v1/projects/:pid/artifacts/:aid/versions`：版本列表
  - `POST /v1/projects/:pid/artifacts/:aid/rollback`：回滚到指定版本
- **任务类型 (Task):**
  - `novel_foundation`: 小说基底（标题 + 简介）
  - `worldview`: 世界观设定
  - `characters`: 角色与关系网络
  - `outline`: 卷章大纲

#### 1.2.3 一揽子生成（Foundation 初始化）

一次性生成完整设定包（Plan → Validate → Apply），适合项目冷启动。

- **主要入口:**
  - HTTP Handler: `internal/interfaces/http/handler/foundation.go`
  - DTO: `internal/interfaces/http/dto/foundation.go`
  - Plan/Generate/Validate/Apply: `internal/application/story/foundation_*.go`
- **HTTP API:**
  - `POST /v1/projects/:pid/foundation/preview`
  - `GET|POST /v1/projects/:pid/foundation/stream`
  - `POST /v1/projects/:pid/foundation/generate`（支持 `Idempotency-Key`）
  - `POST /v1/projects/:pid/foundation/apply`

#### 1.2.4 Eino 编排升级（Chain / Graph / ToolCalling / ChatTemplate / Callback）— **新增**

在不改变现有 API 与数据结构的前提下，将设定生成链路升级为可组合、可观测、可扩展的 Eino 工作流：

- Prompt 统一管理（go:embed ChatTemplate）：`internal/workflow/prompt/*`（含 `artifact_v2` / `artifact_patch_v1`）
- Foundation / ProjectCreation：Chain 重构主路径（Prompt → LLM → Parse → Validate → Normalize）
- Artifact：Graph + ToolCalling（ReAct 回路）按需获取上下文 + 校验失败修复回路（Validate → Repair → Re-run）
- 增量 Patch 模式（JSON Patch）：先支持 `novel_foundation/worldview` 顶层字段；服务端应用 patch 后仍输出完整 JSON
- 上下文滚动摘要（Redis）：长会话自动压缩历史（summary + recent turns）并注入 Prompt，降低 token 成本
- 可观测性：Eino 全局 callbacks + Prometheus 指标：`internal/observability/eino/*`
- 安全：ProjectCreation 增加服务端“确定性确认门控”，避免模型幻觉触发误创建

---

## 2. 当前仓库目标与运行模式

### 2.1 默认：CRUD + 对话创作（当前仓库默认）

- 目标：基础 CRUD + 对话驱动创作（孵化/迭代/Foundation）可运行；章节生成/检索等核心业务仍保留占位实现。
- 开关：`features.core.enabled=false`
- 关键行为：API Gateway 启动时不会强依赖核心 gRPC 服务可达。

### 2.2 core（仅用于后续逐步实现核心能力时开启）

- 开关：`features.core.enabled=true`
- 说明：当前仓库仍以占位为主；开启 core 只会允许网关尝试建立 gRPC client 连接。

---

## 3. 项目目录结构（当前仓库）

```text
z-novel-ai-api/
├── cmd/                         # 各进程入口（main.go）
│   ├── api-gateway/             # HTTP API 网关
│   ├── job-worker/              # 异步任务执行器
│   ├── bootstrap/               # 系统初始化（创建默认租户与管理员）
│   ├── story-gen-svc/           # gRPC：生成服务（占位）
│   ├── rag-retrieval-svc/       # gRPC：检索服务（占位）
│   ├── memory-svc/              # gRPC：记忆服务（占位）
│   ├── validator-svc/           # gRPC：校验服务（占位）
│   ├── admin-svc/               # 预留
│   └── file-svc/                # 预留
│
├── internal/                    # 私有应用代码
│   ├── config/                  # 配置结构/加载
│   ├── domain/                  # 领域层（实体/仓储接口/领域服务）
│   ├── infrastructure/          # 基础设施（DB/Cache/MQ/LLM）
│   ├── application/             # 应用层（story/quota）
│   ├── interfaces/              # 接口适配层（http/grpc）
│   └── wire/                    # 依赖注入（Wire）
│
├── migrations/                  # 迁移（postgres/milvus）
├── configs/                     # 配置文件
├── api/                         # API 定义（proto/openapi）
├── pkg/                         # 可复用公共包
├── docs/                        # 设计文档
└── test/                        # 测试目录
```

---

## 4. HTTP API 清单

### 4.1 已完成：基础 CRUD

- Projects / Volumes / Chapters / Entities / Relations / Events / Jobs
- Auth: register / login / refresh / logout
- Users / Tenants
- System: /health, /ready, /live

### 4.2 已完成：对话驱动创作

- **项目孵化**：`/v1/project-creation-sessions/*`
- **设定迭代**：`/v1/projects/:pid/sessions/*`、`/v1/projects/:pid/artifacts/*`
- **一揽子生成**：`/v1/projects/:pid/foundation/*`

### 4.3 占位：核心业务（统一 501）

- 章节生成：`POST /v1/projects/:pid/chapters/generate`
- 章节重新生成：`POST /v1/chapters/:cid/regenerate`
- SSE 流式生成：`GET /v1/chapters/:cid/stream`
- 检索：`POST /v1/retrieval/search`

---

## 5. 本地运行

```bash
docker compose up -d
make migrate-up
JWT_SECRET="dev-secret" FEATURES_CORE_ENABLED=false go run ./cmd/api-gateway
```

如需调用对话创作功能，请先配置 `llm.providers.*` 对应的 `api_key/base_url/model`。

---

## 6. 当前明确缺口

- `cmd/admin-svc`、`cmd/file-svc`：仅目录预留。
- `api/openapi`：为空目录。
- `test/`：仅目录骨架，无测试用例。
- **动态 RBAC**：当前权限模型为硬编码（静态）。
- **Patch 扩展**：`characters/outline` 的增量编辑策略未启用（数组型 JSON Pointer 易脆）。
- **上下文自动摘要（LLM/结构化）**：当前为轻量滚动压缩，后续可升级为更强语义压缩。

---

## 7. 后续开发方向

- **增量编辑**：支持 AI 仅输出变更部分。
- **章节生成闭环**：补齐章节正文的同步/SSE/异步路径。
- **检索与记忆**：实现向量检索与上下文召回。
- **可观测性**：落地细分指标与告警。
