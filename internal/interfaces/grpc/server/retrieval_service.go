// Package server 提供 gRPC 服务端实现
package server

import (
	"context"
	"fmt"
	"time"

	commonv1 "z-novel-ai-api/api/proto/gen/go/common"
	retrievalv1 "z-novel-ai-api/api/proto/gen/go/retrieval"
	"z-novel-ai-api/internal/infrastructure/embedding"
	"z-novel-ai-api/internal/infrastructure/persistence/milvus"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RetrievalService gRPC RetrievalService 服务端实现
type RetrievalService struct {
	retrievalv1.UnimplementedRetrievalServiceServer

	embedClient *embedding.Client
	milvusRepo  *milvus.Repository
}

func NewRetrievalService(embedClient *embedding.Client, milvusRepo *milvus.Repository) *RetrievalService {
	return &RetrievalService{
		embedClient: embedClient,
		milvusRepo:  milvusRepo,
	}
}

func (s *RetrievalService) Search(ctx context.Context, req *retrievalv1.SearchRequest) (*retrievalv1.SearchResponse, error) {
	if s.embedClient == nil || s.milvusRepo == nil {
		return nil, status.Error(codes.FailedPrecondition, "retrieval service not initialized")
	}
	if req.GetContext() == nil || req.GetContext().GetTenantId() == "" {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}
	if req.GetProjectId() == "" {
		return nil, status.Error(codes.InvalidArgument, "missing project_id")
	}
	if req.GetQuery() == "" {
		return nil, status.Error(codes.InvalidArgument, "missing query")
	}

	start := time.Now()

	embeddings, err := s.embedClient.Embed(ctx, []string{req.GetQuery()})
	if err != nil || len(embeddings) == 0 {
		return nil, status.Errorf(codes.Internal, "embedding failed: %v", err)
	}

	topK := int(req.GetTopK())
	if topK <= 0 {
		topK = 20
	}
	opt := req.GetOptions()
	vecWeight := float32(0.7)
	kwWeight := float32(0.3)
	if opt != nil {
		vecWeight = float32(opt.GetVectorWeight())
		kwWeight = float32(opt.GetKeywordWeight())
	}

	results, err := s.milvusRepo.HybridSearch(ctx, &milvus.HybridSearchParams{
		TenantID:         req.GetContext().GetTenantId(),
		ProjectID:        req.GetProjectId(),
		QueryVector:      embeddings[0],
		QueryText:        req.GetQuery(),
		CurrentStoryTime: req.GetCurrentStoryTime(),
		TopK:             topK,
		VectorWeight:     vecWeight,
		KeywordWeight:    kwWeight,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "milvus search failed: %v", err)
	}

	resp := &retrievalv1.SearchResponse{
		Segments: []*retrievalv1.ContextSegment{},
		Entities: []*commonv1.EntityRef{},
		Metadata: &retrievalv1.RetrievalMetadata{
			TotalSegments:       int32(len(results)),
			TotalEntities:       0,
			RetrievalDurationMs: time.Since(start).Milliseconds(),
		},
	}

	for _, r := range results {
		resp.Segments = append(resp.Segments, &retrievalv1.ContextSegment{
			Id:        r.ID,
			Text:      r.TextContent,
			ChapterId: r.ChapterID,
			StoryTime: r.StoryTime,
			Score:     float64(r.Score),
			Source:    "vector",
		})
	}

	return resp, nil
}

func (s *RetrievalService) DebugSearch(ctx context.Context, req *retrievalv1.DebugSearchRequest) (*retrievalv1.DebugSearchResponse, error) {
	if req.GetRequest() == nil {
		return nil, status.Error(codes.InvalidArgument, "missing request")
	}

	start := time.Now()
	embeddings, err := s.embedClient.Embed(ctx, []string{req.GetRequest().GetQuery()})
	if err != nil || len(embeddings) == 0 {
		return nil, status.Errorf(codes.Internal, "embedding failed: %v", err)
	}

	searchResp, err := s.Search(ctx, req.GetRequest())
	if err != nil {
		return nil, err
	}

	out := &retrievalv1.DebugSearchResponse{
		Response: searchResp,
		DebugInfo: &retrievalv1.DebugInfo{
			VectorSearchTimeMs:  time.Since(start).Milliseconds(),
			KeywordSearchTimeMs: 0,
			FusionTimeMs:        0,
			TotalCandidates:     int32(len(searchResp.GetSegments())),
			FilteredCandidates:  int32(len(searchResp.GetSegments())),
		},
	}
	if req.GetIncludeEmbedding() {
		out.QueryEmbedding = embeddings[0]
	}
	return out, nil
}

func (s *RetrievalService) IndexDocument(ctx context.Context, req *retrievalv1.IndexDocumentRequest) (*retrievalv1.IndexDocumentResponse, error) {
	if s.embedClient == nil || s.milvusRepo == nil {
		return nil, status.Error(codes.FailedPrecondition, "retrieval service not initialized")
	}
	if req.GetContext() == nil || req.GetContext().GetTenantId() == "" {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}
	if req.GetProjectId() == "" || req.GetDocumentId() == "" || req.GetContent() == "" {
		return nil, status.Error(codes.InvalidArgument, "missing required fields")
	}

	embeddings, err := s.embedClient.Embed(ctx, []string{req.GetContent()})
	if err != nil || len(embeddings) == 0 {
		return nil, status.Errorf(codes.Internal, "embedding failed: %v", err)
	}

	seg := &milvus.StorySegment{
		ID:          req.GetDocumentId(),
		Vector:      embeddings[0],
		TenantID:    req.GetContext().GetTenantId(),
		ProjectID:   req.GetProjectId(),
		ChapterID:   "",
		StoryTime:   0,
		SegmentType: req.GetDocumentType(),
		TextContent: req.GetContent(),
	}
	if chapterID, ok := req.GetMetadata()["chapter_id"]; ok {
		seg.ChapterID = chapterID
	}
	if err := s.milvusRepo.InsertSegments(ctx, seg.TenantID, seg.ProjectID, []*milvus.StorySegment{seg}); err != nil {
		return nil, status.Errorf(codes.Internal, "insert failed: %v", err)
	}

	return &retrievalv1.IndexDocumentResponse{
		VectorId: seg.ID,
		Success:  true,
	}, nil
}

func requireTenant(ctx *commonv1.TenantContext) (string, error) {
	if ctx == nil || ctx.GetTenantId() == "" {
		return "", fmt.Errorf("missing tenant context")
	}
	return ctx.GetTenantId(), nil
}
