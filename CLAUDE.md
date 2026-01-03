# z-novel-ai-api（AI 小说生成后端）当前状态与待办清单

本文档用于快速回答两个问题：
1) 项目目前是什么状态（能跑哪些链路、哪些是占位/半成品）  
2) 还差什么（按优先级列出待完成项，便于下一轮工程迭代）

更新时间：2026-01-03

---

## 1. 项目目标（简述）

构建一个面向“多租户 AI 小说生成”的后端系统，采用 API Gateway（HTTP）+ 内部 gRPC 微服务 + 异步任务（Redis Streams）+ PostgreSQL RLS 的架构，实现：
- 章节生成（异步 + SSE 流式）
- RAG 检索（Embedding + Milvus）
- 校验（validator-svc）
- 记忆（memory-svc，实体/状态回写）
- 任务状态与进度（/v1/jobs/:jid 轮询）

---

## 2. 项目目前的情况

### 2.1 基础设施（docs 01-03）
- 目录结构、配置加载（Viper + ENV 替换）、OTel tracing / metrics 基础已具备。
- `docker-compose.yaml` 提供：PostgreSQL、Redis、Milvus、Jaeger、Prometheus（注意端口冲突，见“本地运行方式”）。

### 2.2 PostgreSQL（docs 04）——已补齐 RLS 生效闭环
- RLS 多租户隔离：迁移中启用，并已实现“请求级事务 + 事务内 set_config”闭环，避免连接池导致 `app.current_tenant_id` 丢失。
- 关键约束：`set_config(..., is_local=TRUE)` 是事务级变量，必须让本次请求 DB 读写落在同一事务连接上。
- HTTP 侧实现：`internal/interfaces/http/middleware/db_transaction.go`
  - 特别处理：对 SSE `/stream` 路由跳过事务（避免长连接占用事务连接池），SSE handler 采用短事务读写。

### 2.3 Redis（docs 05）
- Producer/Consumer（Streams）已实现，支持 consumer group、重试、DLQ。
- 已实现 `job-worker`：消费 `stream:story:gen`，形成“异步任务执行 + 进度回写”闭环。

### 2.4 API Gateway（docs 07-08）——章节生成与 SSE 已对齐内部链路
- `POST /v1/projects/:pid/chapters/generate`：已从 TODO 改为真实链路：
  - 创建 Chapter（`status=generating`）
  - 创建 GenerationJob（`status=pending`，`progress=0`）
  - 发布 Redis Stream 消息（`chapter_gen`）
  - 返回 `202` + `JobResponse`
- `GET /v1/chapters/:cid/stream`：已从“模拟分块”改为：
  - 调用 `story-gen-svc` 的 gRPC streaming
  - 透传 SSE 事件（`content`/`metadata`/`done`）
  - 流结束后将最终内容回写 DB（短事务）
- `GET /v1/jobs/:jid`：能查看 progress/status（由 job-worker 回写）。

### 2.5 gRPC 内部通信（docs 09）——已从“proto 存在”升级为“可运行闭环”
当前存在 4 个 gRPC 服务入口（可启动、可编译）：
- `cmd/story-gen-svc`：章节生成服务（当前为占位实现）
- `cmd/rag-retrieval-svc`：检索服务（已实现 embedding + milvus 搜索）
- `cmd/memory-svc`：记忆服务（最小可用：`UpdateEntityState` 会落库 `entities.current_state`）
- `cmd/validator-svc`：校验服务（最小可用：空内容判 invalid）

并且 API Gateway 已完成 Retrieval 的 HTTP→gRPC 调用链闭环：
- `POST /v1/retrieval/search`、`POST /v1/retrieval/debug` 通过 gRPC 调用 `rag-retrieval-svc`。
- 注意：这两个 HTTP 请求体要求 `project_id`（与 proto 和 Milvus 分区检索一致）。

### 2.6 异步任务与进度（docs 15 / walkthrough 对齐）
- 新增 `JobRepository.MarkRunning`、`JobRepository.UpdateProgress`，用于 job-worker 细粒度更新任务状态。
- `job-worker` 当前处理 `chapter_gen`：
  - `pending -> running`（设置 started_at）
  - progress：5%（启动）→ 80%（生成后）→ 100%（落库完成）
  - 成功：更新 chapter 内容与 metadata；更新 job result/progress/status
  - 失败：写入 job error，并交由 Consumer 的 retry/DLQ 机制处理

---

## 3. 当前可跑通的核心链路（E2E）

### 3.1 异步章节生成（非 SSE）
Client → API Gateway（HTTP）→ Postgres（创建 chapter/job）→ Redis Streams（chapter_gen）→ job-worker  
→ story-gen-svc（gRPC unary GenerateChapter，占位生成）→ Postgres（回写 chapter 内容 + job 结果/进度）

