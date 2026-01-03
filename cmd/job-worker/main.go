// Package main 异步任务执行器入口（job-worker）
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	commonv1 "z-novel-ai-api/api/proto/gen/go/common"
	storyv1 "z-novel-ai-api/api/proto/gen/go/story"
	"z-novel-ai-api/internal/config"
	"z-novel-ai-api/internal/domain/entity"
	"z-novel-ai-api/internal/infrastructure/messaging"
	"z-novel-ai-api/internal/infrastructure/persistence/postgres"
	"z-novel-ai-api/internal/infrastructure/persistence/redis"
	grpcclient "z-novel-ai-api/internal/interfaces/grpc/client"
	"z-novel-ai-api/pkg/logger"
	"z-novel-ai-api/pkg/tracer"

	"github.com/google/uuid"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	logger.Init(cfg.Observability.Logging.Level, cfg.Observability.Logging.Format)
	ctx := context.Background()

	shutdown, err := tracer.Init(ctx, tracer.Config{
		ServiceName: "job-worker",
		Endpoint:    cfg.Observability.Tracing.Endpoint,
		SampleRate:  cfg.Observability.Tracing.SampleRate,
		Enabled:     cfg.Observability.Tracing.Enabled,
	})
	if err != nil {
		logger.Fatal(ctx, "failed to init tracer", err)
	}
	defer func() { _ = shutdown(ctx) }()

	pgClient, err := postgres.NewClient(&cfg.Database.Postgres)
	if err != nil {
		logger.Fatal(ctx, "failed to init postgres", err)
	}
	defer func() { _ = pgClient.Close() }()

	redisClient, err := redis.NewClient(&cfg.Cache.Redis)
	if err != nil {
		logger.Fatal(ctx, "failed to init redis", err)
	}
	defer func() { _ = redisClient.Close() }()

	txMgr := postgres.NewTxManager(pgClient)
	tenantCtx := postgres.NewTenantContext(pgClient)
	jobRepo := postgres.NewJobRepository(pgClient)
	chapterRepo := postgres.NewChapterRepository(pgClient)

	storyConn, err := grpcclient.Dial(ctx, cfg.Clients.GRPC.StoryGenServiceAddr, cfg.Clients.GRPC.DialTimeout)
	if err != nil {
		logger.Fatal(ctx, "failed to dial story-gen-svc", err)
	}
	defer func() { _ = storyConn.Close() }()
	storyClient := storyv1.NewStoryGenServiceClient(storyConn)

	consumer := messaging.NewConsumer(redisClient.Redis(), messaging.ConsumerConfig{
		Stream:       messaging.StreamStoryGen,
		Group:        messaging.ConsumerGroupGenWorker,
		ConsumerName: hostnameConsumerName(),
		BlockTimeout: cfg.Messaging.RedisStream.BlockTimeout,
		RetryLimit:   cfg.Messaging.RedisStream.RetryLimit,
		Backoff: messaging.BackoffConfig{
			Initial:    cfg.Messaging.RedisStream.RetryBackoff.Initial,
			Max:        cfg.Messaging.RedisStream.RetryBackoff.Max,
			Multiplier: cfg.Messaging.RedisStream.RetryBackoff.Multiplier,
		},
	})

	consumer.RegisterHandler("chapter_gen", func(_ context.Context, msg *messaging.Message) error {
		var payload messaging.GenerationJobMessage
		if err := msg.UnmarshalPayload(&payload); err != nil {
			return err
		}

		return txMgr.WithTransaction(ctx, func(txCtx context.Context) error {
			if err := tenantCtx.SetTenant(txCtx, payload.TenantID); err != nil {
				return err
			}

			job, err := jobRepo.GetByID(txCtx, payload.JobID)
			if err != nil {
				return err
			}
			if job == nil {
				return fmt.Errorf("job not found: %s", payload.JobID)
			}
			if job.Status == entity.JobStatusCancelled {
				return nil
			}

			if err := jobRepo.MarkRunning(txCtx, payload.JobID); err != nil {
				return err
			}
			_ = jobRepo.UpdateProgress(txCtx, payload.JobID, 5)

			outline, _ := payload.Params["outline"].(string)
			targetWordCount := int32(0)
			if v, ok := payload.Params["target_word_count"].(float64); ok {
				targetWordCount = int32(v)
			}

			traceID := uuid.NewString()
			genResp, err := storyClient.GenerateChapter(txCtx, &storyv1.GenerateChapterRequest{
				Context: &commonv1.TenantContext{
					TenantId: payload.TenantID,
					TraceId:  traceID,
				},
				ProjectId:       payload.ProjectID,
				ChapterId:       payload.ChapterID,
				Outline:         outline,
				TargetWordCount: targetWordCount,
			})
			if err != nil {
				_ = jobRepo.UpdateProgress(txCtx, payload.JobID, 100)
				_ = jobRepo.SetResult(txCtx, payload.JobID, nil, err.Error())
				return err
			}
			_ = jobRepo.UpdateProgress(txCtx, payload.JobID, 80)

			chapter, err := chapterRepo.GetByID(txCtx, payload.ChapterID)
			if err != nil {
				return err
			}
			if chapter == nil {
				return fmt.Errorf("chapter not found: %s", payload.ChapterID)
			}

			chapter.SetContent(genResp.GetContent())
			chapter.Status = entity.ChapterStatusCompleted
			if m := genResp.GetMetadata(); m != nil {
				chapter.GenerationMetadata = &entity.GenerationMetadata{
					Model:            m.GetModel(),
					Provider:         m.GetProvider(),
					PromptTokens:     int(m.GetPromptTokens()),
					CompletionTokens: int(m.GetCompletionTokens()),
					Temperature:      m.GetTemperature(),
					GeneratedAt:      time.Unix(m.GetGeneratedAt(), 0).Format(time.RFC3339),
				}
			}

			if err := chapterRepo.Update(txCtx, chapter); err != nil {
				return err
			}

			result, _ := json.Marshal(map[string]interface{}{
				"chapter_id": chapter.ID,
				"word_count": genResp.GetWordCount(),
			})
			_ = jobRepo.UpdateProgress(txCtx, payload.JobID, 100)
			return jobRepo.SetResult(txCtx, payload.JobID, result, "")
		})
	})

	if err := consumer.Start(ctx); err != nil {
		logger.Fatal(ctx, "failed to start consumer", err)
	}

	log := logger.FromContext(ctx)
	log.Info("job-worker started")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("job-worker shutting down")
	consumer.Stop()
}

func hostnameConsumerName() string {
	host, err := os.Hostname()
	if err != nil || host == "" {
		host = "worker"
	}
	return fmt.Sprintf("%s-%d", host, os.Getpid())
}
