// Package main 异步任务执行器入口（job-worker）
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	commonv1 "z-novel-ai-api/api/proto/gen/go/common"
	storyv1 "z-novel-ai-api/api/proto/gen/go/story"
	"z-novel-ai-api/internal/application/quota"
	"z-novel-ai-api/internal/application/story"
	"z-novel-ai-api/internal/config"
	"z-novel-ai-api/internal/domain/entity"
	"z-novel-ai-api/internal/infrastructure/llm"
	"z-novel-ai-api/internal/infrastructure/messaging"
	"z-novel-ai-api/internal/infrastructure/persistence/postgres"
	"z-novel-ai-api/internal/infrastructure/persistence/redis"
	grpcclient "z-novel-ai-api/internal/interfaces/grpc/client"
	einoobs "z-novel-ai-api/internal/observability/eino"
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

	// 初始化 Eino 全局 callbacks（指标/追踪/日志）
	einoobs.Init()

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
	tenantRepo := postgres.NewTenantRepository(pgClient)
	chapterRepo := postgres.NewChapterRepository(pgClient)
	projectRepo := postgres.NewProjectRepository(pgClient)

	llmUsageRepo := postgres.NewLLMUsageEventRepository(pgClient)

	llmFactory := llm.NewEinoFactory(cfg)
	foundationGenerator := story.NewFoundationGenerator(llmFactory)
	tokenQuotaChecker := quota.NewTokenQuotaChecker(jobRepo, llmUsageRepo)

	storyConn, err := grpcclient.Dial(ctx, cfg.Clients.GRPC.StoryGenServiceAddr, cfg.Clients.GRPC.DialTimeout)
	if err != nil {
		logger.Fatal(ctx, "failed to dial story-gen-svc", err)
	}
	defer func() { _ = storyConn.Close() }()
	storyClient := storyv1.NewStoryGenServiceClient(storyConn)

	consumer := messaging.NewConsumer(redisClient.Redis(), messaging.ConsumerConfig{
		Stream:        messaging.StreamStoryGen,
		Group:         messaging.ConsumerGroupGenWorker,
		ConsumerName:  hostnameConsumerName(),
		BlockTimeout:  cfg.Messaging.RedisStream.BlockTimeout,
		ClaimInterval: cfg.Messaging.RedisStream.ClaimInterval,
		RetryLimit:    cfg.Messaging.RedisStream.RetryLimit,
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
			if job.Status == entity.JobStatusCompleted {
				return nil
			}

			job.Start()
			job.UpdateProgress(5)
			if err := jobRepo.Update(txCtx, job); err != nil {
				return err
			}

			outline, _ := payload.Params["outline"].(string)
			targetWordCount := int32(0)
			if v, ok := payload.Params["target_word_count"].(float64); ok {
				targetWordCount = int32(v)
			}

			if payload.ChapterID == nil {
				job.Fail("chapter_id is required for chapter_gen job")
				_ = jobRepo.Update(txCtx, job)
				return fmt.Errorf("chapter_id is required")
			}

			traceID := uuid.NewString()
			genResp, err := storyClient.GenerateChapter(txCtx, &storyv1.GenerateChapterRequest{
				Context: &commonv1.TenantContext{
					TenantId: payload.TenantID,
					TraceId:  traceID,
				},
				ProjectId:       payload.ProjectID,
				ChapterId:       *payload.ChapterID,
				Outline:         outline,
				TargetWordCount: targetWordCount,
			})
			if err != nil {
				job.Fail(err.Error())
				_ = jobRepo.Update(txCtx, job)
				return err
			}
			job.UpdateProgress(80)
			if err := jobRepo.Update(txCtx, job); err != nil {
				return err
			}

			chapter, err := chapterRepo.GetByID(txCtx, *payload.ChapterID)
			if err != nil {
				job.Fail(err.Error())
				_ = jobRepo.Update(txCtx, job)
				return err
			}
			if chapter == nil {
				err := fmt.Errorf("chapter not found: %s", *payload.ChapterID)
				job.Fail(err.Error())
				_ = jobRepo.Update(txCtx, job)
				return err
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
				job.Fail(err.Error())
				_ = jobRepo.Update(txCtx, job)
				return err
			}

			result, _ := json.Marshal(map[string]interface{}{
				"chapter_id": chapter.ID,
				"word_count": genResp.GetWordCount(),
			})
			job.Complete(result)
			return jobRepo.Update(txCtx, job)
		})
	})

	consumer.RegisterHandler("foundation_gen", func(handlerCtx context.Context, msg *messaging.Message) error {
		var payload messaging.GenerationJobMessage
		if err := msg.UnmarshalPayload(&payload); err != nil {
			return err
		}

		return txMgr.WithTransaction(handlerCtx, func(txCtx context.Context) error {
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

			tenant, err := tenantRepo.GetByID(txCtx, payload.TenantID)
			if err != nil {
				return err
			}
			if tenant == nil {
				return fmt.Errorf("tenant not found: %s", payload.TenantID)
			}

			if _, _, err := tokenQuotaChecker.CheckDailyTokens(txCtx, payload.TenantID, tenant.Quota); err != nil {
				var exceeded quota.TokenQuotaExceededError
				if errors.As(err, &exceeded) {
					job.Fail(err.Error())
					_ = jobRepo.Update(txCtx, job)
					return nil
				}
				return err
			}

			project, err := projectRepo.GetByID(txCtx, payload.ProjectID)
			if err != nil {
				return err
			}
			if project == nil {
				return fmt.Errorf("project not found: %s", payload.ProjectID)
			}

			in, err := buildFoundationInput(project, payload.Params)
			if err != nil {
				job.Fail(err.Error())
				return jobRepo.Update(txCtx, job)
			}

			job.Start()
			if err := jobRepo.Update(txCtx, job); err != nil {
				return err
			}

			out, err := foundationGenerator.Generate(txCtx, in)
			if err != nil {
				job.Fail(err.Error())
				_ = jobRepo.Update(txCtx, job)
				return err
			}
			if err := story.ValidateFoundationPlan(out.Plan); err != nil {
				job.Fail(err.Error())
				_ = jobRepo.Update(txCtx, job)
				return nil
			}

			resultBytes, _ := json.Marshal(out.Plan)
			job.SetLLMMetrics(out.Meta.Provider, out.Meta.Model, out.Meta.PromptTokens, out.Meta.CompletionTokens)
			job.Complete(resultBytes)
			return jobRepo.Update(txCtx, job)
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

func buildFoundationInput(project *entity.Project, params map[string]interface{}) (*story.FoundationGenerateInput, error) {
	if project == nil {
		return nil, fmt.Errorf("project is nil")
	}

	prompt, _ := params["prompt"].(string)
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return nil, fmt.Errorf("missing prompt")
	}

	provider, _ := params["provider"].(string)
	modelName, _ := params["model"].(string)

	var temperature *float32
	if v, ok := params["temperature"].(float64); ok {
		f := float32(v)
		temperature = &f
	}

	var maxTokens *int
	if v, ok := params["max_tokens"].(float64); ok {
		i := int(v)
		maxTokens = &i
	}

	attachments := make([]story.TextAttachment, 0)
	if raw, ok := params["attachments"].([]interface{}); ok {
		for _, item := range raw {
			m, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			name, _ := m["name"].(string)
			content, _ := m["content"].(string)
			if strings.TrimSpace(content) == "" {
				continue
			}
			attachments = append(attachments, story.TextAttachment{
				Name:    name,
				Content: content,
			})
		}
	}

	return &story.FoundationGenerateInput{
		ProjectTitle:       project.Title,
		ProjectDescription: project.Description,
		Prompt:             prompt,
		Attachments:        attachments,
		Provider:           strings.TrimSpace(provider),
		Model:              strings.TrimSpace(modelName),
		Temperature:        temperature,
		MaxTokens:          maxTokens,
	}, nil
}
