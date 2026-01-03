// Package handler 提供 HTTP 请求处理器
package handler

import (
	"context"
	"fmt"
	"io"
	"time"

	commonv1 "z-novel-ai-api/api/proto/gen/go/common"
	storyv1 "z-novel-ai-api/api/proto/gen/go/story"
	"z-novel-ai-api/internal/domain/entity"
	"z-novel-ai-api/internal/domain/repository"
	"z-novel-ai-api/internal/interfaces/http/dto"
	"z-novel-ai-api/internal/interfaces/http/middleware"
	"z-novel-ai-api/pkg/logger"

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
	ctx := c.Request.Context()
	chapterID := dto.BindChapterID(c)

	tenantID := middleware.GetTenantID(ctx)
	if tenantID == "" {
		dto.BadRequest(c, "missing tenant_id")
		return
	}

	if h.txMgr == nil || h.tenantCtx == nil || h.storyClient == nil {
		dto.ServiceUnavailable(c, "stream dependencies not configured")
		return
	}

	// 短事务读取章节（避免 SSE 持有事务连接）
	var chapter *entity.Chapter
	if err := h.withTenantTx(ctx, tenantID, func(txCtx context.Context) error {
		var err error
		chapter, err = h.chapterRepo.GetByID(txCtx, chapterID)
		return err
	}); err != nil {
		logger.Error(ctx, "failed to get chapter", err)
		dto.InternalError(c, "failed to get chapter")
		return
	}
	if chapter == nil {
		dto.NotFound(c, "chapter not found")
		return
	}

	// 设置 SSE 响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Header("Access-Control-Allow-Origin", "*")

	// 发送开始事件
	c.SSEvent("start", gin.H{
		"chapter_id": chapterID,
		"timestamp":  time.Now().Unix(),
	})

	// 调用 story-gen-svc 流式生成并透传
	stream, err := h.storyClient.StreamGenerateChapter(ctx, &storyv1.GenerateChapterRequest{
		Context: &commonv1.TenantContext{
			TenantId: tenantID,
			TraceId:  c.GetString("trace_id"),
		},
		ProjectId:       chapter.ProjectID,
		ChapterId:       chapter.ID,
		Outline:         chapter.Outline,
		TargetWordCount: 0,
	})
	if err != nil {
		dto.InternalError(c, "failed to start generation stream")
		return
	}

	var contentBuf []rune
	var meta *storyv1.GenerationMetadata
	index := 0

	c.Stream(func(w io.Writer) bool {
		select {
		case <-ctx.Done():
			return false
		default:
		}

		chunk, err := stream.Recv()
		if err != nil {
			// stream end 或错误
			wordCount := len(contentBuf)
			if meta != nil {
				c.SSEvent("metadata", gin.H{
					"word_count":        wordCount,
					"tokens_used":       meta.GetPromptTokens() + meta.GetCompletionTokens(),
					"model":             meta.GetModel(),
					"provider":          meta.GetProvider(),
					"prompt_tokens":     meta.GetPromptTokens(),
					"completion_tokens": meta.GetCompletionTokens(),
				})
			}

			finalContent := string(contentBuf)
			if finalContent != "" {
				_ = h.withTenantTx(ctx, tenantID, func(txCtx context.Context) error {
					ch, err := h.chapterRepo.GetByID(txCtx, chapterID)
					if err != nil || ch == nil {
						return err
					}
					ch.SetContent(finalContent)
					ch.Status = entity.ChapterStatusCompleted
					if meta != nil {
						ch.GenerationMetadata = &entity.GenerationMetadata{
							Model:            meta.GetModel(),
							Provider:         meta.GetProvider(),
							PromptTokens:     int(meta.GetPromptTokens()),
							CompletionTokens: int(meta.GetCompletionTokens()),
							Temperature:      meta.GetTemperature(),
							GeneratedAt:      time.Unix(meta.GetGeneratedAt(), 0).Format(time.RFC3339),
						}
					}
					return h.chapterRepo.Update(txCtx, ch)
				})
			}

			c.SSEvent("done", gin.H{
				"chapter_id":   chapterID,
				"total_chunks": index,
			})
			return false
		}

		if errMsg := chunk.GetError(); errMsg != "" {
			c.SSEvent("error", gin.H{"message": errMsg})
			return false
		}
		if m := chunk.GetMetadata(); m != nil {
			meta = m
			c.SSEvent("metadata", gin.H{
				"model":             m.GetModel(),
				"provider":          m.GetProvider(),
				"prompt_tokens":     m.GetPromptTokens(),
				"completion_tokens": m.GetCompletionTokens(),
				"temperature":       m.GetTemperature(),
				"generated_at":      m.GetGeneratedAt(),
			})
			return true
		}

		text := chunk.GetContentChunk()
		if text != "" {
			contentBuf = append(contentBuf, []rune(text)...)
			c.SSEvent("content", gin.H{
				"chunk": text,
				"index": index,
			})
			index++
			return true
		}

		return true
	})
}

func (h *StreamHandler) withTenantTx(ctx context.Context, tenantID string, fn func(context.Context) error) error {
	if h.txMgr == nil || h.tenantCtx == nil {
		return fmt.Errorf("transaction dependencies not configured")
	}
	return h.txMgr.WithTransaction(ctx, func(txCtx context.Context) error {
		if err := h.tenantCtx.SetTenant(txCtx, tenantID); err != nil {
			return err
		}
		return fn(txCtx)
	})
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
