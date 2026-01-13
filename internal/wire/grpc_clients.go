// Package wire 提供依赖注入配置
package wire

import (
	"context"

	"google.golang.org/grpc"

	memoryv1 "z-novel-ai-api/api/proto/gen/go/memory"
	retrievalv1 "z-novel-ai-api/api/proto/gen/go/retrieval"
	storyv1 "z-novel-ai-api/api/proto/gen/go/story"
	validatorv1 "z-novel-ai-api/api/proto/gen/go/validator"
	"z-novel-ai-api/internal/config"
	grpcclient "z-novel-ai-api/internal/interfaces/grpc/client"
)

type RetrievalGRPCConn *grpc.ClientConn
type StoryGenGRPCConn *grpc.ClientConn
type MemoryGRPCConn *grpc.ClientConn
type ValidatorGRPCConn *grpc.ClientConn

// ProvideRetrievalGRPCConn 提供检索服务 gRPC 连接
func ProvideRetrievalGRPCConn(ctx context.Context, cfg *config.Config) (RetrievalGRPCConn, func(), error) {
	// 注意：已移除 features.* 功能开关；是否启用 gRPC clients 由 Wire 注入图决定。
	conn, err := grpcclient.Dial(ctx, cfg.Clients.GRPC.RetrievalServiceAddr, cfg.Clients.GRPC.DialTimeout)
	if err != nil {
		return nil, nil, err
	}
	return RetrievalGRPCConn(conn), func() { _ = conn.Close() }, nil
}

// ProvideRetrievalGRPCClient 提供检索服务 gRPC Client
func ProvideRetrievalGRPCClient(conn RetrievalGRPCConn) retrievalv1.RetrievalServiceClient {
	if conn == nil {
		return nil
	}
	return retrievalv1.NewRetrievalServiceClient((*grpc.ClientConn)(conn))
}

func ProvideStoryGenGRPCConn(ctx context.Context, cfg *config.Config) (StoryGenGRPCConn, func(), error) {
	conn, err := grpcclient.Dial(ctx, cfg.Clients.GRPC.StoryGenServiceAddr, cfg.Clients.GRPC.DialTimeout)
	if err != nil {
		return nil, nil, err
	}
	return StoryGenGRPCConn(conn), func() { _ = conn.Close() }, nil
}

func ProvideStoryGenGRPCClient(conn StoryGenGRPCConn) storyv1.StoryGenServiceClient {
	if conn == nil {
		return nil
	}
	return storyv1.NewStoryGenServiceClient((*grpc.ClientConn)(conn))
}

func ProvideMemoryGRPCConn(ctx context.Context, cfg *config.Config) (MemoryGRPCConn, func(), error) {
	conn, err := grpcclient.Dial(ctx, cfg.Clients.GRPC.MemoryServiceAddr, cfg.Clients.GRPC.DialTimeout)
	if err != nil {
		return nil, nil, err
	}
	return MemoryGRPCConn(conn), func() { _ = conn.Close() }, nil
}

func ProvideMemoryGRPCClient(conn MemoryGRPCConn) memoryv1.MemoryServiceClient {
	if conn == nil {
		return nil
	}
	return memoryv1.NewMemoryServiceClient((*grpc.ClientConn)(conn))
}

func ProvideValidatorGRPCConn(ctx context.Context, cfg *config.Config) (ValidatorGRPCConn, func(), error) {
	conn, err := grpcclient.Dial(ctx, cfg.Clients.GRPC.ValidatorServiceAddr, cfg.Clients.GRPC.DialTimeout)
	if err != nil {
		return nil, nil, err
	}
	return ValidatorGRPCConn(conn), func() { _ = conn.Close() }, nil
}

func ProvideValidatorGRPCClient(conn ValidatorGRPCConn) validatorv1.ValidatorServiceClient {
	if conn == nil {
		return nil
	}
	return validatorv1.NewValidatorServiceClient((*grpc.ClientConn)(conn))
}
