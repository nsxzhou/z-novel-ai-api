// Package tracer 提供 OpenTelemetry 链路追踪功能
package tracer

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

var tracer trace.Tracer

// Config 追踪配置
type Config struct {
	ServiceName string
	Endpoint    string
	SampleRate  float64
	Enabled     bool
}

// Init 初始化 OpenTelemetry 追踪
func Init(ctx context.Context, cfg Config) (func(context.Context) error, error) {
	if !cfg.Enabled {
		// 追踪未启用，返回空的 shutdown 函数
		tracer = otel.Tracer(cfg.ServiceName)
		return func(context.Context) error { return nil }, nil
	}

	// 创建 OTLP gRPC exporter
	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(cfg.Endpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	// 创建资源
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(cfg.ServiceName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// 创建采样器
	var sampler sdktrace.Sampler
	if cfg.SampleRate >= 1.0 {
		sampler = sdktrace.AlwaysSample()
	} else if cfg.SampleRate <= 0 {
		sampler = sdktrace.NeverSample()
	} else {
		sampler = sdktrace.TraceIDRatioBased(cfg.SampleRate)
	}

	// 创建 TracerProvider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	// 设置全局 TracerProvider
	otel.SetTracerProvider(tp)

	// 设置传播器
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	tracer = tp.Tracer(cfg.ServiceName)

	return tp.Shutdown, nil
}

// Start 开始一个新的 Span
func Start(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	if tracer == nil {
		tracer = otel.Tracer("z-novel-ai-api")
	}
	return tracer.Start(ctx, name, opts...)
}

// SpanFromContext 从 context 获取当前 Span
func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// TraceID 从 context 获取 trace ID
func TraceID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		return span.SpanContext().TraceID().String()
	}
	return ""
}

// SpanID 从 context 获取 span ID
func SpanID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		return span.SpanContext().SpanID().String()
	}
	return ""
}
