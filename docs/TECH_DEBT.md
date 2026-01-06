# 技术债记录

> 记录项目中已知的技术债务，用于后续迭代时优先处理。

---

## TD-001: API Gateway 直连内部服务，未通过 gRPC 调用微服务

**记录时间**: 2026-01-06  
**优先级**: 中  
**影响范围**: 架构层面

### 问题描述

根据设计文档 `docs/09-gRPC内部服务通信规范.md`，系统架构应为：

```
API Gateway (Gin) --> gRPC --> story-gen-svc
                  --> gRPC --> rag-retrieval-svc
                  --> gRPC --> validator-svc
                  --> gRPC --> memory-svc
```

**当前实际情况**：

API Gateway 的 Handler 直接注入 `internal/application/story` 层的服务，跳过了 gRPC 微服务层：

| Handler                  | 直连的内部服务                                         | 应调用的 gRPC 服务 |
| ------------------------ | ------------------------------------------------------ | ------------------ |
| `FoundationHandler`      | `story.FoundationGenerator`, `story.FoundationApplier` | `story-gen-svc`    |
| `ConversationHandler`    | `story.ArtifactGenerator`                              | `story-gen-svc`    |
| `ProjectCreationHandler` | `story.ProjectCreationGenerator`                       | `story-gen-svc`    |

同时，gRPC 服务端实现（`internal/interfaces/grpc/server/`）均为占位实现，返回 `codes.Unimplemented`。

### 现状说明

当前以**单体架构**运行，所有业务逻辑在 `api-gateway` 进程内完成。这在 MVP 阶段是可接受的，但与微服务架构设计存在差异。

### 解决方案（后续迭代）

1. **实现 gRPC 服务**：完成 `story-gen-svc`、`rag-retrieval-svc`、`validator-svc`、`memory-svc` 的真实逻辑
2. **引入 gRPC 客户端**：在 Handler 中通过 gRPC Client 调用独立微服务
3. **独立部署**：将各服务拆分为独立进程/容器

### 相关文档

- [09-gRPC 内部服务通信规范](./09-gRPC内部服务通信规范.md)
- [未实现模块清单](./unimplemented_modules.md)

---

## 如何添加新的技术债

复制以下模板并填写：

```markdown
## TD-XXX: [简短标题]

**记录时间**: YYYY-MM-DD  
**优先级**: 高/中/低  
**影响范围**: [架构/性能/安全/可维护性]

### 问题描述

[详细说明问题]

### 现状说明

[当前为什么可以接受]

### 解决方案（后续迭代）

[如何修复]
```
