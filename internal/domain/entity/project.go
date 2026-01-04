// Package entity 定义领域实体
package entity

import (
	"time"
)

// ProjectStatus 项目状态
type ProjectStatus string

const (
	ProjectStatusDraft     ProjectStatus = "draft"
	ProjectStatusWriting   ProjectStatus = "writing"
	ProjectStatusCompleted ProjectStatus = "completed"
	ProjectStatusArchived  ProjectStatus = "archived"
)

// WorldSettings 世界观设置
type WorldSettings struct {
	TimeSystem string   `json:"time_system,omitempty"`
	Calendar   string   `json:"calendar,omitempty"`
	Locations  []string `json:"locations,omitempty"`
}

// ProjectSettings 项目设置
type ProjectSettings struct {
	DefaultChapterLength int     `json:"default_chapter_length,omitempty"`
	WritingStyle         string  `json:"writing_style,omitempty"`
	POV                  string  `json:"pov,omitempty"`
	Temperature          float64 `json:"temperature,omitempty"`
}

// Project 小说项目实体
type Project struct {
	ID               string           `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TenantID         string           `json:"tenant_id" gorm:"type:uuid;index;not null"`
	OwnerID          string           `json:"owner_id,omitempty" gorm:"type:uuid;index"`
	Title            string           `json:"title" gorm:"type:varchar(255);not null"`
	Description      string           `json:"description,omitempty" gorm:"type:text"`
	Genre            string           `json:"genre,omitempty" gorm:"type:varchar(100)"`
	TargetWordCount  int              `json:"target_word_count,omitempty"`
	CurrentWordCount int              `json:"current_word_count" gorm:"default:0"`
	Settings         *ProjectSettings `json:"settings,omitempty" gorm:"type:jsonb;serializer:json"`
	WorldSettings    *WorldSettings   `json:"world_settings,omitempty" gorm:"type:jsonb;serializer:json"`
	Status           ProjectStatus    `json:"status" gorm:"type:varchar(50);default:'draft'"`
	CreatedAt        time.Time        `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt        time.Time        `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName 指定表名
func (Project) TableName() string {
	return "projects"
}

// NewProject 创建新项目
func NewProject(tenantID, ownerID, title string) *Project {
	now := time.Now()
	return &Project{
		TenantID:         tenantID,
		OwnerID:          ownerID,
		Title:            title,
		CurrentWordCount: 0,
		Status:           ProjectStatusDraft,
		Settings:         &ProjectSettings{},
		WorldSettings: &WorldSettings{
			TimeSystem: "linear",
			Calendar:   "custom",
			Locations:  []string{},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// IsEditable 检查项目是否可编辑
func (p *Project) IsEditable() bool {
	return p.Status == ProjectStatusDraft || p.Status == ProjectStatusWriting
}

// UpdateWordCount 更新字数统计
func (p *Project) UpdateWordCount(delta int) {
	p.CurrentWordCount += delta
	p.UpdatedAt = time.Now()
}
