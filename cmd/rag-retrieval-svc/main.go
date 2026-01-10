// Package main Retrieval gRPC 服务入口
package main

import (
	"context"
	"fmt"
	"os"

	"google.golang.org/grpc"

	retrievalv1 "z-novel-ai-api/api/proto/gen/go/retrieval"
	"z-novel-ai-api/internal/config"
	"z-novel-ai-api/internal/infrastructure/embedding"
	"z-novel-ai-api/internal/infrastructure/persistence/milvus"
	grpcserver "z-novel-ai-api/internal/interfaces/grpc/server"
	"z-novel-ai-api/pkg/logger"
	"z-novel-ai-api/pkg/tracer"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	logger.Init(cfg.Observability.Logging.Level, cfg.Observability.Logging.Format)
	ctx := context.Background()

	shutdown, err := tracer.Init(ctx, tracer.Config{
		ServiceName: "rag-retrieval-svc",
		Endpoint:    cfg.Observability.Tracing.Endpoint,
		SampleRate:  cfg.Observability.Tracing.SampleRate,
		Enabled:     cfg.Observability.Tracing.Enabled,
	})
	if err != nil {
		logger.Fatal(ctx, "failed to init tracer", err)
	}
	defer func() {
		_ = shutdown(ctx)
	}()

	milvusClient, err := milvus.NewClient(ctx, &cfg.Vector.Milvus)
	if err != nil {
		logger.Fatal(ctx, "failed to init milvus", err)
	}
	defer func() { _ = milvusClient.Close() }()

	milvusRepo := milvus.NewRepository(milvusClient)

	// 使用 Eino Embedder
	embedder, err := embedding.NewEinoEmbedder(ctx, &cfg.Embedding)
	if err != nil {
		logger.Fatal(ctx, "failed to init eino embedder", err)
	}

	if err := grpcserver.Run(ctx, cfg, func(s *grpc.Server) {
		retrievalv1.RegisterRetrievalServiceServer(s, grpcserver.NewRetrievalService(embedder, milvusRepo))
	}); err != nil {
		logger.Fatal(ctx, "grpc server exited", err)
	}
}
