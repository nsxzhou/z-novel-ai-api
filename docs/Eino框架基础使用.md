# Eino 框架基础使用指南

---

## 1. 框架概述

### 1.1 什么是 Eino

**Eino**（发音类似 "I know"）是 CloudWeGo 团队开源的 Golang LLM 应用开发框架。它借鉴了 LangChain、LlamaIndex 等优秀框架的设计理念，结合前沿研究与实际应用经验，提供了一套强调**简洁性、可扩展性、可靠性和有效性**的开发框架，更贴合 Golang 编程习惯。

**官方资源**：

- 核心框架：[github.com/cloudwego/eino](https://github.com/cloudwego/eino)
- 组件库：[github.com/cloudwego/eino-ext](https://github.com/cloudwego/eino-ext)
- 示例仓库：[github.com/cloudwego/eino-examples](https://github.com/cloudwego/eino-examples)

### 1.2 核心特性

#### 丰富的组件抽象

Eino 将常用功能封装为可复用的组件抽象：

- **统一接口**：每种组件都有明确定义的输入输出类型、选项类型和流式处理范式
- **透明实现**：编排时只需关注抽象接口，具体实现对外透明
- **可嵌套组合**：复杂组件内部由多个组件组成，但对外仍是标准接口

核心组件包括：`ChatModel`、`Tool`、`ChatTemplate`、`Retriever`、`Document Loader`、`Lambda` 等。

#### 完整的流式处理支持

- **自动流拼接**：框架自动处理流式数据的拼接
- **自动装箱/拆箱**：根据需要在流式与非流式数据间自动转换
- **流合并/复制**：多流汇聚时自动合并，扇出时自动复制

#### 灵活的多模型支持

- **统一接口**：通过统一的 `ChatModel` 接口调用不同 LLM 提供商
- **轻松切换**：通过配置即可切换不同的模型服务
- **OpenAI 兼容**：支持所有 OpenAI 兼容的 API 服务

### 1.3 适用场景

- **智能对话系统**：聊天机器人、AI 客服
- **RAG 应用**：检索增强生成、知识问答
- **内容生成**：文章写作、代码生成、小说创作
- **多模型集成**：需要灵活切换不同 LLM 提供商的场景

---

## 2. 框架接入

本节演示如何在一个全新的 Go 项目中集成 Eino 框架（默认已安装 Go 环境）。

### 2.1 创建新项目

```bash
# 创建项目目录
mkdir my-llm-app && cd my-llm-app

# 初始化 Go 模块
go mod init my-llm-app

# 安装 Eino 核心框架和扩展组件
go get github.com/cloudwego/eino@latest
go get github.com/cloudwego/eino-ext@latest
```

安装完成后，`go.mod` 文件应包含：

```go
module my-llm-app

go 1.21

require (
    github.com/cloudwego/eino v0.4.3
    github.com/cloudwego/eino-ext v0.4.3
)
```

### 2.2 环境配置

创建 `.env` 文件存储 API 密钥：

```bash
# .env
OPENAI_API_KEY=sk-your-openai-api-key
```

> **安全提示**：确保将 `.env` 添加到 `.gitignore`，避免泄露密钥。

### 2.3 快速开始示例

创建 `main.go` 文件，实现一个简单的对话程序：

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/cloudwego/eino-ext/components/model/openai"
    "github.com/cloudwego/eino/schema"
)

func main() {
    ctx := context.Background()

    // 从环境变量读取 API Key
    apiKey := os.Getenv("OPENAI_API_KEY")
    if apiKey == "" {
        log.Fatal("请设置 OPENAI_API_KEY 环境变量")
    }

    // 创建 ChatModel 实例
    chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
        APIKey:  apiKey,
        BaseURL: "https://api.openai.com/v1",
        Model:   "gpt-4o-mini",
    })
    if err != nil {
        log.Fatalf("创建 ChatModel 失败: %v", err)
    }

    // 构建消息
    messages := []*schema.Message{
        schema.SystemMessage("你是一个有帮助的 AI 助手。"),
        schema.UserMessage("介绍一下 Go 语言的特点"),
    }

    // 调用 LLM
    response, err := chatModel.Generate(ctx, messages)
    if err != nil {
        log.Fatalf("调用 LLM 失败: %v", err)
    }

    // 输出结果
    fmt.Printf("AI 回复:\n%s\n", response.Content)
}
```

运行程序：

```bash
# 加载环境变量并运行
export $(cat .env | xargs) && go run main.go
```

### 2.4 推荐的项目结构

随着项目复杂度增加，建议采用如下结构：

```
my-llm-app/
├── main.go                 # 程序入口
├── go.mod                  # Go 模块定义
├── go.sum                  # 依赖校验和
├── .env                    # 环境变量（不提交到 Git）
├── .gitignore              # Git 忽略文件
├── config/
│   └── config.go           # 配置管理
├── llm/
│   ├── client.go           # LLM 客户端封装
│   └── factory.go          # 多模型管理（可选）
└── prompt/
    ├── templates.go        # Prompt 模板
    └── registry.go         # 模板注册（可选）
