// Package errors 提供统一的错误定义
package errors

import (
	"fmt"
	"net/http"
)

// ErrorCode 错误码类型
type ErrorCode string

// 预定义错误码
const (
	// 通用错误 (1xxx)
	CodeSuccess            ErrorCode = "0"
	CodeUnknown            ErrorCode = "1000"
	CodeInvalidParam       ErrorCode = "1001"
	CodeUnauthorized       ErrorCode = "1002"
	CodeForbidden          ErrorCode = "1003"
	CodeNotFound           ErrorCode = "1004"
	CodeConflict           ErrorCode = "1005"
	CodeTooManyRequests    ErrorCode = "1006"
	CodeInternalError      ErrorCode = "1007"
	CodeServiceUnavailable ErrorCode = "1008"

	// 认证授权错误 (2xxx)
	CodeTokenExpired     ErrorCode = "2001"
	CodeTokenInvalid     ErrorCode = "2002"
	CodeTokenMissing     ErrorCode = "2003"
	CodePermissionDenied ErrorCode = "2004"

	// 资源错误 (3xxx)
	CodeProjectNotFound ErrorCode = "3001"
	CodeChapterNotFound ErrorCode = "3002"
	CodeEntityNotFound  ErrorCode = "3003"
	CodeFileNotFound    ErrorCode = "3004"

	// 业务错误 (4xxx)
	CodeGenerationFailed  ErrorCode = "4001"
	CodeValidationFailed  ErrorCode = "4002"
	CodeRetrievalFailed   ErrorCode = "4003"
	CodeMemoryWriteFailed ErrorCode = "4004"
	CodeLLMCallFailed     ErrorCode = "4005"
	CodeEmbeddingFailed   ErrorCode = "4006"

	// 外部服务错误 (5xxx)
	CodeDatabaseError    ErrorCode = "5001"
	CodeCacheError       ErrorCode = "5002"
	CodeVectorDBError    ErrorCode = "5003"
	CodeStorageError     ErrorCode = "5004"
	CodeLLMProviderError ErrorCode = "5005"
)

// AppError 应用错误
type AppError struct {
	Code       ErrorCode `json:"code"`
	Message    string    `json:"message"`
	Detail     string    `json:"detail,omitempty"`
	HTTPStatus int       `json:"-"`
	Err        error     `json:"-"`
}

// Error 实现 error 接口
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap 返回底层错误
func (e *AppError) Unwrap() error {
	return e.Err
}

// WithDetail 添加详细信息
func (e *AppError) WithDetail(detail string) *AppError {
	e.Detail = detail
	return e
}

// WithError 添加底层错误
func (e *AppError) WithError(err error) *AppError {
	e.Err = err
	return e
}

// New 创建新的应用错误
func New(code ErrorCode, message string) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: codeToHTTPStatus(code),
	}
}

// Wrap 包装错误
func Wrap(err error, code ErrorCode, message string) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: codeToHTTPStatus(code),
		Err:        err,
	}
}

// codeToHTTPStatus 错误码转 HTTP 状态码
func codeToHTTPStatus(code ErrorCode) int {
	switch code {
	case CodeSuccess:
		return http.StatusOK
	case CodeInvalidParam:
		return http.StatusBadRequest
	case CodeUnauthorized, CodeTokenExpired, CodeTokenInvalid, CodeTokenMissing:
		return http.StatusUnauthorized
	case CodeForbidden, CodePermissionDenied:
		return http.StatusForbidden
	case CodeNotFound, CodeProjectNotFound, CodeChapterNotFound, CodeEntityNotFound, CodeFileNotFound:
		return http.StatusNotFound
	case CodeConflict:
		return http.StatusConflict
	case CodeTooManyRequests:
		return http.StatusTooManyRequests
	case CodeServiceUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

// 预定义错误
var (
	ErrInvalidParam       = New(CodeInvalidParam, "invalid parameter")
	ErrUnauthorized       = New(CodeUnauthorized, "unauthorized")
	ErrForbidden          = New(CodeForbidden, "forbidden")
	ErrNotFound           = New(CodeNotFound, "resource not found")
	ErrConflict           = New(CodeConflict, "resource conflict")
	ErrTooManyRequests    = New(CodeTooManyRequests, "too many requests")
	ErrInternalError      = New(CodeInternalError, "internal server error")
	ErrServiceUnavailable = New(CodeServiceUnavailable, "service unavailable")

	ErrTokenExpired = New(CodeTokenExpired, "token expired")
	ErrTokenInvalid = New(CodeTokenInvalid, "token invalid")
	ErrTokenMissing = New(CodeTokenMissing, "token missing")

	ErrProjectNotFound = New(CodeProjectNotFound, "project not found")
	ErrChapterNotFound = New(CodeChapterNotFound, "chapter not found")
	ErrEntityNotFound  = New(CodeEntityNotFound, "entity not found")

	ErrGenerationFailed = New(CodeGenerationFailed, "story generation failed")
	ErrValidationFailed = New(CodeValidationFailed, "validation failed")
	ErrLLMCallFailed    = New(CodeLLMCallFailed, "LLM call failed")
)

// IsAppError 检查是否为 AppError
func IsAppError(err error) bool {
	_, ok := err.(*AppError)
	return ok
}

// AsAppError 将错误转换为 AppError
func AsAppError(err error) *AppError {
	if appErr, ok := err.(*AppError); ok {
		return appErr
	}
	return Wrap(err, CodeUnknown, "unknown error")
}
