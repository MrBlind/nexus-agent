"""
配置管理器 - 支持热重载和环境隔离的配置管理
"""

import os
import time
import json
from typing import Dict, Any, Optional, List, Tuple
from dataclasses import dataclass, asdict
from pathlib import Path
import structlog

from ..utils.config import load_config, LLMConfig
from .model_registry import ProviderInfo

logger = structlog.get_logger()


@dataclass
class EnvironmentConfig:
    """环境配置"""
    name: str
    description: str
    providers: Dict[str, ProviderInfo]
    default_provider: str = "openai"
    default_model: str = "gpt-3.5-turbo"
    rate_limits: Dict[str, int] = None  # 每个提供商的速率限制
    cost_limits: Dict[str, float] = None  # 每个提供商的成本限制


@dataclass
class GlobalSettings:
    """全局设置"""
    default_temperature: float = 0.7
    default_max_tokens: int = 2000
    request_timeout: int = 60
    retry_attempts: int = 3
    retry_delay: float = 1.0
    enable_caching: bool = True
    cache_ttl: int = 3600
    log_level: str = "INFO"


class ConfigManager:
    """配置管理器 - 支持热重载和环境隔离"""
    
    def __init__(self, config_dir: str = None):
        self.config_dir = Path(config_dir) if config_dir else Path.cwd() / "config"
        self.environments: Dict[str, EnvironmentConfig] = {}
        self.global_settings = GlobalSettings()
        self.current_environment = "default"
        self.last_reload = 0
        self.reload_interval = 300  # 5分钟检查一次
        self.config_files_mtime: Dict[str, float] = {}
        
        # 确保配置目录存在
        self.config_dir.mkdir(exist_ok=True)
        
        # 初始化加载
        self._load_all_configs()
        
        logger.info("ConfigManager 初始化完成", 
                   config_dir=str(self.config_dir),
                   environments=list(self.environments.keys()),
                   current_env=self.current_environment)
    
    def _load_all_configs(self):
        """加载所有配置"""
        try:
            # 加载基础配置
            self._load_base_config()
            
            # 加载环境配置
            self._load_environment_configs()
            
            # 加载全局设置
            self._load_global_settings()
            
            self.last_reload = time.time()
            logger.info("配置加载完成", environments_count=len(self.environments))
            
        except Exception as e:
            logger.error("配置加载失败", error=str(e))
            # 使用默认配置
            self._create_default_environment()
    
    def _load_base_config(self):
        """从环境变量加载基础配置"""
        try:
            config = load_config()
            providers_config = config.llm.providers
            
            # 创建默认环境
            providers = {}
            
            # 加载各个提供商配置
            if providers_config.openai.api_key:
                providers["openai"] = ProviderInfo(
                    name="openai",
                    api_key=providers_config.openai.api_key,
                    base_url=providers_config.openai.base_url,
                    models={},
                    is_available=True,
                    extra_config={"org_id": providers_config.openai.org_id}
                )
            
            if providers_config.deepseek.api_key:
                providers["deepseek"] = ProviderInfo(
                    name="deepseek",
                    api_key=providers_config.deepseek.api_key,
                    base_url=providers_config.deepseek.base_url,
                    models={},
                    is_available=True
                )
            
            if providers_config.anthropic.api_key:
                providers["anthropic"] = ProviderInfo(
                    name="anthropic",
                    api_key=providers_config.anthropic.api_key,
                    base_url=providers_config.anthropic.base_url,
                    models={},
                    is_available=True
                )
            
            if providers_config.qwen.api_key:
                providers["qwen"] = ProviderInfo(
                    name="qwen",
                    api_key=providers_config.qwen.api_key,
                    base_url=providers_config.qwen.base_url,
                    models={},
                    is_available=True,
                    extra_config={
                        "region": providers_config.qwen.region,
                        "workspace": providers_config.qwen.workspace
                    }
                )
            
            if providers_config.ernie.api_key and providers_config.ernie.secret_key:
                providers["ernie"] = ProviderInfo(
                    name="ernie",
                    api_key=providers_config.ernie.api_key,
                    base_url=providers_config.ernie.base_url,
                    models={},
                    is_available=True,
                    extra_config={"secret_key": providers_config.ernie.secret_key}
                )
            
            if providers_config.chatglm.api_key:
                providers["chatglm"] = ProviderInfo(
                    name="chatglm",
                    api_key=providers_config.chatglm.api_key,
                    base_url=providers_config.chatglm.base_url,
                    models={},
                    is_available=True
                )
            
            # 创建默认环境配置
            self.environments["default"] = EnvironmentConfig(
                name="default",
                description="Default environment from environment variables",
                providers=providers,
                default_provider=config.llm.engine_type,
                default_model=config.llm.model or "gpt-3.5-turbo"
            )
            
        except Exception as e:
            logger.error("加载基础配置失败", error=str(e))
            self._create_default_environment()
    
    def _load_environment_configs(self):
        """从配置文件加载环境配置"""
        env_config_dir = self.config_dir / "environments"
        if not env_config_dir.exists():
            return
        
        for config_file in env_config_dir.glob("*.json"):
            try:
                with open(config_file, 'r', encoding='utf-8') as f:
                    env_data = json.load(f)
                
                env_name = config_file.stem
                
                # 转换提供商配置
                providers = {}
                for provider_name, provider_data in env_data.get("providers", {}).items():
                    providers[provider_name] = ProviderInfo(
                        name=provider_data["name"],
                        api_key=provider_data["api_key"],
                        base_url=provider_data["base_url"],
                        models={},
                        is_available=provider_data.get("is_available", True),
                        extra_config=provider_data.get("extra_config", {})
                    )
                
                self.environments[env_name] = EnvironmentConfig(
                    name=env_name,
                    description=env_data.get("description", f"Environment: {env_name}"),
                    providers=providers,
                    default_provider=env_data.get("default_provider", "openai"),
                    default_model=env_data.get("default_model", "gpt-3.5-turbo"),
                    rate_limits=env_data.get("rate_limits", {}),
                    cost_limits=env_data.get("cost_limits", {})
                )
                
                # 记录文件修改时间
                self.config_files_mtime[str(config_file)] = config_file.stat().st_mtime
                
                logger.info("加载环境配置", 
                           environment=env_name,
                           providers_count=len(providers))
                
            except Exception as e:
                logger.error("加载环境配置文件失败", 
                           file=str(config_file), 
                           error=str(e))
    
    def _load_global_settings(self):
        """加载全局设置"""
        settings_file = self.config_dir / "global_settings.json"
        if not settings_file.exists():
            return
        
        try:
            with open(settings_file, 'r', encoding='utf-8') as f:
                settings_data = json.load(f)
            
            self.global_settings = GlobalSettings(**settings_data)
            self.config_files_mtime[str(settings_file)] = settings_file.stat().st_mtime
            
            logger.info("加载全局设置完成", settings=asdict(self.global_settings))
            
        except Exception as e:
            logger.error("加载全局设置失败", error=str(e))
    
    def _create_default_environment(self):
        """创建默认环境配置"""
        self.environments["default"] = EnvironmentConfig(
            name="default",
            description="Default fallback environment",
            providers={},
            default_provider="openai",
            default_model="gpt-3.5-turbo"
        )
        logger.warning("使用默认兜底环境配置")
    
    def get_provider_config(self, provider: str, environment: str = None) -> Optional[ProviderInfo]:
        """获取提供商配置，支持环境隔离"""
        env_name = environment or self.current_environment
        
        # 检查是否需要重新加载
        if self._should_reload():
            self.reload_configs()
        
        env_config = self.environments.get(env_name)
        if not env_config:
            # 尝试使用默认环境
            env_config = self.environments.get("default")
        
        if env_config:
            return env_config.providers.get(provider)
        
        return None
    
    def get_environment_config(self, environment: str = None) -> Optional[EnvironmentConfig]:
        """获取环境配置"""
        env_name = environment or self.current_environment
        return self.environments.get(env_name)
    
    def get_available_providers(self, environment: str = None) -> List[str]:
        """获取可用的提供商列表"""
        env_config = self.get_environment_config(environment)
        if not env_config:
            return []
        
        return [name for name, provider in env_config.providers.items() 
                if provider.is_available]
    
    def switch_environment(self, environment: str) -> bool:
        """切换环境"""
        if environment in self.environments:
            old_env = self.current_environment
            self.current_environment = environment
            logger.info("环境切换成功", 
                       from_env=old_env, 
                       to_env=environment)
            return True
        else:
            logger.error("环境不存在", environment=environment)
            return False
    
    def get_global_settings(self) -> GlobalSettings:
        """获取全局设置"""
        return self.global_settings
    
    def _should_reload(self) -> bool:
        """检查是否需要重新加载配置"""
        # 检查时间间隔
        if time.time() - self.last_reload < self.reload_interval:
            return False
        
        # 检查配置文件是否有变化
        for file_path, old_mtime in self.config_files_mtime.items():
            try:
                current_mtime = Path(file_path).stat().st_mtime
                if current_mtime > old_mtime:
                    return True
            except FileNotFoundError:
                # 文件被删除，需要重新加载
                return True
        
        return True
    
    def reload_configs(self):
        """重新加载配置"""
        logger.info("重新加载配置")
        self._load_all_configs()
    
    def validate_environment(self, environment: str) -> Tuple[bool, str]:
        """验证环境配置"""
        env_config = self.environments.get(environment)
        if not env_config:
            return False, f"Environment '{environment}' not found"
        
        if not env_config.providers:
            return False, f"No providers configured for environment '{environment}'"
        
        # 检查是否有可用的提供商
        available_providers = [name for name, provider in env_config.providers.items() 
                             if provider.is_available and provider.api_key]
        
        if not available_providers:
            return False, f"No available providers in environment '{environment}'"
        
        # 检查默认提供商是否可用
        if env_config.default_provider not in available_providers:
            return False, f"Default provider '{env_config.default_provider}' is not available"
        
        return True, "Valid"
    
    def get_environment_stats(self, environment: str = None) -> Dict[str, Any]:
        """获取环境统计信息"""
        env_config = self.get_environment_config(environment)
        if not env_config:
            return {}
        
        total_providers = len(env_config.providers)
        available_providers = len([p for p in env_config.providers.values() if p.is_available])
        
        return {
            "name": env_config.name,
            "description": env_config.description,
            "total_providers": total_providers,
            "available_providers": available_providers,
            "default_provider": env_config.default_provider,
            "default_model": env_config.default_model,
            "has_rate_limits": bool(env_config.rate_limits),
            "has_cost_limits": bool(env_config.cost_limits)
        }
    
    def export_environment_config(self, environment: str, file_path: str = None) -> str:
        """导出环境配置到文件"""
        env_config = self.environments.get(environment)
        if not env_config:
            raise ValueError(f"Environment '{environment}' not found")
        
        # 转换为可序列化的格式
        export_data = {
            "name": env_config.name,
            "description": env_config.description,
            "default_provider": env_config.default_provider,
            "default_model": env_config.default_model,
            "providers": {},
            "rate_limits": env_config.rate_limits or {},
            "cost_limits": env_config.cost_limits or {}
        }
        
        for provider_name, provider_info in env_config.providers.items():
            export_data["providers"][provider_name] = {
                "name": provider_info.name,
                "api_key": "***MASKED***",  # 不导出敏感信息
                "base_url": provider_info.base_url,
                "is_available": provider_info.is_available,
                "extra_config": provider_info.extra_config or {}
            }
        
        config_json = json.dumps(export_data, indent=2, ensure_ascii=False)
        
        if file_path:
            with open(file_path, 'w', encoding='utf-8') as f:
                f.write(config_json)
            logger.info("环境配置导出完成", 
                       environment=environment, 
                       file=file_path)
        
        return config_json
