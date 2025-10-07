package dto

import "github.com/mrblind/nexus-agent/internal/domain/model"

type TraceResponse struct {
	ID         string                 `json:"id"`
	SessionID  string                 `json:"session_id"`
	AgentName  string                 `json:"agent_name"`
	Status     string                 `json:"status"`
	StartedAt  string                 `json:"started_at"`
	EndedAt    string                 `json:"ended_at"`
	CostTokens int                    `json:"cost_tokens"`
	CostAPI    float64                `json:"cost_api"`
	Metadata   map[string]interface{} `json:"metadata"`
	CreatedAt  string                 `json:"created_at"`
}

// TraceDetailResponse 包含steps的详情响应
type TraceDetailResponse struct {
	TraceResponse                        // 嵌入基础信息
	Steps         []*model.ExecutionStep `json:"steps"`
}

type TraceListResponse struct {
	Traces      []TraceResponse `json:"traces"`
	Total       int             `json:"total"`
	TotalTokens int             `json:"total_tokens"`
	TotalCost   CostSummary     `json:"total_cost"`
}

type CostSummary struct {
	Tokens  int     `json:"tokens"`
	APICost float64 `json:"api_cost"`
}
