// Package server 提供 gRPC 服务端启动与注册封装
package server

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"

	"z-novel-ai-api/internal/config"
	"z-novel-ai-api/pkg/logger"
)

// Run 启动 gRPC Server，并在退出信号时优雅停止。
func Run(ctx context.Context, cfg *config.Config, register func(s *grpc.Server)) error {
	addr := fmt.Sprintf("%s:%d", cfg.Server.GRPC.Host, cfg.Server.GRPC.Port)

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	s := grpc.NewServer(
		grpc.MaxRecvMsgSize(cfg.Server.GRPC.MaxRecvMsgSize),
		grpc.MaxSendMsgSize(cfg.Server.GRPC.MaxSendMsgSize),
	)

	if register != nil {
		register(s)
	}

	log := logger.FromContext(ctx)
	log.Info("grpc server starting", "addr", addr)

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.Serve(lis)
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		log.Info("grpc server shutting down")
		s.GracefulStop()
		return nil
	case err := <-errCh:
		return fmt.Errorf("grpc server error: %w", err)
	}
}
