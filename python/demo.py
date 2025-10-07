#!/usr/bin/env python3
"""
LLM 模型切换演示脚本
展示如何在实际项目中一行代码切换模型
"""

import os
import asyncio
from src.core.model_factory import create_openai_engine, create_deepseek_engine
from src.models.request import Message, AgentExecuteRequest


async def demo_model_switching():
    """演示模型切换功能"""
    print("🚀 LLM 模型切换演示")
    print("=" * 50)
    
    # 准备测试消息
    messages = [
        Message(role="user", content="请用一句话介绍你自己，并说明你的优势。")
    ]
    
    request = AgentExecuteRequest(
        session_id="demo-session",
        messages=messages,
        temperature=0.7,
        max_tokens=100
    )
    
    # 检查 API 密钥
    openai_key = os.getenv("OPENAI_API_KEY")
    deepseek_key = os.getenv("DEEPSEEK_API_KEY")
    
    if not openai_key and not deepseek_key:
        print("⚠️  请设置环境变量:")
        print("   export OPENAI_API_KEY='your-openai-key'")
        print("   export DEEPSEEK_API_KEY='your-deepseek-key'")
        print("\n💡 演示模型创建（不发送请求）:")
        
        # 演示模型创建
        print("\n1️⃣  创建 OpenAI 引擎:")
        print("   agent = create_openai_engine(api_key='your-key')")
        openai_agent = create_openai_engine(api_key="demo-key")
        print(f"   ✅ 创建成功: {openai_agent.default_model}")
        
        print("\n2️⃣  切换到 DeepSeek（只需修改一行）:")
        print("   agent = create_deepseek_engine(api_key='your-key')")
        deepseek_agent = create_deepseek_engine(api_key="demo-key")
        print(f"   ✅ 创建成功: {deepseek_agent.default_model}")
        
        print("\n🎯 就是这么简单！一行代码切换模型！")
        return
    
    # 实际 API 调用演示
    print("🔥 实际 API 调用演示:")
    
    if openai_key:
        print("\n1️⃣  使用 OpenAI:")
        print("   agent = create_openai_engine(api_key)")
        
        try:
            # 一行代码创建 OpenAI 引擎
            agent = create_openai_engine(api_key=openai_key, model="gpt-3.5-turbo")
            
            response = await agent.execute_agent(request)
            print(f"   📝 响应: {response.message.content}")
            print(f"   💰 成本: ${response.cost:.6f}")
            print(f"   ⏱️  耗时: {response.execution_time:.2f}s")
            
        except Exception as e:
            print(f"   ❌ OpenAI 调用失败: {e}")
    
    if deepseek_key:
        print("\n2️⃣  切换到 DeepSeek（只需修改一行）:")
        print("   agent = create_deepseek_engine(api_key)")
        
        try:
            # 一行代码切换到 DeepSeek 引擎
            agent = create_deepseek_engine(api_key=deepseek_key)
            
            response = await agent.execute_agent(request)
            print(f"   📝 响应: {response.message.content}")
            print(f"   💰 成本: ${response.cost:.6f}")
            print(f"   ⏱️  耗时: {response.execution_time:.2f}s")
            
        except Exception as e:
            print(f"   ❌ DeepSeek 调用失败: {e}")
    
    print("\n" + "=" * 50)
    print("🎉 演示完成！")
    print("\n💡 关键优势:")
    print("   • 一行代码切换模型")
    print("   • 统一的接口，无需修改业务逻辑")
    print("   • 易于扩展新的 LLM 提供商")
    print("   • 支持运行时动态切换")


def show_usage_examples():
    """显示使用示例"""
    print("\n📚 使用示例:")
    print("-" * 30)
    
    print("\n🔧 方法一：直接切换")
    print("""
# 使用 OpenAI
agent = create_openai_engine(api_key="your-openai-key")

# 切换到 DeepSeek（只需修改这一行）
agent = create_deepseek_engine(api_key="your-deepseek-key")
""")
    
    print("🔧 方法二：工厂模式")
    print("""
from src.core.model_factory import ModelFactory

# 使用 OpenAI
agent = ModelFactory.create_engine("openai", api_key="your-key")

# 切换到 DeepSeek
agent = ModelFactory.create_engine("deepseek", api_key="your-key")
""")
    
    print("🔧 方法三：统一客户端")
    print("""
from src.core.unified_llm_client import UnifiedLLMClient

# 创建客户端
client = UnifiedLLMClient(config)

# 运行时切换
client.switch_engine("deepseek", api_key="your-key")
""")


if __name__ == "__main__":
    try:
        asyncio.run(demo_model_switching())
        show_usage_examples()
    except KeyboardInterrupt:
        print("\n\n👋 演示已取消")
    except Exception as e:
        print(f"\n❌ 演示出错: {e}")
        print("请检查依赖是否安装完整: pip install -r requirements.txt")
