// Package server 提供 gRPC 服务端实现
package server

import (
	"context"

	storyv1 "z-novel-ai-api/api/proto/gen/go/story"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// StoryGenService gRPC StoryGenService 服务端（占位实现）
type StoryGenService struct {
	storyv1.UnimplementedStoryGenServiceServer
}

func (s *StoryGenService) GenerateChapter(ctx context.Context, req *storyv1.GenerateChapterRequest) (*storyv1.GenerateChapterResponse, error) {
	return nil, status.Error(codes.Unimplemented, "story generation not implemented")
}

func (s *StoryGenService) StreamGenerateChapter(req *storyv1.GenerateChapterRequest, stream grpc.ServerStreamingServer[storyv1.GenerateChapterChunk]) error {
	return status.Error(codes.Unimplemented, "story streaming not implemented")
}

func (s *StoryGenService) GenerateSummary(ctx context.Context, req *storyv1.GenerateSummaryRequest) (*storyv1.GenerateSummaryResponse, error) {
	return nil, status.Error(codes.Unimplemented, "summary generation not implemented")
}
