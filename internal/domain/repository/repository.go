// Package repository 定义数据访问层接口
package repository

import (
	"context"
)

// TxKey 事务上下文键类型
type TxKey struct{}

// Transactor 事务管理接口
type Transactor interface {
	// WithTransaction 在事务中执行操作
	WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

// Pagination 分页参数
type Pagination struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

// NewPagination 创建分页参数
func NewPagination(page, pageSize int) Pagination {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return Pagination{Page: page, PageSize: pageSize}
}

// Offset 计算偏移量
func (p Pagination) Offset() int {
	return (p.Page - 1) * p.PageSize
}

// Limit 获取限制数量
func (p Pagination) Limit() int {
	return p.PageSize
}

// PagedResult 分页结果
type PagedResult[T any] struct {
	Items      []T   `json:"items"`
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalPages int   `json:"total_pages"`
}

// NewPagedResult 创建分页结果
func NewPagedResult[T any](items []T, total int64, pagination Pagination) *PagedResult[T] {
	totalPages := int(total) / pagination.PageSize
	if int(total)%pagination.PageSize > 0 {
		totalPages++
	}
	return &PagedResult[T]{
		Items:      items,
		Total:      total,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalPages: totalPages,
	}
}

// SortOrder 排序方向
type SortOrder string

const (
	SortOrderAsc  SortOrder = "ASC"
	SortOrderDesc SortOrder = "DESC"
)

// Sort 排序参数
type Sort struct {
	Field string    `json:"field"`
	Order SortOrder `json:"order"`
}

// NewSort 创建排序参数
func NewSort(field string, order SortOrder) Sort {
	return Sort{Field: field, Order: order}
}
