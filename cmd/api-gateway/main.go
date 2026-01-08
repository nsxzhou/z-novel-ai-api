// Package main API Gateway 服务入口
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"z-novel-ai-api/internal/config"
	einoobs "z-novel-ai-api/internal/observability/eino"
	"z-novel-ai-api/internal/wire"
	"z-novel-ai-api/pkg/logger"
	"z-novel-ai-api/pkg/tracer"

	"github.com/joho/godotenv"
)

// Version 版本信息，构建时注入
var (
	Version   = "dev"
	BuildTime = "unknown"
)

func main() {
	// 加载 .env 文件（如果存在）
	_ = godotenv.Load()

	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// 初始化日志
	logger.Init(
		cfg.Observability.Logging.Level,
		cfg.Observability.Logging.Format,
	)

	ctx := context.Background()
	log := logger.FromContext(ctx)
	log.Info("starting api-gateway",
		"version", Version,
		"build_time", BuildTime,
		"env", cfg.App.Env,
	)

	// 初始化追踪
	shutdown, err := tracer.Init(ctx, tracer.Config{
		ServiceName: cfg.App.Name,
		Endpoint:    cfg.Observability.Tracing.Endpoint,
		SampleRate:  cfg.Observability.Tracing.SampleRate,
		Enabled:     cfg.Observability.Tracing.Enabled,
	})
	if err != nil {
		log.Error("failed to init tracer", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := shutdown(ctx); err != nil {
			log.Error("failed to shutdown tracer", "error", err)
		}
	}()

	// 初始化 Eino 全局 callbacks（指标/追踪/日志）
	einoobs.Init()

	// 初始化应用（使用 Wire 注入）
	app, cleanupApp, err := wire.InitializeApp(ctx, cfg)
	if err != nil {
		logger.Fatal(ctx, "failed to initialize app", err)
	}
	defer cleanupApp()

	// 获取引擎
	r := app.Engine()

	// 创建 HTTP 服务器
	addr := fmt.Sprintf("%s:%d", cfg.Server.HTTP.Host, cfg.Server.HTTP.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  cfg.Server.HTTP.ReadTimeout,
		WriteTimeout: cfg.Server.HTTP.WriteTimeout,
		IdleTimeout:  cfg.Server.HTTP.IdleTimeout,
	}

	// 启动服务器
	go func() {
		log.Info("http server starting", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("http server error", "error", err)
			os.Exit(1)
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down server...")

	// 优雅关闭
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("server forced to shutdown", "error", err)
	}

	log.Info("server exited")
}
