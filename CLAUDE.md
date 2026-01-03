# z-novel-ai-api（后端）— 当前状态与目录结构

更新时间：2026-01-03

本文档面向“维护/二次开发”，目标是让你在 1 分钟内明确：
1) 默认能跑什么（CRUD / 核心能力）  
2) 代码入口在哪（服务入口 / 路由 / 数据层）  
3) 仓库结构是什么（目录树）  

---

## 1. 当前仓库目标与运行模式（必须先看）

### 1.1 默认：CRUD-only（当前仓库默认）
- 目标：基础 CRUD 接口可运行；生成/检索/流式等核心业务仅保留占位实现（HTTP 返回 501，gRPC 返回 Unimplemented）。
- 开关：`features.core.enabled=false`
  - 配置文件：`configs/config.yaml`
  - 环境变量覆盖：`FEATURES_CORE_ENABLED=false`
- 关键行为：API Gateway 启动时不会强依赖核心 gRPC 服务可达（避免“只跑 CRUD 也起不来”）。

### 1.2 core（仅用于后续逐步实现核心能力时开启）
- 开关：`features.core.enabled=true`
- 说明：当前仓库仍以占位为主；开启 core 只会允许网关尝试建立 gRPC client 连接，但核心 HTTP/gRPC 仍是占位逻辑，需你后续逐步替换。

---

## 2. 项目目录结构（当前仓库）

说明：这是“真实目录”，用于定位入口与职责边界；以 Go 工程习惯做分层（domain / infrastructure / interfaces）。

```text
z-novel-ai-api/
├── cmd/                         # 各进程入口（main.go）
│   ├── api-gateway/             # HTTP API 网关（CRUD + 占位核心接口）
│   │   └── main.go
│   ├── job-worker/              # 异步任务执行器（当前仍可编译；业务闭环待恢复/实现）
│   │   └── main.go
│   ├── story-gen-svc/           # gRPC：生成服务（占位）
│   │   └── main.go
│   ├── rag-retrieval-svc/       # gRPC：检索服务（占位）
│   │   └── main.go
│   ├── memory-svc/              # gRPC：记忆服务（占位）
│   │   └── main.go
│   ├── validator-svc/           # gRPC：校验服务（占位）
│   │   └── main.go
│   ├── admin-svc/               # 预留（当前为空目录）
│   └── file-svc/                # 预留（当前为空目录）
│
├── internal/                    # 私有应用代码
│   ├── config/                  # 配置结构/加载（Viper + env 覆盖）
│   ├── domain/                  # 领域层（实体/仓储接口/领域服务）
│   │   ├── entity/
│   │   ├── repository/
│   │   └── service/
│   ├── infrastructure/          # 基础设施（DB/Cache/MQ/外部客户端）
│   │   ├── persistence/
│   │   │   ├── postgres/        # Postgres 仓储实现（CRUD + RLS）
│   │   │   ├── redis/
│   │   │   └── milvus/
│   │   ├── messaging/           # Redis Streams Producer/Consumer
│   │   ├── embedding/           # embedding HTTP 客户端
│   │   └── llm/                 # 预留（当前未实现核心 LLM 逻辑）
│   ├── interfaces/              # 接口适配层
│   │   ├── http/                # Gin（router/handler/middleware/dto）
│   │   └── grpc/                # gRPC（client/server）
│   ├── wire/                    # 依赖注入（Wire）
│   └── workflow/                # 预留（工作流/编排骨架）
│
├── migrations/                  # 迁移
│   ├── postgres/
│   └── milvus/
│
├── configs/                     # 配置文件（默认 config.yaml + 环境覆盖）
├── api/                         # API 相关产物/定义
│   ├── proto/                   # proto 源与生成物
│   └── openapi/                 # 预留（当前为空）
├── pkg/                         # 可复用公共包（logger/tracer/errors/utils 等）
├── deployments/                 # 部署编排（docker/helm/k8s）
├── docs/                        # 设计文档（架构/规范）
├── scripts/                     # 辅助脚本
└── test/                        # 测试目录骨架（当前无 *_test.go 用例）
```

---

## 3. 关键入口与调用路径

