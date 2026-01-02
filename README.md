# Z-Novel-AI-API

AI 小说生成后端服务 - 基于 Gin + Eino 的长篇小说生成系统。

## 功能特性

- 🚀 **流式章节生成** - 支持 SSE 实时输出
- 🔍 **三信号 RAG 检索** - 语义向量 + 关键词 + 时间对齐
- ✅ **四维一致性校验** - 设定/角色/状态/情感
- 💾 **记忆回写** - 摘要抽取与时间知识图谱
- 📊 **完整可观测性** - 日志、追踪、指标

## 快速开始

### 环境要求

- Go 1.21+
- PostgreSQL 15+
- Redis 7+
- Milvus 2.3+ (可选，开发可用 PGVector)

### 安装依赖

```bash
# 安装 Go 工具链
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install github.com/google/wire/cmd/wire@latest
go install github.com/cosmtrek/air@latest

# 下载项目依赖
go mod download
```

### 本地开发

```bash
# 启动依赖服务 (PostgreSQL, Redis, Milvus, MinIO)
docker compose -f deployments/docker/docker-compose.dev.yaml up -d

# 运行数据库迁移
make migrate-up

# 启动服务 (开发模式，热重载)
make run-air

# 或直接运行
make run-dev
```

### 构建

```bash
# 构建所有服务
make build

# 构建单个服务
make build-api-gateway
```

## 项目结构

```
z-novel-ai-api/
├── cmd/                    # 服务入口
├── internal/               # 私有应用代码
│   ├── config/            # 配置加载
│   ├── domain/            # 领域模型
│   ├── application/       # 应用层
│   ├── infrastructure/    # 基础设施层
│   ├── interfaces/        # 接口适配层
│   └── workflow/          # Eino 工作流
├── pkg/                    # 公共库
├── api/                    # API 定义
├── configs/                # 配置文件
├── deployments/            # 部署配置
├── migrations/             # 数据库迁移
├── scripts/                # 构建脚本
├── test/                   # 测试
└── docs/                   # 文档
```

## API 端点

| Method | Path     | 描述            |
| ------ | -------- | --------------- |
| GET    | /health  | 健康检查        |
| GET    | /ready   | 就绪检查        |
| GET    | /live    | 存活检查        |
| GET    | /metrics | Prometheus 指标 |

## 配置

配置文件位于 `configs/` 目录：

- `config.yaml` - 主配置
- `config.dev.yaml` - 开发环境
- `config.staging.yaml` - 预发布环境
- `config.prod.yaml` - 生产环境

通过环境变量 `APP_ENV` 指定环境，例如：

```bash
APP_ENV=development go run ./cmd/api-gateway
```

## 开发命令

```bash
make help        # 显示所有可用命令
make test        # 运行测试
make lint        # 代码检查
make fmt         # 格式化代码
make coverage    # 生成覆盖率报告
```

## 文档

- [项目初始化与目录结构规范](docs/01-项目初始化与目录结构规范.md)
- [配置管理与环境变量规范](docs/02-配置管理与环境变量规范.md)
- [日志与可观测性规范](docs/03-日志与可观测性规范.md)
- [AI 小说生成后端方案设计](docs/AI小说生成后端方案设计（Gin+Eino）.md)

## License

MIT
