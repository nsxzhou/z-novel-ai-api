// Package handler 提供 HTTP 请求处理器
package handler

import (
	"io"

	storyv1 "z-novel-ai-api/api/proto/gen/go/story"
	"z-novel-ai-api/internal/domain/repository"
	"z-novel-ai-api/internal/interfaces/http/dto"

	"github.com/gin-gonic/gin"
)

// StreamHandler 流式响应处理器
type StreamHandler struct {
	chapterRepo repository.ChapterRepository
	txMgr       repository.Transactor
	tenantCtx   repository.TenantContextManager
	storyClient storyv1.StoryGenServiceClient
}

// NewStreamHandler 创建流式响应处理器
func NewStreamHandler(
	chapterRepo repository.ChapterRepository,
	txMgr repository.Transactor,
	tenantCtx repository.TenantContextManager,
	storyClient storyv1.StoryGenServiceClient,
) *StreamHandler {
	return &StreamHandler{
		chapterRepo: chapterRepo,
		txMgr:       txMgr,
		tenantCtx:   tenantCtx,
		storyClient: storyClient,
	}
}

// StreamChapter 流式获取章节内容
// @Summary 流式获取章节内容
// @Description 通过 SSE 流式获取章节生成内容
// @Tags Chapters
// @Accept json
// @Produce text/event-stream
// @Param cid path string true "章节 ID"
// @Success 200 "SSE stream"
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /v1/chapters/{cid}/stream [get]
func (h *StreamHandler) StreamChapter(c *gin.Context) {
	dto.Error(c, 501, "chapter streaming not implemented")
}

// StreamGenerate 流式生成内容（内部方法）
// 用于实际的生成流程，由 GenerationService 调用
func (h *StreamHandler) StreamGenerate(c *gin.Context, contentChan <-chan string, metaChan <-chan map[string]interface{}, errChan <-chan error) {
	// 设置 SSE 响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	index := 0

	c.Stream(func(w io.Writer) bool {
		select {
		case chunk, ok := <-contentChan:
			if !ok {
				// 内容通道关闭
				return false
			}
			c.SSEvent("content", gin.H{
				"chunk": chunk,
				"index": index,
			})
			index++
			return true

		case meta, ok := <-metaChan:
			if !ok {
				return true // 元数据通道关闭，继续等待内容
			}
			c.SSEvent("metadata", meta)
			return true

		case err, ok := <-errChan:
			if !ok {
				return true // 错误通道关闭
			}
			c.SSEvent("error", gin.H{
				"message": err.Error(),
			})
			return false

		case <-c.Request.Context().Done():
			// 客户端断开
			return false
		}
	})
}