```

---

## 3. 大模型调用基础

### 3.1 初始化 ChatModel

#### 3.1.1 基础初始化

最简单的方式是直接创建 ChatModel 实例：

```go
package main

import (
    "context"
    "log"
    "os"

    "github.com/cloudwego/eino-ext/components/model/openai"
)

func main() {
    ctx := context.Background()

    // 创建 ChatModel
    chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
        APIKey:  os.Getenv("OPENAI_API_KEY"),
        BaseURL: "https://api.openai.com/v1",
        Model:   "gpt-4o-mini",
    })
    if err != nil {
        log.Fatalf("创建失败: %v", err)
    }

    log.Println("ChatModel 初始化成功")
}
```

#### 3.1.2 配置选项说明

`ChatModelConfig` 支持以下配置项：

| 参数          | 类型          | 必填 | 说明                                  |
| ------------- | ------------- | ---- | ------------------------------------- |
| `APIKey`      | string        | 是   | API 密钥                              |
| `BaseURL`     | string        | 是   | API 基础地址                          |
| `Model`       | string        | 是   | 模型名称                              |
| `MaxTokens`   | \*int         | 否   | 最大输出 token 数（默认值由模型决定） |
| `Temperature` | \*float32     | 否   | 温度参数 0-2（默认 1.0）              |
| `Timeout`     | time.Duration | 否   | 请求超时时间（默认 30s）              |

示例：配置所有选项

```go
maxTokens := 1000
temperature := float32(0.7)

chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
    APIKey:      os.Getenv("OPENAI_API_KEY"),
    BaseURL:     "https://api.openai.com/v1",
    Model:       "gpt-4o",
    MaxTokens:   &maxTokens,
    Temperature: &temperature,
    Timeout:     60 * time.Second,
})
```

#### 3.1.3 管理多个模型（可选）

如果需要同时使用多个模型，可以创建一个简单的管理器：

```go
type LLMClient struct {
    fast   model.BaseChatModel  // 快速模型（如 gpt-4o-mini）
    strong model.BaseChatModel  // 强大模型（如 gpt-4o）
}

func NewLLMClient(ctx context.Context) (*LLMClient, error) {
    apiKey := os.Getenv("OPENAI_API_KEY")

    // 初始化快速模型
    fast, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
        APIKey:  apiKey,
        BaseURL: "https://api.openai.com/v1",
        Model:   "gpt-4o-mini",
    })
    if err != nil {
        return nil, fmt.Errorf("初始化快速模型失败: %w", err)
    }

    // 初始化强大模型
    strong, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
        APIKey:  apiKey,
        BaseURL: "https://api.openai.com/v1",
        Model:   "gpt-4o",
    })
    if err != nil {
        return nil, fmt.Errorf("初始化强大模型失败: %w", err)
    }

    return &LLMClient{
        fast:   fast,
        strong: strong,
    }, nil
}

// 简单任务使用快速模型
func (c *LLMClient) FastModel() model.BaseChatModel {
    return c.fast
}

// 复杂任务使用强大模型
func (c *LLMClient) StrongModel() model.BaseChatModel {
    return c.strong
}
```

### 3.2 发送基础请求

#### 3.2.1 构建消息

Eino 使用 `schema.Message` 表示对话消息，支持三种角色：

```go
import "github.com/cloudwego/eino/schema"

