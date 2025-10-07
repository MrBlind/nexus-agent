package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/kelseyhightower/envconfig"
)

// Config represents the full application configuration loaded from environment variables.
type Config struct {
	Server        ServerConfig        `envconfig:"SERVER"`
	Database      DatabaseConfig      `envconfig:"DATABASE"`
	Redis         RedisConfig         `envconfig:"REDIS"`
	LLM           LLMConfig           `envconfig:"LLM"`
	Budget        BudgetConfig        `envconfig:"BUDGET"`
	Observability ObservabilityConfig `envconfig:"OBS"`
}

type ServerConfig struct {
	Host         string `split_words:"true" default:"0.0.0.0"`
	Port         int    `split_words:"true" default:"8080"`
	ReadTimeout  int    `split_words:"true" default:"1800"`
	WriteTimeout int    `split_words:"true" default:"1800"`
	Debug        bool   `split_words:"true" default:"true"`
}

type DatabaseConfig struct {
	Host         string `split_words:"true" default:"localhost"`
	Port         int    `split_words:"true" default:"5432"`
	Name         string `split_words:"true" default:"nexus"`
	User         string `split_words:"true" default:"postgres"`
	Password     string `split_words:"true" default:"password"`
	SSLMode      string `split_words:"true" default:"disable"`
	MaxOpenConns int    `split_words:"true" default:"25"`
	MaxIdleConns int    `split_words:"true" default:"5"`
}

type RedisConfig struct {
	Addr     string `split_words:"true" default:"localhost:6379"`
	Password string `split_words:"true" default:""`
	DB       int    `split_words:"true" default:"0"`
}

type LLMConfig struct {
	// 通信协议: http, grpc
	Protocol string `split_words:"true" default:"grpc"`
	// HTTP 配置
	BaseURL string `split_words:"true" default:"http://localhost:8000"`
	// gRPC 配置
	GRPCTarget string `split_words:"true" default:"localhost:50051"`

	// 兜底配置（当请求未指定或指定的模型不可用时使用）
	// 实际上不能用（
	FallbackProvider string `split_words:"true" default:"openai"`
	FallbackModel    string `split_words:"true" default:"gpt-3.5-turbo"`

	// 默认提供商和模型（为了向后兼容）
	DefaultProvider string `split_words:"true" default:"deepseek"`
	DefaultModel    string `split_words:"true" default:"deepseek-chat"`
}

type BudgetConfig struct {
	DefaultTotalTokens int     `split_words:"true" default:"10000"`
	DefaultTotalCost   float64 `split_words:"true" default:"0"`
}

type ObservabilityConfig struct {
	LogLevel string `split_words:"true" default:"debug"`
}

// DSN returns the PostgreSQL connection string.
func (c DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host,
		c.Port,
		c.User,
		c.Password,
		c.Name,
		c.SSLMode,
	)
}

// LoadConfig loads configuration from environment variables.
func LoadConfig() (*Config, error) {
	// Load .env file from go directory
	if err := loadEnvFile(".env"); err == nil {
		fmt.Println("✅ 成功加载 .env 文件")
	} else {
		fmt.Println("⚠️ 未找到 .env 文件，使用默认配置")
	}

	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, fmt.Errorf("failed to process config: %w", err)
	}

	fmt.Printf("🔍 配置详情: 数据库=%s:%d, 用户=%s\n", cfg.Database.Host, cfg.Database.Port, cfg.Database.User)

	return &cfg, nil
}

// loadEnvFile loads environment variables from a .env file
func loadEnvFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY=VALUE format
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Set environment variable if not already set
		if os.Getenv(key) == "" {
			os.Setenv(key, value)
			fmt.Printf("📝 设置环境变量: %s=%s\n", key, value)
		}
	}

	return scanner.Err()
}

// GetFallbackProvider 获取兜底提供商
func (c *LLMConfig) GetFallbackProvider() string {
	return c.FallbackProvider
}

// GetFallbackModel 获取兜底模型
func (c *LLMConfig) GetFallbackModel() string {
	return c.FallbackModel
}

// Provider configuration types
type OpenAIConfig struct {
	APIKey  string `json:"api_key"`
	BaseURL string `json:"base_url"`
}

type DeepSeekConfig struct {
	APIKey  string `json:"api_key"`
	BaseURL string `json:"base_url"`
}

type AnthropicConfig struct {
	APIKey  string `json:"api_key"`
	BaseURL string `json:"base_url"`
}

