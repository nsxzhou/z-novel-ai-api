// Package handler 提供 HTTP 请求处理器
package handler

import (
	"errors"
	"strings"
	"time"

	appretrieval "z-novel-ai-api/internal/application/retrieval"
	"z-novel-ai-api/internal/application/story"
	"z-novel-ai-api/internal/domain/repository"
	"z-novel-ai-api/internal/interfaces/http/dto"
	"z-novel-ai-api/internal/interfaces/http/middleware"
	"z-novel-ai-api/pkg/logger"

	"github.com/gin-gonic/gin"
)

type ArtifactHandler struct {
	artifactRepo repository.ArtifactRepository
	indexer      *appretrieval.Indexer
}

func NewArtifactHandler(artifactRepo repository.ArtifactRepository, indexer *appretrieval.Indexer) *ArtifactHandler {
	return &ArtifactHandler{artifactRepo: artifactRepo, indexer: indexer}
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
	branchKey := strings.TrimSpace(c.Query("branch_key"))
	if branchKey != "" && !isValidBranchKey(branchKey) {
		dto.BadRequest(c, "invalid branch_key: "+branchKey)
		return
	}

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
	result, err := h.artifactRepo.ListVersions(ctx, artifactID, branchKey, repository.NewPagination(pageReq.Page, pageReq.PageSize))
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

// ListBranches 列出构件的分支头（每个 branch_key 的最新版本）
// @Summary 列出构件分支
// @Tags Artifacts
// @Accept json
// @Produce json
// @Param pid path string true "项目 ID"
// @Param aid path string true "构件 ID"
// @Success 200 {object} dto.Response[dto.ArtifactBranchListResponse]
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /v1/projects/{pid}/artifacts/{aid}/branches [get]
func (h *ArtifactHandler) ListBranches(c *gin.Context) {
	ctx := c.Request.Context()
	projectID := dto.BindProjectID(c)
	artifactID := dto.BindArtifactID(c)

	art, err := h.artifactRepo.GetArtifactByID(ctx, artifactID)
	if err != nil {
		logger.Error(ctx, "failed to get artifact", err)
		dto.InternalError(c, "failed to list branches")
		return
	}
	if art == nil || art.ProjectID != projectID {
		dto.NotFound(c, "artifact not found")
		return
	}

	heads, err := h.artifactRepo.ListBranchHeads(ctx, artifactID)
	if err != nil {
		logger.Error(ctx, "failed to list artifact branch heads", err)
		dto.InternalError(c, "failed to list branches")
		return
	}

	activeID := ""
	if art.ActiveVersionID != nil {
		activeID = strings.TrimSpace(*art.ActiveVersionID)
	}

	out := make([]*dto.ArtifactBranchHeadResponse, 0, len(heads))
	for i := range heads {
		v := heads[i]
		if v == nil {
			continue
		}
		out = append(out, &dto.ArtifactBranchHeadResponse{
			BranchKey:   v.BranchKey,
			HeadVersion: v.ID,
			HeadNo:      v.VersionNo,
			CreatedAt:   v.CreatedAt.UTC().Format(time.RFC3339),
			CreatedBy:   v.CreatedBy,
			SourceJobID: v.SourceJobID,
			IsActive:    activeID != "" && activeID == v.ID,
		})
	}

	dto.Success(c, &dto.ArtifactBranchListResponse{Branches: out})
}

// CompareVersions 对比两个版本（A/B 并行对比）
// @Summary 对比两个构件版本
// @Tags Artifacts
// @Accept json
// @Produce json
// @Param pid path string true "项目 ID"
// @Param aid path string true "构件 ID"
// @Param from_version_id query string true "起始版本 ID"
// @Param to_version_id query string true "目标版本 ID"
// @Success 200 {object} dto.Response[dto.ArtifactCompareResponse]
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /v1/projects/{pid}/artifacts/{aid}/compare [get]
func (h *ArtifactHandler) CompareVersions(c *gin.Context) {
	ctx := c.Request.Context()
	projectID := dto.BindProjectID(c)
	artifactID := dto.BindArtifactID(c)

	fromID := strings.TrimSpace(c.Query("from_version_id"))
	toID := strings.TrimSpace(c.Query("to_version_id"))
	if fromID == "" || toID == "" {
		dto.BadRequest(c, "from_version_id and to_version_id are required")
		return
	}

	art, err := h.artifactRepo.GetArtifactByID(ctx, artifactID)
	if err != nil {
		logger.Error(ctx, "failed to get artifact", err)
		dto.InternalError(c, "failed to compare versions")
		return
	}
	if art == nil || art.ProjectID != projectID {
		dto.NotFound(c, "artifact not found")
		return
	}

	fromV, err := h.artifactRepo.GetVersionByID(ctx, fromID)
	if err != nil {
		logger.Error(ctx, "failed to get from version", err)
		dto.InternalError(c, "failed to compare versions")
		return
	}
	toV, err := h.artifactRepo.GetVersionByID(ctx, toID)
	if err != nil {
		logger.Error(ctx, "failed to get to version", err)
		dto.InternalError(c, "failed to compare versions")
		return
	}
	if fromV == nil || fromV.ArtifactID != artifactID {
		dto.NotFound(c, "from version not found")
		return
	}
	if toV == nil || toV.ArtifactID != artifactID {
		dto.NotFound(c, "to version not found")
		return
	}

	diff, err := story.CompareArtifactContent(art.Type, fromV.Content, toV.Content)
	if err != nil {
		logger.Error(ctx, "failed to compare artifact content", err)
		dto.InternalError(c, "failed to compare versions")
		return
	}

	dto.Success(c, &dto.ArtifactCompareResponse{
		ArtifactID: art.ID,
		Type:       string(art.Type),
		From:       dto.ToArtifactVersionResponse(fromV),
		To:         dto.ToArtifactVersionResponse(toV),
		Diff:       diff,
	})
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
	tenantID := middleware.GetTenantIDFromGin(c)
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

	// 同步写索引：回滚只切 active_version_id，因此直接用目标版本内容重建索引。
	if h.indexer != nil {
		if err := h.indexer.IndexArtifactJSON(ctx, tenantID, projectID, art.Type, art.ID, version.Content); err != nil && !errors.Is(err, appretrieval.ErrVectorDisabled) {
			logger.Warn(ctx, "failed to index artifact after rollback",
				"error", err.Error(),
				"artifact_id", art.ID,
				"artifact_type", string(art.Type),
			)
		}
	}

	dto.Success(c, &dto.ArtifactRollbackResponse{
		Artifact: dto.ToArtifactResponse(art),
		Version:  dto.ToArtifactVersionResponse(version),
	})
}
