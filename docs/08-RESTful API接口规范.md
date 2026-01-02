# 08 - RESTful API 接口规范

> AI 小说生成后端 REST API 版本控制、路由命名、请求响应格式规范

## 1. 概述

本文档定义了项目的 RESTful API 设计规范，包括版本控制策略、路由命名规则、请求/响应格式、错误处理以及 SSE 流式接口设计。

---

## 2. API 版本控制

### 2.1 版本策略

- **URL 路径版本化**：`/v1/`, `/v2/`
- **向后兼容**：新版本发布后，旧版本至少保留 6 个月
- **废弃通知**：通过 `Deprecation` Header 提前通知

```http
Deprecation: true
Sunset: Sat, 01 Jul 2026 00:00:00 GMT
Link: </v2/chapters>; rel="successor-version"
```

---

## 3. 路由命名规范

### 3.1 命名规则

| 规则         | 说明                 | 示例                      |
| ------------ | -------------------- | ------------------------- |
| 使用复数名词 | 资源集合使用复数     | `/projects`, `/chapters`  |
| 小写字母     | 全部小写，连字符分隔 | `/entity-states`          |
| 层级嵌套     | 父子资源关系清晰     | `/projects/:pid/chapters` |
| 动作使用动词 | 非 CRUD 操作使用动词 | `/chapters/:cid/generate` |

### 3.2 完整路由清单

```go
// internal/interfaces/http/router/routes.go
package router

func registerRoutes(r *gin.Engine, cfg *config.Config) {
    v1 := r.Group("/v1")

    // 项目管理
    projects := v1.Group("/projects")
    {
        projects.GET("", handlers.ListProjects)
        projects.POST("", handlers.CreateProject)
        projects.GET("/:pid", handlers.GetProject)
        projects.PUT("/:pid", handlers.UpdateProject)
        projects.DELETE("/:pid", handlers.DeleteProject)

        // 项目下的章节
        projects.GET("/:pid/chapters", handlers.ListChapters)
        projects.POST("/:pid/chapters", handlers.CreateChapter)
        projects.POST("/:pid/chapters/generate", handlers.GenerateChapter)

        // 项目下的实体
        projects.GET("/:pid/entities", handlers.ListEntities)
        projects.POST("/:pid/entities", handlers.CreateEntity)

        // 项目下的事件
        projects.GET("/:pid/events", handlers.ListEvents)

        // 项目设置
        projects.GET("/:pid/settings", handlers.GetProjectSettings)
        projects.PUT("/:pid/settings", handlers.UpdateProjectSettings)
    }

    // 章节管理
    chapters := v1.Group("/chapters")
    {
        chapters.GET("/:cid", handlers.GetChapter)
        chapters.PUT("/:cid", handlers.UpdateChapter)
        chapters.DELETE("/:cid", handlers.DeleteChapter)
        chapters.GET("/:cid/stream", handlers.StreamChapter)  // SSE
        chapters.POST("/:cid/regenerate", handlers.RegenerateChapter)
    }

    // 实体管理
    entities := v1.Group("/entities")
    {
        entities.GET("/:eid", handlers.GetEntity)
        entities.PUT("/:eid", handlers.UpdateEntity)
        entities.DELETE("/:eid", handlers.DeleteEntity)
        entities.PUT("/:eid/state", handlers.UpdateEntityState)
        entities.GET("/:eid/relations", handlers.GetEntityRelations)
    }

    // 检索调试
    retrieval := v1.Group("/retrieval")
    {
        retrieval.POST("/search", handlers.Search)
        retrieval.POST("/debug", handlers.DebugRetrieval)
    }

    // 任务管理
    jobs := v1.Group("/jobs")
    {
        jobs.GET("/:jid", handlers.GetJob)
        jobs.DELETE("/:jid", handlers.CancelJob)
    }

    // 健康检查
    r.GET("/health", handlers.Health)
    r.GET("/ready", handlers.Ready)
}
```

---

## 4. 请求格式

### 4.1 通用请求头

| Header            | 必填       | 说明                                      |
| ----------------- | ---------- | ----------------------------------------- |
| `Authorization`   | ✅         | Bearer Token                              |
| `Content-Type`    | ✅         | `application/json`                        |
| `Accept`          | ❌         | `application/json` 或 `text/event-stream` |
| `X-Request-ID`    | ❌         | 客户端请求 ID                             |
| `Idempotency-Key` | 写操作推荐 | 幂等键                                    |

### 4.2 分页请求

```http
GET /v1/projects/:pid/chapters?page=1&page_size=20&sort=-created_at
```

