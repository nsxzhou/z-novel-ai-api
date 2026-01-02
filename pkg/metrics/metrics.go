// Package metrics 提供 Prometheus 指标采集功能
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	namespace = "z_novel"
)

var (
	// HTTP 请求指标
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "http",
			Name:      "requests_total",
			Help:      "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: "http",
			Name:      "request_duration_seconds",
			Help:      "HTTP request duration in seconds",
			Buckets:   []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"method", "path"},
	)

	HTTPRequestSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: "http",
			Name:      "request_size_bytes",
			Help:      "HTTP request size in bytes",
			Buckets:   prometheus.ExponentialBuckets(100, 10, 6),
		},
		[]string{"method", "path"},
	)

	HTTPResponseSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: "http",
			Name:      "response_size_bytes",
			Help:      "HTTP response size in bytes",
			Buckets:   prometheus.ExponentialBuckets(100, 10, 6),
		},
		[]string{"method", "path"},
	)

	// 业务指标 - 故事生成
	StoryGenerationTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "story",
			Name:      "generation_total",
			Help:      "Total number of story generations",
		},
		[]string{"tenant_id", "status"},
	)

	StoryGenerationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: "story",
			Name:      "generation_duration_seconds",
			Help:      "Story generation duration in seconds",
			Buckets:   []float64{1, 5, 10, 30, 60, 120, 300},
		},
		[]string{"tenant_id"},
	)

	StoryWordCount = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: "story",
			Name:      "word_count",
			Help:      "Generated story word count",
			Buckets:   []float64{100, 500, 1000, 2000, 3000, 5000, 10000},
		},
		[]string{"tenant_id"},
	)

	// LLM 指标
	LLMTokensUsed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "llm",
			Name:      "tokens_used_total",
			Help:      "Total tokens used for LLM calls",
		},
		[]string{"provider", "model", "type"}, // type: prompt/completion
	)

	LLMCallDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: "llm",
			Name:      "call_duration_seconds",
			Help:      "LLM call duration in seconds",
			Buckets:   []float64{1, 5, 10, 30, 60, 120},
		},
		[]string{"provider", "model"},
	)

	LLMCallTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "llm",
			Name:      "call_total",
			Help:      "Total number of LLM calls",
		},
		[]string{"provider", "model", "status"},
	)

	// 向量检索指标
	MilvusSearchDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: "milvus",
			Name:      "search_duration_seconds",
			Help:      "Milvus search duration in seconds",
			Buckets:   []float64{.01, .05, .1, .25, .5, 1},
		},
		[]string{"collection"},
	)

	MilvusSearchTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "milvus",
			Name:      "search_total",
			Help:      "Total number of Milvus searches",
		},
		[]string{"collection", "status"},
	)

	// 队列指标
	RedisStreamLag = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "redis",
			Name:      "stream_lag",
			Help:      "Redis stream consumer lag",
		},
		[]string{"stream", "consumer_group"},
	)

	RedisStreamProcessed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "redis",
			Name:      "stream_processed_total",
			Help:      "Total number of Redis stream messages processed",
		},
		[]string{"stream", "status"},
	)

	// 校验指标
	ValidationTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "validation",
			Name:      "total",
			Help:      "Total number of validations",
		},
		[]string{"type", "status"},
	)

	// 活跃用户/写作者指标
	ActiveWriters = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "story",
			Name:      "active_writers",
			Help:      "Current number of active writers",
		},
	)
)
