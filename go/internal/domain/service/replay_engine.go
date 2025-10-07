package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mrblind/nexus-agent/internal/domain/model"
)

// 重放引擎核心接口
type ReplayEngine interface {
	// 核心功能：从快照重放
	ReplayFromSnapshot(ctx context.Context, traceID string, snapshotIndex int) (*ReplayResult, error)
	// 确定性重放：使用相同输入
	ReplayDeterministic(ctx context.Context, traceID string) (*ReplayResult, error)
}

type ReplayEngineImpl struct {
	tracer    Tracer
	agentExec AgentExecutor
}

// 核心重放逻辑：状态恢复 + 确定性执行
func (r *ReplayEngineImpl) ReplayFromSnapshot(ctx context.Context, traceID string, snapshotIndex int) (*ReplayResult, error) {
	// 1. 加载原始trace和所有快照
	originalTrace, steps, err := r.tracer.GetTrace(ctx, traceID)
	if err != nil {
		return nil, fmt.Errorf("failed to load trace: %w", err)
	}

	// 2. 找到指定快照
	var targetSnapshot *model.Snapshot
	for _, step := range steps {
		if step.StepType == "snapshot" && step.Sequence == snapshotIndex {
			// 从 JSON 字段解析快照
			var snapshot model.Snapshot
			if err := json.Unmarshal(step.Snapshot, &snapshot); err != nil {
				return nil, fmt.Errorf("failed to unmarshal snapshot: %w", err)
			}
			targetSnapshot = &snapshot
			break
		}
	}

	if targetSnapshot == nil {
		return nil, fmt.Errorf("snapshot %d not found", snapshotIndex)
	}

	// 3. 核心：恢复状态并重放
	newTrace, err := r.tracer.StartTrace(ctx, originalTrace.SessionID, originalTrace.AgentName+"_replay")
	if err != nil {
		return nil, err
	}

	// 4. 从快照状态开始执行
	// 关键：使用相同的输入，确保确定性
	replayCtx := WithTraceID(WithTracer(ctx, r.tracer), newTrace.ID)

	// 恢复状态到快照点
	var restoredState map[string]interface{}
	if err := json.Unmarshal(targetSnapshot.State, &restoredState); err != nil {
		return nil, fmt.Errorf("failed to unmarshal snapshot state: %w", err)
	}

	// 从快照点继续执行剩余步骤
	remainingSteps := steps[snapshotIndex+1:]
	for _, step := range remainingSteps {
		// 使用原始输入重新执行
		err := r.replayStep(replayCtx, step, restoredState)
		if err != nil {
			return nil, fmt.Errorf("replay step failed: %w", err)
		}
	}

	r.tracer.EndTrace(replayCtx, newTrace.ID, model.TraceStatusCompleted)

	return &ReplayResult{
		OriginalTraceID: traceID,
		ReplayTraceID:   newTrace.ID,
		Status:          "success",
	}, nil
}

// 重放单个步骤
func (r *ReplayEngineImpl) replayStep(ctx context.Context, originalStep *model.ExecutionStep, state map[string]interface{}) error {
	switch originalStep.StepType {
	case "llm_call":
		// 关键：使用原始输入，但可能得到不同输出
		// 这就是重放的价值：发现非确定性差异
		return r.replayLLMCall(ctx, originalStep, state)
	case "tool_call":
		return r.replayToolCall(ctx, originalStep, state)
	}
	return nil
}

func (r *ReplayEngineImpl) replayLLMCall(ctx context.Context, step *model.ExecutionStep, state map[string]interface{}) error {
	// 使用相同输入调用LLM，记录新的输出
	// 这里会自动通过Context中的Tracer记录新的执行步骤
	return nil
}

func (r *ReplayEngineImpl) replayToolCall(ctx context.Context, step *model.ExecutionStep, state map[string]interface{}) error {
	// 重放工具调用
	return nil
}

type ReplayResult struct {
	OriginalTraceID string
	ReplayTraceID   string
	Status          string
	Differences     []*ReplayDifference
}

type ReplayDifference struct {
	Step          int
	Field         string
	OriginalValue interface{}
	ReplayValue   interface{}
}

// Agent执行器接口
type AgentExecutor interface {
	ExecuteFromState(ctx context.Context, agentName string, state map[string]interface{}) error
}
