// Package server 提供 gRPC 服务端实现
package server

import (
	"context"

	memoryv1 "z-novel-ai-api/api/proto/gen/go/memory"
	"z-novel-ai-api/internal/domain/repository"

	"github.com/google/uuid"
)

// MemoryService gRPC MemoryService 服务端（待完善）
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
	// 待补充完善
	return &memoryv1.StoreMemoryResponse{
		MemoryId: uuid.NewString(),
		Success:  true,
	}, nil
}

func (s *MemoryService) GetMemory(ctx context.Context, req *memoryv1.GetMemoryRequest) (*memoryv1.GetMemoryResponse, error) {
	if s.txMgr == nil || s.tenantCtx == nil || s.entityRepo == nil {
		return &memoryv1.GetMemoryResponse{}, nil
	}

	tenantID := ""
	if req.GetContext() != nil {
		tenantID = req.GetContext().GetTenantId()
	}

	out := &memoryv1.GetMemoryResponse{Entities: []*memoryv1.EntityMemory{}}
	_ = s.txMgr.WithTransaction(ctx, func(txCtx context.Context) error {
		if tenantID != "" {
			_ = s.tenantCtx.SetTenant(txCtx, tenantID)
		}
		for _, id := range req.GetEntityIds() {
			e, err := s.entityRepo.GetByID(txCtx, id)
			if err != nil || e == nil {
				continue
			}
			out.Entities = append(out.Entities, &memoryv1.EntityMemory{
				EntityId:     e.ID,
				EntityName:   e.Name,
				CurrentState: e.CurrentState,
				History:      []*memoryv1.StateHistory{},
			})
		}
		return nil
	})

	return out, nil
}

func (s *MemoryService) UpdateEntityState(ctx context.Context, req *memoryv1.UpdateEntityStateRequest) (*memoryv1.UpdateEntityStateResponse, error) {
	if s.txMgr == nil || s.tenantCtx == nil || s.entityRepo == nil {
		return &memoryv1.UpdateEntityStateResponse{Success: false}, nil
	}

	tenantID := ""
	if req.GetContext() != nil {
		tenantID = req.GetContext().GetTenantId()
	}

	var prev string
	err := s.txMgr.WithTransaction(ctx, func(txCtx context.Context) error {
		if tenantID != "" {
			if err := s.tenantCtx.SetTenant(txCtx, tenantID); err != nil {
				return err
			}
		}
		e, err := s.entityRepo.GetByID(txCtx, req.GetEntityId())
		if err != nil {
			return err
		}
		if e == nil {
			return nil
		}
		prev = e.CurrentState
		return s.entityRepo.UpdateState(txCtx, e.ID, req.GetNewState())
	})

	if err != nil {
		return &memoryv1.UpdateEntityStateResponse{Success: false, PreviousState: prev}, nil
	}
	return &memoryv1.UpdateEntityStateResponse{Success: true, PreviousState: prev}, nil
}