// System 消息：定义 AI 的角色和行为规范
systemMsg := schema.SystemMessage("你是一个有帮助的 AI 助手")

// User 消息：用户的输入
userMsg := schema.UserMessage("Go 语言有什么特点？")

// Assistant 消息：AI 的历史回复（用于多轮对话）
assistantMsg := schema.AssistantMessage("Go 是一门简洁、高效的编程语言...")
```

**消息类型对照表**：

| 方法                        | 角色      | 用途         | OpenAI 对应 |
| --------------------------- | --------- | ------------ | ----------- |
| `SystemMessage(content)`    | system    | 定义 AI 行为 | system      |
| `UserMessage(content)`      | user      | 用户输入     | user        |
| `AssistantMessage(content)` | assistant | AI 回复      | assistant   |

#### 3.2.2 单轮对话示例

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/cloudwego/eino-ext/components/model/openai"
    "github.com/cloudwego/eino/schema"
)

func main() {
    ctx := context.Background()

    // 初始化 ChatModel
    chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
        APIKey:  os.Getenv("OPENAI_API_KEY"),
        BaseURL: "https://api.openai.com/v1",
        Model:   "gpt-4o-mini",
    })
    if err != nil {
        log.Fatalf("初始化失败: %v", err)
    }

    // 构建消息
    messages := []*schema.Message{
        schema.SystemMessage("你是一个编程专家"),
        schema.UserMessage("用一句话解释什么是闭包"),
    }

    // 调用 LLM
    response, err := chatModel.Generate(ctx, messages)
    if err != nil {
        log.Fatalf("调用失败: %v", err)
    }

    // 输出结果
    fmt.Printf("AI: %s\n", response.Content)
}
```

**输出示例**：

```
AI: 闭包是指函数可以访问并记住其外部作用域中的变量，即使外部函数已经执行完毕。
```

#### 3.2.3 多轮对话示例

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/cloudwego/eino-ext/components/model/openai"
    "github.com/cloudwego/eino/schema"
)

func main() {
    ctx := context.Background()

    // 初始化 ChatModel
    chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
        APIKey:  os.Getenv("OPENAI_API_KEY"),
        BaseURL: "https://api.openai.com/v1",
        Model:   "gpt-4o-mini",
    })
    if err != nil {
        log.Fatalf("初始化失败: %v", err)
    }

    // 多轮对话消息列表
    messages := []*schema.Message{
        schema.SystemMessage("你是一位编程助手"),
        schema.UserMessage("什么是闭包？"),
        schema.AssistantMessage("闭包是指函数可以访问其外部作用域的变量..."),
        schema.UserMessage("能举个 JavaScript 的例子吗？"),
    }

    // 调用 LLM
    response, err := chatModel.Generate(ctx, messages)
    if err != nil {
        log.Fatalf("调用失败: %v", err)
    }

    // 输出结果
    fmt.Printf("AI: %s\n", response.Content)
}
```

### 3.3 接收完整响应

#### 3.3.1 响应结构解析

`chatModel.Generate()` 返回的 `schema.Message` 包含以下关键信息：

```go
type Message struct {
    // 基础内容
    Role    string  // 角色：system/user/assistant
    Content string  // 文本内容

    // 元数据（可选）
    ResponseMeta *ResponseMeta  // 响应元数据
}

type ResponseMeta struct {
    Usage *TokenUsage  // Token 使用量
    // ... 其他元数据
}

