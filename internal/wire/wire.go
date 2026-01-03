//go:build wireinject
// +build wireinject

// Package wire 提供依赖注入配置
package wire

import (
	"context"

	"github.com/google/wire"

	"z-novel-ai-api/internal/config"
	"z-novel-ai-api/internal/infrastructure/messaging"
	"z-novel-ai-api/internal/infrastructure/persistence/milvus"
	"z-novel-ai-api/internal/infrastructure/persistence/postgres"
	"z-novel-ai-api/internal/infrastructure/persistence/redis"
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

// InitializeDataLayer 初始化数据层
func InitializeDataLayer(ctx context.Context, cfg *config.Config) (*DataLayer, func(), error) {
	wire.Build(
		PostgresSet,
		RedisSet,
		MessagingSet,
		MilvusSet,
		wire.Struct(new(DataLayer), "*"),
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
)

// RedisSet Redis 提供者集合
var RedisSet = wire.NewSet(
	ProvideRedisClient,
	redis.NewCache,
	redis.NewRateLimiter,
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
