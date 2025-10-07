package client

import (
	"context"
	"fmt"

	"github.com/mrblind/nexus-agent/internal/config"
	"github.com/mrblind/nexus-agent/internal/domain/model"
)

// LLMClientInterface 定义 LLM 客户端接口
type LLMClientInterface interface {
	ExecuteAgent(ctx context.Context, req *model.LLMRequest) (*model.LLMResponse, error)
	ExecuteAgentStream(ctx context.Context, req *model.LLMRequest, callback StreamCallback) error
	GetSupportedModels(ctx context.Context) (map[string][]string, error)
	Close() error // 添加资源清理方法
}

// LLMClientFactory 创建不同类型的 LLM 客户端
type LLMClientFactory struct{}

// NewLLMClientFactory 创建 LLM 客户端工厂
func NewLLMClientFactory() *LLMClientFactory {
	return &LLMClientFactory{}
}

// CreateClient 根据配置创建相应的 LLM 客户端
func (f *LLMClientFactory) CreateClient(cfg config.LLMConfig) (LLMClientInterface, error) {
	switch cfg.Protocol {
	case "grpc":
		grpcClient, err := NewGRPCLLMClient(cfg.GRPCTarget)
		if err != nil {
			return nil, fmt.Errorf("failed to create gRPC client: %w", err)
		}
		// 将 gRPC 客户端包装为兼容接口
		return &GRPCLLMClientWrapper{client: grpcClient}, nil

	case "http":
		return NewLLMClient(cfg.BaseURL), nil

	default:
		return nil, fmt.Errorf("unsupported protocol: %s", cfg.Protocol)
	}
}

// GRPCLLMClientWrapper 包装 gRPC 客户端以实现 LLMClient 接口
type GRPCLLMClientWrapper struct {
	client *GRPCLLMClient
}

// 实现 LLMClient 接口的方法
func (w *GRPCLLMClientWrapper) ExecuteAgent(ctx context.Context, req *model.LLMRequest) (*model.LLMResponse, error) {
	return w.client.ExecuteAgent(ctx, req)
}

func (w *GRPCLLMClientWrapper) ExecuteAgentStream(ctx context.Context, req *model.LLMRequest, callback StreamCallback) error {
	return w.client.ExecuteAgentStream(ctx, req, callback)
}

func (w *GRPCLLMClientWrapper) GetSupportedModels(ctx context.Context) (map[string][]string, error) {
	return w.client.GetSupportedModels(ctx)
}

// Close 关闭客户端连接
func (w *GRPCLLMClientWrapper) Close() error {
	return w.client.Close()
}
