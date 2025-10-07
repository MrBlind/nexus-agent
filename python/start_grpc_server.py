#!/usr/bin/env python3
"""
独立的 gRPC 服务器启动脚本
"""

import sys
import os
import asyncio
import logging
import structlog
from datetime import datetime
import time
from dotenv import load_dotenv

# 加载环境变量
load_dotenv()

# 强制刷新输出缓冲区，确保日志能实时显示
sys.stdout.reconfigure(line_buffering=True)
sys.stderr.reconfigure(line_buffering=True)

# 添加项目根目录到 Python 路径
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from src.grpc_server.server import run_grpc_server

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

logger = structlog.get_logger()

async def main():
    """启动 gRPC 服务器"""
    logger.info("启动 Nexus Agent gRPC LLM 服务器")
    
    try:
        await run_grpc_server(
            host="0.0.0.0",
            port=50051,
            max_workers=10
        )
    except KeyboardInterrupt:
        logger.info("收到键盘中断，服务器已停止")
    except Exception as e:
        logger.error("gRPC 服务器运行出错", error=str(e))
        raise

if __name__ == "__main__":
    asyncio.run(main())
