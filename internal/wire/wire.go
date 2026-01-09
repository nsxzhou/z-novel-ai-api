//go:build wireinject
// +build wireinject

// Package wire 提供依赖注入配置
package wire

import (
	"context"

	"github.com/google/wire"

	"z-novel-ai-api/internal/application/quota"
	"z-novel-ai-api/internal/application/story"
	"z-novel-ai-api/internal/config"
	"z-novel-ai-api/internal/domain/repository"
	"z-novel-ai-api/internal/infrastructure/llm"
	"z-novel-ai-api/internal/infrastructure/messaging"
	"z-novel-ai-api/internal/infrastructure/persistence/milvus"
	"z-novel-ai-api/internal/infrastructure/persistence/postgres"
	"z-novel-ai-api/internal/infrastructure/persistence/redis"
	"z-novel-ai-api/internal/interfaces/http/handler"
	"z-novel-ai-api/internal/interfaces/http/middleware"
	"z-novel-ai-api/internal/interfaces/http/router"
)

// DataLayer 数据层依赖容器
type DataLayer struct {
	// PostgreSQL
	PgClient      *postgres.Client
	TxManager     *postgres.TxManager
	TenantContext *postgres.TenantContext
	TenantRepo    *postgres.TenantRepository
	UserRepo      *postgres.UserRepository
	ProjectRepo   *postgres.ProjectRepository
	VolumeRepo    *postgres.VolumeRepository
	ChapterRepo   *postgres.ChapterRepository
	EntityRepo    *postgres.EntityRepository
	RelationRepo  *postgres.RelationRepository
	EventRepo     *postgres.EventRepository
	JobRepo       *postgres.JobRepository
	LLMUsageRepo  *postgres.LLMUsageEventRepository
	SessionRepo   *postgres.ConversationSessionRepository
	TurnRepo      *postgres.ConversationTurnRepository
	ArtifactRepo  *postgres.ArtifactRepository
	PCSessionRepo *postgres.ProjectCreationSessionRepository
	PCTurnRepo    *postgres.ProjectCreationTurnRepository

	// Redis
	RedisClient *redis.Client
	Cache       *redis.Cache
	RateLimiter *redis.RateLimiter

	// Messaging
	Producer *messaging.Producer

	// Milvus
	MilvusClient *milvus.Client
	VectorRepo   *milvus.Repository
}

// PostgresOnlyDataLayer 仅包含 PostgreSQL 的数据层（用于 bootstrap）
type PostgresOnlyDataLayer struct {
	PgClient      *postgres.Client
	TxManager     *postgres.TxManager
	TenantContext *postgres.TenantContext
	TenantRepo    *postgres.TenantRepository
	UserRepo      *postgres.UserRepository
	ProjectRepo   *postgres.ProjectRepository
	VolumeRepo    *postgres.VolumeRepository
	ChapterRepo   *postgres.ChapterRepository
	EntityRepo    *postgres.EntityRepository
	RelationRepo  *postgres.RelationRepository
	EventRepo     *postgres.EventRepository
	JobRepo       *postgres.JobRepository
	LLMUsageRepo  *postgres.LLMUsageEventRepository
	SessionRepo   *postgres.ConversationSessionRepository
	TurnRepo      *postgres.ConversationTurnRepository
	ArtifactRepo  *postgres.ArtifactRepository
	PCSessionRepo *postgres.ProjectCreationSessionRepository
	PCTurnRepo    *postgres.ProjectCreationTurnRepository
}

// InitializeDataLayer 初始化数据层
func InitializeDataLayer(ctx context.Context, cfg *config.Config) (*DataLayer, func(), error) {
	wire.Build(
		RepoSet,
		RedisSet,
		MessagingSet,
		MilvusSet,
		wire.Struct(new(DataLayer), "*"),
	)
	return nil, nil, nil
}

// InitializePostgresOnly 仅初始化 PostgreSQL 数据层（用于 bootstrap）
func InitializePostgresOnly(ctx context.Context, cfg *config.Config) (*PostgresOnlyDataLayer, func(), error) {
	wire.Build(
		PostgresSet,
		wire.Struct(new(PostgresOnlyDataLayer), "*"),
	)
	return nil, nil, nil
}

// InitializeApp 初始化整个应用（带路由器）
func InitializeApp(ctx context.Context, cfg *config.Config) (*router.Router, func(), error) {
	wire.Build(
		RepoSet,
		RedisSet,
		MessagingSet,
		GRPCClientsSet,
		RouterSet,
	)
	return nil, nil, nil
}

// PostgresSet PostgreSQL 提供者集合
var PostgresSet = wire.NewSet(
	ProvidePostgresClient,
	postgres.NewTxManager,
	postgres.NewTenantContext,
	postgres.NewTenantRepository,
	postgres.NewUserRepository,
	postgres.NewProjectRepository,
	postgres.NewVolumeRepository,
	postgres.NewChapterRepository,
	postgres.NewEntityRepository,
	postgres.NewRelationRepository,
	postgres.NewEventRepository,
	postgres.NewJobRepository,
	postgres.NewLLMUsageEventRepository,
	postgres.NewConversationSessionRepository,
	postgres.NewConversationTurnRepository,
	postgres.NewArtifactRepository,
	postgres.NewProjectCreationSessionRepository,
	postgres.NewProjectCreationTurnRepository,
)

