// Package messaging 提供消息队列实现
package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"z-novel-ai-api/pkg/logger"
)

// MessageHandler 消息处理函数
type MessageHandler func(ctx context.Context, msg *Message) error

// Consumer 消息消费者
type Consumer struct {
	client       *redis.Client
	stream       Stream
	group        ConsumerGroup
	consumerName string
	blockTimeout time.Duration
	retryLimit   int
	backoff      BackoffConfig

	handlers map[string]MessageHandler
	mu       sync.RWMutex
	running  bool
	stopCh   chan struct{}
}

// ConsumerConfig 消费者配置
type ConsumerConfig struct {
	Stream       Stream
	Group        ConsumerGroup
	ConsumerName string
	BlockTimeout time.Duration
	RetryLimit   int
	Backoff      BackoffConfig
}

// NewConsumer 创建消息消费者
func NewConsumer(client *redis.Client, cfg ConsumerConfig) *Consumer {
	if cfg.BlockTimeout <= 0 {
		cfg.BlockTimeout = 5 * time.Second
	}
	if cfg.RetryLimit <= 0 {
		cfg.RetryLimit = 3
	}
	if cfg.Backoff.Initial <= 0 {
		cfg.Backoff = DefaultBackoffConfig()
	}

	return &Consumer{
		client:       client,
		stream:       cfg.Stream,
		group:        cfg.Group,
		consumerName: cfg.ConsumerName,
		blockTimeout: cfg.BlockTimeout,
		retryLimit:   cfg.RetryLimit,
		backoff:      cfg.Backoff,
		handlers:     make(map[string]MessageHandler),
		stopCh:       make(chan struct{}),
	}
}

// RegisterHandler 注册消息处理器
func (c *Consumer) RegisterHandler(msgType string, handler MessageHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.handlers[msgType] = handler
}

// Start 启动消费者
func (c *Consumer) Start(ctx context.Context) error {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return fmt.Errorf("consumer already running")
	}
	c.running = true
	c.mu.Unlock()

	// 确保消费者组存在
	err := c.client.XGroupCreateMkStream(ctx, string(c.stream), string(c.group), "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return fmt.Errorf("failed to create consumer group: %w", err)
	}

	go c.run(ctx)
	return nil
}

// Stop 停止消费者
func (c *Consumer) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.running {
		close(c.stopCh)
		c.running = false
	}
}

// run 消费循环
func (c *Consumer) run(ctx context.Context) {
	log := logger.FromContext(ctx)
	log.Info("consumer started",
		"stream", c.stream,
		"group", c.group,
		"consumer", c.consumerName,
	)

	for {
		select {
		case <-ctx.Done():
			log.Info("consumer stopped due to context cancellation")
			return
		case <-c.stopCh:
			log.Info("consumer stopped")
			return
		default:
		}

		// 读取消息
		streams, err := c.client.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    string(c.group),
			Consumer: c.consumerName,
			Streams:  []string{string(c.stream), ">"},
			Count:    10,
			Block:    c.blockTimeout,
		}).Result()

		if err != nil {
			if err == redis.Nil {
				continue
			}
			log.Error("failed to read from stream", "error", err)
			time.Sleep(time.Second)
			continue
		}

		for _, stream := range streams {
			for _, xmsg := range stream.Messages {
				c.processMessage(ctx, xmsg)
			}
		}
	}
}

