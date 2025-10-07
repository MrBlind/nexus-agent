package service

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/mrblind/nexus-agent/internal/domain/model"
	"github.com/mrblind/nexus-agent/internal/infrastructure/repository"
)

// CostAnalyzer 成本分析器
type CostAnalyzer interface {
	// 分析会话成本
	Analyze(ctx context.Context, input *CostAnalysisInput) (*CostAnalysisResult, error)

	// 识别成本热点
	IdentifyHotspots(ctx context.Context, traces []*model.Trace, topN int) []*CostHotspot

	// 生成成本优化建议
	// GenerateSuggestions(ctx context.Context, analysis *CostAnalysisResult) []*CostSuggestion
}

// CostAnalysisInput 成本分析输入
type CostAnalysisInput struct {
	SessionID string
	TimeRange *TimeRange      // 使用 analysis_engine.go 中定义的 TimeRange
	GroupBy   []string        // ["model", "step_type", "time"]
	Filters   *FilterCriteria // 使用 data_aggregator.go 中定义的 FilterCriteria
}

// CostAnalysisResult 成本分析结果 - 【代表性接口实现】
type CostAnalysisResult struct {
	SessionID   string            `json:"session_id"`
	Summary     *CostSummary      `json:"summary"`
	Breakdown   *CostBreakdown    `json:"breakdown"`
	Trends      *CostTrend        `json:"trends"`
	Hotspots    []*CostHotspot    `json:"hotspots"`
	Suggestions []*CostSuggestion `json:"suggestions"`
}

// CostSummary 成本总览
type CostSummary struct {
	TotalCost    float64 `json:"total_cost"`
	AvgCost      float64 `json:"avg_cost"`
	MaxCost      float64 `json:"max_cost"`
	MinCost      float64 `json:"min_cost"`
	TotalTokens  int     `json:"total_tokens"`
	TraceCount   int     `json:"trace_count"`
	CostPerToken float64 `json:"cost_per_token"`
}

// CostBreakdown 成本分解
type CostBreakdown struct {
	ByModel    map[string]float64 `json:"by_model"`
	ByStepType map[string]float64 `json:"by_step_type"`
	ByTime     map[string]float64 `json:"by_time"`
}

// CostTrend 成本趋势
type CostTrend struct {
	DataPoints   []*CostDataPoint `json:"data_points"`
	GrowthRate   float64          `json:"growth_rate"` // 增长率
	IsIncreasing bool             `json:"is_increasing"`
}

// CostHotspot 成本热点
type CostHotspot struct {
	TraceID string  `json:"trace_id"`
	Cost    float64 `json:"cost"`
	Tokens  int     `json:"tokens"`
	Reason  string  `json:"reason"`
	Impact  string  `json:"impact"` // "high", "medium", "low"
}

// CostSuggestion 成本优化建议
type CostSuggestion struct {
	Type             string   `json:"type"` // "model_downgrade", "prompt_optimization", "caching"
	Description      string   `json:"description"`
	PotentialSavings float64  `json:"potential_savings"`
	Confidence       float64  `json:"confidence"` // 0-1
	ActionableSteps  []string `json:"actionable_steps"`
}

// CostAnalyzerImpl 成本分析器实现
type CostAnalyzerImpl struct {
	traceRepo  repository.TraceRepository
	stepRepo   repository.StepRepository
	calculator MetricsCalculator
	aggregator DataAggregator
}

func NewCostAnalyzer(
	traceRepo repository.TraceRepository,
	stepRepo repository.StepRepository,
	calculator MetricsCalculator,
	aggregator DataAggregator,
) CostAnalyzer {
	return &CostAnalyzerImpl{
		traceRepo:  traceRepo,
		stepRepo:   stepRepo,
		calculator: calculator,
		aggregator: aggregator,
	}
}

// Analyze 分析会话成本
func (c *CostAnalyzerImpl) Analyze(ctx context.Context, input *CostAnalysisInput) (*CostAnalysisResult, error) {
	// 1. 获取traces
	traces, err := c.traceRepo.GetBySessionID(ctx, input.SessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get traces: %w", err)
	}

	if len(traces) == 0 {
		return &CostAnalysisResult{
			SessionID: input.SessionID,
			Summary:   &CostSummary{},
		}, nil
	}

	// 2. 过滤数据
	if input.Filters != nil {
		traces = c.aggregator.FilterTraces(ctx, traces, input.Filters)
	}

	// 3. 计算成本总览
	summary := c.calculateSummary(traces)

	// 4. 成本分解
	breakdown := c.calculateBreakdown(ctx, traces, input.GroupBy)

	// 5. 趋势分析
	trends := c.analyzeTrends(traces)

	// 6. 识别热点
	hotspots := c.IdentifyHotspots(ctx, traces, 5)

	// 7. 生成建议
	result := &CostAnalysisResult{
		SessionID:   input.SessionID,
		Summary:     summary,
		Breakdown:   breakdown,
		Trends:      trends,
		Hotspots:    hotspots,
		Suggestions: []*CostSuggestion{},
	}
	// suggestions := c.GenerateSuggestions(ctx, result)
	// result.Suggestions = suggestions

	return result, nil
}

