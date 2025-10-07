package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mrblind/nexus-agent/internal/domain/model"
	"github.com/mrblind/nexus-agent/internal/infrastructure/repository"
	"github.com/mrblind/nexus-agent/internal/interface/dto"
	"github.com/mrblind/nexus-agent/internal/interface/middleware"
)

type TraceHandler struct {
	traceRepo repository.TraceRepository
	stepRepo  repository.StepRepository
}

func NewTraceHandler(traceRepo repository.TraceRepository, stepRepo repository.StepRepository) TraceHandler {
	return TraceHandler{
		traceRepo: traceRepo,
		stepRepo:  stepRepo,
	}
}

func (h TraceHandler) Get(c *gin.Context) {
	log := middleware.FromContext(c.Request.Context())

	traceID := c.Param("id")

	// 获取trace
	trace, err := h.traceRepo.GetByID(c.Request.Context(), traceID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get trace")
		c.JSON(http.StatusNotFound, gin.H{"error": "TRACE_NOT_FOUND"})
		return
	}

	// 获取steps（关键修复）
	steps, err := h.stepRepo.GetByTraceID(c.Request.Context(), traceID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get steps")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "FAILED_TO_GET_STEPS"})
		return
	}

	// 返回完整数据
	response := dto.TraceDetailResponse{
		TraceResponse: toTraceResponse(trace),
		Steps:         steps,
	}

	c.JSON(http.StatusOK, response)
}

// GetSessionTraces 获取session下的所有traces（跨表路由）
// GET /api/v1/sessions/:id/traces
func (h TraceHandler) GetSessionTraces(c *gin.Context) {
	log := middleware.FromContext(c.Request.Context())

	// 注意：这里的param是"id"，因为路由是 /sessions/:id/traces
	sessionID := c.Param("id")
	if sessionID == "" {
		log.Error().Msg("session ID is required")
		c.JSON(http.StatusBadRequest, gin.H{"error": "SESSION_ID_REQUIRED"})
		return
	}

	log.Info().Str("session_id", sessionID).Msg("查询session的所有traces")

	// 通过repository获取traces列表
	traces, err := h.traceRepo.GetBySessionID(c.Request.Context(), sessionID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to list traces")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "FAILED_TO_LIST_TRACES"})
		return
	}

	// 计算总成本
	var totalTokens int
	var totalCost float64
	traceResponses := make([]dto.TraceResponse, 0, len(traces))

	for _, trace := range traces {
		totalTokens += trace.CostTokens
		totalCost += trace.CostAPI
		traceResponses = append(traceResponses, toTraceResponse(trace))
	}

	// 构造响应
	response := dto.TraceListResponse{
		Traces: traceResponses,
		Total:  len(traces),
		TotalCost: dto.CostSummary{
			Tokens:  totalTokens,
			APICost: totalCost,
		},
	}

	log.Info().
		Str("session_id", sessionID).
		Int("traces_count", len(traces)).
		Int("total_tokens", totalTokens).
		Float64("total_cost", totalCost).
		Msg("成功获取traces列表")

	c.JSON(http.StatusOK, response)
}

func toTraceResponse(trace *model.Trace) dto.TraceResponse {
	var metadataMap map[string]interface{}
	if err := json.Unmarshal(trace.Metadata, &metadataMap); err != nil {
		// 处理错误，或者使用空map
		metadataMap = make(map[string]interface{})
	}
	endedAt := ""
	if trace.EndedAt != nil {
		endedAt = trace.EndedAt.Format(time.RFC3339)
	}
	return dto.TraceResponse{
		ID:         trace.ID,
		SessionID:  trace.SessionID,
		AgentName:  trace.AgentName,
		Status:     string(trace.Status),
		StartedAt:  trace.StartedAt.Format(time.RFC3339),
		EndedAt:    endedAt,
		CostTokens: trace.CostTokens,
		CostAPI:    trace.CostAPI,
		Metadata:   metadataMap,
		CreatedAt:  trace.CreatedAt.Format(time.RFC3339),
	}
}