**参数说明**：

- `page`：页码，从 1 开始
- `page_size`：每页条数，默认 20，最大 100
- `sort`：排序字段，`-` 前缀表示降序

### 4.3 过滤请求

```http
GET /v1/projects/:pid/entities?type=character&importance=protagonist,major
```

---

## 5. 响应格式

### 5.1 统一响应信封

```json
{
  "code": 200,
  "message": "success",
  "data": { ... },
  "meta": {
    "page": 1,
    "page_size": 20,
    "total": 100,
    "total_pages": 5
  },
  "trace_id": "abc123def456"
}
```

### 5.2 响应结构定义

```go
// internal/interfaces/http/dto/response.go
package dto

type Response[T any] struct {
    Code      int       `json:"code"`
    Message   string    `json:"message"`
    Data      T         `json:"data,omitempty"`
    Meta      *PageMeta `json:"meta,omitempty"`
    TraceID   string    `json:"trace_id"`
}

type PageMeta struct {
    Page       int `json:"page"`
    PageSize   int `json:"page_size"`
    Total      int `json:"total"`
    TotalPages int `json:"total_pages"`
}

func Success[T any](c *gin.Context, data T) {
    c.JSON(200, Response[T]{
        Code:    200,
        Message: "success",
        Data:    data,
        TraceID: c.GetString("trace_id"),
    })
}

func SuccessWithPage[T any](c *gin.Context, data T, meta *PageMeta) {
    c.JSON(200, Response[T]{
        Code:    200,
        Message: "success",
        Data:    data,
        Meta:    meta,
        TraceID: c.GetString("trace_id"),
    })
}
```

### 5.3 具体响应示例

**章节生成响应**：

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "chapter_id": "550e8400-e29b-41d4-a716-446655440000",
    "title": "第一章 初遇",
    "content": "...",
    "summary": "主角在城门口与女主角相遇...",
    "word_count": 2500,
    "metadata": {
      "generated_at": "2026-01-02T10:00:00Z",
      "model": "gpt-4o",
      "tokens_used": 8192,
      "validation_passed": true
    }
  },
  "trace_id": "abc123"
}
```

---

## 6. 错误处理

### 6.1 错误码体系

| 错误码 | HTTP 状态 | 说明             |
| ------ | --------- | ---------------- |
| 200    | 200       | 成功             |
| 400    | 400       | 请求参数错误     |
| 401    | 401       | 未认证           |
| 403    | 403       | 无权限           |
| 404    | 404       | 资源不存在       |
| 409    | 409       | 冲突（重复创建） |
| 422    | 422       | 业务逻辑错误     |
| 429    | 429       | 请求过于频繁     |
| 500    | 500       | 服务器内部错误   |
| 503    | 503       | 服务不可用       |

### 6.2 业务错误码

| 错误码  | 说明           |
| ------- | -------------- |
| `E1001` | 章节生成失败   |
| `E1002` | 一致性校验失败 |
| `E1003` | 检索上下文为空 |
| `E2001` | 实体不存在     |
| `E2002` | 实体状态冲突   |
| `E3001` | 配额超限       |
| `E3002` | LLM 服务不可用 |

### 6.3 错误响应格式

```json
{
  "code": 422,
  "message": "validation failed",
  "error": {
    "error_code": "E1002",
    "details": "角色状态与上一章节矛盾：张三已在第5章死亡",
    "suggestions": ["修改当前章节中张三的出现", "修正第5章中张三的状态"]
  },
  "trace_id": "abc123"
}
```

### 6.4 错误处理实现

```go
// pkg/errors/errors.go
package errors

type AppError struct {
    HTTPCode    int      `json:"-"`
    Code        int      `json:"code"`
    Message     string   `json:"message"`
    ErrorCode   string   `json:"error_code,omitempty"`
    Details     string   `json:"details,omitempty"`
    Suggestions []string `json:"suggestions,omitempty"`
}

func (e *AppError) Error() string {
    return e.Message
}

func NewValidationError(errorCode, details string, suggestions ...string) *AppError {
    return &AppError{
        HTTPCode:    422,
        Code:        422,
        Message:     "validation failed",
        ErrorCode:   errorCode,
        Details:     details,
        Suggestions: suggestions,
    }
}

func NewNotFoundError(resource string) *AppError {
    return &AppError{
        HTTPCode: 404,
        Code:     404,
        Message:  fmt.Sprintf("%s not found", resource),
    }
}

