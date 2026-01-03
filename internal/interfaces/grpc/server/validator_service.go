// Package server 提供 gRPC 服务端实现
package server

import (
	"context"

	validatorv1 "z-novel-ai-api/api/proto/gen/go/validator"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ValidatorService gRPC ValidatorService 服务端（占位实现）
type ValidatorService struct {
	validatorv1.UnimplementedValidatorServiceServer
}

func (s *ValidatorService) ValidateChapter(ctx context.Context, req *validatorv1.ValidateChapterRequest) (*validatorv1.ValidateChapterResponse, error) {
	return nil, status.Error(codes.Unimplemented, "validator not implemented")
}

func (s *ValidatorService) ValidateConsistency(ctx context.Context, req *validatorv1.ValidateConsistencyRequest) (*validatorv1.ValidateConsistencyResponse, error) {
	return nil, status.Error(codes.Unimplemented, "validator not implemented")
}

