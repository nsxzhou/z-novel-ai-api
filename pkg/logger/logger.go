// Package logger 提供结构化日志功能
package logger

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

// ContextKey 用于从 context 中提取值的键类型
type ContextKey string

// 预定义的 context 键
const (
	TraceIDKey   ContextKey = "trace_id"
	SpanIDKey    ContextKey = "span_id"
	TenantIDKey  ContextKey = "tenant_id"
	ProjectIDKey ContextKey = "project_id"
	RequestIDKey ContextKey = "request_id"
	UserIDKey    ContextKey = "user_id"
)

var defaultLogger *slog.Logger

// Init 初始化日志器
func Init(level string, format string) {
	var handler slog.Handler

	opts := &slog.HandlerOptions{
		Level:     parseLevel(level),
		AddSource: true,
	}

	if strings.ToLower(format) == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	defaultLogger = slog.New(handler)
	slog.SetDefault(defaultLogger)
}

// parseLevel 解析日志级别字符串
func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Default 返回默认日志器
func Default() *slog.Logger {
	if defaultLogger == nil {
		Init("info", "json")
	}
	return defaultLogger
}

// FromContext 从 Context 提取追踪信息创建带上下文的 Logger
func FromContext(ctx context.Context) *slog.Logger {
	logger := Default()

	if traceID := ctx.Value(TraceIDKey); traceID != nil {
		logger = logger.With("trace_id", traceID)
	}
	if spanID := ctx.Value(SpanIDKey); spanID != nil {
		logger = logger.With("span_id", spanID)
	}
	if tenantID := ctx.Value(TenantIDKey); tenantID != nil {
		logger = logger.With("tenant_id", tenantID)
	}
	if projectID := ctx.Value(ProjectIDKey); projectID != nil {
		logger = logger.With("project_id", projectID)
	}
	if requestID := ctx.Value(RequestIDKey); requestID != nil {
		logger = logger.With("request_id", requestID)
	}
	if userID := ctx.Value(UserIDKey); userID != nil {
		logger = logger.With("user_id", userID)
	}

	return logger
}

// WithContext 将日志上下文信息注入到 context
func WithContext(ctx context.Context, key ContextKey, value any) context.Context {
	return context.WithValue(ctx, key, value)
}

// Info 记录 INFO 级别日志
func Info(ctx context.Context, msg string, args ...any) {
	FromContext(ctx).Info(msg, args...)
}

// Debug 记录 DEBUG 级别日志
func Debug(ctx context.Context, msg string, args ...any) {
	FromContext(ctx).Debug(msg, args...)
}

// Warn 记录 WARN 级别日志
func Warn(ctx context.Context, msg string, args ...any) {
	FromContext(ctx).Warn(msg, args...)
}

// Error 记录 ERROR 级别日志
func Error(ctx context.Context, msg string, err error, args ...any) {
	if err != nil {
		args = append(args, "error", err.Error())
	}
	FromContext(ctx).Error(msg, args...)
}

// Fatal 记录 Fatal 级别日志并退出
func Fatal(ctx context.Context, msg string, err error, args ...any) {
	Error(ctx, msg, err, args...)
	os.Exit(1)
}
