"""
模型注册表 - 管理所有可用的模型和提供商配置
"""

import os
import time
from typing import Dict, List, Tuple, Any, Optional
from dataclasses import dataclass
import structlog

from ..utils.config import LLMProvidersConfig, load_config

logger = structlog.get_logger()


@dataclass
class ModelInfo:
    """模型信息"""
    name: str
    max_tokens: int
    supports_vision: bool
    cost_per_1k_tokens: float
    description: str = ""


@dataclass
class ProviderInfo:
    """提供商信息"""
    name: str
    api_key: str
    base_url: str
    models: Dict[str, ModelInfo]
    is_available: bool = False
    extra_config: Dict[str, Any] = None


class ModelRegistry:
    """模型注册表 - 管理所有可用的模型"""
    
    def __init__(self):
        self.providers: Dict[str, ProviderInfo] = {}
        self.model_catalog: Dict[str, Dict[str, ModelInfo]] = {}
        self.last_reload = 0
        self.reload_interval = 300  # 5分钟检查一次
        
        # 初始化加载
        self._load_provider_configs()
        self._build_model_catalog()
        
        logger.info("ModelRegistry 初始化完成", 
                   providers_count=len(self.providers),
                   total_models=sum(len(models) for models in self.model_catalog.values()))
    
    def _load_provider_configs(self):
        """从环境变量加载提供商配置"""
        try:
            config = load_config()
            providers_config = config.llm.providers
            
            # OpenAI
            if providers_config.openai.api_key:
                self.providers["openai"] = ProviderInfo(
                    name="openai",
                    api_key=providers_config.openai.api_key,
                    base_url=providers_config.openai.base_url,
                    models={},
                    is_available=True,
                    extra_config={"org_id": providers_config.openai.org_id}
                )
            
            # DeepSeek
            if providers_config.deepseek.api_key:
                self.providers["deepseek"] = ProviderInfo(
                    name="deepseek",
                    api_key=providers_config.deepseek.api_key,
                    base_url=providers_config.deepseek.base_url,
                    models={},
                    is_available=True
                )
            
            # Anthropic
            if providers_config.anthropic.api_key:
                self.providers["anthropic"] = ProviderInfo(
                    name="anthropic",
                    api_key=providers_config.anthropic.api_key,
                    base_url=providers_config.anthropic.base_url,
                    models={},
                    is_available=True
                )
            
            # Qwen
            if providers_config.qwen.api_key:
                self.providers["qwen"] = ProviderInfo(
                    name="qwen",
                    api_key=providers_config.qwen.api_key,
                    base_url=providers_config.qwen.base_url,
                    models={},
                    is_available=True,
                    extra_config={
                        "region": providers_config.qwen.region,
                        "workspace": providers_config.qwen.workspace
                    }
                )
            
            # Ernie
            if providers_config.ernie.api_key and providers_config.ernie.secret_key:
                self.providers["ernie"] = ProviderInfo(
                    name="ernie",
                    api_key=providers_config.ernie.api_key,
                    base_url=providers_config.ernie.base_url,
                    models={},
                    is_available=True,
                    extra_config={"secret_key": providers_config.ernie.secret_key}
                )
            
            # ChatGLM
            if providers_config.chatglm.api_key:
                self.providers["chatglm"] = ProviderInfo(
                    name="chatglm",
                    api_key=providers_config.chatglm.api_key,
                    base_url=providers_config.chatglm.base_url,
                    models={},
                    is_available=True
                )
            
            self.last_reload = time.time()
            logger.info("提供商配置加载完成", available_providers=list(self.providers.keys()))
            
        except Exception as e:
            logger.error("加载提供商配置失败", error=str(e))
    
    def _build_model_catalog(self):
        """构建完整的模型目录"""
        # OpenAI 模型
        openai_models = {
            "gpt-4": ModelInfo(
                name="gpt-4",
                max_tokens=8192,
                supports_vision=True,
                cost_per_1k_tokens=0.03,
                description="GPT-4 最强大的模型"
            ),
            "gpt-4-turbo": ModelInfo(
                name="gpt-4-turbo",
                max_tokens=128000,
                supports_vision=True,
                cost_per_1k_tokens=0.01,
                description="GPT-4 Turbo 高性能版本"
            ),
            "gpt-4o": ModelInfo(
                name="gpt-4o",
                max_tokens=128000,
                supports_vision=True,
                cost_per_1k_tokens=0.005,
                description="GPT-4o 优化版本"
            ),
            "gpt-4o-mini": ModelInfo(
                name="gpt-4o-mini",
                max_tokens=128000,
                supports_vision=True,
                cost_per_1k_tokens=0.0001,
                description="GPT-4o Mini 轻量版本"
            ),
            "gpt-3.5-turbo": ModelInfo(
                name="gpt-3.5-turbo",
                max_tokens=4096,
                supports_vision=False,
                cost_per_1k_tokens=0.001,
                description="GPT-3.5 Turbo 经济实用"
            )
        }
        
        # DeepSeek 模型
        deepseek_models = {
            "deepseek-chat": ModelInfo(
                name="deepseek-chat",
                max_tokens=32768,
                supports_vision=False,
                cost_per_1k_tokens=0.00014,
                description="DeepSeek Chat 通用对话模型"
            ),
            "deepseek-reasoner": ModelInfo(
                name="deepseek-reasoner",
                max_tokens=8192,
                supports_vision=False,
                cost_per_1k_tokens=0.00055,
                description="DeepSeek Reasoner 推理专用模型"
            )
        }
        
        # Anthropic 模型
        anthropic_models = {
            "claude-3-opus-20240229": ModelInfo(
                name="claude-3-opus-20240229",
                max_tokens=200000,
                supports_vision=True,
                cost_per_1k_tokens=0.015,
                description="Claude 3 Opus 最强大版本"
            ),
            "claude-3-sonnet-20240229": ModelInfo(
                name="claude-3-sonnet-20240229",
                max_tokens=200000,
                supports_vision=True,
                cost_per_1k_tokens=0.003,
                description="Claude 3 Sonnet 平衡版本"
            ),
            "claude-3-5-sonnet-20241022": ModelInfo(
                name="claude-3-5-sonnet-20241022",
                max_tokens=200000,
                supports_vision=True,
                cost_per_1k_tokens=0.003,
                description="Claude 3.5 Sonnet 最新版本"
            ),
            "claude-3-haiku-20240307": ModelInfo(
                name="claude-3-haiku-20240307",
                max_tokens=200000,
                supports_vision=True,
                cost_per_1k_tokens=0.00025,
                description="Claude 3 Haiku 快速版本"
            )
        }
        
        # Qwen 模型
        qwen_models = {
            "qwen-turbo": ModelInfo(
                name="qwen-turbo",
                max_tokens=8192,
                supports_vision=False,
                cost_per_1k_tokens=0.0008,
                description="通义千问 Turbo 快速版本"
            ),
            "qwen-plus": ModelInfo(
                name="qwen-plus",
                max_tokens=32768,
                supports_vision=False,
                cost_per_1k_tokens=0.002,
                description="通义千问 Plus 增强版本"
            ),
            "qwen-max": ModelInfo(
                name="qwen-max",
                max_tokens=8192,
                supports_vision=False,
                cost_per_1k_tokens=0.02,
                description="通义千问 Max 最强版本"
            ),
            "qwen-max-longcontext": ModelInfo(
                name="qwen-max-longcontext",
                max_tokens=30000,
                supports_vision=False,
                cost_per_1k_tokens=0.02,
                description="通义千问 Max 长上下文版本"
            )
        }
        
        # Ernie 模型
        ernie_models = {
            "ernie-bot-turbo": ModelInfo(
                name="ernie-bot-turbo",
                max_tokens=8192,
                supports_vision=False,
                cost_per_1k_tokens=0.0008,
                description="文心一言 Turbo 快速版本"
            ),
            "ernie-bot": ModelInfo(
                name="ernie-bot",
                max_tokens=8192,
                supports_vision=False,
                cost_per_1k_tokens=0.0012,
                description="文心一言标准版本"
            ),
            "ernie-bot-4": ModelInfo(
                name="ernie-bot-4",
                max_tokens=8192,
                supports_vision=False,
                cost_per_1k_tokens=0.012,
                description="文心一言 4.0 最新版本"
            ),
            "ernie-speed": ModelInfo(
                name="ernie-speed",
                max_tokens=8192,
                supports_vision=False,
                cost_per_1k_tokens=0.0004,
                description="文心一言 Speed 极速版本"
            )
        }
        
        # ChatGLM 模型
        chatglm_models = {
            "glm-4": ModelInfo(
                name="glm-4",
                max_tokens=8192,
                supports_vision=False,
                cost_per_1k_tokens=0.01,
                description="智谱 GLM-4 最新版本"
            ),
            "glm-4v": ModelInfo(
                name="glm-4v",
                max_tokens=8192,
                supports_vision=True,
                cost_per_1k_tokens=0.01,
                description="智谱 GLM-4V 视觉版本"
            ),
            "glm-3-turbo": ModelInfo(
                name="glm-3-turbo",
                max_tokens=8192,
                supports_vision=False,
                cost_per_1k_tokens=0.0005,
                description="智谱 GLM-3 Turbo 快速版本"
            )
        }
        
        # 构建模型目录
        self.model_catalog = {
            "openai": openai_models,
            "deepseek": deepseek_models,
            "anthropic": anthropic_models,
            "qwen": qwen_models,
            "ernie": ernie_models,
            "chatglm": chatglm_models
        }
        
        # 将模型信息添加到提供商中
        for provider_name, models in self.model_catalog.items():
            if provider_name in self.providers:
                self.providers[provider_name].models = models
        
        logger.info("模型目录构建完成", 
                   total_models=sum(len(models) for models in self.model_catalog.values()))
    
    def get_available_providers(self) -> List[str]:
        """获取可用的提供商列表"""
        return [name for name, provider in self.providers.items() if provider.is_available]
    
    def get_available_models(self, provider: str = None) -> Dict[str, List[str]]:
        """获取可用模型列表"""
        if provider:
            if provider in self.providers and self.providers[provider].is_available:
                return {provider: list(self.model_catalog.get(provider, {}).keys())}
            return {}
        
        result = {}
        for provider_name in self.get_available_providers():
            result[provider_name] = list(self.model_catalog.get(provider_name, {}).keys())
        return result
    
    def validate_model_request(self, provider: str, model: str) -> Tuple[bool, str]:
        """验证模型请求是否有效"""
        # 检查提供商是否存在
        if provider not in self.providers:
            return False, f"Provider '{provider}' not configured"
        
        # 检查提供商是否可用
        if not self.providers[provider].is_available:
            return False, f"Provider '{provider}' is not available"
        
        # 检查模型是否存在
        if model not in self.model_catalog.get(provider, {}):
            available_models = list(self.model_catalog.get(provider, {}).keys())
            return False, f"Model '{model}' not available for provider '{provider}'. Available models: {available_models}"
        
        return True, "Valid"
    
    def get_model_info(self, provider: str, model: str) -> Optional[ModelInfo]:
        """获取特定模型的信息"""
        return self.model_catalog.get(provider, {}).get(model)
    
    def get_provider_info(self, provider: str) -> Optional[ProviderInfo]:
        """获取提供商信息"""
        return self.providers.get(provider)
    
    def reload_if_needed(self):
        """如果需要则重新加载配置"""
        if time.time() - self.last_reload > self.reload_interval:
            logger.info("重新加载模型注册表配置")
            self._load_provider_configs()
            self._build_model_catalog()
    
    def get_all_models_info(self) -> Dict[str, Dict[str, Dict[str, Any]]]:
        """获取所有模型的详细信息"""
        result = {}
        for provider_name, models in self.model_catalog.items():
            if provider_name in self.providers and self.providers[provider_name].is_available:
                result[provider_name] = {}
                for model_name, model_info in models.items():
                    result[provider_name][model_name] = {
                        "name": model_info.name,
                        "max_tokens": model_info.max_tokens,
                        "supports_vision": model_info.supports_vision,
                        "cost_per_1k_tokens": model_info.cost_per_1k_tokens,
                        "description": model_info.description
                    }
        return result

