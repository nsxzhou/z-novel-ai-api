// Package dto 提供 HTTP 层数据传输对象
package dto

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

// PageRequest 分页请求参数
type PageRequest struct {
	Page     int `form:"page" json:"page"`
	PageSize int `form:"page_size" json:"page_size"`
}

// SortRequest 排序请求参数
type SortRequest struct {
	Sort string `form:"sort" json:"sort"`
}

// PagedRequest 分页+排序请求
type PagedRequest struct {
	PageRequest
	SortRequest
}

// Normalize 规范化分页参数
func (r *PageRequest) Normalize() {
	if r.Page < 1 {
		r.Page = 1
	}
	if r.PageSize < 1 {
		r.PageSize = 20
	}
	if r.PageSize > 100 {
		r.PageSize = 100
	}
}

// Offset 计算偏移量
func (r *PageRequest) Offset() int {
	return (r.Page - 1) * r.PageSize
}

// Limit 返回限制数
func (r *PageRequest) Limit() int {
	return r.PageSize
}

// BindPage 从 Gin Context 绑定分页参数
func BindPage(c *gin.Context) PageRequest {
	page := parseIntWithDefault(c.Query("page"), 1)
	pageSize := parseIntWithDefault(c.Query("page_size"), 20)

	req := PageRequest{
		Page:     page,
		PageSize: pageSize,
	}
	req.Normalize()
	return req
}

// BindPagedRequest 从 Gin Context 绑定分页和排序参数
func BindPagedRequest(c *gin.Context) PagedRequest {
	page := BindPage(c)
	sort := c.Query("sort")

	return PagedRequest{
		PageRequest: page,
		SortRequest: SortRequest{Sort: sort},
	}
}

// parseIntWithDefault 解析整数，失败时返回默认值
func parseIntWithDefault(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return v
}

// IDRequest 资源 ID 请求
type IDRequest struct {
	ID string `uri:"id" binding:"required"`
}

// ProjectIDRequest 项目 ID 请求
type ProjectIDRequest struct {
	ProjectID string `uri:"pid" binding:"required"`
}

// ChapterIDRequest 章节 ID 请求
type ChapterIDRequest struct {
	ChapterID string `uri:"cid" binding:"required"`
}

// EntityIDRequest 实体 ID 请求
type EntityIDRequest struct {
	EntityID string `uri:"eid" binding:"required"`
}

// JobIDRequest 任务 ID 请求
type JobIDRequest struct {
	JobID string `uri:"jid" binding:"required"`
}

// VolumeIDRequest 卷 ID 请求
type VolumeIDRequest struct {
	VolumeID string `uri:"vid" binding:"required"`
}

// TenantIDRequest 租户 ID 请求
type TenantIDRequest struct {
	TenantID string `uri:"tid" binding:"required"`
}

// SessionIDRequest 会话 ID 请求
type SessionIDRequest struct {
	SessionID string `uri:"sid" binding:"required"`
}

// ArtifactIDRequest 构件 ID 请求
type ArtifactIDRequest struct {
	ArtifactID string `uri:"aid" binding:"required"`
}

// BindProjectID 从 URI 绑定项目 ID
func BindProjectID(c *gin.Context) string {
	return c.Param("pid")
}

// BindSessionID 从 URI 绑定会话 ID
func BindSessionID(c *gin.Context) string {
	return c.Param("sid")
}

// BindProjectCreationSessionID 从 URI 绑定对话创建项目会话 ID
func BindProjectCreationSessionID(c *gin.Context) string {
	return c.Param("sid")
}

// BindArtifactID 从 URI 绑定构件 ID
func BindArtifactID(c *gin.Context) string {
	return c.Param("aid")
}

// BindChapterID 从 URI 绑定章节 ID
func BindChapterID(c *gin.Context) string {
	return c.Param("cid")
}

// BindEntityID 从 URI 绑定实体 ID
func BindEntityID(c *gin.Context) string {
	return c.Param("eid")
}

// BindJobID 从 URI 绑定任务 ID
func BindJobID(c *gin.Context) string {
	return c.Param("jid")
}

// BindVolumeID 从 URI 绑定卷 ID
func BindVolumeID(c *gin.Context) string {
	return c.Param("vid")
}

// BindTenantID 从 URI 绑定租户 ID
func BindTenantID(c *gin.Context) string {
	return c.Param("tid")
}
