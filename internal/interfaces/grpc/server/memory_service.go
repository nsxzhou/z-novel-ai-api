// Package server 提供 gRPC 服务端实现
package server

import (
	"context"

	memoryv1 "z-novel-ai-api/api/proto/gen/go/memory"
	"z-novel-ai-api/internal/domain/repository"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// MemoryService gRPC MemoryService 服务端（占位实现）
type MemoryService struct {
	memoryv1.UnimplementedMemoryServiceServer

	txMgr      repository.Transactor
	tenantCtx  repository.TenantContextManager
	entityRepo repository.EntityRepository
}

func NewMemoryService(txMgr repository.Transactor, tenantCtx repository.TenantContextManager, entityRepo repository.EntityRepository) *MemoryService {
	return &MemoryService{
		txMgr:      txMgr,
		tenantCtx:  tenantCtx,
		entityRepo: entityRepo,
	}
}

func (s *MemoryService) StoreMemory(ctx context.Context, req *memoryv1.StoreMemoryRequest) (*memoryv1.StoreMemoryResponse, error) {
	return nil, status.Error(codes.Unimplemented, "memory store not implemented")
}

func (s *MemoryService) GetMemory(ctx context.Context, req *memoryv1.GetMemoryRequest) (*memoryv1.GetMemoryResponse, error) {
	return nil, status.Error(codes.Unimplemented, "memory get not implemented")
}

func (s *MemoryService) UpdateEntityState(ctx context.Context, req *memoryv1.UpdateEntityStateRequest) (*memoryv1.UpdateEntityStateResponse, error) {
	return nil, status.Error(codes.Unimplemented, "memory update entity state not implemented")
}

