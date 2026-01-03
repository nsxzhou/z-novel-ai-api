// Package dto 提供 HTTP 层数据传输对象
package dto

import (
	"time"

	"z-novel-ai-api/internal/domain/entity"
)

// CreateChapterRequest 创建章节请求
type CreateChapterRequest struct {
	Title          string `json:"title" binding:"max=255"`
	Outline        string `json:"outline" binding:"max=10000"`
	VolumeID       string `json:"volume_id,omitempty"`
	StoryTimeStart int64  `json:"story_time_start,omitempty"`
	Notes          string `json:"notes" binding:"max=2000"`
}

// UpdateChapterRequest 更新章节请求
type UpdateChapterRequest struct {
	Title          *string `json:"title,omitempty" binding:"omitempty,max=255"`
	Outline        *string `json:"outline,omitempty" binding:"omitempty,max=10000"`
	ContentText    *string `json:"content_text,omitempty"`
	Summary        *string `json:"summary,omitempty" binding:"omitempty,max=5000"`
	Notes          *string `json:"notes,omitempty" binding:"omitempty,max=2000"`
	StoryTimeStart *int64  `json:"story_time_start,omitempty"`
	StoryTimeEnd   *int64  `json:"story_time_end,omitempty"`
	Status         *string `json:"status,omitempty"`
}

// GenerateChapterRequest 生成章节请求
type GenerateChapterRequest struct {
	Title           string             `json:"title" binding:"max=255"`
	Outline         string             `json:"outline" binding:"required,max=10000"`
	VolumeID        string             `json:"volume_id,omitempty"`
	TargetWordCount int                `json:"target_word_count" binding:"gte=500,lte=10000"`
	StoryTimeStart  int64              `json:"story_time_start,omitempty"`
	Notes           string             `json:"notes" binding:"max=2000"`
	Options         *GenerationOptions `json:"options,omitempty"`
}

// GenerationOptions 生成选项
type GenerationOptions struct {
	Model          string  `json:"model,omitempty"`
	Temperature    float64 `json:"temperature,omitempty"`
	SkipValidation bool    `json:"skip_validation,omitempty"`
	MaxRetries     int     `json:"max_retries,omitempty"`
}

// RegenerateChapterRequest 重新生成章节请求
type RegenerateChapterRequest struct {
	Outline         string             `json:"outline,omitempty" binding:"omitempty,max=10000"`
	TargetWordCount int                `json:"target_word_count,omitempty" binding:"omitempty,gte=500,lte=10000"`
	Options         *GenerationOptions `json:"options,omitempty"`
}

// ChapterResponse 章节响应
type ChapterResponse struct {
	ID                 string                      `json:"id"`
	ProjectID          string                      `json:"project_id"`
	VolumeID           string                      `json:"volume_id,omitempty"`
	SeqNum             int                         `json:"seq_num"`
	Title              string                      `json:"title,omitempty"`
	Outline            string                      `json:"outline,omitempty"`
	ContentText        string                      `json:"content_text,omitempty"`
	Summary            string                      `json:"summary,omitempty"`
	Notes              string                      `json:"notes,omitempty"`
	StoryTimeStart     int64                       `json:"story_time_start,omitempty"`
	StoryTimeEnd       int64                       `json:"story_time_end,omitempty"`
	WordCount          int                         `json:"word_count"`
	Status             string                      `json:"status"`
	GenerationMetadata *GenerationMetadataResponse `json:"generation_metadata,omitempty"`
	Version            int                         `json:"version"`
	CreatedAt          time.Time                   `json:"created_at"`
	UpdatedAt          time.Time                   `json:"updated_at"`
}

// GenerationMetadataResponse 生成元数据响应
type GenerationMetadataResponse struct {
	Model            string  `json:"model,omitempty"`
	Provider         string  `json:"provider,omitempty"`
	PromptTokens     int     `json:"prompt_tokens,omitempty"`
	CompletionTokens int     `json:"completion_tokens,omitempty"`
	Temperature      float64 `json:"temperature,omitempty"`
	GeneratedAt      string  `json:"generated_at,omitempty"`
}

// ChapterListResponse 章节列表响应
type ChapterListResponse struct {
	Chapters []*ChapterResponse `json:"chapters"`
}

// ToChapterResponse 将领域实体转换为响应 DTO
func ToChapterResponse(c *entity.Chapter) *ChapterResponse {
	if c == nil {
		return nil
	}

	resp := &ChapterResponse{
		ID:             c.ID,
		ProjectID:      c.ProjectID,
		VolumeID:       c.VolumeID,
		SeqNum:         c.SeqNum,
		Title:          c.Title,
		Outline:        c.Outline,
		ContentText:    c.ContentText,
		Summary:        c.Summary,
		Notes:          c.Notes,
		StoryTimeStart: c.StoryTimeStart,
		StoryTimeEnd:   c.StoryTimeEnd,
		WordCount:      c.WordCount,
		Status:         string(c.Status),
		Version:        c.Version,
		CreatedAt:      c.CreatedAt,
		UpdatedAt:      c.UpdatedAt,
	}

	if c.GenerationMetadata != nil {
		resp.GenerationMetadata = &GenerationMetadataResponse{
			Model:            c.GenerationMetadata.Model,
			Provider:         c.GenerationMetadata.Provider,
			PromptTokens:     c.GenerationMetadata.PromptTokens,
			CompletionTokens: c.GenerationMetadata.CompletionTokens,
			Temperature:      c.GenerationMetadata.Temperature,
			GeneratedAt:      c.GenerationMetadata.GeneratedAt,
		}
	}

	return resp
}

// ToChapterListResponse 将领域实体列表转换为响应 DTO
func ToChapterListResponse(chapters []*entity.Chapter) *ChapterListResponse {
	resp := &ChapterListResponse{
		Chapters: make([]*ChapterResponse, 0, len(chapters)),
	}

	for _, c := range chapters {
		resp.Chapters = append(resp.Chapters, ToChapterResponse(c))
	}

	return resp
}

// ToChapterEntity 将请求 DTO 转换为领域实体
func (r *CreateChapterRequest) ToChapterEntity(projectID string, seqNum int) *entity.Chapter {
	chapter := entity.NewChapter(projectID, r.VolumeID, seqNum)
	chapter.Title = r.Title
	chapter.Outline = r.Outline
	chapter.Notes = r.Notes
	chapter.StoryTimeStart = r.StoryTimeStart

	return chapter
}

// ApplyToChapter 将更新请求应用到章节实体
func (r *UpdateChapterRequest) ApplyToChapter(c *entity.Chapter) {
	if r.Title != nil {
		c.Title = *r.Title
	}
	if r.Outline != nil {
		c.Outline = *r.Outline
	}
	if r.ContentText != nil {
		c.SetContent(*r.ContentText)
	}
	if r.Summary != nil {
		c.Summary = *r.Summary
	}
	if r.Notes != nil {
		c.Notes = *r.Notes
	}
	if r.StoryTimeStart != nil {
		c.StoryTimeStart = *r.StoryTimeStart
	}
	if r.StoryTimeEnd != nil {
		c.StoryTimeEnd = *r.StoryTimeEnd
	}
	if r.Status != nil {
		c.Status = entity.ChapterStatus(*r.Status)
	}

	c.UpdatedAt = time.Now()
}
