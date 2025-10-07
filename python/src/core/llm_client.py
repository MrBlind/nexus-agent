import time
from typing import List, Dict, Any, Optional
import httpx
import structlog
from openai import OpenAI

from ..models.request import Message, AgentExecuteRequest, AgentExecuteResponse

logger = structlog.get_logger()


class LLMClient:
    """LLM client for handling agent execution requests."""
    
    def __init__(self, api_key: str, base_url: Optional[str] = None, model: str = "gpt-4"):
        self.client = OpenAI(
            api_key=api_key,
            base_url=base_url
        )
        self.default_model = model
        self.pricing = {
            "gpt-4": {"input": 0.03, "output": 0.06},  # per 1K tokens
            "gpt-4-vision-preview": {"input": 0.01, "output": 0.03},
            "gpt-3.5-turbo": {"input": 0.001, "output": 0.002}
        }
    
    async def execute_agent(self, request: AgentExecuteRequest) -> AgentExecuteResponse:
        """Execute agent with LLM and return response."""
        start_time = time.time()
        
        try:
            # Convert messages to OpenAI format
            openai_messages = self._convert_messages(request.messages)
            
            # Determine if this is a vision request
            has_images = any(msg.image_url for msg in request.messages)
            model = request.model
            if has_images and "vision" not in model:
                model = "gpt-4-vision-preview"
            
            # Make API call
            response = await self._call_openai(
                messages=openai_messages,
                model=model,
                temperature=request.temperature,
                max_tokens=request.max_tokens,
                tools=request.tools
            )
            
            # Calculate usage and cost
            usage = response.usage.model_dump() if response.usage else {}
            cost = self._calculate_cost(usage, model)
            execution_time = time.time() - start_time
            
            # Extract response message
            assistant_message = Message(
                role="assistant",
                content=response.choices[0].message.content or ""
            )
            
            # Extract tool calls if any
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
            logger.error("LLM execution failed", 
                        session_id=request.session_id, 
                        error=str(e))
            raise
    
    def _convert_messages(self, messages: List[Message]) -> List[Dict[str, Any]]:
        """Convert internal message format to OpenAI format."""
        openai_messages = []
        
        for msg in messages:
            openai_msg = {
                "role": msg.role,
                "content": msg.content
            }
            
            # Handle multimodal content
            if msg.image_url:
                openai_msg["content"] = [
                    {"type": "text", "text": msg.content},
                    {"type": "image_url", "image_url": {"url": msg.image_url}}
                ]
            
            openai_messages.append(openai_msg)
        
        return openai_messages
    
    async def _call_openai(self, **kwargs) -> Any:
        """Make async call to OpenAI API."""
        # For now, using sync client - in production, use async client
        return self.client.chat.completions.create(**kwargs)
    
    def _calculate_cost(self, usage: Dict[str, Any], model: str) -> float:
        """Calculate estimated cost based on token usage."""
        if model not in self.pricing:
            return 0.0
        
        pricing = self.pricing[model]
        input_tokens = usage.get("prompt_tokens", 0)
        output_tokens = usage.get("completion_tokens", 0)
        
        input_cost = (input_tokens / 1000) * pricing["input"]
        output_cost = (output_tokens / 1000) * pricing["output"]
        
        return round(input_cost + output_cost, 6)
