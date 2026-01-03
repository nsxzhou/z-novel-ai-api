// Package entity 定义领域实体
package entity

import (
	"time"
)

// VolumeStatus 卷状态
type VolumeStatus string

const (
	VolumeStatusDraft     VolumeStatus = "draft"
	VolumeStatusWriting   VolumeStatus = "writing"
	VolumeStatusCompleted VolumeStatus = "completed"
)

// Volume 卷/部实体
type Volume struct {
	ID          string       `json:"id"`
	ProjectID   string       `json:"project_id"`
	SeqNum      int          `json:"seq_num"`
	Title       string       `json:"title,omitempty"`
	Description string       `json:"description,omitempty"`
	Summary     string       `json:"summary,omitempty"`
	WordCount   int          `json:"word_count"`
	Status      VolumeStatus `json:"status"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

// NewVolume 创建新卷
func NewVolume(projectID string, seqNum int, title string) *Volume {
	now := time.Now()
	return &Volume{
		ProjectID: projectID,
		SeqNum:    seqNum,
		Title:     title,
		WordCount: 0,
		Status:    VolumeStatusDraft,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// UpdateWordCount 更新字数统计
func (v *Volume) UpdateWordCount(delta int) {
	v.WordCount += delta
	v.UpdatedAt = time.Now()
}
