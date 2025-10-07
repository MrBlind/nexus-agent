package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/mrblind/nexus-agent/internal/config"
	"github.com/mrblind/nexus-agent/internal/domain/model"
	"github.com/mrblind/nexus-agent/internal/infrastructure/client"
)

// LLMService handles LLM-related operations.
type LLMService struct {
	llmClient client.LLMClientInterface
	config    config.LLMConfig
}

// NewLLMService creates a new LLM service.
func NewLLMService(llmClient client.LLMClientInterface, config config.LLMConfig) *LLMService {
	return &LLMService{
		llmClient: llmClient,
		config:    config,
	}
}

// ExecuteChat executes a chat request using the configured LLM.
func (s *LLMService) ExecuteChat(ctx context.Context, sessionID uuid.UUID, messages []model.Message, llmConfig *model.LLMConfig) (*model.LLMResponse, error) {
	// Use provided config or fall back to default
	config := s.getEffectiveLLMConfig(llmConfig)

	// 准备温度和最大令牌数
	var temperature *float64
	var maxTokens *int
	if config.Temperature > 0 {
		temp := config.Temperature
		temperature = &temp
	}
	if config.MaxTokens > 0 {
		tokens := config.MaxTokens
		maxTokens = &tokens
	}

	req := &model.LLMRequest{
		SessionID: sessionID.String(),
		Messages:  messages,
		Config:    *config,

		// 同时设置独立字段，供 gRPC 客户端使用
		Provider:    string(config.Provider),
		Model:       config.Model,
		Temperature: temperature,
		MaxTokens:   maxTokens,
	}

	return s.llmClient.ExecuteAgent(ctx, req)
}

// ExecuteChatStream executes a streaming chat request using the configured LLM.
func (s *LLMService) ExecuteChatStream(ctx context.Context, sessionID uuid.UUID, messages []model.Message, llmConfig *model.LLMConfig, callback client.StreamCallback) error {
	// Use provided config or fall back to default
	config := s.getEffectiveLLMConfig(llmConfig)

	// 准备温度和最大令牌数
	var temperature *float64
	var maxTokens *int
	if config.Temperature > 0 {
		temp := config.Temperature
		temperature = &temp
	}
	if config.MaxTokens > 0 {
		tokens := config.MaxTokens
		maxTokens = &tokens
	}

	req := &model.LLMRequest{
		SessionID: sessionID.String(),
		Messages:  messages,
		Config:    *config,

		// 同时设置独立字段，供 gRPC 客户端使用
		Provider:    string(config.Provider),
		Model:       config.Model,
		Temperature: temperature,
		MaxTokens:   maxTokens,
	}

	return s.llmClient.ExecuteAgentStream(ctx, req, callback)
}

// GetSupportedModels returns supported models for all providers.
func (s *LLMService) GetSupportedModels(ctx context.Context) (map[string][]string, error) {
	return s.llmClient.GetSupportedModels(ctx)
}

// GetDefaultConfig returns the default LLM configuration based on the configured default provider.
func (s *LLMService) GetDefaultConfig() *model.LLMConfig {
	// Get the default provider configuration
	providerConfig, exists := s.config.GetDefaultProviderConfig()
	if !exists {
		// Fallback to OpenAI if default provider config doesn't exist
		providerConfig, _ = s.config.GetProviderConfig("openai")
	}

	// Determine provider type and get appropriate config
	provider := s.getProviderFromType(s.config.DefaultProvider)
	apiKey, baseURL := s.getProviderCredentials(s.config.DefaultProvider, providerConfig)
	modelName := s.getDefaultModelForProvider(s.config.DefaultProvider)

	return &model.LLMConfig{
		Provider:    provider,
		APIKey:      apiKey,
		Model:       modelName,
		BaseURL:     baseURL,
		Temperature: 0.7,
		MaxTokens:   2000,
	}
}

// getProviderFromType converts string provider type to model.LLMProvider
func (s *LLMService) getProviderFromType(providerType string) model.LLMProvider {
	switch providerType {
	case "openai":
		return model.LLMProviderOpenAI
	case "deepseek":
		return model.LLMProviderDeepSeek
	case "anthropic":
		return model.LLMProviderAnthropic
	case "qwen":
		return model.LLMProviderQwen
	case "ernie":
		return model.LLMProviderErnie
	case "chatglm":
		return model.LLMProviderChatGLM
	default:
		return model.LLMProviderOpenAI // Default fallback
	}
}

