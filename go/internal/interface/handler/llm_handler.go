package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mrblind/nexus-agent/internal/domain/model"
	"github.com/mrblind/nexus-agent/internal/domain/service"
	"github.com/mrblind/nexus-agent/internal/infrastructure/client"
	"github.com/mrblind/nexus-agent/internal/interface/dto"
	"github.com/mrblind/nexus-agent/internal/interface/middleware"
	"gorm.io/datatypes"
)

// LLMHandler handles LLM-related HTTP requests.
type LLMHandler struct {
	llmService     *service.LLMService
	sessionService *service.SessionService
	messageService *service.MessageService
	tracer         service.Tracer // 核心追踪器注入
}

// StreamTimeoutConfig 流式超时配置
type StreamTimeoutConfig struct {
	InitialTimeout  time.Duration // 初始超时时间
	ActivityTimeout time.Duration // 活动超时时间（无数据流动时的超时）
	MaxTotalTimeout time.Duration // 绝对最大超时时间
}

// ActivityTracker 活动跟踪器
type ActivityTracker struct {
	mu           sync.RWMutex
	lastActivity time.Time
	totalStart   time.Time
	config       StreamTimeoutConfig
	ctx          context.Context
	cancel       context.CancelFunc
}

// NewActivityTracker 创建新的活动跟踪器
func NewActivityTracker(config StreamTimeoutConfig) *ActivityTracker {
	ctx, cancel := context.WithCancel(context.Background())
	now := time.Now()

	return &ActivityTracker{
		lastActivity: now,
		totalStart:   now,
		config:       config,
		ctx:          ctx,
		cancel:       cancel,
	}
}

// UpdateActivity 更新活动时间
func (at *ActivityTracker) UpdateActivity() {
	at.mu.Lock()
	defer at.mu.Unlock()
	at.lastActivity = time.Now()
}

// Context 返回跟踪器的 context
func (at *ActivityTracker) Context() context.Context {
	return at.ctx
}

// StartMonitoring 开始监控超时
func (at *ActivityTracker) StartMonitoring() {
	go func() {
		ticker := time.NewTicker(1 * time.Second) // 每秒检查一次
		defer ticker.Stop()

		for {
			select {
			case <-at.ctx.Done():
				return
			case <-ticker.C:
				if at.shouldTimeout() {
					at.cancel() // 触发超时
					return
				}
			}
		}
	}()
}

// shouldTimeout 检查是否应该超时
func (at *ActivityTracker) shouldTimeout() bool {
	at.mu.RLock()
	defer at.mu.RUnlock()

	now := time.Now()
	timeSinceLastActivity := now.Sub(at.lastActivity)
	totalTime := now.Sub(at.totalStart)

	// 检查活动超时（无数据流动）
	if timeSinceLastActivity > at.config.ActivityTimeout {
		return true
	}

	// 检查绝对最大超时
	if totalTime > at.config.MaxTotalTimeout {
		return true
	}

	return false
}

// Stop 停止跟踪器
func (at *ActivityTracker) Stop() {
	at.cancel()
}

// getDefaultStreamTimeoutConfig 获取默认的流式超时配置
func getDefaultStreamTimeoutConfig() StreamTimeoutConfig {
	return StreamTimeoutConfig{
		InitialTimeout:  2 * time.Minute,  // 初始等待2分钟
		ActivityTimeout: 30 * time.Second, // 30秒无活动则超时
		MaxTotalTimeout: 30 * time.Minute, // 绝对最大30分钟
	}
}

// NewLLMHandler creates a new LLM handler.
func NewLLMHandler(
	llmService *service.LLMService,
	sessionService *service.SessionService,
	messageService *service.MessageService,
	tracer service.Tracer,
) LLMHandler {
	return LLMHandler{
		llmService:     llmService,
		sessionService: sessionService,
		messageService: messageService,
		tracer:         tracer,
	}
}

