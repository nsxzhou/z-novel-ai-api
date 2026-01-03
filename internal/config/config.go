// Package config 提供配置加载和管理功能
package config

import (
	"time"
)

// Config 应用配置根结构
type Config struct {
	App           AppConfig           `yaml:"app" mapstructure:"app"`
	Server        ServerConfig        `yaml:"server" mapstructure:"server"`
	Clients       ClientsConfig       `yaml:"clients" mapstructure:"clients"`
	Database      DatabaseConfig      `yaml:"database" mapstructure:"database"`
	Cache         CacheConfig         `yaml:"cache" mapstructure:"cache"`
	Vector        VectorConfig        `yaml:"vector" mapstructure:"vector"`
	Storage       StorageConfig       `yaml:"storage" mapstructure:"storage"`
	LLM           LLMConfig           `yaml:"llm" mapstructure:"llm"`
	Embedding     EmbeddingConfig     `yaml:"embedding" mapstructure:"embedding"`
	Messaging     MessagingConfig     `yaml:"messaging" mapstructure:"messaging"`
	Observability ObservabilityConfig `yaml:"observability" mapstructure:"observability"`
	Security      SecurityConfig      `yaml:"security" mapstructure:"security"`
	Features      FeaturesConfig      `yaml:"features" mapstructure:"features"`
}

// AppConfig 应用基础配置
type AppConfig struct {
	Name    string `yaml:"name" mapstructure:"name"`
	Version string `yaml:"version" mapstructure:"version"`
	Env     string `yaml:"env" mapstructure:"env"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	HTTP HTTPServerConfig `yaml:"http" mapstructure:"http"`
	GRPC GRPCServerConfig `yaml:"grpc" mapstructure:"grpc"`
}

// ClientsConfig 外部/内部依赖客户端配置
type ClientsConfig struct {
	GRPC GRPCClientsConfig `yaml:"grpc" mapstructure:"grpc"`
}

// GRPCClientsConfig 内部 gRPC 客户端配置
type GRPCClientsConfig struct {
	// DialTimeout 拨号超时
	DialTimeout time.Duration `yaml:"dial_timeout" mapstructure:"dial_timeout"`

	// RetrievalServiceAddr 检索服务地址 (host:port)
	RetrievalServiceAddr string `yaml:"retrieval_service_addr" mapstructure:"retrieval_service_addr"`
	// StoryGenServiceAddr 小说生成服务地址 (host:port)
	StoryGenServiceAddr string `yaml:"story_gen_service_addr" mapstructure:"story_gen_service_addr"`
	// MemoryServiceAddr 记忆服务地址 (host:port)
	MemoryServiceAddr string `yaml:"memory_service_addr" mapstructure:"memory_service_addr"`
	// ValidatorServiceAddr 校验服务地址 (host:port)
	ValidatorServiceAddr string `yaml:"validator_service_addr" mapstructure:"validator_service_addr"`
}

// HTTPServerConfig HTTP 服务器配置
type HTTPServerConfig struct {
	Host         string        `yaml:"host" mapstructure:"host"`
	Port         int           `yaml:"port" mapstructure:"port"`
	ReadTimeout  time.Duration `yaml:"read_timeout" mapstructure:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout" mapstructure:"write_timeout"`
	IdleTimeout  time.Duration `yaml:"idle_timeout" mapstructure:"idle_timeout"`
}

// GRPCServerConfig gRPC 服务器配置
type GRPCServerConfig struct {
	Host           string `yaml:"host" mapstructure:"host"`
	Port           int    `yaml:"port" mapstructure:"port"`
	MaxRecvMsgSize int    `yaml:"max_recv_msg_size" mapstructure:"max_recv_msg_size"`
	MaxSendMsgSize int    `yaml:"max_send_msg_size" mapstructure:"max_send_msg_size"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Postgres PostgresConfig `yaml:"postgres" mapstructure:"postgres"`
	Milvus   MilvusConfig   `yaml:"milvus" mapstructure:"milvus"`
}

// PostgresConfig PostgreSQL 配置
type PostgresConfig struct {
	Host            string        `yaml:"host" mapstructure:"host"`
	Port            int           `yaml:"port" mapstructure:"port"`
	User            string        `yaml:"user" mapstructure:"user"`
	Password        string        `yaml:"password" mapstructure:"password"`
	Database        string        `yaml:"database" mapstructure:"database"`
	SSLMode         string        `yaml:"ssl_mode" mapstructure:"ssl_mode"`
	MaxOpenConns    int           `yaml:"max_open_conns" mapstructure:"max_open_conns"`
	MaxIdleConns    int           `yaml:"max_idle_conns" mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime" mapstructure:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `yaml:"conn_max_idle_time" mapstructure:"conn_max_idle_time"`
}

