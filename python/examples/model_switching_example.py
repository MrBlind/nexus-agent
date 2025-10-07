"""
LLM 模型切换示例
演示如何在代码中轻松切换不同的 LLM 引擎
"""

import asyncio
import os
from typing import List

# 添加项目路径
import sys
sys.path.append(os.path.join(os.path.dirname(__file__), '..'))

from src.core.model_factory import ModelFactory, create_openai_engine, create_deepseek_engine
from src.core.unified_llm_client import UnifiedLLMClient
from src.utils.config import LLMConfig
from src.models.request import Message, AgentExecuteRequest


async def example_basic_usage():
    """基础使用示例"""
    print("=== 基础使用示例 ===")
    
    # 方法1: 使用工厂模式
    print("\n1. 使用工厂模式创建引擎:")
    
    # 创建 OpenAI 引擎
    openai_engine = ModelFactory.create_engine(
        engine_type="openai",
        api_key=os.getenv("OPENAI_API_KEY", "your-openai-key"),
        model="gpt-4"
    )
    print(f"创建了 OpenAI 引擎，模型: {openai_engine.default_model}")
    
    # 创建 DeepSeek 引擎
    deepseek_engine = ModelFactory.create_engine(
        engine_type="deepseek", 
        api_key=os.getenv("DEEPSEEK_API_KEY", "your-deepseek-key"),
        model="deepseek-chat"
    )
    print(f"创建了 DeepSeek 引擎，模型: {deepseek_engine.default_model}")
    
    # 方法2: 使用便捷函数
    print("\n2. 使用便捷函数:")
    openai_engine2 = create_openai_engine(
        api_key=os.getenv("OPENAI_API_KEY", "your-openai-key")
    )
    deepseek_engine2 = create_deepseek_engine(
        api_key=os.getenv("DEEPSEEK_API_KEY", "your-deepseek-key")
    )
    print(f"OpenAI 引擎: {openai_engine2.default_model}")
    print(f"DeepSeek 引擎: {deepseek_engine2.default_model}")


async def example_unified_client():
    """统一客户端使用示例"""
    print("\n=== 统一客户端使用示例 ===")
    
    # 配置 OpenAI
    openai_config = LLMConfig(
        engine_type="openai",
        api_key=os.getenv("OPENAI_API_KEY", "your-openai-key"),
        model="gpt-4"
    )
    
    # 配置 DeepSeek
    deepseek_config = LLMConfig(
        engine_type="deepseek",
        api_key=os.getenv("DEEPSEEK_API_KEY", "your-deepseek-key"),
        deepseek_api_key=os.getenv("DEEPSEEK_API_KEY", "your-deepseek-key"),
        model="deepseek-chat"
    )
    
    # 创建客户端
    print("\n1. 使用 OpenAI:")
    client = UnifiedLLMClient(openai_config)
    print(f"当前引擎: {client.get_engine_type()}, 模型: {client.get_model()}")
    
    print("\n2. 切换到 DeepSeek:")
    client = UnifiedLLMClient(deepseek_config)
    print(f"当前引擎: {client.get_engine_type()}, 模型: {client.get_model()}")


async def example_runtime_switching():
    """运行时切换示例"""
    print("\n=== 运行时切换示例 ===")
    
    # 初始配置为 OpenAI
    config = LLMConfig(
        engine_type="openai",
        api_key=os.getenv("OPENAI_API_KEY", "your-openai-key"),
        model="gpt-4"
    )
    
    client = UnifiedLLMClient(config)
    print(f"初始引擎: {client.get_engine_type()}, 模型: {client.get_model()}")
    
    # 运行时切换到 DeepSeek
    client.switch_engine(
        engine_type="deepseek",
        api_key=os.getenv("DEEPSEEK_API_KEY", "your-deepseek-key"),
        model="deepseek-chat"
    )
    print(f"切换后引擎: {client.get_engine_type()}, 模型: {client.get_model()}")


async def example_real_request():
    """真实请求示例（需要有效的 API 密钥）"""
    print("\n=== 真实请求示例 ===")
    
    # 检查是否有 API 密钥
    openai_key = os.getenv("OPENAI_API_KEY")
    deepseek_key = os.getenv("DEEPSEEK_API_KEY")
    
    if not openai_key and not deepseek_key:
        print("请设置 OPENAI_API_KEY 或 DEEPSEEK_API_KEY 环境变量来运行真实请求示例")
        return
    
    # 准备测试消息
    messages = [
        Message(role="user", content="你好，请简单介绍一下你自己。")
    ]
    
    request = AgentExecuteRequest(
        session_id="test-session",
        messages=messages,
        model=None,  # 使用默认模型
        temperature=0.7,
        max_tokens=100
    )
    
    # 测试可用的引擎
    if openai_key:
        print("\n测试 OpenAI:")
        config = LLMConfig(
            engine_type="openai",
            api_key=openai_key,
            model="gpt-3.5-turbo"  # 使用便宜的模型进行测试
        )
        client = UnifiedLLMClient(config)
        
        try:
            response = await client.execute_agent(request)
            print(f"响应: {response.message.content[:100]}...")
            print(f"成本: ${response.cost:.6f}")
            print(f"执行时间: {response.execution_time:.2f}s")
        except Exception as e:
            print(f"OpenAI 请求失败: {e}")
    
    if deepseek_key:
        print("\n测试 DeepSeek:")
        config = LLMConfig(
            engine_type="deepseek",
            api_key=deepseek_key,
            deepseek_api_key=deepseek_key,
            model="deepseek-chat"
        )
        client = UnifiedLLMClient(config)
        
        try:
            response = await client.execute_agent(request)
            print(f"响应: {response.message.content[:100]}...")
            print(f"成本: ${response.cost:.6f}")
            print(f"执行时间: {response.execution_time:.2f}s")
        except Exception as e:
            print(f"DeepSeek 请求失败: {e}")


def show_supported_engines():
    """显示支持的引擎"""
    print("=== 支持的引擎 ===")
    engines = ModelFactory.get_supported_engines()
    for engine in engines:
        default_model = ModelFactory.get_default_model(engine)
        print(f"- {engine}: 默认模型 {default_model}")


async def main():
    """主函数"""
    print("LLM 模型切换示例")
    print("=" * 50)
    
    show_supported_engines()
    
    await example_basic_usage()
    await example_unified_client()
    await example_runtime_switching()
    
    # 如果你有有效的 API 密钥，取消注释下面这行
    # await example_real_request()
    
    print("\n" + "=" * 50)
    print("示例完成！")
    print("\n使用说明:")
    print("1. 设置环境变量 OPENAI_API_KEY 和/或 DEEPSEEK_API_KEY")
    print("2. 在代码中只需要一行代码就能切换模型:")
    print("   agent = create_openai_engine(api_key)  # OpenAI")
    print("   agent = create_deepseek_engine(api_key)  # DeepSeek")
    print("3. 或者使用统一客户端进行运行时切换")


if __name__ == "__main__":
    asyncio.run(main())
