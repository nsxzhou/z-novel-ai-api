// Package dto 提供 HTTP 层数据传输对象
package dto

import (
	"time"

	"z-novel-ai-api/internal/domain/entity"
)

// CreateProjectRequest 创建项目请求
type CreateProjectRequest struct {
	Title           string                  `json:"title" binding:"required,max=255"`
	Description     string                  `json:"description" binding:"max=5000"`
	Genre           string                  `json:"genre" binding:"max=50"`
	TargetWordCount int                     `json:"target_word_count" binding:"gte=0"`
	Settings        *ProjectSettingsRequest `json:"settings,omitempty"`
	WorldSettings   *WorldSettingsRequest   `json:"world_settings,omitempty"`
}

// UpdateProjectRequest 更新项目请求
type UpdateProjectRequest struct {
	Title           *string                 `json:"title,omitempty" binding:"omitempty,max=255"`
	Description     *string                 `json:"description,omitempty" binding:"omitempty,max=5000"`
	Genre           *string                 `json:"genre,omitempty" binding:"omitempty,max=50"`
	TargetWordCount *int                    `json:"target_word_count,omitempty" binding:"omitempty,gte=0"`
	Status          *string                 `json:"status,omitempty"`
	Settings        *ProjectSettingsRequest `json:"settings,omitempty"`
	WorldSettings   *WorldSettingsRequest   `json:"world_settings,omitempty"`
}

// ProjectSettingsRequest 项目设置请求
type ProjectSettingsRequest struct {
	DefaultChapterLength int     `json:"default_chapter_length,omitempty"`
	WritingStyle         string  `json:"writing_style,omitempty"`
	POV                  string  `json:"pov,omitempty"`
	Temperature          float64 `json:"temperature,omitempty"`
}

// WorldSettingsRequest 世界观设置请求
type WorldSettingsRequest struct {
	TimeSystem string   `json:"time_system,omitempty"`
	Calendar   string   `json:"calendar,omitempty"`
	Locations  []string `json:"locations,omitempty"`
}

// ProjectResponse 项目响应
type ProjectResponse struct {
	ID               string                   `json:"id"`
	TenantID         string                   `json:"tenant_id,omitempty"`
	OwnerID          string                   `json:"owner_id,omitempty"`
	Title            string                   `json:"title"`
	Description      string                   `json:"description,omitempty"`
	Genre            string                   `json:"genre,omitempty"`
	TargetWordCount  int                      `json:"target_word_count,omitempty"`
	CurrentWordCount int                      `json:"current_word_count"`
	Status           string                   `json:"status"`
	Settings         *ProjectSettingsResponse `json:"settings,omitempty"`
	WorldSettings    *WorldSettingsResponse   `json:"world_settings,omitempty"`
	CreatedAt        time.Time                `json:"created_at"`
	UpdatedAt        time.Time                `json:"updated_at"`
}

// ProjectSettingsResponse 项目设置响应
type ProjectSettingsResponse struct {
	DefaultChapterLength int     `json:"default_chapter_length,omitempty"`
	WritingStyle         string  `json:"writing_style,omitempty"`
	POV                  string  `json:"pov,omitempty"`
	Temperature          float64 `json:"temperature,omitempty"`
}

// WorldSettingsResponse 世界观设置响应
type WorldSettingsResponse struct {
	TimeSystem string   `json:"time_system,omitempty"`
	Calendar   string   `json:"calendar,omitempty"`
	Locations  []string `json:"locations,omitempty"`
}

// ProjectListResponse 项目列表响应
type ProjectListResponse struct {
	Projects []*ProjectResponse `json:"projects"`
}

