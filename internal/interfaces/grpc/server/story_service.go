// Package server 提供 gRPC 服务端实现
package server

import (
	"context"
	"fmt"
	"time"

	storyv1 "z-novel-ai-api/api/proto/gen/go/story"

	"google.golang.org/grpc"
)

// StoryGenService gRPC StoryGenService 服务端（待完善）
type StoryGenService struct {
	storyv1.UnimplementedStoryGenServiceServer
}

func (s *StoryGenService) GenerateChapter(ctx context.Context, req *storyv1.GenerateChapterRequest) (*storyv1.GenerateChapterResponse, error) {
	content := buildChapterContent(req.GetOutline(), int(req.GetTargetWordCount()))
	wordCount := int32(len([]rune(content)))
	temperature := float64(0)
	if opt := req.GetOptions(); opt != nil {
		temperature = opt.GetTemperature()
	}

	return &storyv1.GenerateChapterResponse{
		ChapterId: req.GetChapterId(),
		Content:   content,
		WordCount: wordCount,
		Metadata: &storyv1.GenerationMetadata{
			Model:            chooseModel(req.GetOptions()),
			Provider:         "placeholder",
			PromptTokens:     0,
			CompletionTokens: 0,
			Temperature:      temperature,
			GeneratedAt:      time.Now().Unix(),
		},
	}, nil
}

func (s *StoryGenService) StreamGenerateChapter(req *storyv1.GenerateChapterRequest, stream grpc.ServerStreamingServer[storyv1.GenerateChapterChunk]) error {
	content := buildChapterContent(req.GetOutline(), int(req.GetTargetWordCount()))
	runes := []rune(content)
	chunkSize := 200
	temperature := float64(0)
	if opt := req.GetOptions(); opt != nil {
		temperature = opt.GetTemperature()
	}

	for i := 0; i < len(runes); i += chunkSize {
		end := i + chunkSize
		if end > len(runes) {
			end = len(runes)
		}

		if err := stream.Send(&storyv1.GenerateChapterChunk{
			Payload: &storyv1.GenerateChapterChunk_ContentChunk{
				ContentChunk: string(runes[i:end]),
			},
		}); err != nil {
			return err
		}
	}

	return stream.Send(&storyv1.GenerateChapterChunk{
		Payload: &storyv1.GenerateChapterChunk_Metadata{
			Metadata: &storyv1.GenerationMetadata{
				Model:            chooseModel(req.GetOptions()),
				Provider:         "placeholder",
				PromptTokens:     0,
				CompletionTokens: 0,
				Temperature:      temperature,
				GeneratedAt:      time.Now().Unix(),
			},
		},
	})
}

func (s *StoryGenService) GenerateSummary(ctx context.Context, req *storyv1.GenerateSummaryRequest) (*storyv1.GenerateSummaryResponse, error) {
	content := []rune(req.GetContent())
	maxLen := int(req.GetMaxLength())
	if maxLen <= 0 || maxLen > len(content) {
		maxLen = len(content)
	}
	return &storyv1.GenerateSummaryResponse{
		Summary:    string(content[:maxLen]),
		TokensUsed: 0,
	}, nil
}

func buildChapterContent(outline string, targetWordCount int) string {
	if outline == "" {
		outline = "（无大纲）"
	}
	if targetWordCount <= 0 {
		targetWordCount = 2000
	}
	return fmt.Sprintf("章节大纲：%s\n\n（目标字数：%d）\n", outline, targetWordCount)
}

func chooseModel(opt *storyv1.GenerationOptions) string {
	if opt != nil && opt.GetModel() != "" {
		return opt.GetModel()
	}
	return "default"
}
