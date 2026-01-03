// Package main Memory gRPC 服务入口
package main

import (
	"context"
	"fmt"
	"os"

	"google.golang.org/grpc"

	memoryv1 "z-novel-ai-api/api/proto/gen/go/memory"
	"z-novel-ai-api/internal/config"
	"z-novel-ai-api/internal/infrastructure/persistence/postgres"
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
		ServiceName: "memory-svc",
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

	pgClient, err := postgres.NewClient(&cfg.Database.Postgres)
	if err != nil {
		logger.Fatal(ctx, "failed to init postgres", err)
	}
	defer func() { _ = pgClient.Close() }()

	txMgr := postgres.NewTxManager(pgClient)
	tenantCtx := postgres.NewTenantContext(pgClient)
	entityRepo := postgres.NewEntityRepository(pgClient)

	if err := grpcserver.Run(ctx, cfg, func(s *grpc.Server) {
		memoryv1.RegisterMemoryServiceServer(s, grpcserver.NewMemoryService(txMgr, tenantCtx, entityRepo))
	}); err != nil {
		logger.Fatal(ctx, "grpc server exited", err)
	}
}
