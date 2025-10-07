import time
from typing import List, Dict, Any, Optional, AsyncGenerator
import structlog
import httpx
import json

from ..llm_interface import LLMEngine
from ...models.request import Message, AgentExecuteRequest, AgentExecuteResponse

logger = structlog.get_logger()


class DeepSeekEngine(LLMEngine):
    """DeepSeek LLM 引擎实现"""
    
    def __init__(self, api_key: str, base_url: Optional[str] = None, model: str = "deepseek-chat"):
        # DeepSeek 默认 API 地址
        if base_url is None:
            base_url = "https://api.deepseek.com"
        super().__init__(api_key, base_url, model)
        
        # 创建 HTTP 客户端
        self.client = httpx.AsyncClient(
            base_url=base_url,
            headers={
                "Authorization": f"Bearer {api_key}",
                "Content-Type": "application/json"
            },
            timeout=300.0
        )
    
    def _get_pricing(self) -> Dict[str, Dict[str, float]]:
        """获取 DeepSeek 模型定价信息 (美元/1K tokens)"""
        return {
            "deepseek-chat": {"input": 0.00014, "output": 0.00028},  # 非常便宜
            "deepseek-reasoner": {"input": 0.00055, "output": 0.0022}  # 推理模型稍贵
        }
    
    async def execute_agent(self, request: AgentExecuteRequest) -> AgentExecuteResponse:
        """执行代理请求并返回响应"""
        start_time = time.time()
        
        try:
            # 转换消息格式
            deepseek_messages = self._convert_messages(request.messages)
            
            # 使用指定模型或默认模型
            model = request.model or self.default_model
            
            # 调用 API
            response = await self._call_api(
                messages=deepseek_messages,
                model=model,
                temperature=request.temperature,
                max_tokens=request.max_tokens,
                tools=request.tools
            )
            
            # 计算使用量和成本
            usage = response.get("usage", {})
            
            # 确保 usage 中的基本字段存在且为整数类型
            usage_clean = {
                "prompt_tokens": usage.get("prompt_tokens", 0),
                "completion_tokens": usage.get("completion_tokens", 0),
                "total_tokens": usage.get("total_tokens", 0)
            }
            
            # 保留其他字段但确保类型安全，过滤掉复杂的嵌套对象
            for key, value in usage.items():
                if key not in usage_clean:
                    # 只保留简单类型的字段
                    if isinstance(value, (int, float, str, bool)):
                        usage_clean[key] = value
                    # 对于复杂类型，可以选择性地保留或转换
                    elif key == "prompt_tokens_details" and isinstance(value, dict):
                        # 将嵌套字典展平为简单字段
                        usage_clean["prompt_cache_hit_tokens"] = value.get("cached_tokens", 0)
                    elif key in ["prompt_cache_hit_tokens", "prompt_cache_miss_tokens"]:
                        usage_clean[key] = int(value) if isinstance(value, (int, float)) else 0
            
            cost = self._calculate_cost(usage_clean, model)
            execution_time = time.time() - start_time
            
            # 提取响应消息
            choice = response["choices"][0]
            assistant_message = Message(
                role="assistant",
                content=choice["message"]["content"] or ""
            )
            
            # 提取工具调用（如果有）
            tool_calls = None
            if choice["message"].get("tool_calls"):
                tool_calls = [
                    {
                        "id": call["id"],
                        "type": call["type"],
                        "function": {
                            "name": call["function"]["name"],
                            "arguments": call["function"]["arguments"]
                        }
                    }
                    for call in choice["message"]["tool_calls"]
                ]

            response = AgentExecuteResponse(
                session_id=request.session_id,
                message=assistant_message,
                usage=usage_clean,
                cost=cost,
                execution_time=execution_time,
                tool_calls=tool_calls
            )

            return response
            
        except Exception as e:
            logger.error("DeepSeek 执行失败", 
                        session_id=request.session_id, 
                        error=str(e))
            raise
    
    def _convert_messages(self, messages: List[Message]) -> List[Dict[str, Any]]:
        """将内部消息格式转换为 DeepSeek 格式"""
        deepseek_messages = []
        
        for msg in messages:
            deepseek_msg = {
                "role": msg.role,
                "content": msg.content
            }
            
            # DeepSeek 目前不支持图像，如果有图像 URL，只保留文本
            if msg.image_url:
                logger.warning("DeepSeek 暂不支持图像输入，将忽略图像内容", 
                             image_url=msg.image_url)
            
            deepseek_messages.append(deepseek_msg)
        
        return deepseek_messages
    
    async def _call_api(self, **kwargs) -> Dict[str, Any]:
        """调用 DeepSeek API"""
        # 构建请求数据
        data = {
            "model": kwargs.get("model", self.default_model),
            "messages": kwargs.get("messages", []),
            "temperature": kwargs.get("temperature", 0.7),
            "stream": False  # 暂不支持流式
        }
        
        # 添加可选参数
        if kwargs.get("max_tokens"):
            data["max_tokens"] = kwargs["max_tokens"]
        
        if kwargs.get("tools"):
            data["tools"] = kwargs["tools"]
            data["tool_choice"] = "auto"
        
        try:
            response = await self.client.post("/chat/completions", json=data)
            response.raise_for_status()
            return response.json()
            
        except httpx.HTTPStatusError as e:
            logger.error("DeepSeek API 请求失败", 
                        status_code=e.response.status_code,
                        response_text=e.response.text)
            raise Exception(f"DeepSeek API 错误: {e.response.status_code} - {e.response.text}")
        
        except Exception as e:
            logger.error("DeepSeek API 调用异常", error=str(e))
            raise
    
    async def __aenter__(self):
        """异步上下文管理器入口"""
        return self
    
    async def execute_agent_stream(self, request: AgentExecuteRequest) -> AsyncGenerator[Dict[str, Any], None]:
        """执行代理请求并返回流式响应"""
        start_time = time.time()
        
        try:
            # 转换消息格式
            deepseek_messages = self._convert_messages(request.messages)
            
            # 使用指定模型或默认模型
            model = request.model or self.default_model
            
            # 构建流式请求数据
            data = {
                "model": model,
                "messages": deepseek_messages,
                "temperature": request.temperature,
                "stream": True
            }
            
            # 添加可选参数
            if request.max_tokens:
                data["max_tokens"] = request.max_tokens
            
            if request.tools:
                data["tools"] = request.tools
                data["tool_choice"] = "auto"
            
            # 发起流式请求
            async with self.client.stream("POST", "/chat/completions", json=data) as response:
                response.raise_for_status()
                
                # 累积响应数据
                full_content = ""
                usage_data = {}
                
                # 处理流式响应
                async for line in response.aiter_lines():
                    if line.startswith("data: "):
                        data_str = line[6:]  # 移除 "data: " 前缀
                        
                        if data_str == "[DONE]":
                            break
                        
                        try:
                            chunk_data = json.loads(data_str)
                            
                            if "choices" in chunk_data and len(chunk_data["choices"]) > 0:
                                choice = chunk_data["choices"][0]
                                
                                # 处理内容增量
                                if "delta" in choice and "content" in choice["delta"]:
                                    content_delta = choice["delta"]["content"]
                                    if content_delta:
                                        full_content += content_delta
                                        
                                        yield {
                                            "type": "content_delta",
                                            "session_id": request.session_id,
                                            "content": content_delta,
                                            "timestamp": time.time()
                                        }
                                
                                # 处理工具调用
                                if "delta" in choice and "tool_calls" in choice["delta"]:
                                    for tool_call in choice["delta"]["tool_calls"]:
                                        yield {
                                            "type": "tool_call",
                                            "session_id": request.session_id,
                                            "tool_call": {
                                                "id": tool_call.get("id", ""),
                                                "type": tool_call.get("type", ""),
                                                "function": {
                                                    "name": tool_call.get("function", {}).get("name", ""),
                                                    "arguments": tool_call.get("function", {}).get("arguments", "")
                                                }
                                            },
                                            "timestamp": time.time()
                                        }
                            
                            # 处理使用量信息
                            if "usage" in chunk_data:
                                usage_data = chunk_data["usage"]
                        
                        except json.JSONDecodeError:
                            continue
                
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
            logger.error("DeepSeek 流式执行失败", 
                        session_id=request.session_id, 
                        error=str(e))
            yield {
                "type": "error",
                "session_id": request.session_id,
                "error": str(e),
                "timestamp": time.time()
            }
    
    def _calculate_cost(self, usage: Dict[str, Any], model: str) -> float:
        """基于令牌使用量计算估算成本"""

        if model not in self.pricing:
            return 0.0
        
        hit_tokens = usage.get("prompt_cache_hit_tokens", 0.0)
        miss_tokens = usage.get("prompt_cache_miss_tokens", 0.0)
        # 百万tokens输入（缓存命中）	0.2
        # 百万tokens输入（缓存未命中）	2
        # 百万tokens输出            	3
        # deepseek 模型
        input_cost = hit_tokens * 0.000002 + miss_tokens * 0.00002
        output_cost = usage.get("completion_tokens", 0) * 0.00003
        
        return round(input_cost + output_cost, 10)

    async def __aexit__(self, exc_type, exc_val, exc_tb):
        """异步上下文管理器出口"""
        await self.client.aclose()
