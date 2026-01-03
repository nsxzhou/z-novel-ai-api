// Package handler 提供 HTTP 请求处理器
package handler

import (
	commonv1 "z-novel-ai-api/api/proto/gen/go/common"
	retrievalv1 "z-novel-ai-api/api/proto/gen/go/retrieval"
	"z-novel-ai-api/internal/interfaces/http/dto"
	"z-novel-ai-api/internal/interfaces/http/middleware"
	"z-novel-ai-api/pkg/logger"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RetrievalHandler 检索处理器
type RetrievalHandler struct {
	client retrievalv1.RetrievalServiceClient
}

// NewRetrievalHandler 创建检索处理器
func NewRetrievalHandler(client retrievalv1.RetrievalServiceClient) *RetrievalHandler {
	return &RetrievalHandler{
		client: client,
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

	var req dto.SearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequest(c, "invalid request body: "+err.Error())
		return
	}

	// 设置默认值
	if req.TopK <= 0 {
		req.TopK = 20
	}
	if req.Options == nil {
		req.Options = &dto.RetrievalOption{
			VectorWeight:    0.7,
			KeywordWeight:   0.3,
			IncludeEntities: true,
			IncludeEvents:   true,
		}
	}

	tenantID := middleware.GetTenantID(ctx)
	traceID := c.GetString("trace_id")

	logger.Info(ctx, "search request received",
		"tenant_id", tenantID,
		"project_id", req.ProjectID,
		"query", req.Query,
		"top_k", req.TopK,
	)

	if h.client == nil {
		dto.ServiceUnavailable(c, "retrieval service not configured")
		return
	}

	resp, err := h.client.Search(ctx, &retrievalv1.SearchRequest{
		Context: &commonv1.TenantContext{
			TenantId: tenantID,
			UserId:   middleware.GetUserID(ctx),
			TraceId:  traceID,
		},
		ProjectId:        req.ProjectID,
		Query:            req.Query,
		CurrentStoryTime: req.CurrentStoryTime,
		TopK:             int32(req.TopK),
		Options: &retrievalv1.RetrievalOptions{
			VectorWeight:    req.Options.VectorWeight,
			KeywordWeight:   req.Options.KeywordWeight,
			IncludeEntities: req.Options.IncludeEntities,
			IncludeEvents:   req.Options.IncludeEvents,
			EntityTypes:     req.Options.EntityTypes,
		},
	})
	if err != nil {
		st := status.Convert(err)
		if st.Code() == codes.Unimplemented {
			dto.Error(c, 501, st.Message())
			return
		}
		logger.Error(ctx, "retrieval search failed", err)
		dto.InternalError(c, "retrieval search failed")
		return
	}

	httpResp := &dto.SearchResponse{
		Segments: make([]*dto.ContextSegment, 0, len(resp.GetSegments())),
		Entities: make([]*dto.EntityRef, 0, len(resp.GetEntities())),
	}
	for _, s := range resp.GetSegments() {
		httpResp.Segments = append(httpResp.Segments, &dto.ContextSegment{
			ID:        s.GetId(),
			Text:      s.GetText(),
			ChapterID: s.GetChapterId(),
			StoryTime: s.GetStoryTime(),
			Score:     s.GetScore(),
			Source:    s.GetSource(),
		})
	}
	for _, e := range resp.GetEntities() {
		httpResp.Entities = append(httpResp.Entities, &dto.EntityRef{
			ID:   e.GetId(),
			Name: e.GetName(),
			Type: e.GetType(),
		})
	}
	if m := resp.GetMetadata(); m != nil {
		httpResp.Metadata = &dto.RetrievalMeta{
			TotalSegments:       int(m.GetTotalSegments()),
			TotalEntities:       int(m.GetTotalEntities()),
			RetrievalDurationMs: m.GetRetrievalDurationMs(),
		}
	}

	dto.Success(c, httpResp)
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

	var req dto.DebugRetrievalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequest(c, "invalid request body: "+err.Error())
		return
	}

	// 设置默认值
	if req.TopK <= 0 {
		req.TopK = 20
	}
	if req.Options == nil {
		req.Options = &dto.RetrievalOption{
			VectorWeight:    0.7,
			KeywordWeight:   0.3,
			IncludeEntities: true,
			IncludeEvents:   true,
		}
	}

	tenantID := middleware.GetTenantID(ctx)
	traceID := c.GetString("trace_id")

	logger.Info(ctx, "debug retrieval request received",
		"tenant_id", tenantID,
		"project_id", req.ProjectID,
		"query", req.Query,
		"include_scores", req.IncludeScores,
		"include_embedding", req.IncludeEmbedding,
	)

	if h.client == nil {
		dto.ServiceUnavailable(c, "retrieval service not configured")
		return
	}

	resp, err := h.client.DebugSearch(ctx, &retrievalv1.DebugSearchRequest{
		Request: &retrievalv1.SearchRequest{
			Context: &commonv1.TenantContext{
				TenantId: tenantID,
				UserId:   middleware.GetUserID(ctx),
				TraceId:  traceID,
			},
			ProjectId:        req.ProjectID,
			Query:            req.Query,
			CurrentStoryTime: req.CurrentStoryTime,
			TopK:             int32(req.TopK),
			Options: &retrievalv1.RetrievalOptions{
				VectorWeight:    req.Options.VectorWeight,
				KeywordWeight:   req.Options.KeywordWeight,
				IncludeEntities: req.Options.IncludeEntities,
				IncludeEvents:   req.Options.IncludeEvents,
				EntityTypes:     req.Options.EntityTypes,
			},
		},
		IncludeScores:    req.IncludeScores,
		IncludeEmbedding: req.IncludeEmbedding,
	})
	if err != nil {
		st := status.Convert(err)
		if st.Code() == codes.Unimplemented {
			dto.Error(c, 501, st.Message())
			return
		}
		logger.Error(ctx, "retrieval debug search failed", err)
		dto.InternalError(c, "retrieval debug search failed")
		return
	}

	httpResp := &dto.DebugRetrievalResponse{}
	if sr := resp.GetResponse(); sr != nil {
		httpResp.Segments = make([]*dto.ContextSegment, 0, len(sr.GetSegments()))
		httpResp.Entities = make([]*dto.EntityRef, 0, len(sr.GetEntities()))
		for _, s := range sr.GetSegments() {
			httpResp.Segments = append(httpResp.Segments, &dto.ContextSegment{
				ID:        s.GetId(),
				Text:      s.GetText(),
				ChapterID: s.GetChapterId(),
				StoryTime: s.GetStoryTime(),
				Score:     s.GetScore(),
				Source:    s.GetSource(),
			})
		}
		for _, e := range sr.GetEntities() {
			httpResp.Entities = append(httpResp.Entities, &dto.EntityRef{
				ID:   e.GetId(),
				Name: e.GetName(),
				Type: e.GetType(),
			})
		}
		if m := sr.GetMetadata(); m != nil {
			httpResp.Metadata = &dto.RetrievalMeta{
				TotalSegments:       int(m.GetTotalSegments()),
				TotalEntities:       int(m.GetTotalEntities()),
				RetrievalDurationMs: m.GetRetrievalDurationMs(),
			}
		}
	}
	httpResp.QueryEmbedding = resp.GetQueryEmbedding()
	if di := resp.GetDebugInfo(); di != nil {
		httpResp.DebugInfo = &dto.DebugInfo{
			VectorSearchTime:   di.GetVectorSearchTimeMs(),
			KeywordSearchTime:  di.GetKeywordSearchTimeMs(),
			FusionTime:         di.GetFusionTimeMs(),
			TotalCandidates:    int(di.GetTotalCandidates()),
			FilteredCandidates: int(di.GetFilteredCandidates()),
		}
	}

	dto.Success(c, httpResp)
}
