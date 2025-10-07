#!/usr/bin/env python3
"""
默认配置自动切换演示
展示如何通过修改默认配置来自动调用不同的 LLM 接口
"""

import os
import sys
import asyncio
import structlog

# 添加项目路径
sys.path.append(os.path.dirname(__file__))

from src.core.unified_llm_client import UnifiedLLMClient
from src.utils.config import LLMConfig, LLMProvidersConfig, OpenAIConfig, DeepSeekConfig
from src.models.request import Message, AgentExecuteRequest

# 配置日志
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


def create_demo_config():
    """创建演示配置"""
    return LLMProvidersConfig(
        openai=OpenAIConfig(
            api_key=os.getenv("OPENAI_API_KEY", "demo-openai-key"),
            base_url="https://api.openai.com/v1"
        ),
        deepseek=DeepSeekConfig(
            api_key=os.getenv("DEEPSEEK_API_KEY", "demo-deepseek-key"),
            base_url="https://api.deepseek.com"
        )
    )


async def demo_config_switching():
    """演示配置切换"""
    print("🚀 默认配置自动切换演示")
    print("🎯 展示如何通过修改配置自动调用不同的 LLM 接口")
    print("=" * 70)
    
    # 创建提供商配置
    providers_config = create_demo_config()
    
    # 准备测试消息
    messages = [
        Message(role="user", content="请用一句话介绍你自己")
    ]
    
    test_request = AgentExecuteRequest(
        session_id="demo-session",
        messages=messages,
        temperature=0.7,
        max_tokens=50
    )
    
    # 演示场景
    scenarios = [
        {
            "name": "场景 1: 默认使用 OpenAI",
            "config": LLMConfig(
                engine_type="openai",  # 修改这里切换引擎
                model=None,  # 使用默认模型
                providers=providers_config
            ),
            "description": "只需设置 engine_type='openai'，系统自动使用 OpenAI 的配置和默认模型"
        },
        {
            "name": "场景 2: 切换到 DeepSeek",
            "config": LLMConfig(
                engine_type="deepseek",  # 修改这里切换引擎
                model=None,  # 使用默认模型
                providers=providers_config
            ),
            "description": "只需修改 engine_type='deepseek'，系统自动切换到 DeepSeek 配置"
        },
        {
            "name": "场景 3: 使用 OpenAI 的特定模型",
            "config": LLMConfig(
                engine_type="openai",
                model="gpt-3.5-turbo",  # 指定模型
                providers=providers_config
            ),
            "description": "指定使用 OpenAI 的 gpt-3.5-turbo 模型"
        },
        {
            "name": "场景 4: 使用 DeepSeek 的代码模型",
            "config": LLMConfig(
                engine_type="deepseek",
                model="deepseek-coder",  # 指定模型
                providers=providers_config
            ),
            "description": "指定使用 DeepSeek 的代码专用模型"
        }
    ]
    
    for i, scenario in enumerate(scenarios, 1):
        print(f"\n{i}️⃣ {scenario['name']}")
        print(f"   📝 {scenario['description']}")
        
        try:
            # 创建客户端 - 系统会自动根据配置选择引擎
            client = UnifiedLLMClient(scenario['config'])
            
            print(f"   🔧 自动选择: {client.get_engine_type()} 引擎")
            print(f"   🎯 使用模型: {client.get_model()}")
            
            # 模拟 API 调用（实际环境中会发送真实请求）
            print(f"   📡 模拟调用: {client.get_engine_type()} API")
            print(f"   ✅ 配置自动切换成功")
            
        except Exception as e:
            print(f"   ❌ 配置切换失败: {e}")


def demo_runtime_switching():
    """演示运行时切换"""
    print(f"\n{'='*20} 运行时动态切换演示 {'='*20}")
    
    providers_config = create_demo_config()
    
    # 创建初始配置
    config = LLMConfig(
        engine_type="openai",
        model="gpt-4",
        providers=providers_config
    )
    
    try:
        client = UnifiedLLMClient(config)
        
        print(f"🔄 运行时切换演示:")
        print(f"   1️⃣ 初始状态: {client.get_engine_type()} - {client.get_model()}")
        
        # 运行时切换到 DeepSeek
        client.switch_engine("deepseek")
        print(f"   2️⃣ 切换引擎: {client.get_engine_type()} - {client.get_model()}")
        
        # 切换模型
        client.switch_engine("deepseek", model="deepseek-coder")
        print(f"   3️⃣ 切换模型: {client.get_engine_type()} - {client.get_model()}")
        
        # 切换回 OpenAI
        client.switch_engine("openai", model="gpt-3.5-turbo")
        print(f"   4️⃣ 切换回来: {client.get_engine_type()} - {client.get_model()}")
        
        print("   ✅ 运行时切换演示完成")
        
    except Exception as e:
        print(f"   ❌ 运行时切换失败: {e}")


