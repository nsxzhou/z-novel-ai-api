// Package config 提供配置加载功能
package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Load 加载配置文件
// 按优先级加载：默认配置 -> 环境配置 -> 环境变量
func Load() (*Config, error) {
	v := viper.New()

	// 设置配置文件路径
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath("./configs")
	v.AddConfigPath("/etc/z-novel-ai")
	v.AddConfigPath(".")

	// 读取默认配置
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// 读取环境特定配置
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "development"
	}

	v.SetConfigName("config." + env)
	if err := v.MergeInConfig(); err != nil {
		// 环境配置可选，不存在不报错
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to merge env config: %w", err)
		}
	}

	// 绑定环境变量
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// 设置默认值
	setDefaults(v)

	// 解析配置
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

// MustLoad 加载配置，失败时 panic
func MustLoad() *Config {
	cfg, err := Load()
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}
	return cfg
}

// setDefaults 设置配置默认值
func setDefaults(v *viper.Viper) {
	// 应用默认值
	v.SetDefault("app.name", "z-novel-ai-api")
	v.SetDefault("app.version", "v0.0.0")
	v.SetDefault("app.env", "development")

	// HTTP 服务器默认值
	v.SetDefault("server.http.host", "0.0.0.0")
	v.SetDefault("server.http.port", 8080)
	v.SetDefault("server.http.read_timeout", "30s")
	v.SetDefault("server.http.write_timeout", "60s")
	v.SetDefault("server.http.idle_timeout", "120s")

	// gRPC 服务器默认值
	v.SetDefault("server.grpc.host", "0.0.0.0")
	v.SetDefault("server.grpc.port", 9090)
	v.SetDefault("server.grpc.max_recv_msg_size", 16777216)
	v.SetDefault("server.grpc.max_send_msg_size", 16777216)

	// 数据库默认值
	v.SetDefault("database.postgres.host", "localhost")
	v.SetDefault("database.postgres.port", 5432)
	v.SetDefault("database.postgres.user", "postgres")
	v.SetDefault("database.postgres.database", "z_novel_ai")
	v.SetDefault("database.postgres.ssl_mode", "disable")
	v.SetDefault("database.postgres.max_open_conns", 50)
	v.SetDefault("database.postgres.max_idle_conns", 10)
	v.SetDefault("database.postgres.conn_max_lifetime", "30m")
	v.SetDefault("database.postgres.conn_max_idle_time", "5m")

	// Redis 默认值
	v.SetDefault("cache.redis.host", "localhost")
	v.SetDefault("cache.redis.port", 6379)
	v.SetDefault("cache.redis.db", 0)
	v.SetDefault("cache.redis.pool_size", 100)
	v.SetDefault("cache.redis.min_idle_conns", 10)
	v.SetDefault("cache.redis.dial_timeout", "5s")
	v.SetDefault("cache.redis.read_timeout", "3s")
	v.SetDefault("cache.redis.write_timeout", "3s")

	// Milvus 默认值
	v.SetDefault("vector.milvus.host", "localhost")
	v.SetDefault("vector.milvus.port", 19530)
	v.SetDefault("vector.milvus.collection_prefix", "z_novel")
	v.SetDefault("vector.milvus.index_type", "HNSW")
	v.SetDefault("vector.milvus.metric_type", "COSINE")
	v.SetDefault("vector.milvus.hnsw_m", 16)
	v.SetDefault("vector.milvus.hnsw_ef_construction", 200)

	// 可观测性默认值
	v.SetDefault("observability.logging.level", "info")
	v.SetDefault("observability.logging.format", "json")
	v.SetDefault("observability.logging.output", "stdout")
	v.SetDefault("observability.tracing.enabled", true)
	v.SetDefault("observability.tracing.exporter", "otlp")
	v.SetDefault("observability.tracing.endpoint", "localhost:4317")
	v.SetDefault("observability.tracing.sample_rate", 1.0)
	v.SetDefault("observability.metrics.enabled", true)
	v.SetDefault("observability.metrics.port", 9091)
	v.SetDefault("observability.metrics.path", "/metrics")

	// 安全默认值
	v.SetDefault("security.jwt.issuer", "z-novel-ai")
	v.SetDefault("security.jwt.expiration", "24h")
	v.SetDefault("security.jwt.refresh_expiration", "168h")
	v.SetDefault("security.rate_limit.enabled", true)
	v.SetDefault("security.rate_limit.requests_per_second", 100)
	v.SetDefault("security.rate_limit.burst", 200)

	// 功能开关默认值
	v.SetDefault("features.validation.enabled", true)
	v.SetDefault("features.validation.default_pass_on_failure", true)
	v.SetDefault("features.memory_writeback.enabled", true)
	v.SetDefault("features.memory_writeback.async", true)
}
