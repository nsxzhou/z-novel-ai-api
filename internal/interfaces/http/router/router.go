// Package router 提供 HTTP 路由配置
package router

import (
	"z-novel-ai-api/internal/config"
	"z-novel-ai-api/internal/domain/repository"
	"z-novel-ai-api/internal/interfaces/http/handler"
	"z-novel-ai-api/internal/interfaces/http/middleware"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Router HTTP 路由器
type Router struct {
	engine *gin.Engine
	cfg    *config.Config

	// Middleware deps
	rateLimiter  middleware.RateLimiter
	transactor   repository.Transactor
	tenantCtxMgr repository.TenantContextManager

	// Handlers
	authHandler      *handler.AuthHandler
	healthHandler    *handler.HealthHandler
	projectHandler   *handler.ProjectHandler
	volumeHandler    *handler.VolumeHandler
	chapterHandler   *handler.ChapterHandler
	entityHandler    *handler.EntityHandler
	jobHandler       *handler.JobHandler
	retrievalHandler *handler.RetrievalHandler
	streamHandler    *handler.StreamHandler
	userHandler      *handler.UserHandler
	tenantHandler    *handler.TenantHandler
	eventHandler     *handler.EventHandler
	relationHandler  *handler.RelationHandler
}

// RouterHandlers 路由器组件处理器集合
type RouterHandlers struct {
	Auth      *handler.AuthHandler
	Health    *handler.HealthHandler
	Project   *handler.ProjectHandler
	Volume    *handler.VolumeHandler
	Chapter   *handler.ChapterHandler
	Entity    *handler.EntityHandler
	Job       *handler.JobHandler
	Retrieval *handler.RetrievalHandler
	Stream    *handler.StreamHandler
	User      *handler.UserHandler
	Tenant    *handler.TenantHandler
	Event     *handler.EventHandler
	Relation  *handler.RelationHandler

	// Middleware deps
	RateLimiter  middleware.RateLimiter
	Transactor   repository.Transactor
	TenantCtxMgr repository.TenantContextManager
}

// NewWithDeps 创建带依赖的路由器（推荐）
func NewWithDeps(cfg *config.Config, handlers *RouterHandlers) *Router {
	// 设置 Gin 模式
	if cfg.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()

	r := &Router{
		engine:           engine,
		cfg:              cfg,
		rateLimiter:      handlers.RateLimiter,
		transactor:       handlers.Transactor,
		tenantCtxMgr:     handlers.TenantCtxMgr,
		authHandler:      handlers.Auth,
		healthHandler:    handlers.Health,
		projectHandler:   handlers.Project,
		volumeHandler:    handlers.Volume,
		chapterHandler:   handlers.Chapter,
		entityHandler:    handlers.Entity,
		jobHandler:       handlers.Job,
		retrievalHandler: handlers.Retrieval,
		streamHandler:    handlers.Stream,
		userHandler:      handlers.User,
		tenantHandler:    handlers.Tenant,
		eventHandler:     handlers.Event,
		relationHandler:  handlers.Relation,
	}

	r.setupMiddleware()
	r.setupSystemRoutes()
	r.setupBusinessRoutes()

	return r
}

// Engine 返回 Gin Engine
func (r *Router) Engine() *gin.Engine {
	return r.engine
}

// setupMiddleware 配置中间件
func (r *Router) setupMiddleware() {
	// 1. Panic 恢复（最外层）
	r.engine.Use(middleware.Recovery())

	// 2. 分布式追踪
	if r.cfg.Observability.Tracing.Enabled {
		r.engine.Use(middleware.Trace(r.cfg.App.Name))
		r.engine.Use(middleware.TraceContext())
	}

	// 3. 请求 ID
	r.engine.Use(middleware.RequestID())

	// 4. CORS
	r.engine.Use(middleware.CORS(middleware.CORSConfig{
		AllowedOrigins: r.cfg.Security.CORS.AllowedOrigins,
		AllowedMethods: r.cfg.Security.CORS.AllowedMethods,
		AllowedHeaders: r.cfg.Security.CORS.AllowedHeaders,
	}))

	// 5. 指标收集
	if r.cfg.Observability.Metrics.Enabled {
		r.engine.Use(middleware.Metrics())
	}

	// 6. 审计日志
	r.engine.Use(middleware.Audit())
}

// setupSystemRoutes 配置系统路由
func (r *Router) setupSystemRoutes() {
	// 系统端点（不需要认证）
	r.engine.GET("/health", r.healthHandler.Health)
	r.engine.GET("/ready", r.healthHandler.Ready)
	r.engine.GET("/live", r.healthHandler.Live)

	// Prometheus 指标端点
	if r.cfg.Observability.Metrics.Enabled {
		r.engine.GET(r.cfg.Observability.Metrics.Path, gin.WrapH(promhttp.Handler()))
	}
}

// setupBusinessRoutes 配置业务路由
func (r *Router) setupBusinessRoutes() {
	// API v1 路由组
	v1 := r.engine.Group("/v1")

	// 添加认证中间件（全功能模式）
	v1.Use(middleware.Auth(middleware.AuthConfig{
		Secret:    r.cfg.Security.JWT.Secret,
		Issuer:    r.cfg.Security.JWT.Issuer,
		SkipPaths: append(middleware.DefaultSkipPaths, "/v1/auth/login", "/v1/auth/refresh"),
		Enabled:   true,
	}))

	// 添加租户中间件
	v1.Use(middleware.Tenant(middleware.TenantConfig{
		Enabled:         true,
		DefaultTenantID: "default-tenant", // 开发环境默认值
	}))

	// 添加限流中间件 (全功能方案)
	v1.Use(middleware.RateLimit(middleware.RateLimitConfig{
		Enabled:           r.cfg.Security.RateLimit.Enabled,
		RequestsPerSecond: r.cfg.Security.RateLimit.RequestsPerSecond,
		Burst:             r.cfg.Security.RateLimit.Burst,
	}, r.rateLimiter))

	// 通过事务绑定租户上下文（确保 RLS 生效）
	v1.Use(middleware.DBTransaction(r.transactor, r.tenantCtxMgr))

	// 注册业务路由
	RegisterV1Routes(
		v1,
		r.authHandler,
		r.projectHandler,
		r.volumeHandler,
		r.chapterHandler,
		r.entityHandler,
		r.jobHandler,
		r.retrievalHandler,
		r.streamHandler,
		r.userHandler,
		r.tenantHandler,
		r.eventHandler,
		r.relationHandler,
	)
}
