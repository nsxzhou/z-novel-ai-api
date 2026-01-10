// Package handler 提供 HTTP 请求处理器
package handler

import (
	"strings"
	"time"

	"z-novel-ai-api/internal/application/retrieval"
	"z-novel-ai-api/internal/interfaces/http/dto"
	"z-novel-ai-api/internal/interfaces/http/middleware"

	"github.com/gin-gonic/gin"
)

// RetrievalHandler 检索处理器
type RetrievalHandler struct {
	engine *retrieval.Engine
}

// NewRetrievalHandler 创建检索处理器
func NewRetrievalHandler(engine *retrieval.Engine) *RetrievalHandler {
	return &RetrievalHandler{
		engine: engine,
	}
}

// Search 检索上下文
// @Summary 检索上下文
// @Description 检索与查询相关的上下文信息
// @Tags Retrieval
// @Accept json
// @Produce json
// @Param body body dto.SearchRequest true "检索请求"
// @Success 200 {object} dto.Response[dto.SearchResponse]
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /v1/retrieval/search [post]
func (h *RetrievalHandler) Search(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID := middleware.GetTenantIDFromGin(c)

	var req dto.SearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequest(c, "invalid request body: "+err.Error())
		return
	}
	projectID := strings.TrimSpace(req.ProjectID)
	query := strings.TrimSpace(req.Query)
	if projectID == "" || query == "" {
		dto.BadRequest(c, "project_id and query are required")
		return
	}

	topK := req.TopK
	if topK <= 0 {
		topK = 10
	}

	if h.engine == nil {
		dto.InternalError(c, "retrieval engine not configured")
		return
	}

	start := time.Now()
	out, err := h.engine.Search(ctx, retrieval.SearchInput{
		TenantID:         tenantID,
		ProjectID:        projectID,
		Query:            query,
		CurrentStoryTime: req.CurrentStoryTime,
		TopK:             topK,
		IncludeEntities:  true,
	})
	if err != nil {
		dto.BadRequest(c, err.Error())
		return
	}

	resp := mapSearchOutput(out, time.Since(start))
	dto.Success(c, resp)
}

// DebugRetrieval 调试检索
// @Summary 调试检索
// @Description 检索并返回详细的调试信息
// @Tags Retrieval
// @Accept json
// @Produce json
// @Param body body dto.DebugRetrievalRequest true "调试检索请求"
// @Success 200 {object} dto.Response[dto.DebugRetrievalResponse]
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /v1/retrieval/debug [post]
func (h *RetrievalHandler) DebugRetrieval(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID := middleware.GetTenantIDFromGin(c)

	var req dto.DebugRetrievalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequest(c, "invalid request body: "+err.Error())
		return
	}
	projectID := strings.TrimSpace(req.ProjectID)
	query := strings.TrimSpace(req.Query)
	if projectID == "" || query == "" {
		dto.BadRequest(c, "project_id and query are required")
		return
	}

	topK := req.TopK
	if topK <= 0 {
		topK = 10
	}

	if h.engine == nil {
		dto.InternalError(c, "retrieval engine not configured")
		return
	}

	start := time.Now()
	out, err := h.engine.DebugSearch(ctx, retrieval.SearchInput{
		TenantID:         tenantID,
		ProjectID:        projectID,
		Query:            query,
		CurrentStoryTime: req.CurrentStoryTime,
		TopK:             topK,
		IncludeEntities:  true,
		IncludeEmbedding: req.IncludeEmbedding,
	})
	if err != nil {
		dto.BadRequest(c, err.Error())
		return
	}

	debugResp := &dto.DebugRetrievalResponse{
		SearchResponse: *mapSearchOutput(out, time.Since(start)),
	}
	if req.IncludeEmbedding {
		debugResp.QueryEmbedding = out.QueryEmbedding
	}
	if out.Debug != nil {
		debugResp.DebugInfo = &dto.DebugInfo{
			VectorSearchTime:   out.Debug.VectorSearchTimeMs,
			KeywordSearchTime:  0,
			FusionTime:         0,
			TotalCandidates:    out.Debug.TotalCandidates,
			FilteredCandidates: out.Debug.FilteredCandidates,
		}
	}

	dto.Success(c, debugResp)
}

func mapSearchOutput(out *retrieval.SearchOutput, elapsed time.Duration) *dto.SearchResponse {
	resp := &dto.SearchResponse{
		Segments: []*dto.ContextSegment{},
		Entities: []*dto.EntityRef{},
		Metadata: &dto.RetrievalMeta{
			TotalSegments:       0,
			TotalEntities:       0,
			RetrievalDurationMs: elapsed.Milliseconds(),
		},
	}
	if out == nil {
		return resp
	}
	if strings.TrimSpace(out.DisabledReason) != "" {
		resp.Metadata.DisabledReason = strings.TrimSpace(out.DisabledReason)
	}

	for i := range out.Segments {
		s := out.Segments[i]
		cs := &dto.ContextSegment{
			ID:           s.ID,
			Text:         s.Text,
			StoryTime:    s.StoryTime,
			Score:        s.Score,
			Source:       s.Source,
			DocType:      s.DocType,
			Title:        s.ChapterTitle,
			ArtifactID:   s.ArtifactID,
			ArtifactType: s.ArtifactType,
			RefPath:      s.RefPath,
		}
		if strings.TrimSpace(s.DocType) == "chapter" {
			cs.ChapterID = s.ChapterID
		}
		resp.Segments = append(resp.Segments, cs)
	}
	for i := range out.Entities {
		e := out.Entities[i]
		resp.Entities = append(resp.Entities, &dto.EntityRef{
			ID:   e.ID,
			Name: e.Name,
			Type: e.Type,
		})
	}

	resp.Metadata.TotalSegments = len(resp.Segments)
	resp.Metadata.TotalEntities = len(resp.Entities)
	return resp
}