// GetSupportedModels returns the list of supported models.
func (h LLMHandler) GetSupportedModels(c *gin.Context) {
	log := middleware.FromContext(c.Request.Context())
	log.Info().Msg("获取支持的模型列表")

	models, err := h.llmService.GetSupportedModels(c.Request.Context())
	if err != nil {
		log.Error().Err(err).Msg("获取支持的模型失败")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "GET_MODELS_FAILED", "message": err.Error()})
		return
	}

	log.Info().Int("providers_count", len(models)).Msg("成功获取支持的模型")

	// Transform to response format
	providers := make(map[string]dto.ProviderInfo)
	for provider, modelList := range models {
		var defaultModel string
		if len(modelList) > 0 {
			defaultModel = modelList[0]
		}

		providers[provider] = dto.ProviderInfo{
			Name:         provider,
			Models:       modelList,
			DefaultModel: defaultModel,
			RequiresKey:  true, // All providers require API keys for now
		}
	}

	response := dto.SupportedModelsResponse{
		Providers: providers,
	}

	c.JSON(http.StatusOK, response)
}

// GetDefaultConfig returns the default LLM configuration.
func (h LLMHandler) GetDefaultConfig(c *gin.Context) {
	log := middleware.FromContext(c.Request.Context())
	log.Info().Msg("获取默认 LLM 配置")

	config := h.llmService.GetDefaultConfig()
	response := toLLMConfigResponse(config)

	log.Info().
		Str("provider", string(config.Provider)).
		Str("model", config.Model).
		Msg("成功获取默认配置")

	c.JSON(http.StatusOK, response)
}

