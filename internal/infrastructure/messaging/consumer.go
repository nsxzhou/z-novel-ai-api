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
	client        *redis.Client
	stream        Stream
	group         ConsumerGroup
	consumerName  string
	blockTimeout  time.Duration
	claimInterval time.Duration
	reclaimIdle   time.Duration
	retryLimit    int
	backoff       BackoffConfig

	handlers map[string]MessageHandler
	mu       sync.RWMutex
	running  bool
	stopCh   chan struct{}
}

// ConsumerConfig 消费者配置
type ConsumerConfig struct {
	Stream        Stream
	Group         ConsumerGroup
	ConsumerName  string
	BlockTimeout  time.Duration
	ClaimInterval time.Duration
	RetryLimit    int
	Backoff       BackoffConfig
}

// NewConsumer 创建消息消费者
func NewConsumer(client *redis.Client, cfg ConsumerConfig) *Consumer {
	if cfg.BlockTimeout <= 0 {
		cfg.BlockTimeout = 5 * time.Second
	}
	if cfg.ClaimInterval <= 0 {
		cfg.ClaimInterval = 30 * time.Second
	}
	if cfg.RetryLimit <= 0 {
		cfg.RetryLimit = 3
	}
	if cfg.Backoff.Initial <= 0 {
		cfg.Backoff = DefaultBackoffConfig()
	}

	return &Consumer{
		client:        client,
		stream:        cfg.Stream,
		group:         cfg.Group,
		consumerName:  cfg.ConsumerName,
		blockTimeout:  cfg.BlockTimeout,
		claimInterval: cfg.ClaimInterval,
		reclaimIdle:   maxDuration(5*time.Minute, cfg.Backoff.Max*2),
		retryLimit:    cfg.RetryLimit,
		backoff:       cfg.Backoff,
		handlers:      make(map[string]MessageHandler),
		stopCh:        make(chan struct{}),
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

	lastClaim := time.Now().Add(-c.claimInterval)

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

		c.processDuePending(ctx)
		if time.Since(lastClaim) >= c.claimInterval {
			c.reclaimStale(ctx)
			lastClaim = time.Now()
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

	// 解析消息
	var msg Message
	dataStr, ok := xmsg.Values["data"].(string)
	if !ok {
		logger.FromContext(ctx).Error("invalid message format", "message_id", xmsg.ID)
		c.ack(ctx, xmsg.ID)
		return
	}

	if err := json.Unmarshal([]byte(dataStr), &msg); err != nil {
		logger.FromContext(ctx).Error("failed to unmarshal message", "error", err, "message_id", xmsg.ID)
		c.ack(ctx, xmsg.ID)
		return
	}

	// 注入日志上下文（便于观测：tenant_id/project_id/request_id）
	if msg.TenantID != "" {
		ctx = logger.WithContext(ctx, logger.TenantIDKey, msg.TenantID)
	}
	if msg.ProjectID != "" {
		ctx = logger.WithContext(ctx, logger.ProjectIDKey, msg.ProjectID)
	}
	if reqID := msg.GetMetadata("request_id"); reqID != "" {
		ctx = logger.WithContext(ctx, logger.RequestIDKey, reqID)
	}
	if traceID := msg.GetMetadata("trace_id"); traceID != "" {
		ctx = logger.WithContext(ctx, logger.TraceIDKey, traceID)
	}

	log := logger.FromContext(ctx)

	span.SetAttributes(
		attribute.String("message.id", msg.ID),
		attribute.String("message.type", msg.Type),
		attribute.String("tenant_id", msg.TenantID),
		attribute.String("project_id", msg.ProjectID),
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
	log.Info("message left pending for retry",
		"message_id", msg.ID,
		"retry_count", retryCount,
	)
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

func (c *Consumer) processDuePending(ctx context.Context) {
	pending, err := c.client.XPendingExt(ctx, &redis.XPendingExtArgs{
		Stream:   string(c.stream),
		Group:    string(c.group),
		Start:    "-",
		End:      "+",
		Count:    20,
		Consumer: c.consumerName,
	}).Result()
	if err != nil {
		if err == redis.Nil {
			return
		}
		logger.FromContext(ctx).Error("failed to query pending messages", "error", err)
		return
	}

	for i := range pending {
		p := pending[i]
		retryCount := int(p.RetryCount)
		if retryCount >= c.retryLimit {
			claimed, claimErr := c.client.XClaim(ctx, &redis.XClaimArgs{
				Stream:   string(c.stream),
				Group:    string(c.group),
				Consumer: c.consumerName,
				MinIdle:  0,
				Messages: []string{p.ID},
			}).Result()
			if claimErr != nil {
				logger.FromContext(ctx).Error("failed to claim pending message for DLQ", "error", claimErr, "message_id", p.ID)
				continue
			}

			for _, xmsg := range claimed {
				raw, ok := xmsg.Values["data"].(string)
				if !ok {
					c.ack(ctx, xmsg.ID)
					continue
				}

				var msg Message
				if unmarshalErr := json.Unmarshal([]byte(raw), &msg); unmarshalErr != nil {
					c.ack(ctx, xmsg.ID)
					continue
				}

				c.moveToDLQ(ctx, &msg, fmt.Errorf("message exceeded max retries"))
				c.ack(ctx, xmsg.ID)
			}
			continue
		}

		backoff := c.backoff.CalculateBackoff(retryCount)
		if p.Idle < backoff {
			continue
		}

		claimed, claimErr := c.client.XClaim(ctx, &redis.XClaimArgs{
			Stream:   string(c.stream),
			Group:    string(c.group),
			Consumer: c.consumerName,
			MinIdle:  backoff,
			Messages: []string{p.ID},
		}).Result()
		if claimErr != nil {
			logger.FromContext(ctx).Error("failed to claim pending message", "error", claimErr, "message_id", p.ID)
			continue
		}

		for _, xmsg := range claimed {
			c.processMessage(ctx, xmsg)
		}
	}
}

func (c *Consumer) reclaimStale(ctx context.Context) {
	if c.reclaimIdle <= 0 {
		return
	}

	pending, err := c.client.XPendingExt(ctx, &redis.XPendingExtArgs{
		Stream: string(c.stream),
		Group:  string(c.group),
		Start:  "-",
		End:    "+",
		Count:  20,
	}).Result()
	if err != nil {
		if err == redis.Nil {
			return
		}
		logger.FromContext(ctx).Error("failed to query pending messages for reclaim", "error", err)
		return
	}

	for i := range pending {
		p := pending[i]
		if p.Consumer == c.consumerName {
			continue
		}
		if p.Idle < c.reclaimIdle {
			continue
		}
		if int(p.RetryCount) >= c.retryLimit {
			claimed, claimErr := c.client.XClaim(ctx, &redis.XClaimArgs{
				Stream:   string(c.stream),
				Group:    string(c.group),
				Consumer: c.consumerName,
				MinIdle:  c.reclaimIdle,
				Messages: []string{p.ID},
			}).Result()
			if claimErr != nil {
				logger.FromContext(ctx).Error("failed to claim stale message for DLQ", "error", claimErr, "message_id", p.ID)
				continue
			}
			for _, xmsg := range claimed {
				raw, ok := xmsg.Values["data"].(string)
				if !ok {
					c.ack(ctx, xmsg.ID)
					continue
				}

				var msg Message
				if unmarshalErr := json.Unmarshal([]byte(raw), &msg); unmarshalErr != nil {
					c.ack(ctx, xmsg.ID)
					continue
				}
				c.moveToDLQ(ctx, &msg, fmt.Errorf("message exceeded max retries"))
				c.ack(ctx, xmsg.ID)
			}
			continue
		}

		claimed, claimErr := c.client.XClaim(ctx, &redis.XClaimArgs{
			Stream:   string(c.stream),
			Group:    string(c.group),
			Consumer: c.consumerName,
			MinIdle:  c.reclaimIdle,
			Messages: []string{p.ID},
		}).Result()
		if claimErr != nil {
			logger.FromContext(ctx).Error("failed to reclaim pending message", "error", claimErr, "message_id", p.ID)
			continue
		}

		for _, xmsg := range claimed {
			c.processMessage(ctx, xmsg)
		}
	}
}

func maxDuration(a, b time.Duration) time.Duration {
	if a > b {
		return a
	}
	return b
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
