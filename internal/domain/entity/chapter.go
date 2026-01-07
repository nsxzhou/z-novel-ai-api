// Package entity 定义领域实体
package entity

import (
	"time"
)

// ChapterStatus 章节状态
type ChapterStatus string

const (
	ChapterStatusDraft      ChapterStatus = "draft"
	ChapterStatusGenerating ChapterStatus = "generating"
	ChapterStatusReview     ChapterStatus = "review"
	ChapterStatusCompleted  ChapterStatus = "completed"
)

// GenerationMetadata 生成元数据
type GenerationMetadata struct {
	Model            string  `json:"model,omitempty"`
	Provider         string  `json:"provider,omitempty"`
	PromptTokens     int     `json:"prompt_tokens,omitempty"`
	CompletionTokens int     `json:"completion_tokens,omitempty"`
	Temperature      float64 `json:"temperature,omitempty"`
	GeneratedAt      string  `json:"generated_at,omitempty"`
}

// Chapter 章节实体
type Chapter struct {
	ID                 string              `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ProjectID          string              `json:"project_id" gorm:"type:uuid;index;not null"`
	AIKey              string              `json:"ai_key,omitempty" gorm:"column:ai_key;type:varchar(128);index"`
	VolumeID           string              `json:"volume_id,omitempty" gorm:"type:uuid;index"`
	SeqNum             int                 `json:"seq_num" gorm:"not null"`
	Title              string              `json:"title,omitempty" gorm:"type:varchar(255)"`
	Outline            string              `json:"outline,omitempty" gorm:"type:text"`
	ContentText        string              `json:"content_text,omitempty" gorm:"type:text"`
	Summary            string              `json:"summary,omitempty" gorm:"type:text"`
	Notes              string              `json:"notes,omitempty" gorm:"type:text"`
	StoryTimeStart     int64               `json:"story_time_start,omitempty"`
	StoryTimeEnd       int64               `json:"story_time_end,omitempty"`
	WordCount          int                 `json:"word_count" gorm:"default:0"`
	Status             ChapterStatus       `json:"status" gorm:"type:varchar(50);default:'draft'"`
	GenerationMetadata *GenerationMetadata `json:"generation_metadata,omitempty" gorm:"type:jsonb;serializer:json"`
	Version            int                 `json:"version" gorm:"default:1"`
	CreatedAt          time.Time           `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt          time.Time           `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName 指定表名
func (Chapter) TableName() string {
	return "chapters"
}

// NewChapter 创建新章节
func NewChapter(projectID, volumeID string, seqNum int) *Chapter {
	now := time.Now()
	return &Chapter{
		ProjectID: projectID,
		VolumeID:  volumeID,
		SeqNum:    seqNum,
		WordCount: 0,
		Status:    ChapterStatusDraft,
		Version:   1,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// SetContent 设置章节内容
func (c *Chapter) SetContent(content string) {
	c.ContentText = content
	c.WordCount = len([]rune(content))
	c.UpdatedAt = time.Now()
}

// IsEditable 检查章节是否可编辑
func (c *Chapter) IsEditable() bool {
	return c.Status == ChapterStatusDraft || c.Status == ChapterStatusReview
}

// IncrementVersion 增加版本号
func (c *Chapter) IncrementVersion() {
	c.Version++
	c.UpdatedAt = time.Now()
}
