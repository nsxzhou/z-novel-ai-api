// Package handler 提供 HTTP 请求处理器
package handler

import (
	retrievalv1 "z-novel-ai-api/api/proto/gen/go/retrieval"
	"z-novel-ai-api/internal/interfaces/http/dto"

	"github.com/gin-gonic/gin"
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
	dto.Error(c, 501, "retrieval not implemented")
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
	dto.Error(c, 501, "retrieval debug not implemented")
}