type TokenUsage struct {
    PromptTokens     int  // 输入 token 数
    CompletionTokens int  // 输出 token 数
    TotalTokens      int  // 总 token 数
}
```

#### 3.3.2 获取 Token 使用量

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/cloudwego/eino-ext/components/model/openai"
    "github.com/cloudwego/eino/components/model"
    "github.com/cloudwego/eino/schema"
)

func main() {
    ctx := context.Background()

    // 初始化 ChatModel
    chatModel, _ := openai.NewChatModel(ctx, &openai.ChatModelConfig{
        APIKey:  os.Getenv("OPENAI_API_KEY"),
        BaseURL: "https://api.openai.com/v1",
        Model:   "gpt-4o-mini",
    })

    // 构建消息
    messages := []*schema.Message{
        schema.SystemMessage("你是一位数学老师"),
        schema.UserMessage("什么是质数？"),
    }

    // 调用 LLM（可选参数）
    response, err := chatModel.Generate(ctx, messages,
        model.WithMaxTokens(500),    // 限制最大输出 token 数
        model.WithTemperature(0.7),  // 设置温度参数
    )
    if err != nil {
        log.Fatalf("调用失败: %v", err)
    }

    // 输出内容
    fmt.Printf("AI 回复:\n%s\n\n", response.Content)

    // 输出 Token 使用量
    if response.ResponseMeta != nil && response.ResponseMeta.Usage != nil {
        usage := response.ResponseMeta.Usage
        fmt.Printf("Token 使用量:\n")
        fmt.Printf("  输入: %d tokens\n", usage.PromptTokens)
        fmt.Printf("  输出: %d tokens\n", usage.CompletionTokens)
        fmt.Printf("  总计: %d tokens\n", usage.TotalTokens)
    }
}
```

**输出示例**：

```
AI 回复:
质数是指只能被1和它本身整除的大于1的自然数。例如：2、3、5、7、11等。

Token 使用量:
  输入: 28 tokens
  输出: 42 tokens
  总计: 70 tokens
```

### 3.4 流式响应

#### 3.4.1 为什么使用流式响应

流式响应（Streaming）适用于以下场景：

- **用户体验优化**：逐步展示内容，降低等待感知
- **长文本生成**：实时显示生成进度
- **实时交互**：聊天场景中即时反馈

#### 3.4.2 流式调用示例

```go
package main

import (
    "context"
    "fmt"
    "io"
    "log"
    "os"

    "github.com/cloudwego/eino-ext/components/model/openai"
    "github.com/cloudwego/eino/components/model"
    "github.com/cloudwego/eino/schema"
)

func main() {
    ctx := context.Background()

    // 初始化 ChatModel
    chatModel, _ := openai.NewChatModel(ctx, &openai.ChatModelConfig{
        APIKey:  os.Getenv("OPENAI_API_KEY"),
        BaseURL: "https://api.openai.com/v1",
        Model:   "gpt-4o-mini",
    })

    // 构建消息
    messages := []*schema.Message{
        schema.SystemMessage("你是一位小说作家"),
        schema.UserMessage("写一段关于星际探险的故事开头，大约100字"),
    }

    // 调用流式接口
    streamReader, err := chatModel.Stream(ctx, messages,
        model.WithMaxTokens(200),
        model.WithTemperature(0.8),
    )
    if err != nil {
        log.Fatalf("启动流式响应失败: %v", err)
    }

    fmt.Println("AI 正在生成...")
    fmt.Println("----------------------------------------")

    // 逐块接收数据
    for {
        chunk, err := streamReader.Recv()
        if err == io.EOF {
            // 流结束
            fmt.Println("\n----------------------------------------")
            fmt.Println("生成完成")
            break
        }
        if err != nil {
            log.Fatalf("接收流数据失败: %v", err)
        }

        // 实时打印生成的内容
        fmt.Print(chunk.Content)
    }
}
```

**输出示例**（逐字显示）：

```
AI 正在生成...
----------------------------------------
2157年，星际飞船"探索者号"穿越虫洞，来到一个未知星系。船长艾莉娅望着舷窗外的陌生星云，心中既兴奋又忐忑。突然，雷达显示前方有异常能量波动...
----------------------------------------
生成完成
```

#### 3.4.3 Web API 中的流式响应

在 HTTP 接口中使用 Server-Sent Events (SSE) 实现流式传输：

