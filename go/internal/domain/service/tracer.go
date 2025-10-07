package service

import (
	"context"

	"github.com/mrblind/nexus-agent/internal/domain/model"
)

// 核心追踪接口
type Tracer interface {
	StartTrace(ctx context.Context, sessionID string, agentName string) (*model.Trace, error)
	RecordStep(ctx context.Context, step *model.ExecutionStep) error
	RecordSnapshot(ctx context.Context, traceID string, snapshot *model.Snapshot) error
	EndTrace(ctx context.Context, traceID string, status model.TraceStatus) error
	GetTrace(ctx context.Context, traceID string) (*model.Trace, []*model.ExecutionStep, error)
	EndTraceWithError(ctx context.Context, traceID string, status model.TraceStatus, err error) error
}

// Context钩子函数 - 核心注入机制
type contextKey string

const (
	tracerKey  contextKey = "tracer"
	traceIDKey contextKey = "trace_id"
)

// 注入Tracer到Context
func WithTracer(ctx context.Context, tracer Tracer) context.Context {
	return context.WithValue(ctx, tracerKey, tracer)
}

// 从Context获取Tracer - 自动追踪的关键
func GetTracerFromContext(ctx context.Context) Tracer {
	if tracer, ok := ctx.Value(tracerKey).(Tracer); ok {
		return tracer
	}
	return nil
}

// 注入TraceID到Context
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey, traceID)
}

// 从Context获取TraceID
func GetTraceIDFromContext(ctx context.Context) string {
	if traceID, ok := ctx.Value(traceIDKey).(string); ok {
		return traceID
	}
	return ""
}
