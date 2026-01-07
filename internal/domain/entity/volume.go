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
	ID          string       `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ProjectID   string       `json:"project_id" gorm:"type:uuid;index;not null"`
	AIKey       string       `json:"ai_key,omitempty" gorm:"column:ai_key;type:varchar(128);index"`
	SeqNum      int          `json:"seq_num" gorm:"not null"`
	Title       string       `json:"title,omitempty" gorm:"type:varchar(255)"`
	Description string       `json:"description,omitempty" gorm:"type:text"`
	Summary     string       `json:"summary,omitempty" gorm:"type:text"`
	WordCount   int          `json:"word_count" gorm:"default:0"`
	Status      VolumeStatus `json:"status" gorm:"type:varchar(50);default:'draft'"`
	CreatedAt   time.Time    `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time    `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName 指定表名
func (Volume) TableName() string {
	return "volumes"
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
