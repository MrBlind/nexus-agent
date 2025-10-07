import structlog
from contextlib import asynccontextmanager
from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware

from .utils.config import load_config
from .core.unified_llm_client import UnifiedLLMClient
from .api.routes import create_router
from .api.llm_routes import router as llm_router

# Configure structured logging
structlog.configure(
    processors=[
        structlog.stdlib.filter_by_level,
        structlog.stdlib.add_logger_name,
        structlog.stdlib.add_log_level,
        structlog.stdlib.PositionalArgumentsFormatter(),
        structlog.processors.TimeStamper(fmt="iso", utc=False),
        structlog.processors.StackInfoRenderer(),
        structlog.processors.format_exc_info,
        structlog.processors.UnicodeDecoder(),
        structlog.processors.JSONRenderer()
    ],
    context_class=dict,
    logger_factory=structlog.stdlib.LoggerFactory(),
    wrapper_class=structlog.stdlib.BoundLogger,
    cache_logger_on_first_use=True,
)

logger = structlog.get_logger()


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Application lifespan management."""
    logger.info("Starting Nexus Agent LLM Service")
    yield
    logger.info("Shutting down Nexus Agent LLM Service")


def create_app() -> FastAPI:
    """Create and configure FastAPI application."""
    # Load configuration
    config = load_config()
    
    # Initialize unified LLM client
    llm_client = UnifiedLLMClient(config.llm)
    
    # Create FastAPI app
    app = FastAPI(
        title="Nexus Agent LLM Service",
        description="LLM service for AI agent execution and observability",
        version="1.0.0",
        lifespan=lifespan
    )
    
    # Add CORS middleware
    app.add_middleware(
        CORSMiddleware,
        allow_origins=["*"],  # Configure appropriately for production
        allow_credentials=True,
        allow_methods=["*"],
        allow_headers=["*"],
    )
    
    # Include API routes
    api_router = create_router(llm_client)
    app.include_router(api_router, prefix="/api/v1")
    
    # Include LLM routes (for dynamic configuration from Go)
    app.include_router(llm_router)
    
    return app


# Create app instance
app = create_app()


if __name__ == "__main__":
    import uvicorn
    
    config = load_config()
    uvicorn.run(
        "main:app",
        host=config.server.host,
        port=config.server.port,
        reload=config.server.debug,
        log_level="debug" if config.server.debug else "info"
    )
