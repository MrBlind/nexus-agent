from typing import List, Optional, Dict, Any
from pydantic import BaseModel, Field


class Message(BaseModel):
    role: str = Field(..., description="Message role: user, assistant, system")
    content: str = Field(..., description="Message content")
    image_url: Optional[str] = Field(None, description="Optional image URL for multimodal")


class AgentExecuteRequest(BaseModel):
    session_id: str = Field(..., description="Session identifier")
    messages: List[Message] = Field(..., description="Conversation messages")
    model: str = Field(default="gpt-4", description="LLM model to use")
    temperature: float = Field(default=0.7, ge=0.0, le=2.0, description="Sampling temperature")
    max_tokens: Optional[int] = Field(default=1000, gt=0, description="Maximum tokens to generate")
    tools: Optional[List[Dict[str, Any]]] = Field(default=None, description="Available tools")


class AgentExecuteResponse(BaseModel):
    session_id: str
    message: Message
    usage: Dict[str, Any] = Field(default_factory=dict, description="Token usage statistics")
    cost: float = Field(default=0.0, description="Estimated cost")
    execution_time: float = Field(default=0.0, description="Execution time in seconds")
    tool_calls: Optional[List[Dict[str, Any]]] = Field(default=None, description="Tool calls made")


class HealthResponse(BaseModel):
    status: str = Field(default="healthy")
    version: str = Field(default="1.0.0")
    timestamp: str
