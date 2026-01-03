// Package main StoryGen gRPC 服务入口
package main

import (
	"context"
	"fmt"
	"os"

	"google.golang.org/grpc"

	storyv1 "z-novel-ai-api/api/proto/gen/go/story"
	"z-novel-ai-api/internal/config"
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
		ServiceName: "story-gen-svc",
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

	if err := grpcserver.Run(ctx, cfg, func(s *grpc.Server) {
		storyv1.RegisterStoryGenServiceServer(s, &grpcserver.StoryGenService{})
	}); err != nil {
		logger.Fatal(ctx, "grpc server exited", err)
	}
}
