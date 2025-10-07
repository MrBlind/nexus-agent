package dto

// LLMConfigResponse represents LLM configuration in API responses.
type LLMConfigResponse struct {
	Provider    string  `json:"provider"`
	Model       string  `json:"model"`
	BaseURL     string  `json:"base_url,omitempty"`
	Temperature float64 `json:"temperature"`
	MaxTokens   int     `json:"max_tokens"`
}

// SupportedModelsResponse represents supported models for each provider.
type SupportedModelsResponse struct {
	Providers map[string]ProviderInfo `json:"providers"`
}

// ProviderInfo contains information about a specific LLM provider.
type ProviderInfo struct {
	Name         string   `json:"name"`
	Models       []string `json:"models"`
	DefaultModel string   `json:"default_model"`
	RequiresKey  bool     `json:"requires_key"`
}

// ChatRequest represents a chat request.
type ChatRequest struct {
	Messages    []MessageRequest `json:"messages" binding:"required"`
	Provider    string           `json:"provider,omitempty"`
	Model       string           `json:"model,omitempty"`
	Temperature float64          `json:"temperature,omitempty"`
	MaxTokens   int              `json:"max_tokens,omitempty"`
}

// MessageRequest represents a message in a chat request.
type MessageRequest struct {
	Role    string `json:"role" binding:"required,oneof=user assistant system"`
	Content string `json:"content" binding:"required"`
}

// ChatResponse represents a chat response.
type ChatResponse struct {
	Message       MessageResponse  `json:"message"`
	Usage         map[string]int   `json:"usage"`
	Cost          float64          `json:"cost"`
	ExecutionTime float64          `json:"execution_time"`
	ToolCalls     []map[string]any `json:"tool_calls,omitempty"`
	TraceID       string           `json:"trace_id,omitempty"` // 核心追踪ID
}

// MessageResponse represents a message in a chat response.
type MessageResponse struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
