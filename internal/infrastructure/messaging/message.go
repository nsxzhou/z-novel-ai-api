// Package messaging 提供消息队列实现
package messaging

import (
	"encoding/json"
	"time"
)

// Message 消息结构
type Message struct {
	ID        string            `json:"id"`
	Type      string            `json:"type"`
	TenantID  string            `json:"tenant_id"`
	ProjectID string            `json:"project_id"`
	Payload   json.RawMessage   `json:"payload"`
	Metadata  map[string]string `json:"metadata"`
	CreatedAt time.Time         `json:"created_at"`
}

// NewMessage 创建新消息
func NewMessage(id, msgType, tenantID, projectID string, payload interface{}) (*Message, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return &Message{
		ID:        id,
		Type:      msgType,
		TenantID:  tenantID,
		ProjectID: projectID,
		Payload:   payloadBytes,
		Metadata:  make(map[string]string),
		CreatedAt: time.Now(),
	}, nil
}

// SetMetadata 设置元数据
func (m *Message) SetMetadata(key, value string) {
	if m.Metadata == nil {
		m.Metadata = make(map[string]string)
	}
	m.Metadata[key] = value
}

// GetMetadata 获取元数据
func (m *Message) GetMetadata(key string) string {
	if m.Metadata == nil {
		return ""
	}
	return m.Metadata[key]
}

// UnmarshalPayload 解析消息载荷
func (m *Message) UnmarshalPayload(v interface{}) error {
	return json.Unmarshal(m.Payload, v)
}

// Stream 流定义
type Stream string

const (
	StreamStoryGen     Stream = "stream:story:gen"
	StreamMemoryUpdate Stream = "stream:memory:update"
	StreamAuditLog     Stream = "stream:audit:log"
)

// DLQStream 获取对应的死信队列流名称
func (s Stream) DLQStream() string {
	return "dlq:" + string(s)
}

// ConsumerGroup 消费者组定义
type ConsumerGroup string

const (
	ConsumerGroupGenWorker ConsumerGroup = "cg-gen-worker"
	ConsumerGroupMemWriter ConsumerGroup = "cg-mem-writer"
	ConsumerGroupArchiver  ConsumerGroup = "cg-archiver"
)

// BackoffConfig 退避配置
type BackoffConfig struct {
	Initial    time.Duration
	Max        time.Duration
	Multiplier float64
}

// DefaultBackoffConfig 默认退避配置
func DefaultBackoffConfig() BackoffConfig {
	return BackoffConfig{
		Initial:    time.Second,
		Max:        time.Minute,
		Multiplier: 2,
	}
}

// CalculateBackoff 计算退避时间
func (c BackoffConfig) CalculateBackoff(retryCount int) time.Duration {
	backoff := c.Initial
	for i := 0; i < retryCount; i++ {
		backoff = time.Duration(float64(backoff) * c.Multiplier)
		if backoff > c.Max {
			backoff = c.Max
			break
		}
	}
	return backoff
}
