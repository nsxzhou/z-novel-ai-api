// Package handler 提供 HTTP 请求处理器
package handler

import (
	"net/http"

	"z-novel-ai-api/internal/domain/repository"
	"z-novel-ai-api/internal/interfaces/http/dto"
	"z-novel-ai-api/pkg/errors"
	"z-novel-ai-api/pkg/logger"

	"github.com/gin-gonic/gin"
)

// VolumeHandler 卷处理器
type VolumeHandler struct {
	volumeRepo repository.VolumeRepository
}

// NewVolumeHandler 创建卷处理器
func NewVolumeHandler(volumeRepo repository.VolumeRepository) *VolumeHandler {
	return &VolumeHandler{
		volumeRepo: volumeRepo,
	}
}

// ListVolumes 获取卷列表
// @Summary 获取项目卷列表
// @Description 获取指定项目下的所有卷信息
// @Tags Volumes
// @Accept json
// @Produce json
// @Param pid path string true "项目 ID"
// @Success 200 {object} dto.Response[dto.VolumeListResponse]
// @Failure 500 {object} dto.ErrorResponse
// @Router /v1/projects/{pid}/volumes [get]
func (h *VolumeHandler) ListVolumes(c *gin.Context) {
	ctx := c.Request.Context()
	projectID := dto.BindProjectID(c)

	result, err := h.volumeRepo.ListByProject(ctx, projectID)
	if err != nil {
		logger.Error(ctx, "failed to list volumes", err)
		dto.InternalError(c, "failed to list volumes")
		return
	}

	resp := dto.ToVolumeListResponse(result)
	dto.Success(c, resp)
}

// CreateVolume 创建卷
// @Summary 创建卷
// @Description 在指定项目下创建新卷
// @Tags Volumes
// @Accept json
// @Produce json
// @Param pid path string true "项目 ID"
// @Param body body dto.CreateVolumeRequest true "卷信息"
// @Success 201 {object} dto.Response[dto.VolumeResponse]
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /v1/projects/{pid}/volumes [post]
func (h *VolumeHandler) CreateVolume(c *gin.Context) {
	ctx := c.Request.Context()
	projectID := dto.BindProjectID(c)

	var req dto.CreateVolumeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequest(c, "invalid request body: "+err.Error())
		return
	}

	// 获取下一个序号
	nextSeq, err := h.volumeRepo.GetNextSeqNum(ctx, projectID)
	if err != nil {
		logger.Error(ctx, "failed to get next seq num", err)
		dto.InternalError(c, "failed to create volume")
		return
	}

	volume := req.ToVolumeEntity(projectID, nextSeq)

	if err := h.volumeRepo.Create(ctx, volume); err != nil {
		logger.Error(ctx, "failed to create volume", err)
		dto.InternalError(c, "failed to create volume")
		return
	}

	resp := dto.ToVolumeResponse(volume)
	dto.Created(c, resp)
}

// GetVolume 获取卷详情
// @Summary 获取卷详情
// @Description 获取指定卷的详细记录
// @Tags Volumes
// @Accept json
// @Produce json
// @Param vid path string true "卷 ID"
// @Success 200 {object} dto.Response[dto.VolumeResponse]
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /v1/volumes/{vid} [get]
func (h *VolumeHandler) GetVolume(c *gin.Context) {
	ctx := c.Request.Context()
	volumeID := dto.BindVolumeID(c)

	volume, err := h.volumeRepo.GetByID(ctx, volumeID)
	if err != nil {
		if errors.IsAppError(err) {
			appErr := errors.AsAppError(err)
			c.JSON(appErr.HTTPStatus, dto.ErrorResponse{
				Code:    appErr.HTTPStatus,
				Message: appErr.Message,
				TraceID: c.GetString("trace_id"),
			})
			return
		}
		logger.Error(ctx, "failed to get volume", err)
		dto.InternalError(c, "failed to get volume")
		return
	}

	if volume == nil {
		dto.NotFound(c, "volume not found")
		return
	}

	resp := dto.ToVolumeResponse(volume)
	dto.Success(c, resp)
}

// UpdateVolume 更新卷
// @Summary 更新卷信息
// @Description 更新卷的标题、描述、状态等
// @Tags Volumes
// @Accept json
// @Produce json
// @Param vid path string true "卷 ID"
// @Param body body dto.UpdateVolumeRequest true "更新内容"
// @Success 200 {object} dto.Response[dto.VolumeResponse]
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /v1/volumes/{vid} [put]
func (h *VolumeHandler) UpdateVolume(c *gin.Context) {
	ctx := c.Request.Context()
	volumeID := dto.BindVolumeID(c)

	var req dto.UpdateVolumeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequest(c, "invalid request body: "+err.Error())
		return
	}

	// 获取现有卷
	volume, err := h.volumeRepo.GetByID(ctx, volumeID)
	if err != nil {
		logger.Error(ctx, "failed to get volume", err)
		dto.InternalError(c, "failed to get volume")
		return
	}

	if volume == nil {
		dto.NotFound(c, "volume not found")
		return
	}

	// 应用更新
	req.ApplyToVolume(volume)

	// 保存更新
	if err := h.volumeRepo.Update(ctx, volume); err != nil {
		logger.Error(ctx, "failed to update volume", err)
		dto.InternalError(c, "failed to update volume")
		return
	}

	resp := dto.ToVolumeResponse(volume)
	dto.Success(c, resp)
}

// DeleteVolume 删除卷
// @Summary 删除卷
// @Description 删除指定卷记录
// @Tags Volumes
// @Accept json
// @Produce json
// @Param vid path string true "卷 ID"
// @Success 204 "No Content"
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /v1/volumes/{vid} [delete]
func (h *VolumeHandler) DeleteVolume(c *gin.Context) {
	ctx := c.Request.Context()
	volumeID := dto.BindVolumeID(c)

	if err := h.volumeRepo.Delete(ctx, volumeID); err != nil {
		if errors.IsAppError(err) {
			appErr := errors.AsAppError(err)
			c.JSON(appErr.HTTPStatus, dto.ErrorResponse{
				Code:    appErr.HTTPStatus,
				Message: appErr.Message,
				TraceID: c.GetString("trace_id"),
			})
			return
		}
		logger.Error(ctx, "failed to delete volume", err)
		dto.InternalError(c, "failed to delete volume")
		return
	}

	c.Status(http.StatusNoContent)
}

// ReorderVolumes 卷重新排序
// @Summary 卷重新排序
// @Description 调整项目下卷的顺序
// @Tags Volumes
// @Accept json
// @Produce json
// @Param pid path string true "项目 ID"
// @Param body body dto.ReorderVolumesRequest true "排序列表"
// @Success 200 {object} dto.Response[map[string]interface{}]
// @Failure 500 {object} dto.ErrorResponse
// @Router /v1/projects/{pid}/volumes/reorder [post]
func (h *VolumeHandler) ReorderVolumes(c *gin.Context) {
	ctx := c.Request.Context()
	projectID := dto.BindProjectID(c)

	var req dto.ReorderVolumesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequest(c, "invalid request body: "+err.Error())
		return
	}

	if err := h.volumeRepo.ReorderVolumes(ctx, projectID, req.VolumeIDs); err != nil {
		logger.Error(ctx, "failed to reorder volumes", err)
		dto.InternalError(c, "failed to reorder volumes")
		return
	}

	dto.Success(c, gin.H{"message": "volumes reordered"})
}