// Chat handles chat requests for a specific session.
func (h LLMHandler) Chat(c *gin.Context) {
	log := middleware.FromContext(c.Request.Context())

	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		log.Error().Err(err).Msg("无效的会话ID")
		c.JSON(http.StatusBadRequest, gin.H{"error": "INVALID_SESSION_ID"})
		return
	}

	log.Info().Str("session_id", sessionID.String()).Msg("处理聊天请求")

	// 创建基于活动的动态超时跟踪器
	timeoutConfig := getDefaultStreamTimeoutConfig()
	activityTracker := NewActivityTracker(timeoutConfig)
	defer activityTracker.Stop()

	go func() {
		<-c.Request.Context().Done()
		activityTracker.Stop() // 如果客户端断开，停止跟踪器
	}()

	// 核心钩子1: 启动追踪
	var trace *model.Trace
	ctx := service.WithTracer(c.Request.Context(), h.tracer)
	trace, err = h.tracer.StartTrace(ctx, sessionID.String(), "simple_chat")
	if err != nil {
		log.Error().Err(err).Msg("启动追踪失败")
		// 不阻断业务，继续执行
	} else {
		// 填充metadata
		metadata := map[string]interface{}{
			"user_agent":    c.GetHeader("User-Agent"),
			"ip_address":    c.ClientIP(),
			"request_id":    c.GetHeader("X-Request-ID"),
			"source":        "http_api",
			"agent_version": "v1.0.0",
		}
		metadataJSON, err := json.Marshal(metadata)
		if err != nil {
			log.Error().Err(err).Msg("Failed to marshal metadata")
			metadataJSON = []byte("{}")
		}
		trace.Metadata = datatypes.JSON(metadataJSON)
		ctx = service.WithTraceID(ctx, trace.ID)
		log.Info().Str("trace_id", trace.ID).Msg("追踪已启动")
	}

	// 禁用此请求的写超时
	if rc := http.NewResponseController(c.Writer); rc != nil {
		rc.SetWriteDeadline(time.Time{}) // 清除写超时
	}

	// Verify session exists
	// session放内存或者redis中进行校验可能会比较好
	_, err = h.sessionService.Get(c.Request.Context(), sessionID)
	if err != nil {
		log.Error().Err(err).Str("session_id", sessionID.String()).Msg("会话不存在")
		c.JSON(http.StatusNotFound, gin.H{"error": "SESSION_NOT_FOUND"})
		return
	}

	var req dto.ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error().Err(err).Msg("无效的请求参数")
		c.JSON(http.StatusBadRequest, gin.H{"error": "INVALID_INPUT", "message": err.Error()})
		return
	}

	log.Info().
		Str("session_id", sessionID.String()).
		Str("provider", req.Provider).
		Str("model", req.Model).
		Int("messages_count", len(req.Messages)).
		Msg("开始执行聊天")

	// Convert DTO messages to domain messages
	messages := make([]model.Message, len(req.Messages))
	for i, msg := range req.Messages {
		messages[i] = model.Message{
			ID:        uuid.New(),
			SessionID: sessionID,
			Role:      msg.Role,
			Content:   msg.Content,
		}
	}

	// Build LLM config from request
	var llmConfig *model.LLMConfig
	if req.Provider != "" || req.Model != "" {
		llmConfig = &model.LLMConfig{
			Provider:    model.LLMProvider(req.Provider),
			Model:       req.Model,
			Temperature: req.Temperature,
			MaxTokens:   req.MaxTokens,
		}

		// Validate config
		if err := h.llmService.ValidateConfig(llmConfig); err != nil {
			log.Error().Err(err).
				Str("provider", string(llmConfig.Provider)).
				Str("model", llmConfig.Model).
				Msg("LLM 配置验证失败")
			c.JSON(http.StatusBadRequest, gin.H{"error": "INVALID_CONFIG", "message": err.Error()})
			return
		}

		log.Info().
			Str("provider", string(llmConfig.Provider)).
			Str("model", llmConfig.Model).
			Msg("LLM 配置验证成功")
	}

	// 核心钩子2: 记录pre_llm快照
	if trace != nil {
		stateData := map[string]interface{}{
			"messages_count": len(messages),
			"model":          llmConfig.Model,
			"provider":       llmConfig.Provider,
		}
		stateJSON, err := json.Marshal(stateData)
		if err != nil {
			log.Error().Err(err).Msg("Failed to marshal snapshot state")
		} else {
			// 序列化输入数据
			inputJSON, err := json.Marshal(messages)
			if err != nil {
				log.Error().Err(err).Msg("Failed to marshal input messages")
				inputJSON = []byte("{}")
			}

			// 序列化上下文数据（LLM配置）
			contextData := map[string]interface{}{
				"llm_config": llmConfig,
				"session_id": sessionID.String(),
			}
			contextJSON, err := json.Marshal(contextData)
			if err != nil {
				log.Error().Err(err).Msg("Failed to marshal context")
				contextJSON = []byte("{}")
			}

			h.tracer.RecordSnapshot(ctx, trace.ID, &model.Snapshot{
				Stage:   "pre_llm",
				Input:   datatypes.JSON(inputJSON),
				State:   datatypes.JSON(stateJSON),
				Context: datatypes.JSON(contextJSON),
			})
		}
	}

	// Execute chat
	startTime := time.Now()
	response, err := h.llmService.ExecuteChat(ctx, sessionID, messages, llmConfig)
	latency := time.Since(startTime)

	if err != nil {
		// 核心钩子3: 记录失败
		if trace != nil {
			log.Info().Msgf("结束追踪: %+v", trace)
			h.tracer.EndTrace(ctx, trace.ID, model.TraceStatusFailed)
		}
		log.Error().Err(err).
			Str("session_id", sessionID.String()).
			Msg("聊天执行失败")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "CHAT_EXECUTION_FAILED", "message": err.Error()})
		return
	}

	// 核心钩子4: 记录执行步骤
	if trace != nil {
		// 序列化输入数据
		inputJSON, err := json.Marshal(messages)
		if err != nil {
			log.Error().Err(err).Msg("Failed to marshal input messages")
			inputJSON = []byte("{}")
		}

		// 序列化输出数据
		outputJSON, err := json.Marshal(response)
		if err != nil {
			log.Error().Err(err).Msg("Failed to marshal response")
			outputJSON = []byte("{}")
		}

		step := &model.ExecutionStep{
			ID:         uuid.New().String(),
			TraceID:    trace.ID,
			Sequence:   1,
			StepType:   "llm_call",
			Input:      datatypes.JSON(inputJSON),
			Output:     datatypes.JSON(outputJSON),
			CostTokens: response.Usage["total_tokens"],
			CostAPI:    response.Cost,
			LatencyMs:  int(latency.Milliseconds()),
		}
		h.tracer.RecordStep(ctx, step)

		// 核心钩子5: 记录post_llm快照（包含完整的输出和更新后状态）
		stateData := map[string]interface{}{
			"messages_count": len(messages) + 1, // 包含AI响应
			"model":          llmConfig.Model,
			"provider":       llmConfig.Provider,
			"total_tokens":   response.Usage["total_tokens"],
			"cost":           response.Cost,
		}
		stateJSON, err := json.Marshal(stateData)
		if err != nil {
			log.Error().Err(err).Msg("Failed to marshal post-llm state")
		} else {
			// 序列化上下文数据
			contextData := map[string]interface{}{
				"llm_config": llmConfig,
				"session_id": sessionID.String(),
				"latency_ms": latency.Milliseconds(),
			}
			contextJSON, err := json.Marshal(contextData)
			if err != nil {
				log.Error().Err(err).Msg("Failed to marshal context")
				contextJSON = []byte("{}")
			}

			h.tracer.RecordSnapshot(ctx, trace.ID, &model.Snapshot{
				Stage:   "post_llm",
				Input:   datatypes.JSON(inputJSON),
				Output:  datatypes.JSON(outputJSON),
				State:   datatypes.JSON(stateJSON),
				Context: datatypes.JSON(contextJSON),
			})
		}

		// 结束追踪
		log.Info().Msgf("结束追踪: %+v", trace)
		h.tracer.EndTrace(ctx, trace.ID, model.TraceStatusCompleted)
	}

	log.Info().
		Str("session_id", sessionID.String()).
		Float64("cost", response.Cost).
		Float64("execution_time", response.ExecutionTime).
		Int("total_tokens", response.Usage["total_tokens"]).
		Msg("聊天执行成功")

	// save user messages
	for i := range messages {
		messages[i].CreatedAt = time.Now()
		if err := h.messageService.SaveMessage(c.Request.Context(), &messages[i]); err != nil {
			log.Error().Err(err).Msg("保存消息失败")
		}
	}

	assistantMsg := &model.Message{
		ID:        uuid.New(),
		SessionID: sessionID,
		Role:      response.Message.Role,
		Content:   response.Message.Content,
		Tokens:    response.Usage["total_tokens"],
		CreatedAt: time.Now(),
	}

	if err := h.messageService.SaveMessage(c.Request.Context(), assistantMsg); err != nil {
		log.Error().Err(err).Msg("保存消息失败")
	}

	if err := h.sessionService.UpdateBudget(c.Request.Context(), sessionID, response.Usage["total_tokens"], response.Cost); err != nil {
		log.Error().Err(err).Msg("更新会话预算失败")
	}

	// 核心钩子5: 在响应中返回trace_id
	chatResponse := dto.ChatResponse{
		Message: dto.MessageResponse{
			Role:    response.Message.Role,
			Content: response.Message.Content,
		},
		Usage:         response.Usage,
		Cost:          response.Cost,
		ExecutionTime: response.ExecutionTime,
		ToolCalls:     response.ToolCalls,
	}

	// 如果有追踪，添加trace_id到响应
	if trace != nil {
		chatResponse.TraceID = trace.ID
	}

	c.JSON(http.StatusOK, chatResponse)
}

