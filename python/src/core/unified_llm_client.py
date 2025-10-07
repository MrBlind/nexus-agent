import structlog
from typing import Optional, AsyncGenerator, Dict, Any

from .model_factory import ModelFactory
from .llm_interface import LLMEngine
from ..models.request import AgentExecuteRequest, AgentExecuteResponse
from ..utils.config import LLMConfig

logger = structlog.get_logger()


class UnifiedLLMClient:
    """统一的 LLM 客户端，支持多种引擎"""
    
    def __init__(self, config: LLMConfig):
        self.config = config
        self.engine = self._create_engine()
    
    def _create_engine(self) -> LLMEngine:
        """根据配置创建相应的引擎"""
        engine_type = self.config.engine_type.lower()
        
        # 根据引擎类型从多提供商配置中获取相应的配置
        api_key, base_url = self._get_provider_config(engine_type)
        
        if not api_key:
            raise ValueError(f"缺少 {engine_type} 引擎的 API 密钥，请在环境变量中配置 LLM_PROVIDERS_{engine_type.upper()}_API_KEY")
        
        # 使用配置的模型或引擎默认模型
        model = self.config.model or ModelFactory.get_default_model(engine_type)
        
        logger.info("创建 LLM 引擎", 
                   engine_type=engine_type, 
                   model=model,
                   base_url=base_url)
        
        return ModelFactory.create_engine(
            engine_type=engine_type,
            api_key=api_key,
            base_url=base_url,
            model=model
        )
    
    def _get_provider_config(self, engine_type: str) -> tuple[str, str]:
        """根据引擎类型获取提供商配置"""
        providers = self.config.providers
        
        if engine_type == "openai":
            return providers.openai.api_key, providers.openai.base_url
        elif engine_type == "deepseek":
            return providers.deepseek.api_key, providers.deepseek.base_url
        elif engine_type == "anthropic":
            return providers.anthropic.api_key, providers.anthropic.base_url
        elif engine_type == "qwen":
            return providers.qwen.api_key, providers.qwen.base_url
        elif engine_type == "ernie":
            # Ernie 需要两个密钥
            if providers.ernie.api_key and providers.ernie.secret_key:
                return providers.ernie.api_key, providers.ernie.base_url
            else:
                return "", providers.ernie.base_url
        elif engine_type == "chatglm":
            return providers.chatglm.api_key, providers.chatglm.base_url
        else:
            raise ValueError(f"不支持的引擎类型: {engine_type}")
    
    def get_available_providers(self) -> dict[str, bool]:
        """获取可用的提供商列表"""
        available = {}
        for engine_type in ModelFactory.get_all_engines():
            api_key, _ = self._get_provider_config(engine_type)
            available[engine_type] = bool(api_key)
        return available
    
    async def execute_agent(self, request: AgentExecuteRequest) -> AgentExecuteResponse:
        """执行代理请求"""
        return await self.engine.execute_agent(request)
    
    async def execute_agent_stream(self, request: AgentExecuteRequest) -> AsyncGenerator[Dict[str, Any], None]:
        """执行代理请求并返回流式响应"""
        async for chunk in self.engine.execute_agent_stream(request):
            yield chunk
    
    def get_engine_type(self) -> str:
        """获取当前使用的引擎类型"""
        return self.config.engine_type
    
    def get_model(self) -> str:
        """获取当前使用的模型"""
        return self.engine.default_model
    
    def switch_engine(self, engine_type: str, api_key: Optional[str] = None, 
                     base_url: Optional[str] = None, model: Optional[str] = None):
        """动态切换引擎（运行时切换）"""
        # 更新配置
        old_engine = self.config.engine_type
        self.config.engine_type = engine_type
        
        # 更新提供商特定的配置
        if api_key or base_url:
            self._update_provider_config(engine_type, api_key, base_url)
        
        if model:
            self.config.model = model
        
        try:
            # 重新创建引擎
            self.engine = self._create_engine()
            
            logger.info("切换 LLM 引擎成功", 
                       old_engine=old_engine,
                       new_engine=engine_type, 
                       new_model=self.engine.default_model)
        except Exception as e:
            # 如果切换失败，恢复原来的配置
            self.config.engine_type = old_engine
            logger.error("切换 LLM 引擎失败，已恢复原配置", 
                        engine_type=engine_type, 
                        error=str(e))
            raise
    
    def _update_provider_config(self, engine_type: str, api_key: Optional[str], base_url: Optional[str]):
        """更新特定提供商的配置"""
        providers = self.config.providers
        
        if engine_type == "openai":
            if api_key:
                providers.openai.api_key = api_key
            if base_url:
                providers.openai.base_url = base_url
        elif engine_type == "deepseek":
            if api_key:
                providers.deepseek.api_key = api_key
            if base_url:
                providers.deepseek.base_url = base_url
        elif engine_type == "anthropic":
            if api_key:
                providers.anthropic.api_key = api_key
            if base_url:
                providers.anthropic.base_url = base_url
        elif engine_type == "qwen":
            if api_key:
                providers.qwen.api_key = api_key
            if base_url:
                providers.qwen.base_url = base_url
        elif engine_type == "ernie":
            if api_key:
                providers.ernie.api_key = api_key
            if base_url:
                providers.ernie.base_url = base_url
        elif engine_type == "chatglm":
            if api_key:
                providers.chatglm.api_key = api_key
            if base_url:
                providers.chatglm.base_url = base_url
