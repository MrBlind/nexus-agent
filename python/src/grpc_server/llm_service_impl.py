"""
gRPC LLM 服务实现 - 支持智能模型选择和配置管理
"""

import time
from datetime import datetime
from typing import Dict, Any
import structlog
import grpc
from google.protobuf.timestamp_pb2 import Timestamp

from ..proto import llm_service_pb2, llm_service_pb2_grpc
from ..core.unified_llm_client import UnifiedLLMClient
from ..core.model_factory import ModelFactory
from ..core.model_registry import ModelRegistry
from ..core.model_selector import ModelSelector, ModelRequirements
from ..core.config_manager import ConfigManager
from ..models.request import Message, AgentExecuteRequest
from ..utils.config import LLMConfig

logger = structlog.get_logger()


class LLMServiceImpl(llm_service_pb2_grpc.LLMServiceServicer):
    """LLM gRPC 服务实现 - 支持智能模型选择"""
    
    def __init__(self):
        # 初始化核心组件
        self.config_manager = ConfigManager()
        self.model_registry = ModelRegistry()
        self.model_selector = ModelSelector(self.model_registry)
        
        # 获取支持的模型信息
        self.supported_models = self._get_supported_models()
        
        logger.info("LLM gRPC 服务初始化完成",
                   available_providers=len(self.model_registry.get_available_providers()),
                   total_models=sum(len(models) for models in self.model_registry.get_available_models().values()))
    
    def _get_supported_models(self) -> Dict[str, Any]:
        """获取支持的模型信息"""
        providers = {}
        
        # 从模型注册表获取所有模型信息
        all_models_info = self.model_registry.get_all_models_info()
        
        for provider_name, models_info in all_models_info.items():
            # 提取模型名称和定价信息
            model_names = list(models_info.keys())
            pricing = {name: info["cost_per_1k_tokens"] for name, info in models_info.items()}
            
            # 获取默认模型
            default_model = ""
            if model_names:
                # 选择成本最低的模型作为默认模型
                default_model = min(models_info.keys(), 
                                  key=lambda m: models_info[m]["cost_per_1k_tokens"])
            
            providers[provider_name] = llm_service_pb2.ProviderInfo(
                name=provider_name,
                models=model_names,
                default_model=default_model,
                requires_key=True,
                pricing=pricing
            )
        
        return providers
    
    async def ExecuteAgent(self, request, context):
        """执行 LLM 代理请求 - 支持智能模型选择"""
        start_time = time.time()
        
        try:
            logger.info("收到 gRPC ExecuteAgent 请求", 
                       session_id=request.session_id,
                       provider=request.provider,
                       model=request.model,
                       temperature=request.temperature,
                       max_tokens=request.max_tokens)
            
            # 转换 gRPC 请求到内部格式
            messages = [
                Message(
                    role=msg.role,
                    content=msg.content,
                    image_url=msg.image_url if msg.image_url else None
                )
                for msg in request.messages
            ]
            
            # 检测是否需要视觉支持
            has_images = any(msg.image_url for msg in messages)
            
            # 创建模型需求
            requirements = ModelRequirements(
                supports_vision=has_images if has_images else None,
                performance_priority="balanced"  # 可以根据请求参数调整
            )
            
            # 如果请求明确指定了 provider 和 model，优先使用
            if request.provider and request.model:
                # 验证模型是否可用
                is_valid, validation_message = self.model_registry.validate_model_request(
                    request.provider, request.model
                )
                
                if is_valid:
                    selected_provider = request.provider
                    selected_model = request.model
                    selection_reason = "explicitly_specified"
                else:
                    # 如果指定的模型不可用，回退到智能选择
                    logger.warning("请求指定的模型不可用，回退到智能选择",
                                 requested_provider=request.provider,
                                 requested_model=request.model,
                                 reason=validation_message)
                    selected_provider, selected_model, selection_reason = self.model_selector.select_best_model(
                        preferred_provider=request.provider,
                        preferred_model=request.model,
                        requirements=requirements
                    )
            else:
                # 没有指定，使用智能选择
                selected_provider, selected_model, selection_reason = self.model_selector.select_best_model(
                    preferred_provider=None,
                    preferred_model=None,
                    requirements=requirements
                )
            
            logger.info("模型选择完成",
                       requested_provider=request.provider,
                       requested_model=request.model,
                       selected_provider=selected_provider,
                       selected_model=selected_model,
                       reason=selection_reason)
            
            # 获取提供商配置
            provider_config = self.config_manager.get_provider_config(selected_provider)
            if not provider_config:
                raise ValueError(f"Provider '{selected_provider}' configuration not found")
            
            # 创建 LLM 配置 - 使用默认配置并设置引擎类型和模型
            llm_config = LLMConfig(
                engine_type=selected_provider,
                model=selected_model,
                timeout=60
            )
            
            # 创建统一 LLM 客户端
            client = UnifiedLLMClient(llm_config)
            
            # 创建执行请求
            agent_request = AgentExecuteRequest(
                session_id=request.session_id,
                messages=messages,
                model=selected_model,
                temperature=request.temperature if request.temperature > 0 else 0.7,
                max_tokens=request.max_tokens if request.max_tokens > 0 else 2000,
                tools=list(request.tools) if request.tools else None
            )
            
            # 执行请求
            response = await client.execute_agent(agent_request)
            
            # 转换响应到 gRPC 格式
            grpc_response = llm_service_pb2.ExecuteAgentResponse(
                session_id=response.session_id,
                message=llm_service_pb2.Message(
                    role=response.message.role,
                    content=response.message.content
                ),
                usage=llm_service_pb2.Usage(
                    prompt_tokens=response.usage.get("prompt_tokens", 0),
                    completion_tokens=response.usage.get("completion_tokens", 0),
                    total_tokens=response.usage.get("total_tokens", 0)
                ),
                cost=response.cost,
                execution_time=response.execution_time
            )
            
            # 添加工具调用（如果有）
            if response.tool_calls:
                for tool_call in response.tool_calls:
                    grpc_tool_call = llm_service_pb2.ToolCall(
                        id=tool_call["id"],
                        type=tool_call["type"],
                        function=llm_service_pb2.ToolFunction(
                            name=tool_call["function"]["name"],
                            arguments=tool_call["function"]["arguments"]
                        )
                    )
                    grpc_response.tool_calls.append(grpc_tool_call)
            
            # 添加时间戳
            timestamp = Timestamp()
            timestamp.FromDatetime(datetime.utcnow())
            grpc_response.timestamp.CopyFrom(timestamp)
            
            logger.info("gRPC ExecuteAgent 请求完成",
                       session_id=request.session_id,
                       provider=selected_provider,
                       model=selected_model,
                       cost=response.cost,
                       execution_time=response.execution_time)
            
            return grpc_response
            
        except Exception as e:
            logger.error("gRPC ExecuteAgent 请求失败",
                        session_id=request.session_id,
                        error=str(e))
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(f"执行失败: {str(e)}")
            return llm_service_pb2.ExecuteAgentResponse()
    
    async def ExecuteAgentStream(self, request, context):
        """执行 LLM 代理请求 - 流式响应"""
        try:
            logger.info("gRPC ExecuteAgentStream 请求开始",
                       session_id=request.session_id,
                       provider=request.provider,
                       model=request.model)

            # 转换消息格式
            messages = [
                Message(role=msg.role, content=msg.content, image_url=msg.image_url)
                for msg in request.messages
            ]
            
            # 智能模型选择
            requirements = ModelRequirements(
                max_cost_per_1k=0.1,  # 默认成本限制
                min_tokens=4000,     # 默认上下文长度要求
                supports_vision=any(msg.image_url for msg in messages)
            )
            
            selected_provider, selected_model, selection_reason = self.model_selector.select_best_model(
                preferred_provider=request.provider if request.provider else None,
                preferred_model=request.model if request.model else None,
                requirements=requirements
            )
            
            logger.info("智能模型选择完成",
                       session_id=request.session_id,
                       selected_provider=selected_provider,
                       selected_model=selected_model,
                       selection_reason=selection_reason,
                       original_provider=request.provider,
                       original_model=request.model)
            
            # 获取提供商配置
            provider_config = self.config_manager.get_provider_config(selected_provider)
            if not provider_config:
                raise ValueError(f"Provider '{selected_provider}' configuration not found")
            
            # 创建 LLM 配置 - 使用默认配置并设置引擎类型和模型
            llm_config = LLMConfig(
                engine_type=selected_provider,
                model=selected_model,
                timeout=60
            )
            
            # 创建统一 LLM 客户端
            client = UnifiedLLMClient(llm_config)
            
            # 创建执行请求
            agent_request = AgentExecuteRequest(
                session_id=request.session_id,
                messages=messages,
                model=selected_model,
                temperature=request.temperature if request.temperature > 0 else 0.7,
                max_tokens=request.max_tokens if request.max_tokens > 0 else 2000,
                tools=list(request.tools) if request.tools else None
            )
            
            # 执行流式请求
            async for chunk in client.execute_agent_stream(agent_request):
                # 创建时间戳
                timestamp = Timestamp()
                timestamp.FromDatetime(datetime.utcnow())
                
                # 根据chunk类型创建相应的响应
                if chunk["type"] == "content_delta":
                    stream_response = llm_service_pb2.ExecuteAgentStreamResponse(
                        session_id=chunk["session_id"],
                        type=llm_service_pb2.ExecuteAgentStreamResponse.CONTENT_DELTA,
                        content_delta=chunk["content"],
                        timestamp=timestamp
                    )
                    yield stream_response
                
                elif chunk["type"] == "tool_call":
                    tool_call = chunk["tool_call"]
                    grpc_tool_call = llm_service_pb2.ToolCall(
                        id=tool_call["id"],
                        type=tool_call["type"],
                        function=llm_service_pb2.ToolFunction(
                            name=tool_call["function"]["name"],
                            arguments=tool_call["function"]["arguments"]
                        )
                    )
                    
                    stream_response = llm_service_pb2.ExecuteAgentStreamResponse(
                        session_id=chunk["session_id"],
                        type=llm_service_pb2.ExecuteAgentStreamResponse.TOOL_CALL,
                        tool_call=grpc_tool_call,
                        timestamp=timestamp
                    )
                    yield stream_response
                
                elif chunk["type"] == "final_response":
                    usage = chunk.get("usage", {})
                    grpc_usage = llm_service_pb2.Usage(
                        prompt_tokens=usage.get("prompt_tokens", 0),
                        completion_tokens=usage.get("completion_tokens", 0),
                        total_tokens=usage.get("total_tokens", 0)
                    )
                    
                    stream_response = llm_service_pb2.ExecuteAgentStreamResponse(
                        session_id=chunk["session_id"],
                        type=llm_service_pb2.ExecuteAgentStreamResponse.FINAL_RESPONSE,
                        usage=grpc_usage,
                        cost=chunk.get("cost", 0.0),  # 从chunk顶级获取cost，而不是从usage中
                        execution_time=chunk.get("execution_time", 0.0),
                        timestamp=timestamp
                    )
                    yield stream_response
                
                elif chunk["type"] == "error":
                    stream_response = llm_service_pb2.ExecuteAgentStreamResponse(
                        session_id=chunk["session_id"],
                        type=llm_service_pb2.ExecuteAgentStreamResponse.ERROR,
                        error_message=chunk["error"],
                        timestamp=timestamp
                    )
                    yield stream_response
                    break
            
            logger.info("gRPC ExecuteAgentStream 请求完成",
                       session_id=request.session_id,
                       provider=selected_provider,
                       model=selected_model)
            
        except Exception as e:
            logger.error("gRPC ExecuteAgentStream 请求失败",
                        session_id=request.session_id,
                        error=str(e))
            
            # 发送错误响应
            timestamp = Timestamp()
            timestamp.FromDatetime(datetime.utcnow())
            
            error_response = llm_service_pb2.ExecuteAgentStreamResponse(
                session_id=request.session_id,
                type=llm_service_pb2.ExecuteAgentStreamResponse.ERROR,
                error_message=str(e),
                timestamp=timestamp
            )
            yield error_response
    
    async def GetSupportedModels(self, request, context):
        """获取支持的模型列表"""
        try:
            logger.info("收到 gRPC GetSupportedModels 请求")
            
            response = llm_service_pb2.GetSupportedModelsResponse()
            
            for provider_name, provider_info in self.supported_models.items():
                response.providers[provider_name].CopyFrom(provider_info)
            
            logger.info("gRPC GetSupportedModels 请求完成",
                       providers_count=len(self.supported_models))
            
            return response
            
        except Exception as e:
            logger.error("gRPC GetSupportedModels 请求失败", error=str(e))
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(f"获取模型列表失败: {str(e)}")
            return llm_service_pb2.GetSupportedModelsResponse()
    
    async def ValidateConfig(self, request, context):
        """验证配置"""
        try:
            logger.info("收到 gRPC ValidateConfig 请求",
                       provider=request.provider,
                       model=request.model)
            
            # 验证提供商和模型
            is_valid, validation_message = self.model_registry.validate_model_request(
                request.provider, request.model
            )
            
            if not is_valid:
                return llm_service_pb2.ValidateConfigResponse(
                    valid=False,
                    error_message=validation_message
                )
            
            # 验证温度参数
            if request.temperature < 0 or request.temperature > 2:
                return llm_service_pb2.ValidateConfigResponse(
                    valid=False,
                    error_message="温度参数必须在 0-2 之间"
                )
            
            # 验证最大令牌数
            if request.max_tokens <= 0 or request.max_tokens > 200000:
                return llm_service_pb2.ValidateConfigResponse(
                    valid=False,
                    error_message="最大令牌数必须在 1-200000 之间"
                )
            
            # 检查模型的最大令牌数限制
            model_info = self.model_registry.get_model_info(request.provider, request.model)
            if model_info and request.max_tokens > model_info.max_tokens:
                return llm_service_pb2.ValidateConfigResponse(
                    valid=False,
                    error_message=f"请求的最大令牌数 ({request.max_tokens}) 超过模型限制 ({model_info.max_tokens})"
                )
            
            logger.info("gRPC ValidateConfig 请求完成", valid=True)
            
            return llm_service_pb2.ValidateConfigResponse(
                valid=True,
                error_message=""
            )
            
        except Exception as e:
            logger.error("gRPC ValidateConfig 请求失败", error=str(e))
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(f"配置验证失败: {str(e)}")
            return llm_service_pb2.ValidateConfigResponse(
                valid=False,
                error_message=f"验证失败: {str(e)}"
            )
    
    async def HealthCheck(self, request, context):
        """健康检查"""
        try:
            logger.debug("收到 gRPC HealthCheck 请求")
            
            # 检查各组件状态
            available_providers = self.model_registry.get_available_providers()
            total_models = sum(len(models) for models in self.model_registry.get_available_models().values())
            
            timestamp = Timestamp()
            timestamp.FromDatetime(datetime.utcnow())
            
            response = llm_service_pb2.HealthCheckResponse(
                status="healthy",
                version="2.0.0",
                timestamp=timestamp,
                details={
                    "available_providers": ",".join(available_providers),
                    "providers_count": str(len(available_providers)),
                    "total_models": str(total_models),
                    "model_registry": "active",
                    "model_selector": "active",
                    "config_manager": "active"
                }
            )
            
            return response
            
        except Exception as e:
            logger.error("gRPC HealthCheck 请求失败", error=str(e))
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(f"健康检查失败: {str(e)}")
            return llm_service_pb2.HealthCheckResponse(
                status="unhealthy",
                version="2.0.0"
            )
