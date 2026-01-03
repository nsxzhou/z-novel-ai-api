# AI Novel AI API Project Status

AI 小说生成后端系统，基于 Go (Gin) + Eino (LLM 编排) 构建。

## 🚀 项目进度概览 (Phase 1-2)

| 阶段           | 模块       | 文档编号 | 状态 | 核心功能                                  |
| :------------- | :--------- | :------- | :--- | :---------------------------------------- |
| **基础建设**   | 目录结构   | 01       | ✅   | 标准 Go 项目布局、监控/日志/追踪基础      |
|                | 配置管理   | 02       | ✅   | Viper + ENV 环境变量预处理器              |
|                | 可观测性   | 03       | ✅   | OpenTelemetry Tracing, Zap Logging        |
| **数据持久层** | PostgreSQL | 04       | ✅   | RLS 多租户隔离、事务管理、自动平滑迁移    |
|                | Redis      | 05       | ✅   | Read-Through 缓存、限流、Streams 消息队列 |
|                | 向量数据库 | 06       | ✅   | Milvus 混合检索 (RRF)、HNSW 索引          |
| **API 层**     | 网关设计   | 07       | ⏳   | Gin 框架、JWT、限流、多租户中间件         |
|                | API 规范   | 08       | ⏳   | RESTful 统一响应、错误码体系              |
|                | gRPC 通信  | 09       | ⏳   | 内部服务微服务化设计                      |
| **核心业务**   | Eino 编排  | 10       | ⏳   | Graph 节点流转、生成工作流                |
|                | 小说生成   | 11       | ⏳   | 生成工作流、任务调度                      |
|                | RAG 检索   | 12       | ⏳   | 背景库语义搜索、召回增强                  |
|                | 校验/记忆  | 13/14    | ⏳   | 内容一致性检查、实体记忆存储              |

## 🛠 技术栈

- **Language**: Go 1.23+
- **Database**: PostgreSQL 16 (RLS), Redis 7 (Streams), Milvus 2.4
- **Framework**: Gin (Web), Google Eino (LLM Orchestration)
- **Observability**: OpenTelemetry, Jaeger, Prometheus, Zap
- **DI**: Google Wire
- **Deployment**: Kubernetes

## 📦 已实现的组件 (Docs 01-06)

### 1. Data Layer (Postgres)

- `internal/persistence/postgres`: 客户端实现、事务管理器、租户上下文。
- `migrations/postgres`: 包含租户、用户、项目、章节、实体、关系、事件、任务等 14 个平滑迁移脚本。
- **Repositories**: 已完成 Tenant, User, Project, Volume, Chapter, Entity, Relation, Event, Job 的所有 Repository 实现。

### 2. Cache & Messaging (Redis)

- `internal/persistence/redis`: 连接池管理、限流器 (Sliding Window)。
- `internal/persistence/redis/cache`: 支持 Singleflight, Read-Through, Write-Through 模式。
- `internal/infrastructure/messaging`: 基于 Redis Streams 的高性能生产者与消费者（支持 Consumer Group, Retry, DLQ）。

### 3. Vector Database (Milvus)

- `internal/persistence/milvus`: 客户端、Schema 定义、向量 Repository。
- **特色功能**: 支持混合检索 (Semantic + Keyword)、RRF (Reciprocal Rank Fusion) 重排整合、多租户 Partition 隔离。

### 4. Dependency Injection

- `internal/wire`: 已完成数据层所有组件的 Wire 自动注入配置。

## ⌨️ 常用开发命令

```bash
# 生成 Wire 依赖代码
wire ./internal/wire

# 运行全项目编译验证
go build ./...

# 运行代码格式化
go fmt ./...

# 更新依赖
go mod tidy
```

## 📅 下一步计划 (Phase 3: API Layer)

1. [ ] 实现 Gin API 网关 (`docs/07`)
2. [ ] 开发统一错误处理与标准响应 (`docs/08`)
3. [ ] 整合 JWT 与租户中间件 (`docs/19`)
4. [ ] 编写核心业务 API 入口