// getProviderCredentials extracts API key and base URL from provider config
func (s *LLMService) getProviderCredentials(providerType string, providerConfig interface{}) (string, string) {
	switch providerType {
	case "openai":
		if cfg, ok := providerConfig.(config.OpenAIConfig); ok {
			return cfg.APIKey, cfg.BaseURL
		}
	case "deepseek":
		if cfg, ok := providerConfig.(config.DeepSeekConfig); ok {
			return cfg.APIKey, cfg.BaseURL
		}
	case "anthropic":
		if cfg, ok := providerConfig.(config.AnthropicConfig); ok {
			return cfg.APIKey, cfg.BaseURL
		}
	case "qwen":
		if cfg, ok := providerConfig.(config.QwenConfig); ok {
			return cfg.APIKey, cfg.BaseURL
		}
	case "ernie":
		if cfg, ok := providerConfig.(config.ErnieConfig); ok {
			return cfg.APIKey, cfg.BaseURL
		}
	case "chatglm":
		if cfg, ok := providerConfig.(config.ChatGLMConfig); ok {
			return cfg.APIKey, cfg.BaseURL
		}
	}

	// Fallback to empty credentials
	return "", ""
}

// getDefaultModelForProvider returns the default model for a given provider
func (s *LLMService) getDefaultModelForProvider(providerType string) string {
	switch providerType {
	case "openai":
		return s.config.DefaultModel
	case "deepseek":
		return "deepseek-chat"
	case "anthropic":
		return "claude-3-sonnet-20240229"
	case "qwen":
		return "qwen-turbo"
	case "ernie":
		return "ernie-bot-turbo"
	case "chatglm":
		return "glm-4"
	default:
		return s.config.DefaultModel
	}
}

// getEffectiveLLMConfig returns the effective configuration, using defaults when needed.
func (s *LLMService) getEffectiveLLMConfig(userConfig *model.LLMConfig) *model.LLMConfig {
	config := s.GetDefaultConfig()

	if userConfig != nil {
		// Update provider if specified
		if userConfig.Provider != "" {
			config.Provider = userConfig.Provider

			// Get the appropriate API key and base URL for the new provider
			providerType := string(userConfig.Provider)
			if providerConfig, exists := s.config.GetProviderConfig(providerType); exists {
				apiKey, baseURL := s.getProviderCredentials(providerType, providerConfig)
				config.APIKey = apiKey
				if baseURL != "" {
					config.BaseURL = baseURL
				}
			}
		}

		if userConfig.Model != "" {
			config.Model = userConfig.Model
		}
		if userConfig.BaseURL != "" {
			config.BaseURL = userConfig.BaseURL
		}
		if userConfig.Temperature > 0 {
			config.Temperature = userConfig.Temperature
		}
		if userConfig.MaxTokens > 0 {
			config.MaxTokens = userConfig.MaxTokens
		}
	}

	// Set provider based on model if not explicitly set
	if config.Provider == "" {
		config.Provider = s.inferProviderFromModel(config.Model)
	}

	// Ensure we have the correct API key for the final provider
	if config.APIKey == "" {
		providerType := string(config.Provider)
		if providerConfig, exists := s.config.GetProviderConfig(providerType); exists {
			apiKey, _ := s.getProviderCredentials(providerType, providerConfig)
			config.APIKey = apiKey
		}
	}

	return config
}

// inferProviderFromModel infers the provider from the model name.
func (s *LLMService) inferProviderFromModel(modelName string) model.LLMProvider {
	switch {
	// DeepSeek models
	case modelName == "deepseek-chat" || modelName == "deepseek-reasoner":
		return model.LLMProviderDeepSeek

	// Anthropic Claude models
	case modelName == "claude-3-opus-20240229" || modelName == "claude-3-sonnet-20240229" ||
		modelName == "claude-3-haiku-20240307" || modelName == "claude-3-5-sonnet-20241022":
		return model.LLMProviderAnthropic

	// Qwen models
	case modelName == "qwen-turbo" || modelName == "qwen-plus" || modelName == "qwen-max" || modelName == "qwen-max-longcontext":
		return model.LLMProviderQwen

	// Ernie models
	case modelName == "ernie-bot-turbo" || modelName == "ernie-bot" || modelName == "ernie-bot-4" || modelName == "ernie-speed":
		return model.LLMProviderErnie

	// ChatGLM models
	case modelName == "glm-4" || modelName == "glm-4v" || modelName == "glm-3-turbo":
		return model.LLMProviderChatGLM

	// OpenAI models (default)
	default:
		return model.LLMProviderOpenAI
	}
}

// ValidateConfig validates an LLM configuration.
func (s *LLMService) ValidateConfig(config *model.LLMConfig) error {
	if config.Provider == "" {
		return fmt.Errorf("provider is required")
	}

	if config.Model == "" {
		return fmt.Errorf("model is required")
	}

	if config.Temperature < 0 || config.Temperature > 2 {
		return fmt.Errorf("temperature must be between 0 and 2")
	}

	if config.MaxTokens <= 0 || config.MaxTokens > 32000 {
		return fmt.Errorf("max_tokens must be between 1 and 32000")
	}

	return nil
}
