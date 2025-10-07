package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mrblind/nexus-agent/internal/domain/service"
	"github.com/mrblind/nexus-agent/internal/interface/middleware"
)

// AnalysisHandler 分析相关的HTTP处理器
type AnalysisHandler struct {
	costAnalyzer        service.CostAnalyzer
	performanceAnalyzer service.PerformanceAnalyzer
	promptAnalyzer      service.PromptAnalyzer
}

func NewAnalysisHandler(
	costAnalyzer service.CostAnalyzer,
	performanceAnalyzer service.PerformanceAnalyzer,
	promptAnalyzer service.PromptAnalyzer,
) AnalysisHandler {
	return AnalysisHandler{
		costAnalyzer:        costAnalyzer,
		performanceAnalyzer: performanceAnalyzer,
		promptAnalyzer:      promptAnalyzer,
	}
}

// AnalyzeCost 成本分析接口 - 【代表性接口实现】
// GET /api/v1/analysis/cost?session_id=xxx&start_time=xxx&end_time=xxx
func (h AnalysisHandler) AnalyzeCost(c *gin.Context) {
	log := middleware.FromContext(c.Request.Context())

	sessionID := c.Query("session_id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "SESSION_ID_REQUIRED"})
		return
	}

	startTime := c.Query("start_time")
	endTime := c.Query("end_time")

	var timeRange *service.TimeRange
	if startTime != "" && endTime != "" {
		timeRange = &service.TimeRange{
			StartTime: startTime,
			EndTime:   endTime,
		}
	}

	// 构建分析输入
	input := &service.CostAnalysisInput{
		SessionID: sessionID,
		TimeRange: timeRange,
		GroupBy:   []string{"model", "step_type", "time"},
	}

	// 执行成本分析
	result, err := h.costAnalyzer.Analyze(c.Request.Context(), input)
	if err != nil {
		log.Error().Err(err).Msg("成本分析失败")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "COST_ANALYSIS_FAILED"})
		return
	}

	log.Info().
		Str("session_id", sessionID).
		Float64("total_cost", result.Summary.TotalCost).
		Int("trace_count", result.Summary.TraceCount).
		Msg("成本分析成功")

	c.JSON(http.StatusOK, result)
}

// AnalyzePerformance 性能分析接口
// GET /api/v1/analysis/performance?session_id=xxx
func (h AnalysisHandler) AnalyzePerformance(c *gin.Context) {
	// TODO: 实现性能分析HTTP接口
	c.JSON(http.StatusNotImplemented, gin.H{"error": "NOT_IMPLEMENTED"})
}

// AnalyzePrompt 提示效果分析接口
// GET /api/v1/analysis/prompt?session_id=xxx&prompt_id=xxx
func (h AnalysisHandler) AnalyzePrompt(c *gin.Context) {
	// TODO: 实现提示分析HTTP接口
	c.JSON(http.StatusNotImplemented, gin.H{"error": "NOT_IMPLEMENTED"})
}

// ComparePrompts 对比两个提示
// GET /api/v1/analysis/prompt/compare?prompt_id_1=xxx&prompt_id_2=xxx
func (h AnalysisHandler) ComparePrompts(c *gin.Context) {
	// TODO: 实现提示对比HTTP接口
	c.JSON(http.StatusNotImplemented, gin.H{"error": "NOT_IMPLEMENTED"})
}

// AnalyzeABTest A/B测试分析接口
// GET /api/v1/analysis/abtest/:test_id
func (h AnalysisHandler) AnalyzeABTest(c *gin.Context) {
	// TODO: 实现A/B测试分析HTTP接口
	c.JSON(http.StatusNotImplemented, gin.H{"error": "NOT_IMPLEMENTED"})
}

// GetCostHotspots 获取成本热点
// GET /api/v1/analysis/cost/hotspots?session_id=xxx&top_n=5
func (h AnalysisHandler) GetCostHotspots(c *gin.Context) {
	// TODO: 实现成本热点查询接口
	c.JSON(http.StatusNotImplemented, gin.H{"error": "NOT_IMPLEMENTED"})
}

// GetPerformanceBottlenecks 获取性能瓶颈
// GET /api/v1/analysis/performance/bottlenecks?session_id=xxx
func (h AnalysisHandler) GetPerformanceBottlenecks(c *gin.Context) {
	// TODO: 实现性能瓶颈查询接口
	c.JSON(http.StatusNotImplemented, gin.H{"error": "NOT_IMPLEMENTED"})
}
