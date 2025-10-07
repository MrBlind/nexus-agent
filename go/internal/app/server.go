package app

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mrblind/nexus-agent/internal/config"
	"github.com/mrblind/nexus-agent/internal/domain/service"
	"github.com/mrblind/nexus-agent/internal/infrastructure/client"
	"github.com/mrblind/nexus-agent/internal/infrastructure/repository"
	"github.com/mrblind/nexus-agent/internal/interface/handler"
	"github.com/mrblind/nexus-agent/internal/interface/middleware"
	"github.com/mrblind/nexus-agent/pkg/logger"
)

type Server struct {
	config *config.Config
	log    logger.Logger

	pgClient    *client.PostgresClient
	redisClient *client.RedisClient
	llmClient   client.LLMClientInterface

	sessionHandler  handler.SessionHandler
	llmHandler      handler.LLMHandler
	traceHandler    handler.TraceHandler
	analysisHandler handler.AnalysisHandler
}

func New(cfg *config.Config, log logger.Logger) (*Server, error) {
	pg, err := client.NewPostgresClient(cfg.Database, cfg.Server.Debug)
	if err != nil {
		return nil, fmt.Errorf("init postgres: %w", err)
	}

	// 自动迁移数据库表
	log.Info().Msg("开始数据库表迁移...")
	if err := pg.AutoMigrate(); err != nil {
		return nil, fmt.Errorf("database migration failed: %w", err)
	}
	log.Info().Msg("数据库表迁移完成")

	redis, err := client.NewRedisClient(cfg.Redis)
	if err != nil {
		return nil, fmt.Errorf("init redis: %w", err)
	}

	// 使用工厂模式根据配置创建相应的LLM客户端
	factory := client.NewLLMClientFactory()
	llmClient, err := factory.CreateClient(cfg.LLM)
	if err != nil {
		return nil, fmt.Errorf("init llm client: %w", err)
	}

	repo := repository.NewSessionRepository(pg.DB)
	messageRepo := repository.NewMessageRepository(pg.DB)
	traceRepo := repository.NewTraceRepository(pg.DB)
	stepRepo := repository.NewStepRepository(pg.DB)

	sessionService := service.NewSessionService(repo, cfg.Budget)
	messageService := service.NewMessageService(messageRepo)
	llmService := service.NewLLMService(llmClient, cfg.LLM)

	tracer := service.NewTracer(traceRepo, stepRepo)

	// 初始化分析引擎组件
	calculator := service.NewMetricsCalculator()
	aggregator := service.NewDataAggregator(stepRepo)
	costAnalyzer := service.NewCostAnalyzer(traceRepo, stepRepo, calculator, aggregator)
	performanceAnalyzer := service.NewPerformanceAnalyzer(traceRepo, stepRepo, calculator, aggregator)
	promptAnalyzer := service.NewPromptAnalyzer(traceRepo, stepRepo, calculator, aggregator)

	sessionHandler := handler.NewSessionHandler(sessionService)
	llmHandler := handler.NewLLMHandler(llmService, sessionService, messageService, tracer)
	traceHandler := handler.NewTraceHandler(traceRepo, stepRepo)
	analysisHandler := handler.NewAnalysisHandler(costAnalyzer, performanceAnalyzer, promptAnalyzer)

	return &Server{
		config:          cfg,
		log:             log,
		pgClient:        pg,
		redisClient:     redis,
		llmClient:       llmClient,
		sessionHandler:  sessionHandler,
		llmHandler:      llmHandler,
		traceHandler:    traceHandler,
		analysisHandler: analysisHandler,
	}, nil
}

func (s *Server) Run(ctx context.Context) error {
	// 确保在服务器关闭时清理资源
	defer func() {
		if s.llmClient != nil {
			if err := s.llmClient.Close(); err != nil {
				s.log.Error().Err(err).Msg("Failed to close LLM client")
			} else {
				s.log.Info().Msg("LLM client closed successfully")
			}
		}
	}()

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.RequestID())
	router.Use(middleware.Logger(s.log))
	router.Use(middleware.ContextLogger(s.log)) // 添加上下文日志中间件

	v1 := router.Group("/api/v1")

	// Session routes
	sessions := v1.Group("/sessions")
	sessions.POST("", s.sessionHandler.Create)
	sessions.GET("/", s.sessionHandler.GetList)
	sessions.GET("/:id", s.sessionHandler.Get)
	sessions.DELETE("/:id", s.sessionHandler.Delete)
	sessions.POST("/:id/chat", s.llmHandler.Chat)
	sessions.POST("/:id/chat/stream", s.llmHandler.ChatStream)
	sessions.GET("/:id/traces", s.traceHandler.GetSessionTraces)

	// LLM configuration routes
	llm := v1.Group("/llm")
	llm.GET("/models", s.llmHandler.GetSupportedModels)
	llm.GET("/config", s.llmHandler.GetDefaultConfig)

	// Trace routes
	trace := v1.Group("/trace")
	trace.GET("/:id", s.traceHandler.Get)

	// Analysis routes
	analysis := v1.Group("/analysis")
	analysis.GET("/cost", s.analysisHandler.AnalyzeCost)                                  // 成本分析
	analysis.GET("/cost/hotspots", s.analysisHandler.GetCostHotspots)                     // 成本热点
	analysis.GET("/performance", s.analysisHandler.AnalyzePerformance)                    // 性能分析
	analysis.GET("/performance/bottlenecks", s.analysisHandler.GetPerformanceBottlenecks) // 性能瓶颈
	analysis.GET("/prompt", s.analysisHandler.AnalyzePrompt)                              // 提示效果分析
	analysis.GET("/prompt/compare", s.analysisHandler.ComparePrompts)                     // 提示对比
	analysis.GET("/abtest/:test_id", s.analysisHandler.AnalyzeABTest)                     // A/B测试分析

	addr := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  time.Duration(s.config.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(s.config.Server.WriteTimeout) * time.Second,
	}

	s.log.Info().Msgf("server running at %s", addr)

	go func() {
		<-ctx.Done()
		s.log.Info().Msg("Received shutdown signal, closing server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			s.log.Error().Err(err).Msg("Server shutdown error")
		}
	}()

	return server.ListenAndServe()
}