```go
package handler

import (
    "io"
    "net/http"

    "github.com/gin-gonic/gin"
    "github.com/cloudwego/eino/schema"
)

// StreamChatHandler 处理流式对话请求
func (h *Handler) StreamChatHandler(c *gin.Context) {
    var req struct {
        Messages []struct {
            Role    string `json:"role"`
            Content string `json:"content"`
        } `json:"messages"`
    }

    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // 转换为 Eino 消息格式
    messages := make([]*schema.Message, len(req.Messages))
    for i, msg := range req.Messages {
        messages[i] = &schema.Message{
            Role:    msg.Role,
            Content: msg.Content,
        }
    }

    // 获取 ChatModel
    chatModel, err := h.getChatModel(c.Request.Context())
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    // 启动流式响应
    streamReader, err := chatModel.Stream(c.Request.Context(), messages)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    // 设置 SSE 响应头
    c.Header("Content-Type", "text/event-stream")
    c.Header("Cache-Control", "no-cache")
    c.Header("Connection", "keep-alive")
    c.Header("X-Accel-Buffering", "no")

    // 流式传输
    c.Stream(func(w io.Writer) bool {
        chunk, err := streamReader.Recv()
        if err == io.EOF {
            c.SSEvent("done", gin.H{"message": "生成完成"})
            return false
        }
        if err != nil {
            c.SSEvent("error", gin.H{"error": err.Error()})
            return false
        }

        c.SSEvent("content", gin.H{"delta": chunk.Content})
        return true
    })
}
```

**前端调用示例**：

```javascript
const eventSource = new EventSource("/api/chat/stream");

eventSource.addEventListener("content", (event) => {
  const data = JSON.parse(event.data);
  // 追加内容到界面
  document.getElementById("output").textContent += data.delta;
});

eventSource.addEventListener("done", (event) => {
  console.log("生成完成");
  eventSource.close();
});

eventSource.addEventListener("error", (event) => {
  const data = JSON.parse(event.data);
  console.error("错误:", data.error);
  eventSource.close();
});
```

---

## 4. Prompt 工程

### 4.1 基础 Prompt 构建

#### 4.1.1 Prompt 的重要性

Prompt 是与 LLM 交互的关键，好的 Prompt 应该：

- **明确具体**：清晰地表达任务需求
- **结构化**：便于 LLM 理解和遵循
- **可复用**：通过模板化实现重用
- **可维护**：集中管理，便于版本控制

#### 4.1.2 直接构建消息

最简单的方式是直接创建消息列表：

```go
import "github.com/cloudwego/eino/schema"

// 单轮对话
messages := []*schema.Message{
    schema.UserMessage("解释什么是递归"),
}

// 带系统提示的对话
messages := []*schema.Message{
    schema.SystemMessage("你是一位资深程序员，擅长用简单的语言解释复杂概念。"),
    schema.UserMessage("解释什么是递归"),
}

// 多轮对话
messages := []*schema.Message{
    schema.SystemMessage("你是编程助手"),
    schema.UserMessage("什么是闭包?"),
    schema.AssistantMessage("闭包是指函数可以访问其外部作用域的变量..."),
    schema.UserMessage("能举个JavaScript的例子吗?"),
}
```

### 4.2 模板使用方法

#### 4.2.1 为什么使用模板

模板的优势：

- **复用性**：同一模板可用于多个场景
- **可维护性**：模板集中管理，修改方便
- **可测试性**：可以独立测试模板效果
- **版本控制**：支持模板版本管理和灰度发布

#### 4.2.2 Eino 的 Prompt 模板

Eino 的 `prompt.ChatTemplate` 提供了强大的模板管理能力：

```go
import (
    "github.com/cloudwego/eino/components/prompt"
    "github.com/cloudwego/eino/schema"
)

// 创建模板
template := prompt.FromMessages(
    schema.FString,
    schema.SystemMessage("你是一位{{.role}}，擅长{{.expertise}}。"),
    schema.UserMessage("{{.task}}"),
)

// 填充变量
messages, err := template.Format(ctx, map[string]any{
    "role":      "小说作家",
    "expertise": "撰写科幻故事",
    "task":      "创作一个关于时间旅行的故事开头",
})
```

#### 4.2.3 完整示例

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/cloudwego/eino-ext/components/model/openai"
    "github.com/cloudwego/eino/components/prompt"
    "github.com/cloudwego/eino/schema"
)

