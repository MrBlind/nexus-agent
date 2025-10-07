package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/mrblind/nexus-agent/internal/app"
	"github.com/mrblind/nexus-agent/internal/config"
	"github.com/mrblind/nexus-agent/pkg/logger"
)

func main() {
	fmt.Println("🚀 启动 Nexus Agent 服务器...")
	
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	
	fmt.Printf("✅ 配置加载成功，数据库用户: %s\n", cfg.Database.User)
	fmt.Println("🔧 开始初始化日志器...")

	// Initialize logger with rotation
	logLevel := cfg.Observability.LogLevel
	if cfg.Server.Debug {
		logLevel = "debug"
	}
	
	// 创建日志目录路径 (格式: logs/2025010112)
	logDate := time.Now().Format("2006010215") // YYYYMMDDHH
	logDir := filepath.Join("..", "logs", logDate)
	
	// 使用带轮转功能的日志器
	appLogger, err := logger.NewWithRotation(logLevel, logDir, "nexus-agent-go", 100) // 100MB 轮转
	if err != nil {
		// 如果日志轮转失败，回退到标准日志器
		fmt.Printf("⚠️ 日志轮转初始化失败，使用标准日志器: %v\n", err)
		appLogger = logger.New(logLevel)
	} else {
		fmt.Printf("📁 日志文件: %s/nexus-agent-go.log\n", logDir)
	}
	
	fmt.Println("📱 日志器初始化完成，开始初始化应用...")

	// Initialize application
	server, err := app.New(cfg, appLogger)
	if err != nil {
		appLogger.Fatal().Err(err).Msg("Failed to initialize server")
	}

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		appLogger.Info().Msg("Shutting down gracefully...")
		cancel()
	}()

	// Start server
	if err := server.Run(ctx); err != nil {
		appLogger.Fatal().Err(err).Msg("Server failed")
	}
}