### 3.2 SSE 流式章节生成
Client → API Gateway（SSE）→ story-gen-svc（gRPC streaming StreamGenerateChapter，占位分块）  
→ API Gateway 透传 SSE → Postgres（流结束后写入最终内容）

### 3.3 检索（RAG 基础）
Client → API Gateway（HTTP）→ rag-retrieval-svc（gRPC）→ embedding-svc（HTTP /embed）→ Milvus Search → 返回 segments

---

## 4. 本地运行方式（建议）

### 4.1 启动基础设施
```bash
docker compose up -d
```

注意：
- `docker-compose.yaml` 中 Prometheus 占用 9090；而配置默认 `server.grpc.port=9090`，会冲突。
- 建议为各 gRPC 服务使用不同端口（示例见下）。

### 4.2 启动各进程（推荐端口分配示例）
建议端口（仅示例）：
- story-gen-svc: 50051
- rag-retrieval-svc: 50052
- memory-svc: 50053
- validator-svc: 50054
- api-gateway（HTTP）: 8080

示例（每个终端各跑一个）：
```bash
# story-gen-svc
SERVER_GRPC_PORT=50051 go run ./cmd/story-gen-svc

# rag-retrieval-svc
SERVER_GRPC_PORT=50052 EMBEDDING_ENDPOINT="http://localhost:8000" go run ./cmd/rag-retrieval-svc

# memory-svc
SERVER_GRPC_PORT=50053 go run ./cmd/memory-svc

# validator-svc
SERVER_GRPC_PORT=50054 go run ./cmd/validator-svc

# job-worker（连接 story-gen-svc）
STORY_GEN_GRPC_ADDR="localhost:50051" go run ./cmd/job-worker

# api-gateway（连接各 gRPC 服务）
RETRIEVAL_GRPC_ADDR="localhost:50052" \
STORY_GEN_GRPC_ADDR="localhost:50051" \
MEMORY_GRPC_ADDR="localhost:50053" \
VALIDATOR_GRPC_ADDR="localhost:50054" \
go run ./cmd/api-gateway
```

Embedding 服务：
- 本仓库目前只实现了 `internal/infrastructure/embedding/client.go`（HTTP 客户端），未实现 embedding-svc 服务端。
- 需要你单独启动一个兼容 `POST /embed` 的服务（返回 `{ "embeddings": [][]float32, "tokens_used": int }`）。

---

## 5. 待完成的部分（按优先级，清晰可执行）

### P0（必须补齐，否则“业务正确性/可用性”不成立）
1. story-gen-svc 真正实现（docs 10/11）
   - 接入 LLM provider（当前 `internal/infrastructure/llm` 目录为空）
   - 章节生成：Prompt → Retrieval（rag-retrieval-svc）→ 生成 →（可选）Validator → 输出
   - StreamGenerateChapter：逐 token/分段输出 + 生成元数据统计
2. embedding-svc 服务端实现
   - 与 `internal/infrastructure/embedding/client.go` 的协议对齐
   - 与配置 `embedding.endpoint` 对齐，并加入重试/超时策略
3. 任务取消/幂等
   - `JobHandler.CancelJob` 目前仅更新 DB；job-worker 需要在执行前/执行中检查 cancelled 并提前停止
   - 幂等键（Idempotency-Key）需要贯穿：HTTP → job 记录 → stream 消息 → worker 去重

### P1（核心对标能力）
1. memory-svc 完整闭环（docs 14）
   - 当前 `StoreMemory` 仅 ACK（无持久化）；需要解析实体/状态并写入 entity_states/history
   - 补齐 `EntityStateRepository` 的 Postgres 实现（目前缺失）
2. validator-svc 完整闭环（docs 13）
   - 当前仅最小判断；需要真正输出 issues、可选 auto_fix、与 story-gen 重试联动
3. Gateway 进一步解耦为“只做网关”
   - projects/chapters/entities/events 等 handler 目前仍直连 repository（非 gRPC）
   - 按服务边界逐步迁移到对应 svc（管理/记忆/生成/检索/校验）

### P2（工程质量/运维能力）
1. gRPC 拦截器/可观测性
   - trace/metrics 在 gRPC 侧的拦截器与上下文传递（Tenant/TraceId）
   - 服务间调用的超时、重试策略统一化
2. 配置与部署完善
   - 各 svc 独立配置文件或通过 ENV 覆盖（避免共享同一 `server.grpc.port`）
   - docker-compose 增加 embedding-svc
3. 测试与回归
   - job-worker 的消费/重试/DLQ 行为增加集成测试
   - RLS：为关键仓储写最小集成用例（tenant A 不能读 tenant B）

---

## 6. 常用开发命令

```bash
# 生成 wire（本仓库使用 go generate 驱动 internal/wire）
go generate ./internal/wire/...

# 生成 proto（按 scripts/gen-proto.sh 或 make proto）
make proto

# 编译/测试/静态检查
go test ./...
go vet ./...
```
