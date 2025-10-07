package service

import (
	"context"

	"github.com/mrblind/nexus-agent/internal/domain/model"
)

// AnalysisEngine 分析引擎接口
// 负责对追踪数据进行深度分析，提供优化建议和性能洞察
type AnalysisEngine interface {
	// 分析单个追踪的执行情况
	AnalyzeTrace(ctx context.Context, traceID string) (*TraceAnalysis, error)

	// 分析会话中的所有追踪
	AnalyzeSession(ctx context.Context, sessionID string) (*SessionAnalysis, error)

	// 对比两个追踪的执行差异
	CompareTraces(ctx context.Context, traceID1, traceID2 string) (*TraceComparison, error)

	// 分析成本使用情况
	AnalyzeCost(ctx context.Context, sessionID string, timeRange *TimeRange) (*CostAnalysis, error)

	// 生成优化建议
	GenerateOptimizationSuggestions(ctx context.Context, traceID string) ([]*OptimizationSuggestion, error)

	// 分析性能瓶颈
	AnalyzePerformance(ctx context.Context, sessionID string) (*PerformanceAnalysis, error)
}

// 追踪分析结果
type TraceAnalysis struct {
	TraceID      string                    `json:"trace_id"`
	SessionID    string                    `json:"session_id"`
	Status       model.TraceStatus         `json:"status"`
	TotalSteps   int                       `json:"total_steps"`
	TotalCost    float64                   `json:"total_cost"`
	TotalTokens  int                       `json:"total_tokens"`
	TotalLatency int                       `json:"total_latency"`
	StepDetails  []*StepAnalysis           `json:"step_details"`
	Bottlenecks  []*Bottleneck             `json:"bottlenecks"`
	Suggestions  []*OptimizationSuggestion `json:"suggestions"`
}

// 步骤分析
type StepAnalysis struct {
	StepID      string  `json:"step_id"`
	StepType    string  `json:"step_type"`
	Sequence    int     `json:"sequence"`
	Cost        float64 `json:"cost"`
	Tokens      int     `json:"tokens"`
	LatencyMs   int     `json:"latency_ms"`
	SuccessRate float64 `json:"success_rate"` // 基于历史数据的成功率
}

// 会话分析结果
type SessionAnalysis struct {
	SessionID        string           `json:"session_id"`
	TotalTraces      int              `json:"total_traces"`
	SuccessfulRuns   int              `json:"successful_runs"`
	FailedRuns       int              `json:"failed_runs"`
	TotalCost        float64          `json:"total_cost"`
	TotalTokens      int              `json:"total_tokens"`
	AvgLatency       float64          `json:"avg_latency"`
	CostTrend        []*CostDataPoint `json:"cost_trend"`
	PerformanceTrend []*PerfDataPoint `json:"performance_trend"`
}

// 追踪对比结果
type TraceComparison struct {
	TraceID1    string        `json:"trace_id_1"`
	TraceID2    string        `json:"trace_id_2"`
	Similarity  float64       `json:"similarity"` // 0-1的相似度分数
	Differences []*Difference `json:"differences"`
	CostDiff    float64       `json:"cost_diff"`
	LatencyDiff int           `json:"latency_diff"`
	TokenDiff   int           `json:"token_diff"`
}

// 成本分析结果
type CostAnalysis struct {
	SessionID   string             `json:"session_id"`
	TimeRange   *TimeRange         `json:"time_range"`
	TotalCost   float64            `json:"total_cost"`
	AvgCost     float64            `json:"avg_cost"`
	MaxCost     float64            `json:"max_cost"`
	CostByStep  map[string]float64 `json:"cost_by_step"`
	CostByModel map[string]float64 `json:"cost_by_model"`
	CostTrend   []*CostDataPoint   `json:"cost_trend"`
}

// 性能分析结果
type PerformanceAnalysis struct {
	SessionID     string         `json:"session_id"`
	AvgLatency    float64        `json:"avg_latency"`
	MaxLatency    int            `json:"max_latency"`
	LatencyByStep map[string]int `json:"latency_by_step"`
	Bottlenecks   []*Bottleneck  `json:"bottlenecks"`
	Throughput    float64        `json:"throughput"` // 请求/秒
}

// 优化建议
type OptimizationSuggestion struct {
	Type        string  `json:"type"` // "model_selection", "prompt_optimization", "tool_usage"
	Description string  `json:"description"`
	Impact      float64 `json:"impact"`     // 预期改进程度 (0-1)
	Confidence  float64 `json:"confidence"` // 建议置信度 (0-1)
	Details     string  `json:"details"`
}

// 性能瓶颈
type Bottleneck struct {
	StepType    string `json:"step_type"`
	StepID      string `json:"step_id"`
	Description string `json:"description"`
	Impact      string `json:"impact"` // "high", "medium", "low"
}

// 时间范围
type TimeRange struct {
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
}

// 成本数据点
type CostDataPoint struct {
	Timestamp string  `json:"timestamp"`
	Cost      float64 `json:"cost"`
}

// 性能数据点
type PerfDataPoint struct {
	Timestamp string `json:"timestamp"`
	Latency   int    `json:"latency"`
}

// 差异详情
type Difference struct {
	Field        string      `json:"field"`
	Value1       interface{} `json:"value_1"`
	Value2       interface{} `json:"value_2"`
	Significance string      `json:"significance"` // "major", "minor"
}
