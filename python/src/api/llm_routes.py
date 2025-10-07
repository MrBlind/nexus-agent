from typing import Dict, List, Any, Optional
from fastapi import APIRouter, HTTPException
from pydantic import BaseModel, Field

from ..core.unified_llm_client import UnifiedLLMClient
from ..models.request import Message, AgentExecuteRequest
from ..utils.config import LLMConfig

router = APIRouter(prefix="/api/v1", tags=["llm"])


class DynamicLLMConfig(BaseModel):
    """Dynamic LLM configuration from Go service."""
    provider: str = Field(..., description="LLM provider (openai, deepseek)")
    api_key: str = Field(..., description="API key for the provider")
    model: str = Field(..., description="Model name")
    base_url: Optional[str] = Field(None, description="Custom API base URL")
    temperature: float = Field(0.7, ge=0.0, le=2.0, description="Temperature for generation")
    max_tokens: int = Field(2000, gt=0, le=32000, description="Maximum tokens to generate")


class DynamicLLMRequest(BaseModel):
    """Request with dynamic LLM configuration."""
    session_id: str = Field(..., description="Session ID")
    messages: List[Message] = Field(..., description="Chat messages")
    config: DynamicLLMConfig = Field(..., description="LLM configuration")


class LLMResponse(BaseModel):
    """LLM execution response."""
    session_id: str
    message: Message
    usage: Dict[str, Any]
    cost: float
    execution_time: float
    tool_calls: Optional[List[Dict[str, Any]]] = None


@router.post("/execute", response_model=LLMResponse)
async def execute_agent(request: DynamicLLMRequest) -> LLMResponse:
    """Execute agent with dynamic configuration."""
    try:
        # Create LLM config from request
        llm_config = LLMConfig(
            engine_type=request.config.provider,
            api_key=request.config.api_key,
            base_url=request.config.base_url,
            model=request.config.model,
            timeout=60
        )
        
        # Create unified client with dynamic config
        client = UnifiedLLMClient(llm_config)
        
        # Create agent execute request
        agent_request = AgentExecuteRequest(
            session_id=request.session_id,
            messages=request.messages,
            model=request.config.model,
            temperature=request.config.temperature,
            max_tokens=request.config.max_tokens
        )
        
        # Execute the request
        response = await client.execute_agent(agent_request)
        
        return LLMResponse(
            session_id=response.session_id,
            message=response.message,
            usage=response.usage,
            cost=response.cost,
            execution_time=response.execution_time,
            tool_calls=response.tool_calls
        )
        
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"LLM execution failed: {str(e)}")


@router.get("/models", response_model=Dict[str, List[str]])
async def get_supported_models() -> Dict[str, List[str]]:
    """Get supported models for each provider."""
    return {
        "openai": [
            "gpt-4",
            "gpt-4-turbo",
            "gpt-4o",
            "gpt-3.5-turbo",
            "gpt-4-vision-preview"
        ],
        "deepseek": [
            "deepseek-chat",
            "deepseek-reasoner"
        ]
    }


@router.get("/health")
async def health_check():
    """Health check endpoint."""
    return {"status": "healthy", "service": "nexus-agent-llm"}
