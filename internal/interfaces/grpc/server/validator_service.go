// Package server 提供 gRPC 服务端实现
package server

import (
	"context"

	validatorv1 "z-novel-ai-api/api/proto/gen/go/validator"
)

// ValidatorService gRPC ValidatorService 服务端（待完善）
type ValidatorService struct {
	validatorv1.UnimplementedValidatorServiceServer
}

func (s *ValidatorService) ValidateChapter(ctx context.Context, req *validatorv1.ValidateChapterRequest) (*validatorv1.ValidateChapterResponse, error) {
	if req.GetContent() == "" {
		return &validatorv1.ValidateChapterResponse{
			Valid: false,
			Issues: []*validatorv1.ValidationIssue{
				{
					Type:       "content",
					Severity:   "error",
					Message:    "empty content",
					Suggestion: "provide non-empty content",
				},
			},
		}, nil
	}
	return &validatorv1.ValidateChapterResponse{Valid: true, Issues: []*validatorv1.ValidationIssue{}}, nil
}

func (s *ValidatorService) ValidateConsistency(ctx context.Context, req *validatorv1.ValidateConsistencyRequest) (*validatorv1.ValidateConsistencyResponse, error) {
	return &validatorv1.ValidateConsistencyResponse{
		Consistent: true,
		Issues:     []*validatorv1.ConsistencyIssue{},
	}, nil
}
