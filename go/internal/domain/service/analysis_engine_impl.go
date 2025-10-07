package service

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/mrblind/nexus-agent/internal/domain/model"
	"github.com/mrblind/nexus-agent/internal/infrastructure/repository"
)

// AnalysisEngineImpl 分析引擎实现
type AnalysisEngineImpl struct {
	traceRepo repository.TraceRepository
	stepRepo  repository.StepRepository
}

// NewAnalysisEngine 创建分析引擎实例
func NewAnalysisEngine(traceRepo repository.TraceRepository, stepRepo repository.StepRepository) AnalysisEngine {
	return &AnalysisEngineImpl{
		traceRepo: traceRepo,
		stepRepo:  stepRepo,
	}
}

// AnalyzeCost 成本分析 - 核心功能实现
func (a *AnalysisEngineImpl) AnalyzeCost(ctx context.Context, sessionID string, timeRange *TimeRange) (*CostAnalysis, error) {
	// 1. 获取会话的所有追踪数据
	traces, err := a.traceRepo.GetBySessionID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get traces: %w", err)
	}

	if len(traces) == 0 {
		return &CostAnalysis{
			SessionID: sessionID,
			TimeRange: timeRange,
		}, nil
	}

	// 2. 成本聚合计算
	costMetrics := a.calculateCostMetrics(traces)

	// 3. 按步骤类型分组成本
	costByStep, err := a.calculateCostByStepType(ctx, traces)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate cost by step: %w", err)
	}

	// 4. 按模型分组成本
	costByModel := a.calculateCostByModel(traces)

	// 5. 生成成本趋势数据
	costTrend := a.generateCostTrend(traces)

	return &CostAnalysis{
		SessionID:   sessionID,
		TimeRange:   timeRange,
		TotalCost:   costMetrics.total,
		AvgCost:     costMetrics.average,
		MaxCost:     costMetrics.maximum,
		CostByStep:  costByStep,
		CostByModel: costByModel,
		CostTrend:   costTrend,
	}, nil
}

// 成本指标结构
type costMetrics struct {
	total   float64
	average float64
	maximum float64
}

// calculateCostMetrics 计算基础成本指标
func (a *AnalysisEngineImpl) calculateCostMetrics(traces []*model.Trace) costMetrics {
	var total, max float64

	for _, trace := range traces {
		total += trace.CostAPI
		if trace.CostAPI > max {
			max = trace.CostAPI
		}
	}

	average := total / float64(len(traces))

	return costMetrics{
		total:   total,
		average: average,
		maximum: max,
	}
}

// calculateCostByStepType 按步骤类型计算成本分布
func (a *AnalysisEngineImpl) calculateCostByStepType(ctx context.Context, traces []*model.Trace) (map[string]float64, error) {
	costByStep := make(map[string]float64)

	for _, trace := range traces {
		steps, err := a.stepRepo.GetByTraceID(ctx, trace.ID)
		if err != nil {
			continue // 跳过错误，不中断整个分析
		}

		for _, step := range steps {
			// 基于token数量估算步骤成本
			stepCost := a.estimateStepCost(step)
			costByStep[step.StepType] += stepCost
		}
	}

	return costByStep, nil
}

// calculateCostByModel 按模型计算成本分布
func (a *AnalysisEngineImpl) calculateCostByModel(traces []*model.Trace) map[string]float64 {
	costByModel := make(map[string]float64)

	for _, trace := range traces {
		// 从metadata中提取模型信息
		model := a.extractModelFromTrace(trace)
		costByModel[model] += trace.CostAPI
	}

	return costByModel
}

// generateCostTrend 生成成本趋势数据
func (a *AnalysisEngineImpl) generateCostTrend(traces []*model.Trace) []*CostDataPoint {
	// 按时间排序
	sort.Slice(traces, func(i, j int) bool {
		return traces[i].StartedAt.Before(traces[j].StartedAt)
	})

	var trend []*CostDataPoint
	var cumulativeCost float64

	for _, trace := range traces {
		cumulativeCost += trace.CostAPI
		trend = append(trend, &CostDataPoint{
			Timestamp: trace.StartedAt.Format(time.RFC3339),
			Cost:      cumulativeCost,
		})
	}

	return trend
}

// estimateStepCost 估算单个步骤的成本
func (a *AnalysisEngineImpl) estimateStepCost(step *model.ExecutionStep) float64 {
	// 简化的成本估算：基于token数量
	// 实际项目中可以根据具体模型的定价来计算
	const avgCostPerToken = 0.00002 // 假设平均每token 0.00002美元
	return float64(step.CostTokens) * avgCostPerToken
}

// extractModelFromTrace 从追踪中提取模型信息
func (a *AnalysisEngineImpl) extractModelFromTrace(trace *model.Trace) string {
	// 从metadata中解析模型信息
	// 这里简化处理，实际需要解析JSON
	if trace.AgentName != "" {
		return trace.AgentName
	}
	return "unknown"
}

// 其他接口方法的占位符实现
func (a *AnalysisEngineImpl) AnalyzeTrace(ctx context.Context, traceID string) (*TraceAnalysis, error) {
	return nil, fmt.Errorf("not implemented yet")
}

func (a *AnalysisEngineImpl) AnalyzeSession(ctx context.Context, sessionID string) (*SessionAnalysis, error) {
	return nil, fmt.Errorf("not implemented yet")
}

func (a *AnalysisEngineImpl) CompareTraces(ctx context.Context, traceID1, traceID2 string) (*TraceComparison, error) {
	return nil, fmt.Errorf("not implemented yet")
}

func (a *AnalysisEngineImpl) GenerateOptimizationSuggestions(ctx context.Context, traceID string) ([]*OptimizationSuggestion, error) {
	return nil, fmt.Errorf("not implemented yet")
}

func (a *AnalysisEngineImpl) AnalyzePerformance(ctx context.Context, sessionID string) (*PerformanceAnalysis, error) {
	return nil, fmt.Errorf("not implemented yet")
}