// CacheConfig 缓存配置
type CacheConfig struct {
	Redis RedisConfig `yaml:"redis" mapstructure:"redis"`
}

// RedisConfig Redis 配置
type RedisConfig struct {
	Host         string        `yaml:"host" mapstructure:"host"`
	Port         int           `yaml:"port" mapstructure:"port"`
	Password     string        `yaml:"password" mapstructure:"password"`
	DB           int           `yaml:"db" mapstructure:"db"`
	PoolSize     int           `yaml:"pool_size" mapstructure:"pool_size"`
	MinIdleConns int           `yaml:"min_idle_conns" mapstructure:"min_idle_conns"`
	DialTimeout  time.Duration `yaml:"dial_timeout" mapstructure:"dial_timeout"`
	ReadTimeout  time.Duration `yaml:"read_timeout" mapstructure:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout" mapstructure:"write_timeout"`
}

// VectorConfig 向量数据库配置
type VectorConfig struct {
	Milvus MilvusConfig `yaml:"milvus" mapstructure:"milvus"`
}

// MilvusConfig Milvus 配置
type MilvusConfig struct {
	Host               string `yaml:"host" mapstructure:"host"`
	Port               int    `yaml:"port" mapstructure:"port"`
	User               string `yaml:"user" mapstructure:"user"`
	Password           string `yaml:"password" mapstructure:"password"`
	CollectionPrefix   string `yaml:"collection_prefix" mapstructure:"collection_prefix"`
	IndexType          string `yaml:"index_type" mapstructure:"index_type"`
	MetricType         string `yaml:"metric_type" mapstructure:"metric_type"`
	HNSWM              int    `yaml:"hnsw_m" mapstructure:"hnsw_m"`
	HNSWEfConstruction int    `yaml:"hnsw_ef_construction" mapstructure:"hnsw_ef_construction"`
}

// StorageConfig 对象存储配置
type StorageConfig struct {
	R2 R2Config `yaml:"r2" mapstructure:"r2"`
}

// R2Config Cloudflare R2 配置
type R2Config struct {
	AccountID       string `yaml:"account_id" mapstructure:"account_id"`
	AccessKeyID     string `yaml:"access_key_id" mapstructure:"access_key_id"`
	SecretAccessKey string `yaml:"secret_access_key" mapstructure:"secret_access_key"`
	Bucket          string `yaml:"bucket" mapstructure:"bucket"`
	PublicURL       string `yaml:"public_url" mapstructure:"public_url"`
}

// LLMConfig LLM 配置
type LLMConfig struct {
	DefaultProvider string                    `yaml:"default_provider" mapstructure:"default_provider"`
	Providers       map[string]ProviderConfig `yaml:"providers" mapstructure:"providers"`
	FallbackChain   []string                  `yaml:"fallback_chain" mapstructure:"fallback_chain"`
}

// ProviderConfig LLM 提供商配置
type ProviderConfig struct {
	APIKey      string        `yaml:"api_key" mapstructure:"api_key"`
	BaseURL     string        `yaml:"base_url" mapstructure:"base_url"`
	Model       string        `yaml:"model" mapstructure:"model"`
	MaxTokens   int           `yaml:"max_tokens" mapstructure:"max_tokens"`
	Temperature float64       `yaml:"temperature" mapstructure:"temperature"`
	Timeout     time.Duration `yaml:"timeout" mapstructure:"timeout"`
}

// EmbeddingConfig Embedding 配置
type EmbeddingConfig struct {
	Provider  string `yaml:"provider" mapstructure:"provider"`
	Model     string `yaml:"model" mapstructure:"model"`
	Dimension int    `yaml:"dimension" mapstructure:"dimension"`
	BatchSize int    `yaml:"batch_size" mapstructure:"batch_size"`
	Endpoint  string `yaml:"endpoint" mapstructure:"endpoint"`
}

// MessagingConfig 消息队列配置
type MessagingConfig struct {
	RedisStream RedisStreamConfig `yaml:"redis_stream" mapstructure:"redis_stream"`
}