func main() {
    ctx := context.Background()

    // 1. 创建模板
    template := prompt.FromMessages(
        schema.FString,
        schema.SystemMessage("你是一位{{.role}}，擅长{{.expertise}}。"),
        schema.UserMessage("{{.task}}"),
    )

    // 2. 填充变量
    messages, err := template.Format(ctx, map[string]any{
        "role":      "小说作家",
        "expertise": "科幻故事创作",
        "task":      "创作一个关于时间旅行的故事开头，100字左右",
    })
    if err != nil {
        log.Fatalf("模板渲染失败: %v", err)
    }

    // 3. 初始化 ChatModel
    chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
        APIKey:  os.Getenv("OPENAI_API_KEY"),
        BaseURL: "https://api.openai.com/v1",
        Model:   "gpt-4o-mini",
    })
    if err != nil {
        log.Fatalf("初始化失败: %v", err)
    }

    // 4. 调用 LLM
    response, err := chatModel.Generate(ctx, messages)
    if err != nil {
        log.Fatalf("调用失败: %v", err)
    }

    // 5. 输出结果
    fmt.Printf("AI 回复:\n%s\n", response.Content)
}
```

### 4.3 动态参数注入

#### 4.3.1 模板变量语法

Eino 使用 Go 的 `text/template` 语法，支持丰富的模板功能：

**支持的语法**：

| 语法                        | 功能       | 示例                                         |
| --------------------------- | ---------- | -------------------------------------------- |
| `{{.变量名}}`               | 变量插值   | `{{.title}}`                                 |
| `{{if .变量}}...{{end}}`    | 条件判断   | `{{if .context}}上下文: {{.context}}{{end}}` |
| `{{range .列表}}...{{end}}` | 遍历列表   | `{{range .items}}- {{.}}{{end}}`             |
| `{{with .对象}}...{{end}}`  | 上下文切换 | `{{with .user}}姓名: {{.name}}{{end}}`       |

#### 4.3.2 条件判断示例

```go
template := prompt.FromMessages(
    schema.FString,
    schema.SystemMessage("你是一位{{.role}}。"),
    schema.UserMessage(`任务：{{.task}}

{{if .context}}
上下文信息：
{{.context}}
{{end}}`),
)

// 使用模板
messages, _ := template.Format(ctx, map[string]any{
    "role":    "编程助手",
    "task":    "帮我审查代码",
    "context": "这是一个 Go 语言项目",  // 如果提供了 context，会包含在消息中
})
```

#### 4.3.3 列表遍历示例

```go
template := prompt.FromMessages(
    schema.FString,
    schema.SystemMessage("你是一位{{.role}}，擅长{{.expertise}}。"),
    schema.UserMessage(`任务：{{.task}}

{{if .requirements}}
具体要求：
{{range .requirements}}- {{.}}
{{end}}{{end}}`),
)

// 使用模板
messages, _ := template.Format(ctx, map[string]any{
    "role":      "小说创作助手",
    "expertise": "科幻故事构思",
    "task":      "帮助用户创作一个科幻小说项目",
    "requirements": []string{
        "故事背景设定在未来100年",
        "包含人工智能元素",
        "具有哲学思考",
    },
})
```

**渲染结果**：

```
[system]: 你是一位小说创作助手，擅长科幻故事构思。

[user]: 任务：帮助用户创作一个科幻小说项目

具体要求：
- 故事背景设定在未来100年
- 包含人工智能元素
- 具有哲学思考
```

#### 4.3.4 完整综合示例

```go
package main

import (
    "context"
    "fmt"

    "github.com/cloudwego/eino/components/prompt"
    "github.com/cloudwego/eino/schema"
)