// processMessage 处理单条消息
func (c *Consumer) processMessage(ctx context.Context, xmsg redis.XMessage) {
	ctx, span := tracer.Start(ctx, "consumer.processMessage",
		trace.WithAttributes(
			attribute.String("stream", string(c.stream)),
			attribute.String("stream.message_id", xmsg.ID),
		))
	defer span.End()

	log := logger.FromContext(ctx)

	// 解析消息
	var msg Message
	dataStr, ok := xmsg.Values["data"].(string)
	if !ok {
		log.Error("invalid message format", "message_id", xmsg.ID)
		c.ack(ctx, xmsg.ID)
		return
	}

	if err := json.Unmarshal([]byte(dataStr), &msg); err != nil {
		log.Error("failed to unmarshal message", "error", err, "message_id", xmsg.ID)
		c.ack(ctx, xmsg.ID)
		return
	}

	span.SetAttributes(
		attribute.String("message.id", msg.ID),
		attribute.String("message.type", msg.Type),
	)

	// 查找处理器
	c.mu.RLock()
	handler, exists := c.handlers[msg.Type]
	c.mu.RUnlock()

	if !exists {
		log.Warn("no handler for message type", "type", msg.Type)
		c.ack(ctx, xmsg.ID)
		return
	}

	// 执行处理器
	if err := handler(ctx, &msg); err != nil {
		span.RecordError(err)
		log.Error("handler failed", "error", err, "message_id", msg.ID)
		c.handleFailure(ctx, xmsg, &msg, err)
		return
	}

	c.ack(ctx, xmsg.ID)
}

// ack 确认消息
func (c *Consumer) ack(ctx context.Context, id string) {
	if err := c.client.XAck(ctx, string(c.stream), string(c.group), id).Err(); err != nil {
		logger.FromContext(ctx).Error("failed to ack message", "error", err, "message_id", id)
	}
}

// handleFailure 处理失败
func (c *Consumer) handleFailure(ctx context.Context, xmsg redis.XMessage, msg *Message, err error) {
	log := logger.FromContext(ctx)

	// 获取重试次数
	retryCount := c.getRetryCount(ctx, xmsg.ID)

	if retryCount >= c.retryLimit {
		// 移入死信队列
		log.Warn("message moved to DLQ after max retries",
			"message_id", msg.ID,
			"retry_count", retryCount,
		)
		c.moveToDLQ(ctx, msg, err)
		c.ack(ctx, xmsg.ID)
		return
	}

	// 计算退避时间
	backoff := c.backoff.CalculateBackoff(retryCount)
	log.Info("scheduling message retry",
		"message_id", msg.ID,
		"retry_count", retryCount+1,
		"backoff", backoff,
	)

	// 延迟重试
	time.AfterFunc(backoff, func() {
		c.client.XClaim(context.Background(), &redis.XClaimArgs{
			Stream:   string(c.stream),
			Group:    string(c.group),
			Consumer: c.consumerName,
			MinIdle:  0,
			Messages: []string{xmsg.ID},
		})
	})
}

// getRetryCount 获取重试次数
func (c *Consumer) getRetryCount(ctx context.Context, messageID string) int {
	// 通过 XPENDING 获取消息的投递次数
	pending, err := c.client.XPendingExt(ctx, &redis.XPendingExtArgs{
		Stream: string(c.stream),
		Group:  string(c.group),
		Start:  messageID,
		End:    messageID,
		Count:  1,
	}).Result()

	if err != nil || len(pending) == 0 {
		return 0
	}

	return int(pending[0].RetryCount)
}

// moveToDLQ 移入死信队列
func (c *Consumer) moveToDLQ(ctx context.Context, msg *Message, err error) {
	dlqStream := c.stream.DLQStream()

	dlqMsg := map[string]interface{}{
		"original_stream": string(c.stream),
		"data":            msg,
		"error":           err.Error(),
		"failed_at":       time.Now().Unix(),
	}

	data, _ := json.Marshal(dlqMsg)
	c.client.XAdd(ctx, &redis.XAddArgs{
		Stream: dlqStream,
		Values: map[string]interface{}{"data": string(data)},
	})
}

// MonitorDLQ 监控死信队列
func (c *Consumer) MonitorDLQ(ctx context.Context, alertThreshold int64) {
	log := logger.FromContext(ctx)
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopCh:
			return
		case <-ticker.C:
			dlqStream := c.stream.DLQStream()
			info, err := c.client.XInfoStream(ctx, dlqStream).Result()
			if err != nil {
				continue
			}

			if info.Length > alertThreshold {
				log.Warn("DLQ has pending messages",
					"stream", dlqStream,
					"count", info.Length,
				)
			}
		}
	}
}
