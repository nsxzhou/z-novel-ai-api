# Go 参数
GO := go
GOFMT := gofmt
GOLINT := golangci-lint

# 版本信息
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

# 服务列表
SERVICES := api-gateway story-gen-svc rag-retrieval-svc validator-svc memory-svc job-worker file-svc admin-svc

.PHONY: all build clean test lint proto wire deps tidy run

# 默认目标
all: lint test build

## 依赖管理
deps:
	$(GO) mod download

tidy:
	$(GO) mod tidy

## 构建
build:
	@for svc in $(SERVICES); do \
		echo "Building $$svc..."; \
		$(GO) build $(LDFLAGS) -o bin/$$svc ./cmd/$$svc 2>/dev/null || echo "$$svc not implemented yet"; \
	done

build-%:
	$(GO) build $(LDFLAGS) -o bin/$* ./cmd/$*

## 运行
run:
	$(GO) run ./cmd/api-gateway

run-dev:
	APP_ENV=development $(GO) run ./cmd/api-gateway

run-air:
	air -c .air.toml

## 测试
test:
	$(GO) test -race -cover ./...

test-v:
	$(GO) test -race -cover -v ./...

test-integration:
	$(GO) test -tags=integration -race ./test/integration/...

coverage:
	$(GO) test -race -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html

## 代码质量
lint:
	$(GOLINT) run ./...

lint-fix:
	$(GOLINT) run --fix ./...

fmt:
	$(GOFMT) -s -w .

vet:
	$(GO) vet ./...

## 代码生成
proto:
	./scripts/gen-proto.sh

wire:
	@for svc in $(SERVICES); do \
		wire ./cmd/$$svc 2>/dev/null || echo "wire not configured for $$svc"; \
	done

generate:
	$(GO) generate ./...

## 数据库迁移
migrate-up:
	migrate -path migrations/postgres -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path migrations/postgres -database "$(DATABASE_URL)" down

migrate-create:
	migrate create -ext sql -dir migrations/postgres -seq $(name)

## Docker
docker-build:
	@for svc in $(SERVICES); do \
		docker build -t z-novel-ai/$$svc:$(VERSION) -f deployments/docker/Dockerfile.$$svc . 2>/dev/null || echo "Dockerfile for $$svc not found"; \
	done

docker-push:
	@for svc in $(SERVICES); do \
		docker push z-novel-ai/$$svc:$(VERSION) 2>/dev/null || echo "Image z-novel-ai/$$svc:$(VERSION) not found"; \
	done

## 清理
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html
	$(GO) clean -cache

## 帮助
help:
	@echo "Available targets:"
	@echo "  all            - Run lint, test, and build"
	@echo "  deps           - Download dependencies"
	@echo "  tidy           - Tidy go.mod"
	@echo "  build          - Build all services"
	@echo "  build-<svc>    - Build specific service"
	@echo "  run            - Run api-gateway"
	@echo "  run-dev        - Run api-gateway in development mode"
	@echo "  run-air        - Run with hot reload (requires air)"
	@echo "  test           - Run tests"
	@echo "  test-v         - Run tests with verbose output"
	@echo "  coverage       - Generate coverage report"
	@echo "  lint           - Run linter"
	@echo "  fmt            - Format code"
	@echo "  proto          - Generate protobuf code"
	@echo "  wire           - Generate dependency injection code"
	@echo "  migrate-up     - Run database migrations"
	@echo "  migrate-down   - Rollback database migrations"
	@echo "  docker-build   - Build Docker images"
	@echo "  clean          - Clean build artifacts"
