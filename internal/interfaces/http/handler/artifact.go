// Package handler 提供 HTTP 请求处理器
package handler

import (
	"z-novel-ai-api/internal/domain/repository"
	"z-novel-ai-api/internal/interfaces/http/dto"
	"z-novel-ai-api/pkg/logger"

	"github.com/gin-gonic/gin"
)

type ArtifactHandler struct {
	artifactRepo repository.ArtifactRepository
}

func NewArtifactHandler(artifactRepo repository.ArtifactRepository) *ArtifactHandler {
	return &ArtifactHandler{artifactRepo: artifactRepo}
}

// ListArtifacts 列出项目下构件
// @Summary 列出项目下构件
// @Tags Artifacts
// @Accept json
// @Produce json
// @Param pid path string true "项目 ID"
// @Success 200 {object} dto.Response[dto.ArtifactListResponse]
// @Failure 500 {object} dto.ErrorResponse
// @Router /v1/projects/{pid}/artifacts [get]
func (h *ArtifactHandler) ListArtifacts(c *gin.Context) {
	ctx := c.Request.Context()
	projectID := dto.BindProjectID(c)

	arts, err := h.artifactRepo.ListArtifactsByProject(ctx, projectID)
	if err != nil {
		logger.Error(ctx, "failed to list artifacts", err)
		dto.InternalError(c, "failed to list artifacts")
		return
	}

	out := make([]*dto.ArtifactResponse, 0, len(arts))
	for i := range arts {
		out = append(out, dto.ToArtifactResponse(arts[i]))
	}
	dto.Success(c, &dto.ArtifactListResponse{Artifacts: out})
}

// ListVersions 列出构件版本
// @Summary 列出构件版本
// @Tags Artifacts
// @Accept json
// @Produce json
// @Param pid path string true "项目 ID"
// @Param aid path string true "构件 ID"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页条数" default(20)
// @Success 200 {object} dto.Response[dto.ArtifactVersionListResponse]
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /v1/projects/{pid}/artifacts/{aid}/versions [get]
func (h *ArtifactHandler) ListVersions(c *gin.Context) {
	ctx := c.Request.Context()
	projectID := dto.BindProjectID(c)
	artifactID := dto.BindArtifactID(c)

	art, err := h.artifactRepo.GetArtifactByID(ctx, artifactID)
	if err != nil {
		logger.Error(ctx, "failed to get artifact", err)
		dto.InternalError(c, "failed to list versions")
		return
	}
	if art == nil || art.ProjectID != projectID {
		dto.NotFound(c, "artifact not found")
		return
	}

	pageReq := dto.BindPage(c)
	result, err := h.artifactRepo.ListVersions(ctx, artifactID, repository.NewPagination(pageReq.Page, pageReq.PageSize))
	if err != nil {
		logger.Error(ctx, "failed to list artifact versions", err)
		dto.InternalError(c, "failed to list versions")
		return
	}

	versions := make([]*dto.ArtifactVersionResponse, 0, len(result.Items))
	for i := range result.Items {
		versions = append(versions, dto.ToArtifactVersionResponse(result.Items[i]))
	}
	dto.SuccessWithPage(c, &dto.ArtifactVersionListResponse{Versions: versions}, dto.NewPageMeta(pageReq.Page, pageReq.PageSize, int(result.Total)))
}

// Rollback 回滚构件到指定版本（只切 active_version_id）
// @Summary 回滚构件到指定版本
// @Tags Artifacts
// @Accept json
// @Produce json
// @Param pid path string true "项目 ID"
// @Param aid path string true "构件 ID"
// @Param body body dto.ArtifactRollbackRequest true "回滚请求"
// @Success 200 {object} dto.Response[dto.ArtifactRollbackResponse]
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /v1/projects/{pid}/artifacts/{aid}/rollback [post]
func (h *ArtifactHandler) Rollback(c *gin.Context) {
	ctx := c.Request.Context()
	projectID := dto.BindProjectID(c)
	artifactID := dto.BindArtifactID(c)

	var req dto.ArtifactRollbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequest(c, "invalid request body: "+err.Error())
		return
	}

	art, err := h.artifactRepo.GetArtifactByID(ctx, artifactID)
	if err != nil {
		logger.Error(ctx, "failed to get artifact", err)
		dto.InternalError(c, "failed to rollback")
		return
	}
	if art == nil || art.ProjectID != projectID {
		dto.NotFound(c, "artifact not found")
		return
	}

	version, err := h.artifactRepo.GetVersionByID(ctx, req.VersionID)
	if err != nil {
		logger.Error(ctx, "failed to get artifact version", err)
		dto.InternalError(c, "failed to rollback")
		return
	}
	if version == nil || version.ArtifactID != artifactID {
		dto.NotFound(c, "version not found")
		return
	}

	if err := h.artifactRepo.SetActiveVersion(ctx, artifactID, version.ID); err != nil {
		logger.Error(ctx, "failed to set active version", err)
		dto.InternalError(c, "failed to rollback")
		return
	}

	dto.Success(c, &dto.ArtifactRollbackResponse{
		Artifact: dto.ToArtifactResponse(art),
		Version:  dto.ToArtifactVersionResponse(version),
	})
}
