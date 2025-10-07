from datetime import datetime
from fastapi import APIRouter, HTTPException, Header
from typing import Optional
import structlog

from ..models.request import AgentExecuteRequest, AgentExecuteResponse, HealthResponse
from ..core.unified_llm_client import UnifiedLLMClient

logger = structlog.get_logger()

router = APIRouter()


class AgentAPI:
    def __init__(self, llm_client: UnifiedLLMClient):
        self.llm_client = llm_client
    
    def setup_routes(self, router: APIRouter):
        """Setup API routes."""
        
        @router.get("/health", response_model=HealthResponse)
        async def health_check():
            """Health check endpoint."""
            return HealthResponse(
                status="healthy",
                version="1.0.0",
                timestamp=datetime.utcnow().isoformat()
            )
        
        @router.post("/agent/execute", response_model=AgentExecuteResponse)
        async def execute_agent(
            request: AgentExecuteRequest,
            x_request_id: Optional[str] = Header(None, alias="X-Request-ID")
        ):
            """Execute agent with LLM."""
            request_id = x_request_id or f"req-{datetime.utcnow().timestamp()}"
            
            logger.info("Agent execution started", 
                       session_id=request.session_id,
                       request_id=request_id,
                       model=request.model)
            
            try:
                response = await self.llm_client.execute_agent(request)
                
                logger.info("Agent execution completed",
                           session_id=request.session_id,
                           request_id=request_id,
                           cost=response.cost,
                           execution_time=response.execution_time)
                
                return response
                
            except Exception as e:
                logger.error("Agent execution failed",
                            session_id=request.session_id,
                            request_id=request_id,
                            error=str(e))
                raise HTTPException(status_code=500, detail=f"Agent execution failed: {str(e)}")


def create_router(llm_client: UnifiedLLMClient) -> APIRouter:
    """Create and configure API router."""
    api = AgentAPI(llm_client)
    api.setup_routes(router)
    return router