type QwenConfig struct {
	APIKey  string `json:"api_key"`
	BaseURL string `json:"base_url"`
}

type ErnieConfig struct {
	APIKey  string `json:"api_key"`
	BaseURL string `json:"base_url"`
}

type ChatGLMConfig struct {
	APIKey  string `json:"api_key"`
	BaseURL string `json:"base_url"`
}

// GetDefaultProviderConfig 获取默认提供商配置
func (c *LLMConfig) GetDefaultProviderConfig() (interface{}, bool) {
	return c.GetProviderConfig(c.DefaultProvider)
}

// GetProviderConfig 获取指定提供商的配置
func (c *LLMConfig) GetProviderConfig(provider string) (interface{}, bool) {
	// 从环境变量加载提供商配置，使用标准的命名约定
	switch provider {
	case "openai":
		apiKey := os.Getenv("LLM_PROVIDERS_OPENAI_API_KEY")
		if apiKey == "" {
			apiKey = os.Getenv("OPENAI_API_KEY") // 兼容旧的命名
		}
		baseURL := os.Getenv("LLM_PROVIDERS_OPENAI_BASE_URL")
		if baseURL == "" {
			baseURL = os.Getenv("OPENAI_BASE_URL") // 兼容旧的命名
		}
		return OpenAIConfig{
			APIKey:  apiKey,
			BaseURL: baseURL,
		}, true
	case "deepseek":
		apiKey := os.Getenv("LLM_PROVIDERS_DEEPSEEK_API_KEY")
		if apiKey == "" {
			apiKey = os.Getenv("DEEPSEEK_API_KEY") // 兼容旧的命名
		}
		baseURL := os.Getenv("LLM_PROVIDERS_DEEPSEEK_BASE_URL")
		if baseURL == "" {
			baseURL = os.Getenv("DEEPSEEK_BASE_URL") // 兼容旧的命名
		}
		return DeepSeekConfig{
			APIKey:  apiKey,
			BaseURL: baseURL,
		}, true
	case "anthropic":
		apiKey := os.Getenv("LLM_PROVIDERS_ANTHROPIC_API_KEY")
		if apiKey == "" {
			apiKey = os.Getenv("ANTHROPIC_API_KEY") // 兼容旧的命名
		}
		baseURL := os.Getenv("LLM_PROVIDERS_ANTHROPIC_BASE_URL")
		if baseURL == "" {
			baseURL = os.Getenv("ANTHROPIC_BASE_URL") // 兼容旧的命名
		}
		return AnthropicConfig{
			APIKey:  apiKey,
			BaseURL: baseURL,
		}, true
	case "qwen":
		apiKey := os.Getenv("LLM_PROVIDERS_QWEN_API_KEY")
		if apiKey == "" {
			apiKey = os.Getenv("QWEN_API_KEY") // 兼容旧的命名
		}
		baseURL := os.Getenv("LLM_PROVIDERS_QWEN_BASE_URL")
		if baseURL == "" {
			baseURL = os.Getenv("QWEN_BASE_URL") // 兼容旧的命名
		}
		return QwenConfig{
			APIKey:  apiKey,
			BaseURL: baseURL,
		}, true
	case "ernie":
		apiKey := os.Getenv("LLM_PROVIDERS_ERNIE_API_KEY")
		if apiKey == "" {
			apiKey = os.Getenv("ERNIE_API_KEY") // 兼容旧的命名
		}
		baseURL := os.Getenv("LLM_PROVIDERS_ERNIE_BASE_URL")
		if baseURL == "" {
			baseURL = os.Getenv("ERNIE_BASE_URL") // 兼容旧的命名
		}
		return ErnieConfig{
			APIKey:  apiKey,
			BaseURL: baseURL,
		}, true
	case "chatglm":
		apiKey := os.Getenv("LLM_PROVIDERS_CHATGLM_API_KEY")
		if apiKey == "" {
			apiKey = os.Getenv("CHATGLM_API_KEY") // 兼容旧的命名
		}
		baseURL := os.Getenv("LLM_PROVIDERS_CHATGLM_BASE_URL")
		if baseURL == "" {
			baseURL = os.Getenv("CHATGLM_BASE_URL") // 兼容旧的命名
		}
		return ChatGLMConfig{
			APIKey:  apiKey,
			BaseURL: baseURL,
		}, true
	default:
		return nil, false
	}
}
