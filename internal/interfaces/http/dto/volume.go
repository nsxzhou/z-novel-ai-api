// Package dto 提供 HTTP 层数据传输对象
package dto

import (
	"time"

	"z-novel-ai-api/internal/domain/entity"
)

// CreateVolumeRequest 创建卷请求
type CreateVolumeRequest struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description"`
	Summary     string `json:"summary"`
}

// UpdateVolumeRequest 更新卷请求
type UpdateVolumeRequest struct {
	Title       *string              `json:"title"`
	Description *string              `json:"description"`
	Summary     *string              `json:"summary"`
	Status      *entity.VolumeStatus `json:"status"`
}

// VolumeResponse 卷响应
type VolumeResponse struct {
	ID          string              `json:"id"`
	ProjectID   string              `json:"project_id"`
	SeqNum      int                 `json:"seq_num"`
	Title       string              `json:"title"`
	Description string              `json:"description,omitempty"`
	Summary     string              `json:"summary,omitempty"`
	WordCount   int                 `json:"word_count"`
	Status      entity.VolumeStatus `json:"status"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
}

// VolumeListResponse 卷列表响应
type VolumeListResponse struct {
	Items []*VolumeResponse `json:"items"`
}

// ReorderVolumesRequest 重新排序卷请求
type ReorderVolumesRequest struct {
	VolumeIDs []string `json:"volume_ids" binding:"required"`
}

// ToVolumeEntity 转换为实体
func (r *CreateVolumeRequest) ToVolumeEntity(projectID string, seqNum int) *entity.Volume {
	return entity.NewVolume(projectID, seqNum, r.Title)
}

// ToVolumeResponse 实体转换为响应
func ToVolumeResponse(v *entity.Volume) *VolumeResponse {
	if v == nil {
		return nil
	}
	return &VolumeResponse{
		ID:          v.ID,
		ProjectID:   v.ProjectID,
		SeqNum:      v.SeqNum,
		Title:       v.Title,
		Description: v.Description,
		Summary:     v.Summary,
		WordCount:   v.WordCount,
		Status:      v.Status,
		CreatedAt:   v.CreatedAt,
		UpdatedAt:   v.UpdatedAt,
	}
}

// ToVolumeListResponse 实体列表转换为响应
func ToVolumeListResponse(volumes []*entity.Volume) *VolumeListResponse {
	items := make([]*VolumeResponse, len(volumes))
	for i, v := range volumes {
		items[i] = ToVolumeResponse(v)
	}
	return &VolumeListResponse{Items: items}
}

// ApplyToVolume 更新实体
func (r *UpdateVolumeRequest) ApplyToVolume(v *entity.Volume) {
	if r.Title != nil {
		v.Title = *r.Title
	}
	if r.Description != nil {
		v.Description = *r.Description
	}
	if r.Summary != nil {
		v.Summary = *r.Summary
	}
	if r.Status != nil {
		v.Status = *r.Status
	}
	v.UpdatedAt = time.Now()
}
