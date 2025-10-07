"""
gRPC 服务器实现
"""

import asyncio
import signal
import sys
from concurrent.futures import ThreadPoolExecutor
import structlog
import grpc
from grpc_reflection.v1alpha import reflection

from ..proto import llm_service_pb2_grpc
from .llm_service_impl import LLMServiceImpl

logger = structlog.get_logger()


async def run_grpc_server(host: str = "0.0.0.0", port: int = 50051, max_workers: int = 10):
    """运行 gRPC 服务器的便捷函数"""
    server = None
    
    try:
        # 创建 gRPC 服务器 - 配置 keepalive 参数
        options = [
            # Keepalive 服务端配置
            ('grpc.keepalive_time_ms', 30000),           # 30秒后发送keepalive ping
            ('grpc.keepalive_timeout_ms', 5000),         # 5秒keepalive超时
            ('grpc.keepalive_permit_without_calls', True), # 允许无调用时的keepalive
            ('grpc.http2.max_pings_without_data', 0),    # 不限制无数据ping数量
            ('grpc.http2.min_time_between_pings_ms', 10000), # 最小ping间隔10秒
            ('grpc.http2.min_ping_interval_without_data_ms', 300000), # 无数据时最小ping间隔5分钟
        ]
        
        server = grpc.aio.server(
            ThreadPoolExecutor(max_workers=max_workers),
            options=options
        )
        
        # 注册服务
        llm_service = LLMServiceImpl()
        llm_service_pb2_grpc.add_LLMServiceServicer_to_server(llm_service, server)
        
        # 启用反射（用于调试）
        SERVICE_NAMES = (
            "nexus.llm.v1.LLMService",  # 直接使用服务名称
            reflection.SERVICE_NAME,
        )
        reflection.enable_server_reflection(SERVICE_NAMES, server)
        
        # 绑定端口
        listen_addr = f"{host}:{port}"
        server.add_insecure_port(listen_addr)
        
        # 启动服务器
        await server.start()
        
        logger.info("gRPC 服务器启动成功",
                   host=host,
                   port=port,
                   max_workers=max_workers)
        
        # 等待服务器终止
        await server.wait_for_termination()
        
    except KeyboardInterrupt:
        logger.info("收到键盘中断，正在关闭服务器")
    except Exception as e:
        logger.error("gRPC 服务器运行出错", error=str(e))
        raise
    finally:
        # 确保服务器正确关闭
        if server:
            logger.info("正在停止 gRPC 服务器")
            await server.stop(5)
            logger.info("gRPC 服务器已停止")


if __name__ == "__main__":
    import logging
    import sys
    import time
    
    # 配置标准日志 - 使用简洁格式，本地时区
    logging.basicConfig(
        level=logging.INFO,
        format='%(asctime)s [%(levelname)s] %(message)s',
        datefmt='%Y-%m-%d %H:%M:%S',
        stream=sys.stdout,
        force=True
    )
    
    # 设置时区为本地时区
    logging.Formatter.converter = time.localtime
    
    # 配置structlog - 使用简洁的文本格式
    structlog.configure(
        processors=[
            structlog.stdlib.filter_by_level,
            structlog.stdlib.add_log_level,
            structlog.stdlib.PositionalArgumentsFormatter(),
            structlog.processors.TimeStamper(fmt="%Y-%m-%d %H:%M:%S", utc=False),
            structlog.processors.format_exc_info,
            structlog.processors.UnicodeDecoder(),
            # 自定义简洁格式
            lambda _, __, event_dict: f"{event_dict.get('timestamp', '')} [{event_dict.get('level', '').upper()}] {event_dict.get('event', '')}" + 
                                     (''.join(f" {k}={v}" for k, v in event_dict.items() if k not in ['timestamp', 'level', 'event']) if len(event_dict) > 3 else '')
        ],
        context_class=dict,
        logger_factory=structlog.stdlib.LoggerFactory(),
        wrapper_class=structlog.stdlib.BoundLogger,
        cache_logger_on_first_use=True,
    )
    
    # 运行服务器
    asyncio.run(run_grpc_server())
