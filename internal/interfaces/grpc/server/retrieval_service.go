// Package server 提供 gRPC 服务端实现
package server

import (
	"context"

	retrievalv1 "z-novel-ai-api/api/proto/gen/go/retrieval"
	"z-novel-ai-api/internal/infrastructure/persistence/milvus"

	"github.com/cloudwego/eino/components/embedding"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RetrievalService gRPC RetrievalService 服务端（占位实现）
type RetrievalService struct {
	retrievalv1.UnimplementedRetrievalServiceServer

	embedder   embedding.Embedder
	milvusRepo *milvus.Repository
}

func NewRetrievalService(embedder embedding.Embedder, milvusRepo *milvus.Repository) *RetrievalService {
	return &RetrievalService{
		embedder:   embedder,
		milvusRepo: milvusRepo,
	}
}

func (s *RetrievalService) Search(ctx context.Context, req *retrievalv1.SearchRequest) (*retrievalv1.SearchResponse, error) {
	return nil, status.Error(codes.Unimplemented, "retrieval search not implemented")
}

func (s *RetrievalService) DebugSearch(ctx context.Context, req *retrievalv1.DebugSearchRequest) (*retrievalv1.DebugSearchResponse, error) {
	return nil, status.Error(codes.Unimplemented, "retrieval debug not implemented")
}

func (s *RetrievalService) IndexDocument(ctx context.Context, req *retrievalv1.IndexDocumentRequest) (*retrievalv1.IndexDocumentResponse, error) {
	return nil, status.Error(codes.Unimplemented, "retrieval indexing not implemented")
}
