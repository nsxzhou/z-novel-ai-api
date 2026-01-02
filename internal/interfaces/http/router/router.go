// Package router 提供 HTTP 路由配置
package router

import (
	"z-novel-ai-api/internal/config"
	"z-novel-ai-api/internal/interfaces/http/handler"
	"z-novel-ai-api/internal/interfaces/http/middleware"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Router HTTP 路由器
type Router struct {
	engine *gin.Engine
	cfg    *config.Config
}

// New 创建新的路由器
func New(cfg *config.Config) *Router {
	// 设置 Gin 模式
	if cfg.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()

	r := &Router{
		engine: engine,
		cfg:    cfg,
	}

	r.setupMiddleware()
	r.setupRoutes()

	return r
}

// Engine 返回 Gin Engine
func (r *Router) Engine() *gin.Engine {
	return r.engine
}

// setupMiddleware 配置中间件
func (r *Router) setupMiddleware() {
	// 基础中间件
	r.engine.Use(middleware.Recovery())
	r.engine.Use(middleware.RequestID())

	// CORS 中间件
	r.engine.Use(middleware.CORS(middleware.CORSConfig{
		AllowedOrigins: r.cfg.Security.CORS.AllowedOrigins,
		AllowedMethods: r.cfg.Security.CORS.AllowedMethods,
		AllowedHeaders: r.cfg.Security.CORS.AllowedHeaders,
	}))

	// 追踪中间件
	if r.cfg.Observability.Tracing.Enabled {
		r.engine.Use(middleware.Trace(r.cfg.App.Name))
		r.engine.Use(middleware.TraceContext())
	}

	// 指标中间件
	if r.cfg.Observability.Metrics.Enabled {
		r.engine.Use(middleware.Metrics())
	}
}

// setupRoutes 配置路由
func (r *Router) setupRoutes() {
	// 健康检查处理器
	healthHandler := handler.NewHealthHandler()

	// 系统端点
	r.engine.GET("/health", healthHandler.Health)
	r.engine.GET("/ready", healthHandler.Ready)
	r.engine.GET("/live", healthHandler.Live)

	// Prometheus 指标端点
	if r.cfg.Observability.Metrics.Enabled {
		r.engine.GET(r.cfg.Observability.Metrics.Path, gin.WrapH(promhttp.Handler()))
	}

	// API v1 路由组
	v1 := r.engine.Group("/v1")
	{
		// 项目相关路由
		projects := v1.Group("/projects")
		{
			_ = projects // TODO: 添加项目相关路由
		}

		// 章节相关路由
		chapters := v1.Group("/chapters")
		{
			_ = chapters // TODO: 添加章节相关路由
		}

		// 实体相关路由
		entities := v1.Group("/entities")
		{
			_ = entities // TODO: 添加实体相关路由
		}

		// 检索调试路由
		retrieval := v1.Group("/retrieval")
		{
			_ = retrieval // TODO: 添加检索相关路由
		}
	}
}