// ToProjectResponse 将领域实体转换为响应 DTO
func ToProjectResponse(p *entity.Project) *ProjectResponse {
	if p == nil {
		return nil
	}

	resp := &ProjectResponse{
		ID:               p.ID,
		TenantID:         p.TenantID,
		OwnerID:          p.OwnerID,
		Title:            p.Title,
		Description:      p.Description,
		Genre:            p.Genre,
		TargetWordCount:  p.TargetWordCount,
		CurrentWordCount: p.CurrentWordCount,
		Status:           string(p.Status),
		CreatedAt:        p.CreatedAt,
		UpdatedAt:        p.UpdatedAt,
	}

	if p.Settings != nil {
		resp.Settings = &ProjectSettingsResponse{
			DefaultChapterLength: p.Settings.DefaultChapterLength,
			WritingStyle:         p.Settings.WritingStyle,
			POV:                  p.Settings.POV,
			Temperature:          p.Settings.Temperature,
		}
	}

	if p.WorldSettings != nil {
		resp.WorldSettings = &WorldSettingsResponse{
			TimeSystem: p.WorldSettings.TimeSystem,
			Calendar:   p.WorldSettings.Calendar,
			Locations:  p.WorldSettings.Locations,
		}
	}

	return resp
}

// ToProjectListResponse 将领域实体列表转换为响应 DTO
func ToProjectListResponse(projects []*entity.Project) *ProjectListResponse {
	resp := &ProjectListResponse{
		Projects: make([]*ProjectResponse, 0, len(projects)),
	}

	for _, p := range projects {
		resp.Projects = append(resp.Projects, ToProjectResponse(p))
	}

	return resp
}

// ToProjectEntity 将请求 DTO 转换为领域实体
func (r *CreateProjectRequest) ToProjectEntity(tenantID, ownerID string) *entity.Project {
	project := entity.NewProject(tenantID, ownerID, r.Title)
	project.Description = r.Description
	project.Genre = r.Genre
	project.TargetWordCount = r.TargetWordCount

	if r.Settings != nil {
		project.Settings = &entity.ProjectSettings{
			DefaultChapterLength: r.Settings.DefaultChapterLength,
			WritingStyle:         r.Settings.WritingStyle,
			POV:                  r.Settings.POV,
			Temperature:          r.Settings.Temperature,
		}
	}

	if r.WorldSettings != nil {
		project.WorldSettings = &entity.WorldSettings{
			TimeSystem: r.WorldSettings.TimeSystem,
			Calendar:   r.WorldSettings.Calendar,
			Locations:  r.WorldSettings.Locations,
		}
	}

	return project
}

// ApplyToProject 将更新请求应用到项目实体
func (r *UpdateProjectRequest) ApplyToProject(p *entity.Project) {
	if r.Title != nil {
		p.Title = *r.Title
	}
	if r.Description != nil {
		p.Description = *r.Description
	}
	if r.Genre != nil {
		p.Genre = *r.Genre
	}
	if r.TargetWordCount != nil {
		p.TargetWordCount = *r.TargetWordCount
	}
	if r.Status != nil {
		p.Status = entity.ProjectStatus(*r.Status)
	}

	if r.Settings != nil {
		if p.Settings == nil {
			p.Settings = &entity.ProjectSettings{}
		}
		if r.Settings.DefaultChapterLength > 0 {
			p.Settings.DefaultChapterLength = r.Settings.DefaultChapterLength
		}
		if r.Settings.WritingStyle != "" {
			p.Settings.WritingStyle = r.Settings.WritingStyle
		}
		if r.Settings.POV != "" {
			p.Settings.POV = r.Settings.POV
		}
		if r.Settings.Temperature > 0 {
			p.Settings.Temperature = r.Settings.Temperature
		}
	}

	if r.WorldSettings != nil {
		if p.WorldSettings == nil {
			p.WorldSettings = &entity.WorldSettings{}
		}
		if r.WorldSettings.TimeSystem != "" {
			p.WorldSettings.TimeSystem = r.WorldSettings.TimeSystem
		}
		if r.WorldSettings.Calendar != "" {
			p.WorldSettings.Calendar = r.WorldSettings.Calendar
		}
		if r.WorldSettings.Locations != nil {
			p.WorldSettings.Locations = r.WorldSettings.Locations
		}
	}

	p.UpdatedAt = time.Now()
}
