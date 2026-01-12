// Package dto 提供 HTTP 层数据传输对象
package dto

import (
	wfmodel "z-novel-ai-api/internal/workflow/model"
)

// ConversationMessageRequest 通用对话消息请求
type ConversationMessageRequest struct {
	Prompt      string                     `json:"prompt" binding:"required"`
	Attachments []FoundationTextAttachment `json:"attachments,omitempty"`

	Provider    string   `json:"provider,omitempty"`
	Model       string   `json:"model,omitempty"`
	Temperature *float32 `json:"temperature,omitempty"`
	MaxTokens   *int     `json:"max_tokens,omitempty"`
}

func (r *ConversationMessageRequest) ToStoryAttachments() []wfmodel.TextAttachment {
	if r == nil {
		return nil
	}
	out := make([]wfmodel.TextAttachment, 0, len(r.Attachments))
	for i := range r.Attachments {
		a := r.Attachments[i]
		out = append(out, wfmodel.TextAttachment{
			Name:    a.Name,
			Content: a.Content,
		})
	}
	return out
}

// ConversationTurnResponse 通用对话轮次响应
type ConversationTurnResponse struct {
	ID        string `json:"id"`
	Role      string `json:"role"`
	Content   string `json:"content"`
	Metadata  any    `json:"metadata,omitempty"`
	CreatedAt string `json:"created_at"`
}

// ConversationTurnListResponse 通用对话轮次列表响应
type ConversationTurnListResponse struct {
	Turns []*ConversationTurnResponse `json:"turns"`
}
