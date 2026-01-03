// Package messaging 提供消息队列实现
package messaging

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("messaging")

// Producer 消息生产者
type Producer struct {
	client *redis.Client
	maxLen int64
}

// NewProducer 创建消息生产者
func NewProducer(client *redis.Client, maxLen int64) *Producer {
	if maxLen <= 0 {
		maxLen = 100000
	}
	return &Producer{
		client: client,
		maxLen: maxLen,
	}
}

// Publish 发布消息到指定流
func (p *Producer) Publish(ctx context.Context, stream Stream, msg *Message) (string, error) {
	ctx, span := tracer.Start(ctx, "producer.Publish",
		trace.WithAttributes(
			attribute.String("stream", string(stream)),
			attribute.String("message.id", msg.ID),
			attribute.String("message.type", msg.Type),
		))
	defer span.End()

	data, err := json.Marshal(msg)
	if err != nil {
		span.RecordError(err)
		return "", fmt.Errorf("failed to marshal message: %w", err)
	}

	result, err := p.client.XAdd(ctx, &redis.XAddArgs{
		Stream: string(stream),
		MaxLen: p.maxLen,
		Approx: true,
		Values: map[string]interface{}{
			"data": string(data),
		},
	}).Result()

	if err != nil {
		span.RecordError(err)
		return "", fmt.Errorf("failed to publish message: %w", err)
	}

	span.SetAttributes(attribute.String("stream.message_id", result))
	return result, nil
}

// PublishGenJob 发布生成任务
func (p *Producer) PublishGenJob(ctx context.Context, job *GenerationJobMessage) (string, error) {
	msg, err := NewMessage(job.JobID, "chapter_gen", job.TenantID, job.ProjectID, job)
	if err != nil {
		return "", err
	}

	msg.SetMetadata("priority", fmt.Sprintf("%d", job.Priority))
	if job.IdempotencyKey != "" {
		msg.SetMetadata("idempotency_key", job.IdempotencyKey)
	}

	return p.Publish(ctx, StreamStoryGen, msg)
}

// PublishMemoryUpdate 发布记忆更新任务
func (p *Producer) PublishMemoryUpdate(ctx context.Context, update *MemoryUpdateMessage) (string, error) {
	msg, err := NewMessage(update.ChapterID, "memory_update", update.TenantID, update.ProjectID, update)
	if err != nil {
		return "", err
	}

	msg.SetMetadata("chapter_version", fmt.Sprintf("%d", update.ChapterVersion))
	return p.Publish(ctx, StreamMemoryUpdate, msg)
}

// PublishAuditLog 发布审计日志
func (p *Producer) PublishAuditLog(ctx context.Context, log *AuditLogMessage) (string, error) {
	msg, err := NewMessage(log.RequestID, "audit", log.TenantID, "", log)
	if err != nil {
		return "", err
	}

	return p.Publish(ctx, StreamAuditLog, msg)
}

// GenerationJobMessage 生成任务消息
type GenerationJobMessage struct {
	JobID          string                 `json:"job_id"`
	TenantID       string                 `json:"tenant_id"`
	ProjectID      string                 `json:"project_id"`
	ChapterID      string                 `json:"chapter_id,omitempty"`
	JobType        string                 `json:"job_type"`
	Priority       int                    `json:"priority"`
	IdempotencyKey string                 `json:"idempotency_key,omitempty"`
	Params         map[string]interface{} `json:"params"`
}

// MemoryUpdateMessage 记忆更新消息
type MemoryUpdateMessage struct {
	TenantID       string `json:"tenant_id"`
	ProjectID      string `json:"project_id"`
	ChapterID      string `json:"chapter_id"`
	ChapterVersion int    `json:"chapter_version"`
	Content        string `json:"content"`
	Summary        string `json:"summary,omitempty"`
}

// AuditLogMessage 审计日志消息
type AuditLogMessage struct {
	TenantID     string                 `json:"tenant_id"`
	UserID       string                 `json:"user_id,omitempty"`
	Action       string                 `json:"action"`
	ResourceType string                 `json:"resource_type"`
	ResourceID   string                 `json:"resource_id,omitempty"`
	RequestID    string                 `json:"request_id"`
	TraceID      string                 `json:"trace_id,omitempty"`
	IPAddress    string                 `json:"ip_address,omitempty"`
	UserAgent    string                 `json:"user_agent,omitempty"`
	Changes      map[string]interface{} `json:"changes,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}