// 统一错误处理
func HandleError(c *gin.Context, err error) {
    if appErr, ok := err.(*AppError); ok {
        c.JSON(appErr.HTTPCode, gin.H{
            "code":      appErr.Code,
            "message":   appErr.Message,
            "error":     appErr,
            "trace_id":  c.GetString("trace_id"),
        })
        return
    }

    c.JSON(500, gin.H{
        "code":     500,
        "message":  "internal server error",
        "trace_id": c.GetString("trace_id"),
    })
}
```

---

## 7. SSE 流式接口

### 7.1 SSE 端点

```http
GET /v1/chapters/:cid/stream
Accept: text/event-stream
```

### 7.2 SSE 事件格式

```
event: content
data: {"chunk": "第一章 初遇\n\n", "index": 0}

event: content
data: {"chunk": "阳光透过云层...", "index": 1}

event: metadata
data: {"word_count": 500, "tokens_used": 1024}

event: done
data: {"chapter_id": "xxx", "total_chunks": 50}
```

### 7.3 SSE 实现

```go
// internal/interfaces/http/handler/stream.go
package handler

import (
    "github.com/gin-gonic/gin"
)

func StreamChapter(c *gin.Context) {
    chapterID := c.Param("cid")

    c.Header("Content-Type", "text/event-stream")
    c.Header("Cache-Control", "no-cache")
    c.Header("Connection", "keep-alive")
    c.Header("X-Accel-Buffering", "no")

    // 获取生成流
    stream, err := storyGenService.StreamGenerate(c.Request.Context(), chapterID)
    if err != nil {
        c.SSEvent("error", gin.H{"message": err.Error()})
        return
    }

    c.Stream(func(w io.Writer) bool {
        select {
        case chunk, ok := <-stream.Chunks:
            if !ok {
                c.SSEvent("done", gin.H{
                    "chapter_id": chapterID,
                    "total_chunks": stream.TotalChunks,
                })
                return false
            }
            c.SSEvent("content", gin.H{
                "chunk": chunk.Text,
                "index": chunk.Index,
            })
            return true

        case meta := <-stream.Metadata:
            c.SSEvent("metadata", meta)
            return true

        case err := <-stream.Errors:
            c.SSEvent("error", gin.H{"message": err.Error()})
            return false

        case <-c.Request.Context().Done():
            return false
        }
    })
}
```

---

## 8. DTO 定义示例

### 8.1 创建章节请求

```go
// internal/interfaces/http/dto/chapter.go
package dto

type CreateChapterRequest struct {
    Title           string `json:"title" validate:"required,max=255"`
    Outline         string `json:"outline" validate:"required,max=5000"`
    VolumeID        string `json:"volume_id" validate:"omitempty,uuid"`
    TargetWordCount int    `json:"target_word_count" validate:"gte=500,lte=10000"`
    StoryTimeStart  int64  `json:"story_time_start"`
    Notes           string `json:"notes" validate:"max=2000"`
}

type ChapterResponse struct {
    ID           string            `json:"id"`
    ProjectID    string            `json:"project_id"`
    VolumeID     string            `json:"volume_id,omitempty"`
    SeqNum       int               `json:"seq_num"`
    Title        string            `json:"title"`
    Content      string            `json:"content"`
    Summary      string            `json:"summary"`
    WordCount    int               `json:"word_count"`
    Status       string            `json:"status"`
    Metadata     *ChapterMetadata  `json:"metadata,omitempty"`
    CreatedAt    time.Time         `json:"created_at"`
    UpdatedAt    time.Time         `json:"updated_at"`
}

type ChapterMetadata struct {
    GeneratedAt      time.Time `json:"generated_at"`
    Model            string    `json:"model"`
    TokensUsed       int       `json:"tokens_used"`
    ValidationPassed bool      `json:"validation_passed"`
}
```

---

## 9. OpenAPI 文档

使用 `swaggo/swag` 生成 OpenAPI 文档：

```go
// @Summary 创建章节生成任务
// @Description 异步创建章节生成任务，返回任务 ID
// @Tags chapters
// @Accept json
// @Produce json
// @Param pid path string true "项目 ID"
// @Param body body dto.CreateChapterRequest true "章节信息"
// @Success 202 {object} dto.Response[dto.JobResponse]
// @Failure 400 {object} dto.ErrorResponse
// @Router /v1/projects/{pid}/chapters/generate [post]
func GenerateChapter(c *gin.Context) {
    // ...
}
```

---

## 10. 相关文档

- [07-Gin API 网关与中间件设计](./07-Gin API 网关与中间件设计.md)
- [09-gRPC 内部服务通信规范](./09-gRPC内部服务通信规范.md)
- [11-小说生成服务设计](./11-小说生成服务设计.md)
