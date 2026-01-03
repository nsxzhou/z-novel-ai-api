// Package dto 提供 HTTP 层数据传输对象
package dto

import (
	"github.com/gin-gonic/gin"
)

// Response 统一响应结构
type Response[T any] struct {
	Code    int       `json:"code"`
	Message string    `json:"message"`
	Data    T         `json:"data,omitempty"`
	Meta    *PageMeta `json:"meta,omitempty"`
	TraceID string    `json:"trace_id,omitempty"`
}

// PageMeta 分页元数据
type PageMeta struct {
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// ErrorDetail 错误详情
type ErrorDetail struct {
	ErrorCode   string   `json:"error_code,omitempty"`
	Details     string   `json:"details,omitempty"`
	Suggestions []string `json:"suggestions,omitempty"`
}

// ErrorResponse 错误响应结构
type ErrorResponse struct {
	Code    int          `json:"code"`
	Message string       `json:"message"`
	Error   *ErrorDetail `json:"error,omitempty"`
	TraceID string       `json:"trace_id,omitempty"`
}

// Success 返回成功响应
func Success[T any](c *gin.Context, data T) {
	c.JSON(200, Response[T]{
		Code:    200,
		Message: "success",
		Data:    data,
		TraceID: c.GetString("trace_id"),
	})
}

// SuccessWithPage 返回带分页的成功响应
func SuccessWithPage[T any](c *gin.Context, data T, meta *PageMeta) {
	c.JSON(200, Response[T]{
		Code:    200,
		Message: "success",
		Data:    data,
		Meta:    meta,
		TraceID: c.GetString("trace_id"),
	})
}

// Created 返回创建成功响应 (201)
func Created[T any](c *gin.Context, data T) {
	c.JSON(201, Response[T]{
		Code:    201,
		Message: "created",
		Data:    data,
		TraceID: c.GetString("trace_id"),
	})
}

// Accepted 返回接受处理响应 (202)
func Accepted[T any](c *gin.Context, data T) {
	c.JSON(202, Response[T]{
		Code:    202,
		Message: "accepted",
		Data:    data,
		TraceID: c.GetString("trace_id"),
	})
}

// NoContent 返回无内容响应 (204)
func NoContent(c *gin.Context) {
	c.Status(204)
}

// Error 返回错误响应
func Error(c *gin.Context, httpCode int, message string) {
	c.JSON(httpCode, ErrorResponse{
		Code:    httpCode,
		Message: message,
		TraceID: c.GetString("trace_id"),
	})
}

// ErrorWithDetail 返回带详情的错误响应
func ErrorWithDetail(c *gin.Context, httpCode int, message string, detail *ErrorDetail) {
	c.JSON(httpCode, ErrorResponse{
		Code:    httpCode,
		Message: message,
		Error:   detail,
		TraceID: c.GetString("trace_id"),
	})
}

// BadRequest 返回 400 错误
func BadRequest(c *gin.Context, message string) {
	Error(c, 400, message)
}

// Unauthorized 返回 401 错误
func Unauthorized(c *gin.Context, message string) {
	Error(c, 401, message)
}

// Forbidden 返回 403 错误
func Forbidden(c *gin.Context, message string) {
	Error(c, 403, message)
}

// NotFound 返回 404 错误
func NotFound(c *gin.Context, message string) {
	Error(c, 404, message)
}

// Conflict 返回 409 错误
func Conflict(c *gin.Context, message string) {
	Error(c, 409, message)
}

// UnprocessableEntity 返回 422 错误
func UnprocessableEntity(c *gin.Context, message string, detail *ErrorDetail) {
	ErrorWithDetail(c, 422, message, detail)
}

// InternalError 返回 500 错误
func InternalError(c *gin.Context, message string) {
	Error(c, 500, message)
}

// ServiceUnavailable 返回 503 错误
func ServiceUnavailable(c *gin.Context, message string) {
	Error(c, 503, message)
}

// NewPageMeta 创建分页元数据
func NewPageMeta(page, pageSize, total int) *PageMeta {
	totalPages := total / pageSize
	if total%pageSize > 0 {
		totalPages++
	}
	return &PageMeta{
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
	}
}