// RedisStreamConfig Redis Stream 配置
type RedisStreamConfig struct {
	MaxLen              int           `yaml:"max_len" mapstructure:"max_len"`
	ConsumerGroupPrefix string        `yaml:"consumer_group_prefix" mapstructure:"consumer_group_prefix"`
	BlockTimeout        time.Duration `yaml:"block_timeout" mapstructure:"block_timeout"`
	ClaimInterval       time.Duration `yaml:"claim_interval" mapstructure:"claim_interval"`
	RetryLimit          int           `yaml:"retry_limit" mapstructure:"retry_limit"`
	RetryBackoff        BackoffConfig `yaml:"retry_backoff" mapstructure:"retry_backoff"`
}

// BackoffConfig 退避配置
type BackoffConfig struct {
	Initial    time.Duration `yaml:"initial" mapstructure:"initial"`
	Max        time.Duration `yaml:"max" mapstructure:"max"`
	Multiplier float64       `yaml:"multiplier" mapstructure:"multiplier"`
}

// ObservabilityConfig 可观测性配置
type ObservabilityConfig struct {
	Logging LoggingConfig `yaml:"logging" mapstructure:"logging"`
	Tracing TracingConfig `yaml:"tracing" mapstructure:"tracing"`
	Metrics MetricsConfig `yaml:"metrics" mapstructure:"metrics"`
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Level  string `yaml:"level" mapstructure:"level"`
	Format string `yaml:"format" mapstructure:"format"`
	Output string `yaml:"output" mapstructure:"output"`
}

// TracingConfig 追踪配置
type TracingConfig struct {
	Enabled    bool    `yaml:"enabled" mapstructure:"enabled"`
	Exporter   string  `yaml:"exporter" mapstructure:"exporter"`
	Endpoint   string  `yaml:"endpoint" mapstructure:"endpoint"`
	SampleRate float64 `yaml:"sample_rate" mapstructure:"sample_rate"`
}

// MetricsConfig 指标配置
type MetricsConfig struct {
	Enabled bool   `yaml:"enabled" mapstructure:"enabled"`
	Port    int    `yaml:"port" mapstructure:"port"`
	Path    string `yaml:"path" mapstructure:"path"`
}

// SecurityConfig 安全配置
type SecurityConfig struct {
	JWT       JWTConfig       `yaml:"jwt" mapstructure:"jwt"`
	RateLimit RateLimitConfig `yaml:"rate_limit" mapstructure:"rate_limit"`
	CORS      CORSConfig      `yaml:"cors" mapstructure:"cors"`
}

// JWTConfig JWT 配置
type JWTConfig struct {
	Secret            string        `yaml:"secret" mapstructure:"secret"`
	Issuer            string        `yaml:"issuer" mapstructure:"issuer"`
	Expiration        time.Duration `yaml:"expiration" mapstructure:"expiration"`
	RefreshExpiration time.Duration `yaml:"refresh_expiration" mapstructure:"refresh_expiration"`
}

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	Enabled           bool `yaml:"enabled" mapstructure:"enabled"`
	RequestsPerSecond int  `yaml:"requests_per_second" mapstructure:"requests_per_second"`
	Burst             int  `yaml:"burst" mapstructure:"burst"`
}

// CORSConfig CORS 配置
type CORSConfig struct {
	AllowedOrigins []string `yaml:"allowed_origins" mapstructure:"allowed_origins"`
	AllowedMethods []string `yaml:"allowed_methods" mapstructure:"allowed_methods"`
	AllowedHeaders []string `yaml:"allowed_headers" mapstructure:"allowed_headers"`
}

// FeaturesConfig 功能开关配置
type FeaturesConfig struct {
	Validation      ValidationFeature      `yaml:"validation" mapstructure:"validation"`
	MemoryWriteback MemoryWritebackFeature `yaml:"memory_writeback" mapstructure:"memory_writeback"`
}

// ValidationFeature 校验功能开关
type ValidationFeature struct {
	Enabled              bool `yaml:"enabled" mapstructure:"enabled"`
	DefaultPassOnFailure bool `yaml:"default_pass_on_failure" mapstructure:"default_pass_on_failure"`
}

// MemoryWritebackFeature 记忆回写功能开关
type MemoryWritebackFeature struct {
	Enabled bool `yaml:"enabled" mapstructure:"enabled"`
	Async   bool `yaml:"async" mapstructure:"async"`
}
