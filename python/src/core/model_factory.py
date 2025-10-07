from typing import Optional, Dict, Type
import structlog

from .llm_interface import LLMEngine
from .engines import OpenAIEngine, DeepSeekEngine

logger = structlog.get_logger()


class ModelFactory:
    """LLM 模型工厂类，用于创建和管理不同的 LLM 引擎"""
    
    # 注册的引擎类型
    _engines: Dict[str, Type[LLMEngine]] = {
        "openai": OpenAIEngine,
        "deepseek": DeepSeekEngine,
        # 未来可以添加更多引擎
        # "anthropic": AnthropicEngine,
        # "qwen": QwenEngine,
        # "ernie": ErnieEngine,
        # "chatglm": ChatGLMEngine,
    }
    
    # 默认模型映射
    _default_models = {
        "openai": "gpt-4",
        "deepseek": "deepseek-chat",
        "anthropic": "claude-3-sonnet-20240229",
        "qwen": "qwen-turbo",
        "ernie": "ernie-bot-turbo",
        "chatglm": "glm-4",
    }
    
    # 所有支持的模型列表（即使没有实现引擎）
    _all_supported_models = {
        "openai": [
            "gpt-4", "gpt-4-turbo", "gpt-4o", "gpt-3.5-turbo", 
            "gpt-4-vision-preview", "gpt-4o-mini"
        ],
        "deepseek": [
            "deepseek-chat", "deepseek-reasoner"
        ],
        "anthropic": [
            "claude-3-opus-20240229", "claude-3-sonnet-20240229", 
            "claude-3-haiku-20240307", "claude-3-5-sonnet-20241022"
        ],
        "qwen": [
            "qwen-turbo", "qwen-plus", "qwen-max", "qwen-max-longcontext"
        ],
        "ernie": [
            "ernie-bot-turbo", "ernie-bot", "ernie-bot-4", "ernie-speed"
        ],
        "chatglm": [
            "glm-4", "glm-4v", "glm-3-turbo"
        ]
    }
    
    @classmethod
    def create_engine(
        cls, 
        engine_type: str, 
        api_key: str, 
        base_url: Optional[str] = None, 
        model: Optional[str] = None
    ) -> LLMEngine:
        """
        创建指定类型的 LLM 引擎
        
        Args:
            engine_type: 引擎类型 ("openai", "deepseek")
            api_key: API 密钥
            base_url: 可选的 API 基础 URL
            model: 可选的模型名称，如果不指定则使用默认模型
            
        Returns:
            LLMEngine: 创建的引擎实例
            
        Raises:
            ValueError: 如果引擎类型不支持
        """
        if engine_type not in cls._engines:
            supported_engines = list(cls._engines.keys())
            raise ValueError(f"不支持的引擎类型: {engine_type}. 支持的类型: {supported_engines}")
        
        # 如果没有指定模型，使用默认模型
        if model is None:
            model = cls._default_models.get(engine_type)
        
        engine_class = cls._engines[engine_type]
        
        return engine_class(
            api_key=api_key,
            base_url=base_url,
            model=model
        )
    
    @classmethod
    def register_engine(cls, engine_type: str, engine_class: Type[LLMEngine], default_model: str = None):
        """
        注册新的引擎类型
        
        Args:
            engine_type: 引擎类型名称
            engine_class: 引擎类
            default_model: 默认模型名称
        """
        cls._engines[engine_type] = engine_class
        if default_model:
            cls._default_models[engine_type] = default_model
        
        logger.info("注册新引擎", engine_type=engine_type, default_model=default_model)
    
    @classmethod
    def get_supported_engines(cls) -> list:
        """获取已实现的引擎类型列表"""
        return list(cls._engines.keys())
    
    @classmethod
    def get_all_engines(cls) -> list:
        """获取所有支持的引擎类型列表（包括未实现的）"""
        return list(cls._all_supported_models.keys())
    
    @classmethod
    def get_default_model(cls, engine_type: str) -> Optional[str]:
        """获取指定引擎类型的默认模型"""
        return cls._default_models.get(engine_type)
    
    @classmethod
    def get_all_models(cls, engine_type: str = None) -> Dict[str, list]:
        """获取所有支持的模型列表"""
        if engine_type:
            models = cls._all_supported_models.get(engine_type, [])
            return {engine_type: models}
        return cls._all_supported_models.copy()
    
    @classmethod
    def is_engine_implemented(cls, engine_type: str) -> bool:
        """检查引擎是否已实现"""
        return engine_type in cls._engines
    
    @classmethod
    def is_engine_supported(cls, engine_type: str) -> bool:
        """检查引擎是否被支持（即使未实现）"""
        return engine_type in cls._all_supported_models
    
    @classmethod
    def get_engine_status(cls) -> Dict[str, Dict[str, any]]:
        """获取所有引擎的状态信息"""
        status = {}
        for engine_type in cls._all_supported_models.keys():
            status[engine_type] = {
                "implemented": cls.is_engine_implemented(engine_type),
                "models": cls._all_supported_models[engine_type],
                "default_model": cls._default_models.get(engine_type),
                "available": False  # 将在运行时根据 API 密钥更新
            }
        return status


# 便捷函数
def create_openai_engine(api_key: str, base_url: Optional[str] = None, model: str = "gpt-4") -> OpenAIEngine:
    """创建 OpenAI 引擎的便捷函数"""
    return ModelFactory.create_engine("openai", api_key, base_url, model)


def create_deepseek_engine(api_key: str, base_url: Optional[str] = None, model: str = "deepseek-chat") -> DeepSeekEngine:
    """创建 DeepSeek 引擎的便捷函数"""
    return ModelFactory.create_engine("deepseek", api_key, base_url, model)