func main() {
    ctx := context.Background()

    // 创建复杂模板（使用多种语法）
    tpl := prompt.FromMessages(
        schema.FString,
        schema.SystemMessage("你是一位{{.role}}，擅长{{.expertise}}。"),
        schema.UserMessage(`任务：{{.task}}

{{if .context}}
上下文：{{.context}}
{{end}}

{{if .requirements}}
要求：
{{range .requirements}}- {{.}}
{{end}}
{{end}}`),
    )

    // 准备变量
    vars := map[string]any{
        "role":      "小说创作助手",
        "expertise": "科幻故事构思",
        "task":      "帮助用户创作一个科幻小说项目",
        "context":   "用户对太空探索和人工智能感兴趣",
        "requirements": []string{
            "故事背景设定在未来100年",
            "包含人工智能元素",
            "具有哲学思考",
        },
    }

    // 渲染模板
    messages, _ := tpl.Format(ctx, vars)

    // 输出结果
    for _, msg := range messages {
        fmt.Printf("[%s]: %s\n\n", msg.Role, msg.Content)
    }
}
```

### 4.4 模板文件管理

对于复杂项目，建议将 Prompt 模板存储在独立的文件中：

**目录结构**：

```
project/
├── prompt/
│   ├── registry.go           # 模板注册表
│   └── templates/
│       ├── coding_v1.yaml    # 编程助手模板
│       ├── writing_v1.yaml   # 创作助手模板
│       └── review_v1.yaml    # 代码审查模板
```

**模板文件示例**（`templates/coding_v1.yaml`）：

````yaml
- role: system
  content: |
    你是一位资深的{{.language}}开发专家。
    你的专长：{{.expertise}}

- role: user
  content: |
    任务：{{.task}}

    {{if .code}}
    代码：
    ```{{.language}}
    {{.code}}
    ```
    {{end}}

    {{if .requirements}}
    要求：
    {{range .requirements}}- {{.}}
    {{end}}
    {{end}}
````

**模板注册表实现**：

```go
package prompt

import (
    "embed"
    "fmt"
    "sync"

    "github.com/cloudwego/eino/components/prompt"
)

//go:embed templates/*.yaml
var templatesFS embed.FS

type PromptName string

const (
    PromptCodingV1  PromptName = "coding_v1"
    PromptWritingV1 PromptName = "writing_v1"
    PromptReviewV1  PromptName = "review_v1"
)

type PromptRegistry struct {
    templates map[PromptName]prompt.ChatTemplate
    mu        sync.RWMutex
}

func NewPromptRegistry() *PromptRegistry {
    return &PromptRegistry{
        templates: make(map[PromptName]prompt.ChatTemplate),
    }
}

func (r *PromptRegistry) Register(name PromptName, tpl prompt.ChatTemplate) {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.templates[name] = tpl
}

func (r *PromptRegistry) Get(name PromptName) (prompt.ChatTemplate, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()

    tpl, ok := r.templates[name]
    if !ok {
        return nil, fmt.Errorf("template %s not found", name)
    }
    return tpl, nil
}

var defaultRegistry = NewPromptRegistry()

func init() {
    loadTemplate(PromptCodingV1, "templates/coding_v1.yaml")
    loadTemplate(PromptWritingV1, "templates/writing_v1.yaml")
    loadTemplate(PromptReviewV1, "templates/review_v1.yaml")
}

func loadTemplate(name PromptName, path string) {
    data, err := templatesFS.ReadFile(path)
    if err != nil {
        panic(fmt.Sprintf("failed to read template %s: %v", name, err))
    }

    tpl, err := prompt.FromYAML(string(data))
    if err != nil {
        panic(fmt.Sprintf("failed to parse template %s: %v", name, err))
    }

    defaultRegistry.Register(name, tpl)
}

func GetTemplate(name PromptName) (prompt.ChatTemplate, error) {
    return defaultRegistry.Get(name)
}
```

**使用示例**：

```go
package main

import (
    "context"
    "fmt"
    "log"

    "your-project/prompt"
    "github.com/cloudwego/eino-ext/components/model/openai"
)

func main() {
    ctx := context.Background()

    // 获取模板
    tpl, err := prompt.GetTemplate(prompt.PromptCodingV1)
    if err != nil {
        log.Fatal(err)
    }

    // 填充变量
    messages, _ := tpl.Format(ctx, map[string]any{
        "language":  "Go",
        "expertise": "并发编程和性能优化",
        "task":      "审查这段代码的性能问题",
        "code":      "func process() { ... }",
        "requirements": []string{
            "检查并发安全性",
            "优化性能瓶颈",
        },
    })

    // 调用 LLM
    chatModel, _ := openai.NewChatModel(ctx, &openai.ChatModelConfig{
        APIKey:  os.Getenv("OPENAI_API_KEY"),
        BaseURL: "https://api.openai.com/v1",
        Model:   "gpt-4o-mini",
    })

    response, _ := chatModel.Generate(ctx, messages)
    fmt.Printf("AI: %s\n", response.Content)
}
```

---

## 5. 常见问题

### Q1: 如何切换不同的 LLM 提供商？

**答**：Eino 支持所有 OpenAI 兼容的 API。只需修改配置中的 `BaseURL` 和 `Model`：

```go
// OpenAI
chatModel, _ := openai.NewChatModel(ctx, &openai.ChatModelConfig{
    APIKey:  os.Getenv("OPENAI_API_KEY"),
    BaseURL: "https://api.openai.com/v1",
    Model:   "gpt-4o-mini",
})