### 3.1 API Gateway（HTTP）
- 入口：`cmd/api-gateway/main.go`
- 路由注册：
  - Engine + 中间件：`internal/interfaces/http/router/router.go`
  - v1 业务路由：`internal/interfaces/http/router/routes.go`
- 数据访问：handler 直接依赖 `internal/infrastructure/persistence/postgres/*_repo.go`（当前仍是“直连仓储”，未迁移为对应微服务）。

### 3.2 gRPC 服务（均为占位）
- 服务端实现：`internal/interfaces/grpc/server/*.go`（统一返回 `codes.Unimplemented`）
- 入口（启动 gRPC server）：`cmd/*-svc/main.go`

### 3.3 依赖注入（Wire）
- `wire.InitializeApp`：`internal/wire/wire_gen.go`
- gRPC client 拨号逻辑：`internal/wire/grpc_clients.go`
  - `features.core.enabled=false` 时：返回 nil client，避免启动强依赖

---

## 4. HTTP API 清单（按“CRUD vs 核心占位”分组）

路由总表：`internal/interfaces/http/router/routes.go`

### 4.1 已完成：基础 CRUD（可运行）
- Projects：`GET|POST /v1/projects`，`GET|PUT|DELETE /v1/projects/:pid`，`GET|PUT /v1/projects/:pid/settings`
- Volumes：`GET|POST /v1/projects/:pid/volumes`，`POST /v1/projects/:pid/volumes/reorder`，`GET|PUT|DELETE /v1/volumes/:vid`
- Chapters：`GET|POST /v1/projects/:pid/chapters`，`GET|PUT|DELETE /v1/chapters/:cid`
- Entities：`GET|POST /v1/projects/:pid/entities`，`GET|PUT|DELETE /v1/entities/:eid`，`PUT /v1/entities/:eid/state`，`GET /v1/entities/:eid/relations`
- Events：`GET|POST /v1/projects/:pid/events`，`GET|PUT|DELETE /v1/events/:evid`
- Relations：`GET|POST /v1/projects/:pid/relations`，`GET|PUT|DELETE /v1/relations/:rid`
- Jobs（查询/取消/项目内列表）：`GET /v1/projects/:pid/jobs`，`GET /v1/jobs/:jid`，`DELETE /v1/jobs/:jid`
- Users/Tenants（当前视角）：`GET|PUT /v1/users/me`，`GET /v1/users`，`GET|PUT /v1/tenants/current`
- System：`GET /health`，`GET /ready`，`GET /live`

### 4.2 占位：核心业务（统一 501）
- 章节生成（异步入口）：`POST /v1/projects/:pid/chapters/generate`
- 章节重新生成：`POST /v1/chapters/:cid/regenerate`
- SSE 流式生成：`GET /v1/chapters/:cid/stream`
- 检索：`POST /v1/retrieval/search`，`POST /v1/retrieval/debug`

---

## 5. 数据层与多租户（PostgreSQL RLS）

### 5.1 RLS 约束与上下文设置
- RLS 在迁移中启用：`migrations/postgres/000007_enable_rls.up.sql`
- 网关侧通过“请求级事务 + 事务内 set_config”确保 `app.current_tenant_id` 生效：
  - 中间件：`internal/interfaces/http/middleware/db_transaction.go`

### 5.2 EntityStateRepository（实体状态历史）
- 领域接口定义：`internal/domain/repository/entity_repository.go`
- 表结构：`migrations/postgres/000005_create_entities_relations.up.sql`（`entity_states`）
- Postgres 实现已提供：`internal/infrastructure/persistence/postgres/entity_state_repo.go`
- 现状说明：仓储已具备；但当前业务层仍以 `entities.current_state` 为主，是否落“历史”需后续 memory-svc/写入链路决定。

---

## 6. 本地运行（CRUD-only）

前置依赖：PostgreSQL + Redis（API Gateway 依赖 DB 与限流/缓存）。

```bash
docker compose up -d
make migrate-up
JWT_SECRET="dev-secret" FEATURES_CORE_ENABLED=false go run ./cmd/api-gateway
```

---

## 7. 当前明确缺口（避免误判）

- `cmd/admin-svc`、`cmd/file-svc`：仅目录预留，暂无入口与实现。
- `api/openapi`：为空目录，当前未提供 OpenAPI 产物。
- `test/`：仅目录骨架，仓库无 `*_test.go` 用例。