// calculateSummary 计算成本总览
func (c *CostAnalyzerImpl) calculateSummary(traces []*model.Trace) *CostSummary {
	costs := make([]float64, len(traces))
	var totalTokens int

	for i, trace := range traces {
		costs[i] = trace.CostAPI
		totalTokens += trace.CostTokens
	}

	totalCost := c.calculator.Sum(costs)
	avgCost := c.calculator.Avg(costs)
	maxCost := c.calculator.Max(costs)
	minCost := c.calculator.Min(costs)

	var costPerToken float64
	if totalTokens > 0 {
		costPerToken = totalCost / float64(totalTokens)
	}

	return &CostSummary{
		TotalCost:    totalCost,
		AvgCost:      avgCost,
		MaxCost:      maxCost,
		MinCost:      minCost,
		TotalTokens:  totalTokens,
		TraceCount:   len(traces),
		CostPerToken: costPerToken,
	}
}

// calculateBreakdown 计算成本分解
func (c *CostAnalyzerImpl) calculateBreakdown(ctx context.Context, traces []*model.Trace, groupBy []string) *CostBreakdown {
	breakdown := &CostBreakdown{
		ByModel:    make(map[string]float64),
		ByStepType: make(map[string]float64),
		ByTime:     make(map[string]float64),
	}

	// 按模型分组
	for _, trace := range traces {
		model := extractModelFromMetadata(trace)
		breakdown.ByModel[model] += trace.CostAPI
	}

	// 按步骤类型分组
	for _, trace := range traces {
		steps, err := c.stepRepo.GetByTraceID(ctx, trace.ID)
		if err != nil {
			continue
		}
		for _, step := range steps {
			stepCost := estimateStepCost(step)
			breakdown.ByStepType[step.StepType] += stepCost
		}
	}

	// 按时间分组 (按天)
	for _, trace := range traces {
		day := trace.StartedAt.Format("2006-01-02")
		breakdown.ByTime[day] += trace.CostAPI
	}

	return breakdown
}

// analyzeTrends 分析成本趋势
func (c *CostAnalyzerImpl) analyzeTrends(traces []*model.Trace) *CostTrend {
	// 按时间排序
	sort.Slice(traces, func(i, j int) bool {
		return traces[i].StartedAt.Before(traces[j].StartedAt)
	})

	var dataPoints []*CostDataPoint
	var cumulativeCost float64

	for _, trace := range traces {
		cumulativeCost += trace.CostAPI
		dataPoints = append(dataPoints, &CostDataPoint{
			Timestamp: trace.StartedAt.Format(time.RFC3339),
			Cost:      cumulativeCost,
		})
	}

	// 计算增长率 (简化版)
	growthRate := 0.0
	isIncreasing := false
	if len(traces) > 1 {
		firstCost := traces[0].CostAPI
		lastCost := traces[len(traces)-1].CostAPI
		if firstCost > 0 {
			growthRate = (lastCost - firstCost) / firstCost
			isIncreasing = lastCost > firstCost
		}
	}

	return &CostTrend{
		DataPoints:   dataPoints,
		GrowthRate:   growthRate,
		IsIncreasing: isIncreasing,
	}
}

// IdentifyHotspots 识别成本热点
func (c *CostAnalyzerImpl) IdentifyHotspots(ctx context.Context, traces []*model.Trace, topN int) []*CostHotspot {
	// 按成本排序
	sort.Slice(traces, func(i, j int) bool {
		return traces[i].CostAPI > traces[j].CostAPI
	})

	var hotspots []*CostHotspot
	for i := 0; i < topN && i < len(traces); i++ {
		trace := traces[i]

		// 判断成本影响级别
		impact := "low"
		if trace.CostAPI > 1.0 {
			impact = "high"
		} else if trace.CostAPI > 0.1 {
			impact = "medium"
		}

		reason := fmt.Sprintf("使用了%s模型", extractModelFromMetadata(trace))

		hotspots = append(hotspots, &CostHotspot{
			TraceID: trace.ID,
			Cost:    trace.CostAPI,
			Tokens:  trace.CostTokens,
			Reason:  reason,
			Impact:  impact,
		})
	}

	return hotspots
}

// GenerateSuggestions 生成成本优化建议
// func (c *CostAnalyzerImpl) GenerateSuggestions(ctx context.Context, analysis *CostAnalysisResult) []*CostSuggestion {
// 	var suggestions []*CostSuggestion

// 	return suggestions
// }

// 辅助函数
func extractModelFromMetadata(trace *model.Trace) string {
	// TODO: 从metadata中解析模型信息
	if trace.AgentName != "" {
		return trace.AgentName
	}
	return "unknown"
}

func estimateStepCost(step *model.ExecutionStep) float64 {
	const avgCostPerToken = 0.00002
	return float64(step.CostTokens) * avgCostPerToken
}
