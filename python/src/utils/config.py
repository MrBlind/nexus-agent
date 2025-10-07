import os
from typing import Optional
from pydantic import BaseModel, Field
from pydantic_settings import BaseSettings


class ServerConfig(BaseModel):
    host: str = Field(default="0.0.0.0")
    port: int = Field(default=8000)
    debug: bool = Field(default=False)


class LLMConfig(BaseModel):
    # 默认引擎类型和模型
    engine_type: str = Field(default="openai", description="Default LLM engine type")
    model: Optional[str] = Field(default=None, description="Default model name")
    timeout: int = Field(default=60)
    
    # 多提供商配置
    providers: "LLMProvidersConfig" = Field(default_factory=lambda: LLMProvidersConfig())


class LLMProvidersConfig(BaseModel):
    # OpenAI 配置
    openai: "OpenAIConfig" = Field(default_factory=lambda: OpenAIConfig())
    
    # DeepSeek 配置
    deepseek: "DeepSeekConfig" = Field(default_factory=lambda: DeepSeekConfig())
    
    # Anthropic Claude 配置
    anthropic: "AnthropicConfig" = Field(default_factory=lambda: AnthropicConfig())
    
    # 阿里云通义千问配置
    qwen: "QwenConfig" = Field(default_factory=lambda: QwenConfig())
    
    # 百度文心一言配置
    ernie: "ErnieConfig" = Field(default_factory=lambda: ErnieConfig())
    
    # 智谱 ChatGLM 配置
    chatglm: "ChatGLMConfig" = Field(default_factory=lambda: ChatGLMConfig())


class OpenAIConfig(BaseSettings):
    api_key: str = Field(default="", description="OpenAI API key")
    base_url: str = Field(default="https://api.openai.com/v1", description="OpenAI API base URL")
    org_id: str = Field(default="", description="OpenAI organization ID")
    
    class Config:
        env_prefix = "LLM_PROVIDERS_OPENAI_"


class DeepSeekConfig(BaseSettings):
    api_key: str = Field(default="", description="DeepSeek API key")
    base_url: str = Field(default="https://api.deepseek.com", description="DeepSeek API base URL")
    
    class Config:
        env_prefix = "LLM_PROVIDERS_DEEPSEEK_"


class AnthropicConfig(BaseSettings):
    api_key: str = Field(default="", description="Anthropic API key")
    base_url: str = Field(default="https://api.anthropic.com", description="Anthropic API base URL")
    
    class Config:
        env_prefix = "LLM_PROVIDERS_ANTHROPIC_"


class QwenConfig(BaseSettings):
    api_key: str = Field(default="", description="Qwen API key")
    base_url: str = Field(default="https://dashscope.aliyuncs.com/api/v1", description="Qwen API base URL")
    region: str = Field(default="cn-hangzhou", description="Qwen region")
    workspace: str = Field(default="", description="Qwen workspace")
    
    class Config:
        env_prefix = "LLM_PROVIDERS_QWEN_"


class ErnieConfig(BaseSettings):
    api_key: str = Field(default="", description="Ernie API key")
    secret_key: str = Field(default="", description="Ernie secret key")
    base_url: str = Field(default="https://aip.baidubce.com", description="Ernie API base URL")
    
    class Config:
        env_prefix = "LLM_PROVIDERS_ERNIE_"


class ChatGLMConfig(BaseSettings):
    api_key: str = Field(default="", description="ChatGLM API key")
    base_url: str = Field(default="https://open.bigmodel.cn/api/paas/v4", description="ChatGLM API base URL")
    
    class Config:
        env_prefix = "LLM_PROVIDERS_CHATGLM_"


class RedisConfig(BaseModel):
    host: str = Field(default="localhost")
    port: int = Field(default=6379)
    password: Optional[str] = Field(default=None)
    db: int = Field(default=0)


class Config(BaseSettings):
    server: ServerConfig = Field(default_factory=ServerConfig)
    llm: LLMConfig = Field(default_factory=LLMConfig)
    redis: RedisConfig = Field(default_factory=RedisConfig)
    
    class Config:
        env_nested_delimiter = "_"
        env_prefix = ""


def load_config() -> Config:
    """Load configuration from environment variables."""
    # 手动创建提供商配置以避免嵌套问题
    providers_config = LLMProvidersConfig(
        openai=OpenAIConfig(),
        deepseek=DeepSeekConfig(),
        anthropic=AnthropicConfig(),
        qwen=QwenConfig(),
        ernie=ErnieConfig(),
        chatglm=ChatGLMConfig()
    )
    
    llm_config = LLMConfig(providers=providers_config)
    
    return Config(
        server=ServerConfig(),
        llm=llm_config,
        redis=RedisConfig()
    )