// ChatStream handles streaming chat requests for a specific session.
func (h LLMHandler) ChatStream(c *gin.Context) {
	log := middleware.FromContext(c.Request.Context())

	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		log.Error().Err(err).Msg("无效的会话ID")
		c.JSON(http.StatusBadRequest, gin.H{"error": "INVALID_SESSION_ID"})
		return
	}

	log.Info().Str("session_id", sessionID.String()).Msg("处理流式聊天请求")

	// 核心钩子1: 启动追踪
	var trace *model.Trace
	ctx := service.WithTracer(c.Request.Context(), h.tracer)
	trace, err = h.tracer.StartTrace(ctx, sessionID.String(), "stream_chat")
	if err != nil {
		log.Error().Err(err).Msg("启动追踪失败")
		// 不阻断业务，继续执行
	} else {
		// 填充metadata
		metadata := map[string]interface{}{
			"user_agent":    c.GetHeader("User-Agent"),
			"ip_address":    c.ClientIP(),
			"request_id":    c.GetHeader("X-Request-ID"),
			"source":        "http_api_stream",
			"agent_version": "v1.0.0",
		}
		metadataJSON, err := json.Marshal(metadata)
		if err != nil {
			log.Error().Err(err).Msg("Failed to marshal metadata")
			metadataJSON = []byte("{}")
		}
		trace.Metadata = datatypes.JSON(metadataJSON)
		ctx = service.WithTraceID(ctx, trace.ID)
		log.Info().Str("trace_id", trace.ID).Msg("流式追踪已启动")
	}

	// Verify session exists
	_, err = h.sessionService.Get(c.Request.Context(), sessionID)
	if err != nil {
		log.Error().Err(err).Str("session_id", sessionID.String()).Msg("会话不存在")
		c.JSON(http.StatusNotFound, gin.H{"error": "SESSION_NOT_FOUND"})
		return
	}

	var req dto.ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error().Err(err).Msg("无效的请求参数")
		c.JSON(http.StatusBadRequest, gin.H{"error": "INVALID_INPUT", "message": err.Error()})
		return
	}

	log.Info().
		Str("session_id", sessionID.String()).
		Str("provider", req.Provider).
		Str("model", req.Model).
		Int("messages_count", len(req.Messages)).
		Msg("开始执行流式聊天")

	// Convert DTO messages to domain messages
	messages := make([]model.Message, len(req.Messages))
	for i, msg := range req.Messages {
		messages[i] = model.Message{
			ID:        uuid.New(),
			SessionID: sessionID,
			Role:      msg.Role,
			Content:   msg.Content,
		}
	}

	// Build LLM config from request
	var llmConfig *model.LLMConfig
	if req.Provider != "" || req.Model != "" {
		llmConfig = &model.LLMConfig{
			Provider:    model.LLMProvider(req.Provider),
			Model:       req.Model,
			Temperature: req.Temperature,
			MaxTokens:   req.MaxTokens,
		}

		// Validate config
		if err := h.llmService.ValidateConfig(llmConfig); err != nil {
			log.Error().Err(err).
				Str("provider", string(llmConfig.Provider)).
				Str("model", llmConfig.Model).
				Msg("LLM 配置验证失败")
			c.JSON(http.StatusBadRequest, gin.H{"error": "INVALID_CONFIG", "message": err.Error()})
			return
		}

		log.Info().
			Str("provider", string(llmConfig.Provider)).
			Str("model", llmConfig.Model).
			Msg("LLM 配置验证成功")
	}

	// 核心钩子2: 记录pre_llm快照
	if trace != nil {
		stateData := map[string]interface{}{
			"messages_count": len(messages),
			"model":          llmConfig.Model,
			"provider":       llmConfig.Provider,
		}
		stateJSON, err := json.Marshal(stateData)
		if err != nil {
			log.Error().Err(err).Msg("Failed to marshal snapshot state")
		} else {
			// 序列化输入数据
			inputJSON, err := json.Marshal(messages)
			if err != nil {
				log.Error().Err(err).Msg("Failed to marshal input messages")
				inputJSON = []byte("{}")
			}

			// 序列化上下文数据（LLM配置）
			contextData := map[string]interface{}{
				"llm_config": llmConfig,
				"session_id": sessionID.String(),
			}
			contextJSON, err := json.Marshal(contextData)
			if err != nil {
				log.Error().Err(err).Msg("Failed to marshal context")
				contextJSON = []byte("{}")
			}

			h.tracer.RecordSnapshot(ctx, trace.ID, &model.Snapshot{
				Stage:   "pre_llm",
				Input:   datatypes.JSON(inputJSON),
				State:   datatypes.JSON(stateJSON),
				Context: datatypes.JSON(contextJSON),
			})
		}
	}

	// 禁用此请求的写超时
	if rc := http.NewResponseController(c.Writer); rc != nil {
		rc.SetWriteDeadline(time.Time{}) // 清除写超时
	}

	// Set up SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Headers", "Cache-Control")

	// Create a flusher to send data immediately
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		log.Error().Msg("Streaming unsupported")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "STREAMING_UNSUPPORTED"})
		return
	}

	// 创建基于活动的动态超时跟踪器
	timeoutConfig := getDefaultStreamTimeoutConfig()
	activityTracker := NewActivityTracker(timeoutConfig)
	defer activityTracker.Stop()

	log.Info().
		Str("session_id", sessionID.String()).
		Dur("activity_timeout", timeoutConfig.ActivityTimeout).
		Dur("max_total_timeout", timeoutConfig.MaxTotalTimeout).
		Msg("启动活动超时跟踪器")

	// 开始监控活动超时
	activityTracker.StartMonitoring()

	// 监听客户端断开连接
	go func() {
		<-c.Request.Context().Done()
		activityTracker.Stop() // 如果客户端断开，停止跟踪器
	}()

	// Track accumulated data for session update
	var totalCost float64
	var totalTokens int
	var fullContent string

	// Define callback for streaming data
	callback := func(chunk *client.StreamChunk) error {
		// 更新活动时间 - 每次收到数据都重置活动超时
		activityTracker.UpdateActivity()

		// Create SSE event
		var eventData map[string]interface{}
		// log.Debug().Str("type", chunk.Type).Msg("流式聊天数据")

		switch chunk.Type {
		case "content_delta":
			fullContent += chunk.ContentDelta
			eventData = map[string]interface{}{
				"type":    "content_delta",
				"content": chunk.ContentDelta,
			}

		case "tool_call":
			eventData = map[string]interface{}{
				"type":      "tool_call",
				"tool_call": chunk.ToolCall,
			}

		case "usage_update":
			if chunk.Usage != nil {
				totalTokens = chunk.Usage["total_tokens"]
			}
			eventData = map[string]interface{}{
				"type":  "usage_update",
				"usage": chunk.Usage,
			}

		case "final_response":
			log.Info().Msgf("final_response: %+v", chunk)
			totalCost = chunk.Cost
			if chunk.Usage != nil {
				totalTokens = chunk.Usage["total_tokens"]
			}

			// 核心钩子4: 在final_response中记录执行步骤
			if trace != nil {
				// 序列化输入数据
				inputJSON, err := json.Marshal(messages)
				if err != nil {
					log.Error().Err(err).Msg("Failed to marshal input messages")
					inputJSON = []byte("{}")
				}

				// 构造流式响应对象
				streamResponse := map[string]interface{}{
					"message": map[string]interface{}{
						"role":    "assistant",
						"content": fullContent,
					},
					"usage":          chunk.Usage,
					"cost":           chunk.Cost,
					"execution_time": chunk.ExecTime,
				}

				// 序列化输出数据
				outputJSON, err := json.Marshal(streamResponse)
				if err != nil {
					log.Error().Err(err).Msg("Failed to marshal response")
					outputJSON = []byte("{}")
				}

				// 调试日志：检查成本数据
				log.Info().Msgf("准备记录ExecutionStep - totalTokens: %d, totalCost: %f, chunk.Cost: %f",
					totalTokens, totalCost, chunk.Cost)

				step := &model.ExecutionStep{
					ID:         uuid.New().String(),
					TraceID:    trace.ID,
					Sequence:   1,
					StepType:   "llm_call_stream",
					Input:      datatypes.JSON(inputJSON),
					Output:     datatypes.JSON(outputJSON),
					CostTokens: totalTokens,
					CostAPI:    totalCost,
					LatencyMs:  int(chunk.ExecTime * 1000), // chunk.ExecTime 是秒，转换为毫秒
				}
				h.tracer.RecordStep(ctx, step)
			}

			eventData = map[string]interface{}{
				"type":           "final_response",
				"usage":          chunk.Usage,
				"cost":           chunk.Cost,
				"execution_time": chunk.ExecTime,
			}

		case "error":
			eventData = map[string]interface{}{
				"type":  "error",
				"error": chunk.Error,
			}
		}

		// Convert to JSON
		jsonData, err := json.Marshal(eventData)
		if err != nil {
			log.Error().Err(err).Msg("Failed to marshal stream data")
			return err
		}

		// Send SSE event
		_, err = fmt.Fprintf(c.Writer, "data: %s\n\n", string(jsonData))
		if err != nil {
			log.Error().Err(err).Msg("Failed to write stream data")
			return err
		}

		flusher.Flush()
		return nil
	}

	// Execute streaming chat
	startTime := time.Now()
	err = h.llmService.ExecuteChatStream(activityTracker.Context(), sessionID, messages, llmConfig, callback)
	latency := time.Since(startTime)
	if err != nil {
		// 检查是否是活动超时
		var errorMessage string
		if err == context.Canceled {
			// 检查是活动超时还是客户端断开
			select {
			case <-activityTracker.Context().Done():
				// 活动超时
				if activityTracker.shouldTimeout() {
					at := activityTracker
					at.mu.RLock()
					timeSinceLastActivity := time.Since(at.lastActivity)
					totalTime := time.Since(at.totalStart)
					at.mu.RUnlock()

					if timeSinceLastActivity > at.config.ActivityTimeout {
						errorMessage = fmt.Sprintf("流式响应活动超时：%v 秒无数据传输", int(timeSinceLastActivity.Seconds()))
					} else if totalTime > at.config.MaxTotalTimeout {
						errorMessage = fmt.Sprintf("流式响应总时长超时：已运行 %v 分钟", int(totalTime.Minutes()))
					} else {
						errorMessage = "流式响应超时"
					}
				} else {
					errorMessage = "客户端断开连接"
				}
			case <-c.Request.Context().Done():
				errorMessage = "客户端断开连接"
			default:
				errorMessage = err.Error()
			}
		} else {
			errorMessage = err.Error()
		}

		log.Error().Err(err).
			Str("session_id", sessionID.String()).
			Str("error_type", errorMessage).
			Msg("流式聊天执行失败")

		// Send error event
		errorData := map[string]interface{}{
			"type":  "error",
			"error": errorMessage,
		}
		jsonData, _ := json.Marshal(errorData)
		fmt.Fprintf(c.Writer, "data: %s\n\n", string(jsonData))
		flusher.Flush()
		return
	}

	assistantMsg := &model.Message{
		ID:        uuid.New(),
		SessionID: sessionID,
		Role:      "assistant",
		Content:   fullContent,
		Tokens:    totalTokens,
		CreatedAt: time.Now(),
	}

	if err := h.messageService.SaveMessage(c.Request.Context(), assistantMsg); err != nil {
		log.Error().Err(err).Msg("保存消息失败")
	}

	// Update session budget
	if err := h.sessionService.UpdateBudget(c.Request.Context(), sessionID, totalTokens, totalCost); err != nil {
		log.Error().Err(err).Msg("更新会话预算失败")
	}

	log.Info().
		Str("session_id", sessionID.String()).
		Float64("cost", totalCost).
		Int("total_tokens", totalTokens).
		Msg("流式聊天执行成功")

	// 核心钩子5: 记录post_llm快照
	if trace != nil {
		// 序列化输入数据
		inputJSON, err := json.Marshal(messages)
		if err != nil {
			log.Error().Err(err).Msg("Failed to marshal input messages")
			inputJSON = []byte("{}")
		}

		// 构造最终响应对象
		// finalResponse := map[string]interface{}{
		// 	"message": map[string]interface{}{
		// 		"role":    "assistant",
		// 		"content": fullContent,
		// 	},
		// 	"usage": map[string]int{
		// 		"total_tokens": totalTokens,
		// 	},
		// 	"cost":           totalCost,
		// 	"execution_time": latency.Seconds(),
		// }

		// 序列化输出数据
		outputJSON, err := json.Marshal(fullContent)
		if err != nil {
			log.Error().Err(err).Msg("Failed to marshal response")
			outputJSON = []byte("{}")
		}

		stateData := map[string]interface{}{
			"messages_count": len(messages) + 1, // 包含AI响应
			"model":          llmConfig.Model,
			"provider":       llmConfig.Provider,
			"total_tokens":   totalTokens,
			"cost":           totalCost,
		}
		stateJSON, err := json.Marshal(stateData)
		if err != nil {
			log.Error().Err(err).Msg("Failed to marshal post-llm state")
		} else {
			// 序列化上下文数据
			contextData := map[string]interface{}{
				"llm_config": llmConfig,
				"session_id": sessionID.String(),
				"latency_ms": latency.Milliseconds(),
			}
			contextJSON, err := json.Marshal(contextData)
			if err != nil {
				log.Error().Err(err).Msg("Failed to marshal context")
				contextJSON = []byte("{}")
			}

			h.tracer.RecordSnapshot(ctx, trace.ID, &model.Snapshot{
				Stage:   "post_llm",
				Input:   datatypes.JSON(inputJSON),
				Output:  datatypes.JSON(outputJSON),
				State:   datatypes.JSON(stateJSON),
				Context: datatypes.JSON(contextJSON),
			})
		}

		// 结束追踪
		log.Info().Msgf("chat stream 结束追踪: %+v", trace)
		h.tracer.EndTrace(ctx, trace.ID, model.TraceStatusCompleted)
	}

	// Send final done event
	doneData := map[string]interface{}{
		"type": "done",
	}
	jsonData, _ := json.Marshal(doneData)
	fmt.Fprintf(c.Writer, "data: %s\n\n", string(jsonData))
	flusher.Flush()
}

// toLLMConfigResponse converts domain LLMConfig to DTO response.
func toLLMConfigResponse(config *model.LLMConfig) dto.LLMConfigResponse {
	return dto.LLMConfigResponse{
		Provider:    string(config.Provider),
		Model:       config.Model,
		BaseURL:     config.BaseURL,
		Temperature: config.Temperature,
		MaxTokens:   config.MaxTokens,
	}
}
