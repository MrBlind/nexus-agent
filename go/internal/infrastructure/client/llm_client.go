package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/mrblind/nexus-agent/internal/domain/model"
)

// LLMClient handles communication with the Python LLM service.
type LLMClient struct {
	baseURL string
	client  *http.Client
}

// NewLLMClient creates a new LLM service client.
func NewLLMClient(baseURL string) *LLMClient {
	return &LLMClient{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// ExecuteAgent sends a request to the Python LLM service.
func (c *LLMClient) ExecuteAgent(ctx context.Context, req *model.LLMRequest) (*model.LLMResponse, error) {
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/execute", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("LLM service returned status %d", resp.StatusCode)
	}

	var llmResp model.LLMResponse
	if err := json.NewDecoder(resp.Body).Decode(&llmResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &llmResp, nil
}

// GetSupportedModels returns the list of supported models for each provider.
func (c *LLMClient) GetSupportedModels(ctx context.Context) (map[string][]string, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/models", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("LLM service returned status %d", resp.StatusCode)
	}

	var models map[string][]string
	if err := json.NewDecoder(resp.Body).Decode(&models); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return models, nil
}

// ExecuteAgentStream 执行流式请求 (HTTP客户端暂不支持，返回错误)
func (c *LLMClient) ExecuteAgentStream(ctx context.Context, req *model.LLMRequest, callback StreamCallback) error {
	return fmt.Errorf("HTTP client does not support streaming, please use gRPC client")
}

// Close 关闭HTTP客户端 (无需特殊处理)
func (c *LLMClient) Close() error {
	// HTTP 客户端会自动管理连接池，通常不需要显式关闭
	// 但为了接口一致性，我们提供这个方法
	return nil
}
