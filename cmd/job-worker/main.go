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

	"github.com/joho/godotenv"

	"z-novel-ai-api/internal/application/quota"
	appretrieval "z-novel-ai-api/internal/application/retrieval"
	"z-novel-ai-api/internal/application/story"
	"z-novel-ai-api/internal/config"
	"z-novel-ai-api/internal/domain/entity"
	infraembedding "z-novel-ai-api/internal/infrastructure/embedding"
	"z-novel-ai-api/internal/infrastructure/llm"
	"z-novel-ai-api/internal/infrastructure/messaging"
	"z-novel-ai-api/internal/infrastructure/persistence/milvus"
	"z-novel-ai-api/internal/infrastructure/persistence/postgres"
	"z-novel-ai-api/internal/infrastructure/persistence/redis"
	einoobs "z-novel-ai-api/internal/observability/eino"
	"z-novel-ai-api/pkg/logger"
	"z-novel-ai-api/pkg/tracer"
)

func main() {
	// 加载 .env 文件（如果存在）
	_ = godotenv.Load()

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

	// 1. 初始化基础基础设施
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

	// 额外依赖：Milvus + Embedding（用于同步写索引；不可用时自动降级）
	var vectorRepo *milvus.Repository
	var indexer *appretrieval.Indexer
	var retrievalEngine *appretrieval.Engine
	var milvusClient *milvus.Client
	if client, err := milvus.NewClient(ctx, &cfg.Vector.Milvus); err != nil {
		logger.Warn(ctx, "milvus not available, vector indexing disabled", "error", err.Error())
	} else {
		milvusClient = client
		defer func() { _ = milvusClient.Close() }()
		vectorRepo = milvus.NewRepository(milvusClient)
	}

	embedder, err := infraembedding.NewEinoEmbedder(ctx, &cfg.Embedding)
	if err != nil {
		logger.Warn(ctx, "embedding not available, vector indexing disabled", "error", err.Error())
	} else if vectorRepo != nil {
		vectorPort := milvus.NewRetrievalVectorRepository(vectorRepo)
		indexer = appretrieval.NewIndexer(embedder, vectorPort, cfg.Embedding.BatchSize)
		retrievalEngine = appretrieval.NewEngine(embedder, vectorPort, nil, cfg.Embedding.BatchSize)
	}

	// 2. 初始化 Repositories
	txMgr := postgres.NewTxManager(pgClient)
	tenantCtx := postgres.NewTenantContext(pgClient)
	jobRepo := postgres.NewJobRepository(pgClient)
	tenantRepo := postgres.NewTenantRepository(pgClient)
	chapterRepo := postgres.NewChapterRepository(pgClient)
	projectRepo := postgres.NewProjectRepository(pgClient)
	llmUsageRepo := postgres.NewLLMUsageEventRepository(pgClient)

	// 3. 初始化 Eino 全局 callbacks（搬移到这里以确保 Repo 变量已定义）
	einoobs.Init(tenantRepo, llmUsageRepo, tenantCtx)

	// 4. 初始化应用逻辑
	llmFactory := llm.NewEinoFactory(cfg)
	foundationGenerator := story.NewFoundationGenerator(llmFactory)
	chapterGenerator := story.NewChapterGenerator(llmFactory)
	tokenQuotaChecker := quota.NewTokenQuotaChecker(tenantRepo)

	// 5. 初始化消息消费者
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

	// 注册 chapter_gen 处理器
	consumer.RegisterHandler("chapter_gen", func(_ context.Context, msg *messaging.Message) error {
		var payload messaging.GenerationJobMessage
		if err := msg.UnmarshalPayload(&payload); err != nil {
			return err
		}

		var chapterForIndex *entity.Chapter
		txErr := txMgr.WithTransaction(ctx, func(txCtx context.Context) error {
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

			// 余额检查（不足时不重试，直接标记失败）
			if _, err := tokenQuotaChecker.CheckBalance(txCtx, payload.TenantID, 1000); err != nil {
				var exceeded quota.TokenBalanceExceededError
				if errors.As(err, &exceeded) {
					job.Fail(err.Error())
					_ = jobRepo.Update(txCtx, job)
					if payload.ChapterID != nil {
						_ = markChapterDraft(txCtx, chapterRepo, *payload.ChapterID)
					}
					return nil
				}
				return err
			}

			if payload.ChapterID == nil {
				job.Fail("chapter_id is required for chapter_gen job")
				_ = jobRepo.Update(txCtx, job)
				return fmt.Errorf("chapter_id is required")
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

			project, err := projectRepo.GetByID(txCtx, payload.ProjectID)
			if err != nil {
				job.Fail(err.Error())
				_ = jobRepo.Update(txCtx, job)
				return err
			}
			if project == nil {
				err := fmt.Errorf("project not found: %s", payload.ProjectID)
				job.Fail(err.Error())
				_ = jobRepo.Update(txCtx, job)
				return err
			}

			in, err := buildChapterInput(cfg, project, chapter, payload.Params)
			if err != nil {
				job.Fail(err.Error())
				_ = jobRepo.Update(txCtx, job)
				_ = markChapterDraft(txCtx, chapterRepo, chapter.ID)
				return nil
			}

			// RAG：在生成前召回上下文，注入 Prompt（失败不影响主流程）
			if retrievalEngine != nil {
				ro, rerr := retrievalEngine.Search(txCtx, appretrieval.SearchInput{
					TenantID:         payload.TenantID,
					ProjectID:        payload.ProjectID,
					Query:            in.ChapterOutline,
					CurrentStoryTime: chapter.StoryTimeStart,
					TopK:             12,
					IncludeEntities:  false,
				})
				if rerr == nil && ro != nil && len(ro.Segments) > 0 {
					in.RetrievedContext = appretrieval.BuildPromptContext(ro.Segments, 10, 360)
				}
			}

			if chapter.Status != entity.ChapterStatusGenerating {
				chapter.Status = entity.ChapterStatusGenerating
				_ = chapterRepo.Update(txCtx, chapter)
			}

			job.Start()
			job.UpdateProgress(5)
			if err := jobRepo.Update(txCtx, job); err != nil {
				return err
			}

			out, err := chapterGenerator.Generate(txCtx, in)
			if err != nil {
				job.Fail(err.Error())
				_ = jobRepo.Update(txCtx, job)
				return err
			}

			job.UpdateProgress(80)
			if err := jobRepo.Update(txCtx, job); err != nil {
				return err
			}

			chapter.Outline = in.ChapterOutline
			chapter.SetContent(out.Content)
			chapter.Status = entity.ChapterStatusCompleted
			chapter.GenerationMetadata = &entity.GenerationMetadata{
				Model:            out.Meta.Model,
				Provider:         out.Meta.Provider,
				PromptTokens:     out.Meta.PromptTokens,
				CompletionTokens: out.Meta.CompletionTokens,
				Temperature:      out.Meta.Temperature,
				GeneratedAt:      out.Meta.GeneratedAt.Format(time.RFC3339),
			}

			if err := chapterRepo.Update(txCtx, chapter); err != nil {
				job.Fail(err.Error())
				_ = jobRepo.Update(txCtx, job)
				return err
			}

			if stats, err := projectRepo.GetStats(txCtx, project.ID); err == nil && stats != nil {
				_ = projectRepo.UpdateWordCount(txCtx, project.ID, int(stats.TotalWordCount))
			}

			result, _ := json.Marshal(map[string]interface{}{
				"chapter_id": chapter.ID,
				"word_count": len([]rune(out.Content)),
			})
			job.SetLLMMetrics(out.Meta.Provider, out.Meta.Model, out.Meta.PromptTokens, out.Meta.CompletionTokens)
			job.Complete(result)
			if err := jobRepo.Update(txCtx, job); err != nil {
				return err
			}

			// 事务内仅准备索引输入；索引写入放到事务提交之后执行，避免持有 DB 连接。
			chapterForIndex = &entity.Chapter{
				ID:             chapter.ID,
				ProjectID:      chapter.ProjectID,
				Title:          chapter.Title,
				ContentText:    chapter.ContentText,
				StoryTimeStart: chapter.StoryTimeStart,
				StoryTimeEnd:   chapter.StoryTimeEnd,
			}
			return nil
		})

		if txErr != nil {
			return txErr
		}

		// 同步写索引：章节生成成功后写入向量索引（失败不影响消费 ACK）
		if indexer != nil && chapterForIndex != nil {
			indexCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			if err := indexer.IndexChapter(indexCtx, payload.TenantID, payload.ProjectID, chapterForIndex); err != nil && !errors.Is(err, appretrieval.ErrVectorDisabled) {
				logger.Warn(ctx, "failed to index chapter after job completion",
					"error", err.Error(),
					"chapter_id", chapterForIndex.ID,
				)
			}
		}
		return nil
	})

	// 注册 foundation_gen 处理器
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

			// 余额检查
			if _, err := tokenQuotaChecker.CheckBalance(txCtx, payload.TenantID, 1000); err != nil {
				var exceeded quota.TokenBalanceExceededError
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

func buildChapterInput(cfg *config.Config, project *entity.Project, chapter *entity.Chapter, params map[string]interface{}) (*story.ChapterGenerateInput, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}
	if project == nil {
		return nil, fmt.Errorf("project is nil")
	}
	if chapter == nil {
		return nil, fmt.Errorf("chapter is nil")
	}

	outline, _ := params["outline"].(string)
	outline = strings.TrimSpace(outline)
	if outline == "" {
		outline = strings.TrimSpace(chapter.Outline)
	}
	if outline == "" {
		return nil, fmt.Errorf("missing outline")
	}

	targetWordCount := 0
	if v, ok := params["target_word_count"].(float64); ok {
		targetWordCount = int(v)
	}
	if targetWordCount <= 0 {
		if project.Settings != nil && project.Settings.DefaultChapterLength > 0 {
			targetWordCount = project.Settings.DefaultChapterLength
		} else {
			targetWordCount = 2000
		}
	}

	provider, _ := params["provider"].(string)
	modelName, _ := params["model"].(string)
	provider, modelName, err := resolveProviderModelForWorker(cfg, provider, modelName)
	if err != nil {
		return nil, err
	}

	var temperature *float32
	if v, ok := params["temperature"].(float64); ok && v != 0 {
		t := float32(v)
		temperature = &t
	} else if project.Settings != nil && project.Settings.Temperature != 0 {
		t := float32(project.Settings.Temperature)
		temperature = &t
	}

	writingStyle := ""
	pov := ""
	if project.Settings != nil {
		writingStyle = strings.TrimSpace(project.Settings.WritingStyle)
		pov = strings.TrimSpace(project.Settings.POV)
	}

	return &story.ChapterGenerateInput{
		ProjectTitle:       project.Title,
		ProjectDescription: project.Description,
		ChapterTitle:       chapter.Title,
		ChapterOutline:     outline,
		TargetWordCount:    targetWordCount,
		WritingStyle:       writingStyle,
		POV:                pov,
		Provider:           provider,
		Model:              modelName,
		Temperature:        temperature,
	}, nil
}

func resolveProviderModelForWorker(cfg *config.Config, provider, modelName string) (string, string, error) {
	if cfg == nil {
		return "", "", fmt.Errorf("config is nil")
	}

	p := strings.TrimSpace(provider)
	if p == "" {
		p = strings.TrimSpace(cfg.LLM.DefaultProvider)
	}
	if p == "" {
		return "", "", fmt.Errorf("llm provider not specified")
	}
	if len(p) > 32 {
		return "", "", fmt.Errorf("llm provider too long")
	}

	providerCfg, ok := cfg.LLM.Providers[p]
	if !ok {
		return "", "", fmt.Errorf("llm provider not found: %s", p)
	}

	m := strings.TrimSpace(modelName)
	if m == "" {
		m = strings.TrimSpace(providerCfg.Model)
	}
	if len(m) > 64 {
		return "", "", fmt.Errorf("llm model too long")
	}

	return p, m, nil
}

func markChapterDraft(ctx context.Context, chapterRepo *postgres.ChapterRepository, chapterID string) error {
	if strings.TrimSpace(chapterID) == "" {
		return nil
	}
	chapter, err := chapterRepo.GetByID(ctx, chapterID)
	if err != nil || chapter == nil {
		return err
	}
	if chapter.Status == entity.ChapterStatusGenerating {
		chapter.Status = entity.ChapterStatusDraft
		return chapterRepo.Update(ctx, chapter)
	}
	return nil
}
