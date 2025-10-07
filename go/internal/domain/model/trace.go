package model

import (
	"time"

	"gorm.io/datatypes"
)

type TraceStatus string

const (
	TraceStatusRunning   TraceStatus = "running"
	TraceStatusCompleted TraceStatus = "completed"
	TraceStatusFailed    TraceStatus = "failed"
	TraceStatusError     TraceStatus = "error"
)

type Trace struct {
	ID         string         `json:"id" gorm:"primaryKey"`
	SessionID  string         `json:"session_id" gorm:"index;not null"`
	AgentName  string         `json:"agent_name" gorm:"not null"`
	Status     TraceStatus    `json:"status" gorm:"not null"`
	StartedAt  time.Time      `json:"started_at" gorm:"not null"`
	EndedAt    *time.Time     `json:"ended_at"`
	CostTokens int            `json:"cost_tokens" gorm:"default:0"`
	CostAPI    float64        `json:"cost_api" gorm:"default:0"`
	Metadata   datatypes.JSON `json:"metadata" gorm:"type:jsonb"`
	CreatedAt  time.Time      `json:"created_at" gorm:"autoCreateTime"`
}

type ExecutionStep struct {
	ID         string         `json:"id" gorm:"primaryKey"`
	TraceID    string         `json:"trace_id" gorm:"index;not null"`
	Sequence   int            `json:"sequence" gorm:"not null"`
	StepType   string         `json:"step_type" gorm:"not null"`
	Input      datatypes.JSON `json:"input" gorm:"type:jsonb"`
	Output     datatypes.JSON `json:"output" gorm:"type:jsonb"`
	CostTokens int            `json:"cost_tokens" gorm:"default:0"`
	CostAPI    float64        `json:"cost_api" gorm:"default:0"`
	LatencyMs  int            `json:"latency_ms" gorm:"default:0"`
	Snapshot   datatypes.JSON `json:"snapshot" gorm:"type:jsonb"` // 改为 JSON 存储
	CreatedAt  time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt  time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
}

type Snapshot struct {
	Stage     string         `json:"stage"` // "pre_llm", "post_llm"
	Timestamp time.Time      `json:"timestamp"`
	Input     datatypes.JSON `json:"input"`   // 该阶段的输入
	Output    datatypes.JSON `json:"output"`  // 该阶段的输出（post阶段才有）
	State     datatypes.JSON `json:"state"`   // Agent内部状态
	Context   datatypes.JSON `json:"context"` // 执行上下文
}