// DeepSeek (OpenAI 兼容)
chatModel, _ := openai.NewChatModel(ctx, &openai.ChatModelConfig{
    APIKey:  os.Getenv("DEEPSEEK_API_KEY"),
    BaseURL: "https://api.deepseek.com/v1",
    Model:   "deepseek-chat",
})

// 本地 Ollama
chatModel, _ := openai.NewChatModel(ctx, &openai.ChatModelConfig{
    APIKey:  "ollama",  // Ollama 不需要真实 API Key
    BaseURL: "http://localhost:11434/v1",
    Model:   "llama2",
})
```

### Q2: 如何处理 LLM 调用失败？

**答**：实现重试和降级策略：

```go
import "time"

func callWithRetry(ctx context.Context, chatModel model.BaseChatModel, messages []*schema.Message, maxRetries int) (*schema.Message, error) {
    var lastErr error
    for i := 0; i < maxRetries; i++ {
        resp, err := chatModel.Generate(ctx, messages)
        if err == nil {
            return resp, nil
        }
        lastErr = err

        // 指数退避
        waitTime := time.Second * time.Duration(1<<uint(i))
        time.Sleep(waitTime)

        log.Printf("重试 %d/%d: %v", i+1, maxRetries, err)
    }
    return nil, fmt.Errorf("调用失败（已重试 %d 次）: %w", maxRetries, lastErr)
}
```

### Q3: 如何控制生成内容的长度？

**答**：使用 `MaxTokens` 选项：

```go
import "github.com/cloudwego/eino/components/model"

response, err := chatModel.Generate(ctx, messages,
    model.WithMaxTokens(500),  // 限制最多生成 500 tokens
)
```

> **注意**：`MaxTokens` 是输出 token 数的上限，实际生成可能更短（LLM 可能提前结束）。

### Q4: Temperature 参数如何选择？

**答**：

| Temperature | 特点                   | 适用场景                     |
| ----------- | ---------------------- | ---------------------------- |
| 0.0 - 0.3   | 输出确定性强，重复性高 | 数据分析、代码生成、技术问答 |
| 0.4 - 0.7   | 平衡创造性和稳定性     | 一般对话、摘要生成           |
| 0.8 - 1.0   | 输出多样化，创造性强   | 创意写作、头脑风暴           |
| 1.0 以上    | 输出随机性极高         | 艺术创作（慎用）             |

```go
import "github.com/cloudwego/eino/components/model"

// 代码生成（低温度）
response, _ := chatModel.Generate(ctx, messages,
    model.WithTemperature(0.2),
)

// 创意写作（高温度）
response, _ := chatModel.Generate(ctx, messages,
    model.WithTemperature(0.9),
)
```

### Q5: 如何查看 Token 使用量？

**答**：从响应元数据中获取：

```go
resp, _ := chatModel.Generate(ctx, messages)

if resp.ResponseMeta != nil && resp.ResponseMeta.Usage != nil {
    usage := resp.ResponseMeta.Usage
    fmt.Printf("输入: %d, 输出: %d, 总计: %d\n",
        usage.PromptTokens,
        usage.CompletionTokens,
        usage.TotalTokens,
    )
}
```

---

## 6 参考资料

- [Eino 官方文档](https://github.com/cloudwego/eino)
- [Eino-Ext 扩展组件](https://github.com/cloudwego/eino-ext)
- [Eino 示例代码](https://github.com/cloudwego/eino-examples)
- [OpenAI API 文档](https://platform.openai.com/docs/api-reference)
- [Go text/template 文档](https://pkg.go.dev/text/template)
