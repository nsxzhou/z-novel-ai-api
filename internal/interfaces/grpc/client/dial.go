// Package client 提供 gRPC 客户端连接创建
package client

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Dial 创建到目标地址的 gRPC 连接。
func Dial(ctx context.Context, target string, timeout time.Duration, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	base := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}
	base = append(base, opts...)

	conn, err := grpc.DialContext(ctx, target, base...)
	if err != nil {
		return nil, fmt.Errorf("failed to dial %s: %w", target, err)
	}
	return conn, nil
}
