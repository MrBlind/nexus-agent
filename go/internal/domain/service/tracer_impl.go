package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/mrblind/nexus-agent/internal/domain/model"
	"github.com/mrblind/nexus-agent/internal/infrastructure/repository"
	"gorm.io/datatypes"
)

type TracerImplService struct {
	traceRepo repository.TraceRepository
	stepRepo  repository.StepRepository
}

func NewTracer(traceRepo repository.TraceRepository, stepRepo repository.StepRepository) Tracer {
	return &TracerImplService{
		traceRepo: traceRepo,
		stepRepo:  stepRepo,
	}
}

// 核心功能1: 开始追踪 - 创建trace记录
func (t *TracerImplService) StartTrace(ctx context.Context, sessionID string, agentName string) (*model.Trace, error) {
	trace := &model.Trace{
		ID:         uuid.New().String(),
		SessionID:  sessionID,
		AgentName:  agentName,
		Status:     model.TraceStatusRunning,
		StartedAt:  time.Now(),
		CostTokens: 0,
		CostAPI:    0,
		Metadata:   datatypes.JSON{},
		CreatedAt:  time.Now(),
	}

	if err := t.traceRepo.Create(ctx, trace); err != nil {
		return nil, fmt.Errorf("failed to create trace: %w", err)
	}

	return trace, nil
}

// 核心功能2: 记录执行步骤 - 捕获每个操作
func (t *TracerImplService) RecordStep(ctx context.Context, step *model.ExecutionStep) error {
	if step.ID == "" {
		step.ID = uuid.New().String()
	}
	if step.TraceID == "" {
		return fmt.Errorf("traceID is required")
	}

	now := time.Now()
	step.CreatedAt = now
	step.UpdatedAt = now

	// 自动设置序列号
	if step.Sequence == 0 {
		sequence, err := t.getNextSequence(ctx, step.TraceID)
		if err != nil {
			return fmt.Errorf("failed to get sequence: %w", err)
		}
		step.Sequence = sequence
	}

	if err := t.stepRepo.Create(ctx, step); err != nil {
		return fmt.Errorf("failed to record step: %w", err)
	}

	// 更新trace的成本统计（累加）
	if step.CostTokens > 0 || step.CostAPI > 0 {
		if err := t.traceRepo.UpdateCost(ctx, step.TraceID, step.CostTokens, step.CostAPI); err != nil {
			return fmt.Errorf("failed to update trace cost: %w", err)
		}
	}

	return nil
}

// 获取下一个序列号
func (t *TracerImplService) getNextSequence(ctx context.Context, traceID string) (int, error) {
	steps, err := t.stepRepo.GetByTraceID(ctx, traceID)
	if err != nil {
		return 0, err
	}
	return len(steps) + 1, nil
}

// 核心功能3: 记录快照 - 保存完整状态用于重放（包含输入、输出、状态和上下文）
func (t *TracerImplService) RecordSnapshot(ctx context.Context, traceID string, snapshot *model.Snapshot) error {
	if traceID == "" {
		return fmt.Errorf("traceID is required")
	}
	if snapshot == nil {
		return fmt.Errorf("snapshot is required")
	}

	snapshot.Timestamp = time.Now()

	// 将快照序列化为 JSON
	snapshotJSON, err := json.Marshal(snapshot)
	if err != nil {
		return fmt.Errorf("failed to marshal snapshot: %w", err)
	}

	// 快照作为特殊步骤记录
	step := &model.ExecutionStep{
		ID:        uuid.New().String(),
		TraceID:   traceID,
		StepType:  "snapshot",
		Snapshot:  datatypes.JSON(snapshotJSON),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return t.stepRepo.Create(ctx, step)
}

// 核心功能4: 结束追踪 - 标记完成状态
func (t *TracerImplService) EndTrace(ctx context.Context, traceID string, status model.TraceStatus) error {
	endTime := time.Now()
	return t.traceRepo.UpdateStatus(ctx, traceID, status, endTime)
}

// 核心功能5: 获取完整追踪数据
func (t *TracerImplService) GetTrace(ctx context.Context, traceID string) (*model.Trace, []*model.ExecutionStep, error) {
	trace, err := t.traceRepo.GetByID(ctx, traceID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get trace: %w", err)
	}

	steps, err := t.stepRepo.GetByTraceID(ctx, traceID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get steps: %w", err)
	}

	return trace, steps, nil
}

func (t *TracerImplService) EndTraceWithError(ctx context.Context, traceID string, status model.TraceStatus, err error) error {
	return t.traceRepo.UpdateStatusWithError(ctx, traceID, status, time.Now(), err)
}
