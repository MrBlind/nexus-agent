"""
智能模型选择器 - 根据需求和约束选择最佳模型
"""

from typing import Dict, List, Tuple, Any, Optional
from dataclasses import dataclass
import structlog

from .model_registry import ModelRegistry, ModelInfo, ProviderInfo

logger = structlog.get_logger()


@dataclass
class ModelRequirements:
    """模型需求定义"""
    max_cost_per_1k: Optional[float] = None          # 最大成本限制
    min_tokens: Optional[int] = None                 # 最小token数要求
    supports_vision: Optional[bool] = None           # 是否需要视觉支持
    preferred_providers: Optional[List[str]] = None  # 首选提供商列表
    excluded_providers: Optional[List[str]] = None   # 排除的提供商列表
    performance_priority: str = "balanced"           # 性能优先级: "cost", "performance", "balanced"


@dataclass
class ModelCandidate:
    """模型候选者"""
    provider: str
    model: str
    model_info: ModelInfo
    score: float  # 综合评分
    reason: str   # 选择原因


class ModelSelector:
    """智能模型选择器"""
    
    def __init__(self, registry: ModelRegistry):
        self.registry = registry
        
        # 性能评分权重配置
        self.scoring_weights = {
            "cost": {"cost": 0.7, "performance": 0.2, "availability": 0.1},
            "performance": {"cost": 0.2, "performance": 0.7, "availability": 0.1},
            "balanced": {"cost": 0.4, "performance": 0.4, "availability": 0.2}
        }
        
        # 模型性能评分（基于经验和基准测试）
        self.performance_scores = {
            "openai": {
                "gpt-4": 95,
                "gpt-4-turbo": 92,
                "gpt-4o": 90,
                "gpt-4o-mini": 75,
                "gpt-3.5-turbo": 70
            },
            "deepseek": {
                "deepseek-chat": 80,
                "deepseek-reasoner": 85
            },
            "anthropic": {
                "claude-3-opus-20240229": 93,
                "claude-3-sonnet-20240229": 88,
                "claude-3-5-sonnet-20241022": 90,
                "claude-3-haiku-20240307": 82
            },
            "qwen": {
                "qwen-turbo": 72,
                "qwen-plus": 78,
                "qwen-max": 85,
                "qwen-max-longcontext": 83
            },
            "ernie": {
                "ernie-bot-turbo": 70,
                "ernie-bot": 73,
                "ernie-bot-4": 80,
                "ernie-speed": 65
            },
            "chatglm": {
                "glm-4": 78,
                "glm-4v": 76,
                "glm-3-turbo": 68
            }
        }
        
        logger.info("ModelSelector 初始化完成")
    
    def select_best_model(self, 
                         preferred_provider: str = None, 
                         preferred_model: str = None,
                         requirements: ModelRequirements = None) -> Tuple[str, str, str]:
        """
        智能选择最佳模型
        
        Args:
            preferred_provider: 首选提供商
            preferred_model: 首选模型
            requirements: 模型需求
        
        Returns:
            (provider, model, reason) 元组
        """
        
        # 确保注册表是最新的
        self.registry.reload_if_needed()
        
        requirements = requirements or ModelRequirements()
        
        # 1. 尝试使用首选配置
        if preferred_provider and preferred_model:
            is_valid, reason = self.registry.validate_model_request(preferred_provider, preferred_model)
            if is_valid:
                if self._meets_requirements(preferred_provider, preferred_model, requirements):
                    return preferred_provider, preferred_model, f"使用首选配置: {preferred_provider}/{preferred_model}"
                else:
                    logger.warning("首选模型不满足需求要求", 
                                 provider=preferred_provider, 
                                 model=preferred_model)
        
        # 2. 根据需求筛选候选模型
        candidates = self._get_candidates(requirements)
        
        if not candidates:
            # 3. 如果没有候选模型，使用兜底策略
            fallback_provider, fallback_model, fallback_reason = self._get_fallback_model()
            logger.warning("没有找到满足需求的模型，使用兜底模型", 
                          provider=fallback_provider, 
                          model=fallback_model)
            return fallback_provider, fallback_model, fallback_reason
        
        # 4. 按评分排序并选择最佳模型
        best_candidate = max(candidates, key=lambda c: c.score)
        
        logger.info("智能选择模型完成", 
                   provider=best_candidate.provider,
                   model=best_candidate.model,
                   score=best_candidate.score,
                   reason=best_candidate.reason,
                   total_candidates=len(candidates))
        
        return best_candidate.provider, best_candidate.model, best_candidate.reason
    
    def _get_candidates(self, requirements: ModelRequirements) -> List[ModelCandidate]:
        """获取符合需求的候选模型"""
        candidates = []
        
        for provider_name in self.registry.get_available_providers():
            # 检查提供商是否被排除
            if requirements.excluded_providers and provider_name in requirements.excluded_providers:
                continue
            
            provider_info = self.registry.get_provider_info(provider_name)
            if not provider_info or not provider_info.is_available:
                continue
            
            for model_name, model_info in provider_info.models.items():
                if self._meets_requirements(provider_name, model_name, requirements):
                    score = self._calculate_score(provider_name, model_name, model_info, requirements)
                    reason = self._generate_selection_reason(provider_name, model_name, model_info, requirements)
                    
                    candidates.append(ModelCandidate(
                        provider=provider_name,
                        model=model_name,
                        model_info=model_info,
                        score=score,
                        reason=reason
                    ))
        
        return candidates
    
    def _meets_requirements(self, provider: str, model: str, requirements: ModelRequirements) -> bool:
        """检查模型是否满足需求"""
        model_info = self.registry.get_model_info(provider, model)
        if not model_info:
            return False
        
        # 检查成本限制
        if requirements.max_cost_per_1k and model_info.cost_per_1k_tokens > requirements.max_cost_per_1k:
            return False
        
        # 检查最小token数
        if requirements.min_tokens and model_info.max_tokens < requirements.min_tokens:
            return False
        
        # 检查视觉支持
        if requirements.supports_vision is not None and model_info.supports_vision != requirements.supports_vision:
            return False
        
        # 检查首选提供商
        if requirements.preferred_providers and provider not in requirements.preferred_providers:
            return False
        
        return True
    
    def _calculate_score(self, provider: str, model: str, model_info: ModelInfo, requirements: ModelRequirements) -> float:
        """计算模型综合评分"""
        weights = self.scoring_weights.get(requirements.performance_priority, self.scoring_weights["balanced"])
        
        # 成本评分 (成本越低评分越高)
        max_cost = 0.1  # 假设最高成本为 0.1
        cost_score = max(0, (max_cost - model_info.cost_per_1k_tokens) / max_cost * 100)
        
        # 性能评分
        performance_score = self.performance_scores.get(provider, {}).get(model, 50)
        
        # 可用性评分 (基于提供商的稳定性和可靠性)
        availability_scores = {
            "openai": 95,
            "anthropic": 90,
            "deepseek": 85,
            "qwen": 80,
            "ernie": 75,
            "chatglm": 70
        }
        availability_score = availability_scores.get(provider, 50)
        
        # 首选提供商加分
        preference_bonus = 0
        if requirements.preferred_providers and provider in requirements.preferred_providers:
            preference_bonus = 10
        
        # 计算加权总分
        total_score = (
            cost_score * weights["cost"] +
            performance_score * weights["performance"] +
            availability_score * weights["availability"] +
            preference_bonus
        )
        
        return total_score
    
    def _generate_selection_reason(self, provider: str, model: str, model_info: ModelInfo, requirements: ModelRequirements) -> str:
        """生成选择原因"""
        reasons = []
        
        if requirements.performance_priority == "cost":
            reasons.append(f"成本优化 (${model_info.cost_per_1k_tokens:.4f}/1k tokens)")
        elif requirements.performance_priority == "performance":
            performance_score = self.performance_scores.get(provider, {}).get(model, 50)
            reasons.append(f"性能优先 (评分: {performance_score})")
        else:
            reasons.append(f"平衡选择 (成本: ${model_info.cost_per_1k_tokens:.4f}, 性能优秀)")
        
        if model_info.supports_vision and requirements.supports_vision:
            reasons.append("支持视觉功能")
        
        if model_info.max_tokens > 32000:
            reasons.append("长上下文支持")
        
        return ", ".join(reasons)
    
    def _get_fallback_model(self) -> Tuple[str, str, str]:
        """获取兜底模型"""
        # 优先级顺序：OpenAI > Anthropic > DeepSeek > 其他
        fallback_priority = [
            ("openai", "gpt-3.5-turbo"),
            ("openai", "gpt-4o-mini"),
            ("anthropic", "claude-3-haiku-20240307"),
            ("deepseek", "deepseek-chat"),
            ("qwen", "qwen-turbo"),
            ("ernie", "ernie-speed"),
            ("chatglm", "glm-3-turbo")
        ]
        
        for provider, model in fallback_priority:
            is_valid, _ = self.registry.validate_model_request(provider, model)
            if is_valid:
                return provider, model, f"兜底模型: {provider}/{model}"
        
        # 如果所有兜底模型都不可用，返回第一个可用的模型
        available_models = self.registry.get_available_models()
        for provider, models in available_models.items():
            if models:
                return provider, models[0], f"最后兜底: {provider}/{models[0]}"
        
        # 如果没有任何可用模型，返回错误
        raise RuntimeError("没有任何可用的 LLM 模型")
    
    def get_model_recommendations(self, requirements: ModelRequirements = None) -> List[ModelCandidate]:
        """获取模型推荐列表"""
        requirements = requirements or ModelRequirements()
        candidates = self._get_candidates(requirements)
        
        # 按评分排序
        candidates.sort(key=lambda c: c.score, reverse=True)
        
        return candidates[:5]  # 返回前5个推荐
    
    def analyze_model_suitability(self, provider: str, model: str, requirements: ModelRequirements = None) -> Dict[str, Any]:
        """分析特定模型的适用性"""
        requirements = requirements or ModelRequirements()
        
        is_valid, validation_msg = self.registry.validate_model_request(provider, model)
        if not is_valid:
            return {
                "suitable": False,
                "reason": validation_msg,
                "score": 0,
                "details": {}
            }
        
        model_info = self.registry.get_model_info(provider, model)
        meets_req = self._meets_requirements(provider, model, requirements)
        score = self._calculate_score(provider, model, model_info, requirements)
        
        return {
            "suitable": meets_req,
            "score": score,
            "reason": self._generate_selection_reason(provider, model, model_info, requirements),
            "details": {
                "cost_per_1k_tokens": model_info.cost_per_1k_tokens,
                "max_tokens": model_info.max_tokens,
                "supports_vision": model_info.supports_vision,
                "performance_score": self.performance_scores.get(provider, {}).get(model, 50),
                "description": model_info.description
            }
        }

