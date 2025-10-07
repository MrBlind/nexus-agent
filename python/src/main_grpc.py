"""
主应用文件 - 支持 gRPC 服务
"""

import asyncio
import structlog
from contextlib import asynccontextmanager

from .grpc_server import run_grpc_server
from .utils.config import load_config

# 配置结构化日志
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


async def main():
    """主函数 - 启动 gRPC 服务"""
    logger.info("启动 Nexus Agent gRPC LLM 服务")
    
    # 加载配置
    config = load_config()
    
    # 启动 gRPC 服务器
    await run_grpc_server(
        host=config.server.host,
        port=50051,  # gRPC 端口
        max_workers=10
    )


if __name__ == "__main__":
    try:
        asyncio.run(main())
    except KeyboardInterrupt:
        logger.info("服务已停止")
    except Exception as e:
        logger.error("服务启动失败", error=str(e))
        raise
