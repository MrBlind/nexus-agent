import time
from typing import List, Dict, Any, Optional, AsyncGenerator
import structlog
from openai import OpenAI

from ..llm_interface import LLMEngine
from ...models.request import Message, AgentExecuteRequest, AgentExecuteResponse

logger = structlog.get_logger()


class OpenAIEngine(LLMEngine):
    """OpenAI LLM 引擎实现"""
    
    def __init__(self, api_key: str, base_url: Optional[str] = None, model: str = "gpt-4"):
        super().__init__(api_key, base_url, model)
        self.client = OpenAI(
            api_key=api_key,
            base_url=base_url
        )
    
    def _get_pricing(self) -> Dict[str, Dict[str, float]]:
        """获取 OpenAI 模型定价信息"""
        return {
            "gpt-4": {"input": 0.03, "output": 0.06},  # per 1K tokens
            "gpt-4-vision-preview": {"input": 0.01, "output": 0.03},
            "gpt-3.5-turbo": {"input": 0.001, "output": 0.002},
            "gpt-4-turbo": {"input": 0.01, "output": 0.03},
            "gpt-4o": {"input": 0.005, "output": 0.015}
        }
    
    async def execute_agent(self, request: AgentExecuteRequest) -> AgentExecuteResponse:
        """执行代理请求并返回响应"""
        start_time = time.time()
        
        try:
            # 转换消息格式
            openai_messages = self._convert_messages(request.messages)
            
            # 判断是否为视觉请求
            has_images = any(msg.image_url for msg in request.messages)
            model = request.model or self.default_model
            if has_images and "vision" not in model:
                model = "gpt-4-vision-preview"
            
            # 调用 API
            response = await self._call_api(
                messages=openai_messages,
                model=model,
                temperature=request.temperature,
                max_tokens=request.max_tokens,
                tools=request.tools
            )
            
            # 计算使用量和成本
            usage = response.usage.model_dump() if response.usage else {}
            cost = self._calculate_cost(usage, model)
            execution_time = time.time() - start_time
            
            # 提取响应消息
            assistant_message = Message(
                role="assistant",
                content=response.choices[0].message.content or ""
            )
            
            # 提取工具调用（如果有）
            tool_calls = None
            if response.choices[0].message.tool_calls:
                tool_calls = [
                    {
                        "id": call.id,
                        "type": call.type,
                        "function": {
                            "name": call.function.name,
                            "arguments": call.function.arguments
                        }
                    }
                    for call in response.choices[0].message.tool_calls
                ]
            
            return AgentExecuteResponse(
                session_id=request.session_id,
                message=assistant_message,
                usage=usage,
                cost=cost,
                execution_time=execution_time,
                tool_calls=tool_calls
            )
            
        except Exception as e:
            logger.error("OpenAI 执行失败", 
                        session_id=request.session_id, 
                        error=str(e))
            raise
    
    def _convert_messages(self, messages: List[Message]) -> List[Dict[str, Any]]:
        """将内部消息格式转换为 OpenAI 格式"""
        openai_messages = []
        
        for msg in messages:
            openai_msg = {
                "role": msg.role,
                "content": msg.content
            }
            
            # 处理多模态内容
            if msg.image_url:
                openai_msg["content"] = [
                    {"type": "text", "text": msg.content},
                    {"type": "image_url", "image_url": {"url": msg.image_url}}
                ]
            
            openai_messages.append(openai_msg)
        
        return openai_messages
    
    async def execute_agent_stream(self, request: AgentExecuteRequest) -> AsyncGenerator[Dict[str, Any], None]:
        """执行代理请求并返回流式响应"""
        start_time = time.time()
        
        try:
            # 转换消息格式
            openai_messages = self._convert_messages(request.messages)
            
            # 判断是否为视觉请求
            has_images = any(msg.image_url for msg in request.messages)
            model = request.model or self.default_model
            if has_images and "vision" not in model:
                model = "gpt-4-vision-preview"
            
            # 调用流式 API
            stream = self.client.chat.completions.create(
                messages=openai_messages,
                model=model,
                temperature=request.temperature,
                max_tokens=request.max_tokens,
                tools=request.tools,
                stream=True
            )
            
            # 累积响应数据
            full_content = ""
            usage_data = {}
            tool_calls = []
            
            # 处理流式响应
            for chunk in stream:
                if chunk.choices and len(chunk.choices) > 0:
                    choice = chunk.choices[0]
                    
                    # 处理内容增量
                    if choice.delta and choice.delta.content:
                        content_delta = choice.delta.content
                        full_content += content_delta
                        
                        yield {
                            "type": "content_delta",
                            "session_id": request.session_id,
                            "content": content_delta,
                            "timestamp": time.time()
                        }
                    
                    # 处理工具调用
                    if choice.delta and choice.delta.tool_calls:
                        for tool_call in choice.delta.tool_calls:
                            yield {
                                "type": "tool_call",
                                "session_id": request.session_id,
                                "tool_call": {
                                    "id": tool_call.id,
                                    "type": tool_call.type,
                                    "function": {
                                        "name": tool_call.function.name if tool_call.function else "",
                                        "arguments": tool_call.function.arguments if tool_call.function else ""
                                    }
                                },
                                "timestamp": time.time()
                            }
                
                # 处理使用量信息
                if chunk.usage:
                    usage_data = chunk.usage.model_dump()
            
            # 发送最终响应
            execution_time = time.time() - start_time
            cost = self._calculate_cost(usage_data, model)
            
            yield {
                "type": "final_response",
                "session_id": request.session_id,
                "usage": usage_data,
                "cost": cost,
                "execution_time": execution_time,
                "timestamp": time.time()
            }
            
        except Exception as e:
            logger.error("OpenAI 流式执行失败", 
                        session_id=request.session_id, 
                        error=str(e))
            yield {
                "type": "error",
                "session_id": request.session_id,
                "error": str(e),
                "timestamp": time.time()
            }
    
    async def _call_api(self, **kwargs) -> Any:
        """调用 OpenAI API"""
        # 目前使用同步客户端 - 生产环境中应使用异步客户端
        return self.client.chat.completions.create(**kwargs)