def demo_provider_availability():
    """演示提供商可用性检查"""
    print(f"\n{'='*20} 提供商可用性检查演示 {'='*20}")
    
    # 创建部分配置的提供商（模拟真实场景）
    providers_config = LLMProvidersConfig(
        openai=OpenAIConfig(
            api_key=os.getenv("OPENAI_API_KEY", ""),  # 可能有也可能没有
            base_url="https://api.openai.com/v1"
        ),
        deepseek=DeepSeekConfig(
            api_key=os.getenv("DEEPSEEK_API_KEY", ""),  # 可能有也可能没有
            base_url="https://api.deepseek.com"
        )
    )
    
    config = LLMConfig(
        engine_type="openai",
        providers=providers_config
    )
    
    try:
        client = UnifiedLLMClient(config)
        available = client.get_available_providers()
        
        print("🔍 提供商可用性检查:")
        for provider, is_available in available.items():
            if ModelFactory.is_engine_implemented(provider):
                status = "✅ 可用" if is_available else "❌ 缺少密钥"
                impl_status = "已实现"
            else:
                status = "⏳ 计划中"
                impl_status = "未实现"
            
            print(f"   • {provider}: {status} ({impl_status})")
        
        # 显示建议
        available_engines = [p for p, avail in available.items() if avail and ModelFactory.is_engine_implemented(p)]
        if available_engines:
            print(f"\n💡 建议使用: {', '.join(available_engines)}")
        else:
            print(f"\n⚠️  请配置至少一个提供商的 API 密钥")
        
    except Exception as e:
        print(f"❌ 可用性检查失败: {e}")


def show_configuration_examples():
    """显示配置示例"""
    print(f"\n{'='*20} 配置示例 {'='*20}")
    
    print("🔧 环境变量配置示例:")
    print("""
# 方法 1: 通过环境变量配置
export LLM_ENGINE_TYPE=openai          # 默认引擎
export LLM_MODEL=gpt-4                 # 默认模型
export LLM_PROVIDERS_OPENAI_API_KEY=your_openai_key
export LLM_PROVIDERS_DEEPSEEK_API_KEY=your_deepseek_key

# 方法 2: 通过 .env 文件配置
# 在 .env 文件中设置:
LLM_ENGINE_TYPE=deepseek               # 切换到 DeepSeek
LLM_MODEL=deepseek-chat               # 使用对话模型
""")
    
    print("💻 代码配置示例:")
    print("""
# 创建配置
config = LLMConfig(
    engine_type="openai",              # 只需修改这里
    model="gpt-4",                     # 可选：指定模型
    providers=providers_config
)

# 系统自动根据配置调用相应的接口
client = UnifiedLLMClient(config)     # 自动选择 OpenAI

# 运行时切换
client.switch_engine("deepseek")      # 切换到 DeepSeek
""")


def main():
    """主演示函数"""
    print("🌟 Nexus Agent - 默认配置自动切换功能演示")
    print("📋 功能: 根据配置自动调用相关的 LLM 接口")
    print("🎯 优势: 一行配置修改，自动切换整个 LLM 后端")
    
    # 运行演示
    asyncio.run(demo_config_switching())
    demo_runtime_switching()
    demo_provider_availability()
    show_configuration_examples()
    
    print(f"\n{'='*70}")
    print("🎉 演示完成！")
    
    print("\n✨ 核心特性总结:")
    print("   • 🔧 配置驱动: 修改配置自动切换 LLM 引擎")
    print("   • 🚀 零代码切换: 不需要修改业务逻辑代码")
    print("   • 🔄 运行时切换: 支持动态切换引擎和模型")
    print("   • 🛡️ 智能验证: 自动检查配置有效性")
    print("   • 📊 状态透明: 清晰显示当前使用的引擎和模型")
    
    print("\n🚀 使用步骤:")
    print("   1. 配置多个提供商的 API 密钥")
    print("   2. 设置默认引擎类型 (LLM_ENGINE_TYPE)")
    print("   3. 系统自动使用相应的配置和接口")
    print("   4. 需要切换时只需修改配置即可")
    
    print("\n💡 实际应用:")
    print("   • 开发环境使用便宜的 DeepSeek")
    print("   • 生产环境使用稳定的 OpenAI")
    print("   • A/B 测试不同模型的效果")
    print("   • 根据任务类型选择专用模型")


if __name__ == "__main__":
    try:
        main()
    except KeyboardInterrupt:
        print("\n\n👋 演示被中断")
    except Exception as e:
        logger.error("演示失败", error=str(e))
        print(f"\n❌ 演示出错: {e}")
        
# 导入必要的模块
from src.core.model_factory import ModelFactory