// RedisSet Redis 提供者集合
var RedisSet = wire.NewSet(
	ProvideRedisClient,
	redis.NewCache,
	redis.NewRateLimiter,
	wire.Bind(new(middleware.RateLimiter), new(*redis.RateLimiter)),
)

// MessagingSet 消息队列提供者集合
var MessagingSet = wire.NewSet(
	ProvideMessagingProducer,
)

// MilvusSet Milvus 提供者集合
var MilvusSet = wire.NewSet(
	ProvideMilvusClient,
	milvus.NewRepository,
)

// GRPCClientsSet gRPC 客户端提供者集合
var GRPCClientsSet = wire.NewSet(
	ProvideRetrievalGRPCConn,
	ProvideRetrievalGRPCClient,
	ProvideStoryGenGRPCConn,
	ProvideStoryGenGRPCClient,
	ProvideMemoryGRPCConn,
	ProvideMemoryGRPCClient,
	ProvideValidatorGRPCConn,
	ProvideValidatorGRPCClient,
)

// RouterSet 路由器提供者集合
var RouterSet = wire.NewSet(
	ProvideAuthConfig,
	llm.NewEinoFactory,
	story.NewChapterGenerator,
	story.NewFoundationGenerator,
	story.NewArtifactGenerator,
	quota.NewTokenQuotaChecker,
	story.NewFoundationApplier,
	story.NewProjectCreationGenerator,
	handler.NewAuthHandler,
	handler.NewHealthHandler,
	handler.NewProjectHandler,
	handler.NewVolumeHandler,
	handler.NewChapterHandler,
	handler.NewEntityHandler,
	handler.NewFoundationHandler,
	handler.NewConversationHandler,
	handler.NewProjectCreationHandler,
	handler.NewArtifactHandler,
	handler.NewJobHandler,
	handler.NewRetrievalHandler,
	handler.NewStreamHandler,
	handler.NewUserHandler,
	handler.NewTenantHandler,
	handler.NewEventHandler,
	handler.NewRelationHandler,
	wire.Struct(new(router.RouterHandlers), "*"),
	router.NewWithDeps,
)

// RepoSet 整合了具体实现与接口绑定的集合
var RepoSet = wire.NewSet(
	PostgresSet,
	// 接口绑定
	wire.Bind(new(repository.Transactor), new(*postgres.TxManager)),
	wire.Bind(new(repository.TenantContextManager), new(*postgres.TenantContext)),
	wire.Bind(new(repository.TenantRepository), new(*postgres.TenantRepository)),
	wire.Bind(new(repository.UserRepository), new(*postgres.UserRepository)),
	wire.Bind(new(repository.ProjectRepository), new(*postgres.ProjectRepository)),
	wire.Bind(new(repository.VolumeRepository), new(*postgres.VolumeRepository)),
	wire.Bind(new(repository.ChapterRepository), new(*postgres.ChapterRepository)),
	wire.Bind(new(repository.EntityRepository), new(*postgres.EntityRepository)),
	wire.Bind(new(repository.RelationRepository), new(*postgres.RelationRepository)),
	wire.Bind(new(repository.JobRepository), new(*postgres.JobRepository)),
	wire.Bind(new(repository.LLMUsageEventRepository), new(*postgres.LLMUsageEventRepository)),
	wire.Bind(new(repository.EventRepository), new(*postgres.EventRepository)),
	wire.Bind(new(repository.ConversationSessionRepository), new(*postgres.ConversationSessionRepository)),
	wire.Bind(new(repository.ConversationTurnRepository), new(*postgres.ConversationTurnRepository)),
	wire.Bind(new(repository.ArtifactRepository), new(*postgres.ArtifactRepository)),
	wire.Bind(new(repository.ProjectCreationSessionRepository), new(*postgres.ProjectCreationSessionRepository)),
	wire.Bind(new(repository.ProjectCreationTurnRepository), new(*postgres.ProjectCreationTurnRepository)),
)

// ProvidePostgresClient 提供 PostgreSQL 客户端
func ProvidePostgresClient(cfg *config.Config) (*postgres.Client, func(), error) {
	client, err := postgres.NewClient(&cfg.Database.Postgres)
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() {
		client.Close()
	}
	return client, cleanup, nil
}

// ProvideRedisClient 提供 Redis 客户端
func ProvideRedisClient(cfg *config.Config) (*redis.Client, func(), error) {
	client, err := redis.NewClient(&cfg.Cache.Redis)
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() {
		client.Close()
	}
	return client, cleanup, nil
}

// ProvideMessagingProducer 提供消息生产者
func ProvideMessagingProducer(redisClient *redis.Client, cfg *config.Config) *messaging.Producer {
	maxLen := cfg.Messaging.RedisStream.MaxLen
	if maxLen <= 0 {
		maxLen = 100000
	}
	return messaging.NewProducer(redisClient.Redis(), int64(maxLen))
}

// ProvideMilvusClient 提供 Milvus 客户端
func ProvideMilvusClient(ctx context.Context, cfg *config.Config) (*milvus.Client, func(), error) {
	client, err := milvus.NewClient(ctx, &cfg.Database.Milvus)
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() {
		client.Close()
	}
	return client, cleanup, nil
}

// ProvideAuthConfig 提供认证配置
func ProvideAuthConfig(cfg *config.Config) middleware.AuthConfig {
	return middleware.AuthConfig{
		Secret:    cfg.Security.JWT.Secret,
		Issuer:    cfg.Security.JWT.Issuer,
		SkipPaths: middleware.DefaultSkipPaths,
		Enabled:   true,
	}
}
