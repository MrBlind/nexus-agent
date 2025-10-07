package client

import (
	"context"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mrblind/nexus-agent/internal/domain/model"
	llmv1 "github.com/mrblind/nexus-agent/proto"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"
)

// GRPCLLMClient 是基于 gRPC 的 LLM 客户端
type GRPCLLMClient struct {
	conn   *grpc.ClientConn
	client llmv1.LLMServiceClient
	target string

	// 轻量级并发安全方案
	closeOnce sync.Once   // 确保关闭操作只执行一次
	closed    atomic.Bool // 原子布尔值，快速检查关闭状态
}

// NewGRPCLLMClient 创建新的 gRPC LLM 客户端
func NewGRPCLLMClient(target string) (*GRPCLLMClient, error) {
	// 创建 gRPC 连接 - 配置长连接和禁用超时
	kacp := keepalive.ClientParameters{
		Time:                30 * time.Second, // 每30秒发送keepalive ping
		Timeout:             5 * time.Second,  // 等待keepalive ping响应的超时时间
		PermitWithoutStream: false,            // 禁止在没有活动流时发送keepalive ping
	}

	conn, err := grpc.NewClient(
		target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(kacp),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to gRPC server: %w", err)
	}

	client := &GRPCLLMClient{
		conn:   conn,
		client: llmv1.NewLLMServiceClient(conn),
		target: target,
	}

	// 注意: 这里不使用 defer conn.Close() 是正确的设计选择
	// 因为连接需要在整个客户端生命周期内保持活跃
	// 调用方负责在适当的时候调用 client.Close()

	return client, nil
}

// getConnection 快速检查并获取 gRPC 连接
func (c *GRPCLLMClient) getConnection() (llmv1.LLMServiceClient, error) {
	// 使用原子操作进行快速检查，无锁操作
	if c.closed.Load() {
		return nil, fmt.Errorf("gRPC client is closed")
	}

	// 在正常情况下，conn 不会为 nil（除非在关闭过程中）
	// 这里不需要额外的锁保护，因为 gRPC 客户端本身是线程安全的
	if c.conn == nil {
		return nil, fmt.Errorf("gRPC connection is nil")
	}

	return c.client, nil
}

