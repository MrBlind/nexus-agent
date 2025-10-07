package model

import (
	"time"

	"github.com/google/uuid"
)

// Budget contains quota information for a session.
type Budget struct {
	TotalTokens int     `gorm:"column:budget_total_tokens" json:"total_tokens"`
	UsedTokens  int     `gorm:"column:budget_used_tokens" json:"used_tokens"`
	TotalCost   float64 `gorm:"column:budget_total_cost" json:"total_cost"`
	UsedCost    float64 `gorm:"column:budget_used_cost" json:"used_cost"`
}

// Session represents a user session managed by the orchestrator.
type Session struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	UserID    string    `gorm:"column:user_id;not null;index" json:"user_id"`
	Status    string    `gorm:"column:status;not null" json:"status"`
	Budget    Budget    `gorm:"embedded" json:"budget"`
	CreatedAt time.Time `gorm:"column:created_at;not null" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null" json:"updated_at"`
}

// Message stores chat history for a session.
type Message struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	SessionID uuid.UUID `gorm:"type:uuid;index;not null" json:"session_id"`
	Role      string    `gorm:"column:role;not null" json:"role"`
	Content   string    `gorm:"column:content;not null" json:"content"`
	Tokens    int       `gorm:"column:tokens" json:"tokens"`
	CreatedAt time.Time `gorm:"column:created_at;not null" json:"created_at"`
}

// Cost captures cost metrics for an execution trace.
type Cost struct {
	Tokens  int     `gorm:"column:cost_tokens" json:"tokens"`
	APICost float64 `gorm:"column:cost_api" json:"api_cost"`
}

// ExecutionTrace records the lifecycle of an agent execution.
type ExecutionTrace struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	SessionID uuid.UUID `gorm:"type:uuid;index;not null" json:"session_id"`
	Status    string    `gorm:"column:status;not null" json:"status"`
	Cost      Cost      `gorm:"embedded" json:"cost"`
	StartedAt time.Time `gorm:"column:started_at;not null" json:"started_at"`
	EndedAt   time.Time `gorm:"column:ended_at" json:"ended_at"`
}

// LLMProvider represents supported LLM providers.
type LLMProvider string

const (
	LLMProviderOpenAI    LLMProvider = "openai"
	LLMProviderDeepSeek  LLMProvider = "deepseek"
	LLMProviderAnthropic LLMProvider = "anthropic"
	LLMProviderQwen      LLMProvider = "qwen"
	LLMProviderErnie     LLMProvider = "ernie"
	LLMProviderChatGLM   LLMProvider = "chatglm"
)

// LLMRequest represents a request to the LLM service.
type LLMRequest struct {
	SessionID string    `json:"session_id"`
	Messages  []Message `json:"messages"`
	Config    LLMConfig `json:"config"`

	// 核心改进：支持请求级别的模型选择
	Provider string `json:"provider"` // 如: "openai", "deepseek", "anthropic"
	Model    string `json:"model"`    // 如: "gpt-4", "deepseek-chat", "claude-3-sonnet"

	// 可选参数
	Temperature *float64 `json:"temperature,omitempty"`
	MaxTokens   *int     `json:"max_tokens,omitempty"`
	Tools       []string `json:"tools,omitempty"`
}

// LLMConfig represents LLM configuration for requests.
type LLMConfig struct {
	Provider    LLMProvider `json:"provider"`
	APIKey      string      `json:"api_key,omitempty"`
	Model       string      `json:"model"`
	BaseURL     string      `json:"base_url,omitempty"`
	Temperature float64     `json:"temperature"`
	MaxTokens   int         `json:"max_tokens"`
}

// LLMResponse represents a response from the LLM service.
type LLMResponse struct {
	SessionID     string           `json:"session_id"`
	Message       Message          `json:"message"`
	Usage         map[string]int   `json:"usage"`
	Cost          float64          `json:"cost"`
	ExecutionTime float64          `json:"execution_time"`
	ToolCalls     []map[string]any `json:"tool_calls,omitempty"`
}
