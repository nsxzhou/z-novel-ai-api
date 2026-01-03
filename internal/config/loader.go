// Package config 提供配置加载功能
package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/viper"
)

// Load 加载配置文件
// 按优先级加载：默认配置 -> 环境配置 -> 环境变量
func Load() (*Config, error) {
	v := viper.New()
	v.SetConfigType("yaml")

	// 1. 加载默认配置
	if err := loadConfigFile(v, "configs/config.yaml", false); err != nil {
		return nil, err
	}

	// 2. 加载环境特定配置
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "development"
	}
	envFile := fmt.Sprintf("configs/config.%s.yaml", env)
	if err := loadConfigFile(v, envFile, true); err != nil {
		return nil, err
	}

	// 3. 绑定环境变量 (直接覆盖)
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// 设置默认值 (兜底)
	setDefaults(v)

	// 解析配置
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

// loadConfigFile 读取文件，执行环境变量替换，并加载到 viper
func loadConfigFile(v *viper.Viper, path string, optional bool) error {
	content, err := os.ReadFile(path)
	if err != nil {
		if optional && os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	// 执行环境变量替换
	expanded := expandEnv(string(content))

	// 加载到 viper
	reader := strings.NewReader(expanded)
	if v.ConfigFileUsed() == "" {
		if err := v.ReadConfig(reader); err != nil {
			return fmt.Errorf("failed to read processed config %s: %w", path, err)
		}
		// 手动标记已加载文件，防止后续 ReadInConfig 报错
		v.SetConfigFile(path)
	} else {
		if err := v.MergeConfig(reader); err != nil {
			return fmt.Errorf("failed to merge processed config %s: %w", path, err)
		}
	}

	return nil
}

// expandEnv 替换字符串中的 ${VAR:default} 占位符
func expandEnv(s string) string {
	// 匹配 ${VAR} 或 ${VAR:default}
	// g1: 变量名, g2: 默认值部分（含冒号）, g3: 默认值内容
	re := regexp.MustCompile(`\${(\w+)(:([^}]*))?}`)
	return re.ReplaceAllStringFunc(s, func(match string) string {
		submatch := re.FindStringSubmatch(match)
		key := submatch[1]
		hasDefault := submatch[2] != ""
		defVal := submatch[3]

		val, ok := os.LookupEnv(key)
		if ok {
			return val
		}
		if hasDefault {
			return defVal
		}
		return match // 原样返回，或者返回空？保留原样以便识别未定义的变量
	})
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
	v.SetDefault("server.grpc.port", 50051)
	v.SetDefault("server.grpc.max_recv_msg_size", 16777216)
	v.SetDefault("server.grpc.max_send_msg_size", 16777216)

	// gRPC 客户端默认值
	v.SetDefault("clients.grpc.dial_timeout", "3s")
	v.SetDefault("clients.grpc.retrieval_service_addr", "localhost:50052")
	v.SetDefault("clients.grpc.story_gen_service_addr", "localhost:50053")
	v.SetDefault("clients.grpc.memory_service_addr", "localhost:50054")
	v.SetDefault("clients.grpc.validator_service_addr", "localhost:50055")

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
	v.SetDefault("observability.metrics.port", 9464)
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