// ExecuteAgent 通过 gRPC 执行 LLM 代理请求
func (c *GRPCLLMClient) ExecuteAgent(ctx context.Context, req *model.LLMRequest) (*model.LLMResponse, error) {
	// 安全地获取连接
	client, err := c.getConnection()
	if err != nil {
		return nil, err
	}

	// 1. 将 model.LLMRequest 转换为 protobuf 请求
	pbMessages := make([]*llmv1.Message, len(req.Messages))
	for i, msg := range req.Messages {
		pbMessages[i] = &llmv1.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	// 设置默认值
	temperature := 0.7
	if req.Temperature != nil {
		temperature = *req.Temperature
	}

	maxTokens := int32(2000)
	if req.MaxTokens != nil {
		maxTokens = int32(*req.MaxTokens)
	}

	pbRequest := &llmv1.ExecuteAgentRequest{
		SessionId:   req.SessionID,
		Messages:    pbMessages,
		Provider:    req.Provider,
		Model:       req.Model,
		Temperature: temperature,
		MaxTokens:   maxTokens,
		Tools:       req.Tools,
	}

	// 2. 调用 gRPC 服务
	pbResponse, err := client.ExecuteAgent(ctx, pbRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to execute agent via gRPC: %w", err)
	}

	// 3. 将 protobuf 响应转换回 model.LLMResponse
	response := &model.LLMResponse{
		SessionID: pbResponse.SessionId,
		Message: model.Message{
			Role:    pbResponse.Message.Role,
			Content: pbResponse.Message.Content,
		},
		Cost:          pbResponse.Cost,
		ExecutionTime: pbResponse.ExecutionTime,
	}

	// 转换使用量统计
	if pbResponse.Usage != nil {
		response.Usage = map[string]int{
			"prompt_tokens":     int(pbResponse.Usage.PromptTokens),
			"completion_tokens": int(pbResponse.Usage.CompletionTokens),
			"total_tokens":      int(pbResponse.Usage.TotalTokens),
		}
	}

	// 转换工具调用（如果有）
	if len(pbResponse.ToolCalls) > 0 {
		toolCalls := make([]map[string]any, len(pbResponse.ToolCalls))
		for i, toolCall := range pbResponse.ToolCalls {
			toolCalls[i] = map[string]any{
				"id":   toolCall.Id,
				"type": toolCall.Type,
			}
			if toolCall.Function != nil {
				toolCalls[i]["function"] = map[string]any{
					"name":      toolCall.Function.Name,
					"arguments": toolCall.Function.Arguments,
				}
			}
		}
		response.ToolCalls = toolCalls
	}

	return response, nil
}

// StreamChunk 表示流式响应的一个数据块
type StreamChunk struct {
	Type         string                 `json:"type"`
	SessionID    string                 `json:"session_id"`
	ContentDelta string                 `json:"content_delta,omitempty"`
	ToolCall     map[string]interface{} `json:"tool_call,omitempty"`
	Usage        map[string]int         `json:"usage,omitempty"`
	Cost         float64                `json:"cost,omitempty"`
	ExecTime     float64                `json:"execution_time,omitempty"`
	Error        string                 `json:"error,omitempty"`
}

// StreamCallback 定义流式响应的回调函数类型
type StreamCallback func(chunk *StreamChunk) error

// ExecuteAgentStream 通过 gRPC 执行流式 LLM 代理请求
func (c *GRPCLLMClient) ExecuteAgentStream(ctx context.Context, req *model.LLMRequest, callback StreamCallback) error {
	// 安全地获取连接
	client, err := c.getConnection()
	if err != nil {
		return err
	}

	// 1. 将 model.LLMRequest 转换为 protobuf 请求
	pbMessages := make([]*llmv1.Message, len(req.Messages))
	for i, msg := range req.Messages {
		pbMessages[i] = &llmv1.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	// 设置默认值
	temperature := 0.7
	if req.Temperature != nil {
		temperature = *req.Temperature
	}

	maxTokens := int32(2000)
	if req.MaxTokens != nil {
		maxTokens = int32(*req.MaxTokens)
	}

	pbRequest := &llmv1.ExecuteAgentRequest{
		SessionId:   req.SessionID,
		Messages:    pbMessages,
		Provider:    req.Provider,
		Model:       req.Model,
		Temperature: temperature,
		MaxTokens:   maxTokens,
		Tools:       req.Tools,
	}

	// 2. 调用流式 gRPC 服务
	stream, err := client.ExecuteAgentStream(ctx, pbRequest)
	if err != nil {
		return fmt.Errorf("failed to start stream: %w", err)
	}

	// 3. 处理流式响应
	for {

		pbResponse, err := stream.Recv()
		if err == io.EOF {
			// 流结束
			return nil

		}
		if grpcErr, ok := status.FromError(err); ok {
			if grpcErr.Code() == codes.Canceled {
				log.Info().Msg("stream canceled by user")
				return nil // 用户取消，不算错误
			}
		}
		// return callback(err)
		if err != nil {
			return fmt.Errorf("stream error: %w", err)
		}

		// 转换响应为 StreamChunk
		chunk := &StreamChunk{
			SessionID: pbResponse.SessionId,
		}

		// 根据响应类型设置相应字段
		switch pbResponse.Type {
		case llmv1.ExecuteAgentStreamResponse_CONTENT_DELTA:
			chunk.Type = "content_delta"
			chunk.ContentDelta = pbResponse.ContentDelta

		case llmv1.ExecuteAgentStreamResponse_TOOL_CALL:
			chunk.Type = "tool_call"
			if pbResponse.ToolCall != nil {
				chunk.ToolCall = map[string]interface{}{
					"id":   pbResponse.ToolCall.Id,
					"type": pbResponse.ToolCall.Type,
				}
				if pbResponse.ToolCall.Function != nil {
					chunk.ToolCall["function"] = map[string]interface{}{
						"name":      pbResponse.ToolCall.Function.Name,
						"arguments": pbResponse.ToolCall.Function.Arguments,
					}
				}
			}

		case llmv1.ExecuteAgentStreamResponse_USAGE_UPDATE:
			chunk.Type = "usage_update"
			if pbResponse.Usage != nil {
				chunk.Usage = map[string]int{
					"prompt_tokens":     int(pbResponse.Usage.PromptTokens),
					"completion_tokens": int(pbResponse.Usage.CompletionTokens),
					"total_tokens":      int(pbResponse.Usage.TotalTokens),
				}
			}

		case llmv1.ExecuteAgentStreamResponse_FINAL_RESPONSE:
			chunk.Type = "final_response"
			chunk.Cost = pbResponse.Cost
			chunk.ExecTime = pbResponse.ExecutionTime
			if pbResponse.Usage != nil {
				chunk.Usage = map[string]int{
					"prompt_tokens":     int(pbResponse.Usage.PromptTokens),
					"completion_tokens": int(pbResponse.Usage.CompletionTokens),
					"total_tokens":      int(pbResponse.Usage.TotalTokens),
				}
			}

		case llmv1.ExecuteAgentStreamResponse_ERROR:
			chunk.Type = "error"
			chunk.Error = pbResponse.ErrorMessage
		}

		// 调用回调函数
		if err := callback(chunk); err != nil {
			return fmt.Errorf("callback error: %w", err)
		}

		// 如果是错误或最终响应，结束流
		if chunk.Type == "error" || chunk.Type == "final_response" {
			return nil
		}
	}
}

// GetSupportedModels 通过 gRPC 获取支持的模型
func (c *GRPCLLMClient) GetSupportedModels(ctx context.Context) (map[string][]string, error) {
	// 安全地获取连接
	client, err := c.getConnection()
	if err != nil {
		return nil, err
	}

	// 创建 gRPC 请求
	req := &llmv1.GetSupportedModelsRequest{}

	// 调用 gRPC 服务
	resp, err := client.GetSupportedModels(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get supported models via gRPC: %w", err)
	}

	// 转换响应格式
	result := make(map[string][]string)
	for providerName, providerInfo := range resp.Providers {
		result[providerName] = providerInfo.Models
	}

	return result, nil
}

// ValidateConfig 通过 gRPC 验证配置
func (c *GRPCLLMClient) ValidateConfig(ctx context.Context, provider, model string, temperature float64, maxTokens int) error {
	// 安全地获取连接
	client, err := c.getConnection()
	if err != nil {
		return err
	}

	// 创建 gRPC 请求
	req := &llmv1.ValidateConfigRequest{
		Provider:    provider,
		Model:       model,
		Temperature: temperature,
		MaxTokens:   int32(maxTokens),
	}

	// 调用 gRPC 服务
	resp, err := client.ValidateConfig(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to validate config via gRPC: %w", err)
	}

	// 检查验证结果
	if !resp.Valid {
		return fmt.Errorf("config validation failed: %s", resp.ErrorMessage)
	}

	return nil
}

// HealthCheck 通过 gRPC 进行健康检查
func (c *GRPCLLMClient) HealthCheck(ctx context.Context) error {
	// 安全地获取连接
	client, err := c.getConnection()
	if err != nil {
		return err
	}

	// 创建 gRPC 请求
	req := &llmv1.HealthCheckRequest{}

	// 调用 gRPC 服务
	resp, err := client.HealthCheck(ctx, req)
	if err != nil {
		return fmt.Errorf("health check failed via gRPC: %w", err)
	}

	// 检查健康状态
	if resp.Status != "healthy" && resp.Status != "ok" {
		return fmt.Errorf("service is not healthy: %s", resp.Status)
	}

	return nil
}

// Close 关闭 gRPC 连接（使用 sync.Once 确保只执行一次）
func (c *GRPCLLMClient) Close() error {
	var closeErr error

	// sync.Once 确保关闭逻辑只执行一次，无论被调用多少次
	c.closeOnce.Do(func() {
		// 先标记为已关闭，阻止新的请求
		c.closed.Store(true)

		// 关闭连接
		if c.conn != nil {
			closeErr = c.conn.Close()
			if closeErr != nil {
				closeErr = fmt.Errorf("failed to close gRPC connection to %s: %w", c.target, closeErr)
			}
			c.conn = nil // 清空连接引用
		}
	})

	return closeErr
}

// GetTarget 返回连接目标
func (c *GRPCLLMClient) GetTarget() string {
	return c.target
}

// IsClosed 检查客户端是否已关闭（无锁原子操作）
func (c *GRPCLLMClient) IsClosed() bool {
	return c.closed.Load()
}
