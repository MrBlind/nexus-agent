package service

import (
	"math"
	"sort"
)

// MetricsCalculator 指标计算器 - 可复用的统计计算层
type MetricsCalculator interface {
	// 求和
	Sum(values []float64) float64

	// 平均值
	Avg(values []float64) float64

	// 最大值
	Max(values []float64) float64

	// 最小值
	Min(values []float64) float64

	// 百分位数 (p: 0-100, 如95表示95%分位数)
	Percentile(values []float64, p float64) float64

	// 标准差
	StdDev(values []float64) float64
}

// MetricsCalculatorImpl 指标计算器实现
type MetricsCalculatorImpl struct{}

func NewMetricsCalculator() MetricsCalculator {
	return &MetricsCalculatorImpl{}
}

func (m *MetricsCalculatorImpl) Sum(values []float64) float64 {
	// TODO: 实现求和逻辑
	sum := 0.0
	for _, value := range values {
		sum += value
	}
	return sum
}

func (m *MetricsCalculatorImpl) Avg(values []float64) float64 {
	// TODO: 实现平均值逻辑
	sum := 0.0
	for _, value := range values {
		sum += value
	}
	return sum / float64(len(values))
}

func (m *MetricsCalculatorImpl) Max(values []float64) float64 {
	// TODO: 实现最大值逻辑
	max := values[0]
	for _, value := range values {
		if value > max {
			max = value
		}
	}
	return max
}

func (m *MetricsCalculatorImpl) Min(values []float64) float64 {
	// TODO: 实现最小值逻辑
	min := values[0]
	for _, value := range values {
		if value < min {
			min = value
		}
	}
	return min
}

func (m *MetricsCalculatorImpl) Percentile(values []float64, p float64) float64 {
	// TODO: 实现百分位数逻辑
	sort.Float64s(values)
	return values[int(float64(len(values))*p/100)]
}

func (m *MetricsCalculatorImpl) StdDev(values []float64) float64 {
	// TODO: 实现标准差逻辑
	mean := m.Avg(values)
	sum := 0.0
	for _, value := range values {
		sum += math.Pow(value-mean, 2)
	}
	return math.Sqrt(sum / float64(len(values)))
}
