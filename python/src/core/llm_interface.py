from abc import ABC, abstractmethod
from typing import List, Dict, Any, Optional, AsyncGenerator
from ..models.request import AgentExecuteRequest, AgentExecuteResponse


class LLMEngine(ABC):
    """抽象基类定义 LLM 引擎接口"""
    
    def __init__(self, api_key: str, base_url: Optional[str] = None, model: str = None):
        self.api_key = api_key
        self.base_url = base_url
        self.default_model = model
        self.pricing = self._get_pricing()
    
    @abstractmethod
    def _get_pricing(self) -> Dict[str, Dict[str, float]]:
        """获取模型定价信息"""
        pass
    
    @abstractmethod
    async def execute_agent(self, request: AgentExecuteRequest) -> AgentExecuteResponse:
        """执行代理请求并返回响应"""
        pass
    
    @abstractmethod
    async def execute_agent_stream(self, request: AgentExecuteRequest) -> AsyncGenerator[Dict[str, Any], None]:
        """执行代理请求并返回流式响应"""
        pass
    
    @abstractmethod
    def _convert_messages(self, messages: List[Any]) -> List[Dict[str, Any]]:
        """将内部消息格式转换为特定 API 格式"""
        pass
    
    @abstractmethod
    async def _call_api(self, **kwargs) -> Any:
        """调用具体的 API"""
        pass
    
    def _calculate_cost(self, usage: Dict[str, Any], model: str) -> float:
        """基于令牌使用量计算估算成本"""
        if model not in self.pricing:
            return 0.0
        
        pricing = self.pricing[model]
        input_tokens = usage.get("prompt_tokens", 0)
        output_tokens = usage.get("completion_tokens", 0)
        
        input_cost = (input_tokens / 1000) * pricing["input"]
        output_cost = (output_tokens / 1000) * pricing["output"]
        
        return round(input_cost + output_cost, 6)
