# Nexus Agent Platform

> 🚀 企业级AI Agent平台 - 展示现代分布式系统架构与多语言工程实践

[![Go](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org)
[![Rust](https://img.shields.io/badge/Rust-1.70+-orange.svg)](https://www.rust-lang.org)
[![Python](https://img.shields.io/badge/Python-3.11+-green.svg)](https://www.python.org)
[![License](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

## ✨ 项目概述

Nexus Agent是一个企业级AI Agent执行平台，专注于展示现代软件工程的最佳实践：

- 🏗️ **微服务架构** - Go/Rust/Python多语言协作
- 🔒 **安全沙箱** - WASM隔离 + 策略引擎
- 📊 **可观测性** - 分布式链路追踪 + 实时监控
- ⚡ **高性能** - 异步并发 + 智能路由
- 🌊 **云原生** - Docker + K8s + 自动扩缩容

## 🏗️ 架构图

```
┌─────────────────────────────────────────────────────────────────┐
│                      CLIENT LAYER                               │
│  Web UI  │   CLI    │  REST API  │    gRPC     │    SSE       │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                   ORCHESTRATOR (Go)                             │
│  Router │ Session │ Budget │ Policy │ Workflow │ Observability │
└─────────────────────────────────────────────────────────────────┘
        │                │                 │                │
        ▼                ▼                 ▼                ▼
┌──────────────┐ ┌──────────────┐ ┌──────────────┐ ┌──────────────┐
│ AGENT CORE   │ │ LLM SERVICE  │ │  POSTGRES    │ │    REDIS     │
│   (Rust)     │ │   (Python)   │ │  State DB    │ │  Cache+Queue │
│              │ │              │ │              │ │              │
│ • WASM Box   │ │ • Providers  │ │ • Sessions   │ │ • Sessions   │
│ • Tools      │ │ • Vector     │ │ • Messages   │ │ • Tasks      │
│ • Safety     │ │ • Streaming  │ │ • Audit      │ │ • Metrics    │
└──────────────┘ └──────────────┘ └──────────────┘ └──────────────┘
        │                │
        ▼                ▼
┌──────────────┐ ┌──────────────┐
│   QDRANT     │ │ OBSERVABILITY│
│Vector Memory │ │ Prometheus + │
│   + RAG      │ │ Jaeger +     │
│              │ │ Grafana      │
└──────────────┘ └──────────────┘
```

## 🚀 快速开始

### 前置要求
- Docker & Docker Compose
- Go 1.21+
- Rust 1.70+
- Python 3.11+

### 一键启动
```bash
# 克隆项目
git clone https://github.com/mrblind/nexus-agent.git
cd nexus-agent

# 启动所有服务
make up

# 运行演示
make demo
```

### 服务端口
- **Orchestrator (Go)**: http://localhost:8080
- **Agent Core (Rust)**: http://localhost:8081
- **LLM Service (Python)**: http://localhost:8082
- **Grafana Dashboard**: http://localhost:3000
- **Prometheus**: http://localhost:9090

## 📋 核心功能

### 🎯 Agent执行引擎
- **多步工具调用** - 支持复杂工作流编排
- **并行执行优化** - 智能任务并行化
- **错误恢复机制** - 自动重试 + 优雅降级
- **实时流式响应** - SSE长连接 + 背压处理

### 🔐 安全与治理
- **沙箱隔离执行** - WASM运行时 + 资源限制
- **细粒度权限控制** - OPA策略引擎
- **预算管理系统** - Token计费 + 时间限制
- **审计日志记录** - 完整操作链路追踪

### 📊 可观测性
- **分布式链路追踪** - OpenTelemetry集成
- **实时监控面板** - Prometheus + Grafana
- **性能分析报告** - 延迟分布 + 资源使用
- **错误聚合分析** - 异常模式识别

## 🛠️ 技术栈

### 后端服务
- **Orchestrator**: Go 1.21 + Gin + GORM + Redis
- **Agent Core**: Rust + Tokio + Wasmtime + gRPC
- **LLM Service**: Python + FastAPI + Qdrant + OpenAI

### 数据存储
- **PostgreSQL**: 会话状态 + 审计日志
- **Redis**: 缓存 + 任务队列 + 分布式锁
- **Qdrant**: 向量存储 + 语义检索

### 基础设施
- **Docker**: 容器化部署
- **Kubernetes**: 生产环境编排
- **Prometheus**: 指标采集
- **Grafana**: 可视化面板
- **Jaeger**: 分布式链路追踪

## 📚 文档导航

### 设计文档
- 📖 [架构设计](docs/ARCHITECTURE.md) - 系统设计理念与技术选型
- 📋 [实施规划](docs/IMPLEMENTATION_PLAN.md) - 开发路线图与里程碑
- 🔌 [API规范](docs/API_SPEC.md) - 接口定义与数据模型
- 🚀 [部署指南](docs/DEPLOYMENT.md) - 生产环境部署

### 服务文档
- 🎛️ [Orchestrator服务](services/orchestrator/README.md) - Go并发编程实践
- ⚙️ [Agent Core服务](services/agent-core/README.md) - Rust安全编程展示
- 🧠 [LLM服务](services/llm-service/README.md) - Python异步编程应用

### 运维文档
- 📊 [性能测试](docs/PERFORMANCE.md) - 压力测试与性能调优
- 🔍 [监控告警](docs/MONITORING.md) - 可观测性配置
- 🛡️ [安全指南](docs/SECURITY.md) - 安全策略与合规要求

## 🎯 演示场景

### 场景1: 智能数据分析助手
```bash
curl -X POST http://localhost:8080/v1/sessions \
  -H "Content-Type: application/json" \
  -d '{"user_id": "demo", "budget": {"tokens": 10000, "time_seconds": 3600}}'

curl -X POST http://localhost:8080/v1/messages \
  -H "Content-Type: application/json" \
  -d '{
    "session_id": "xxx",
    "content": "帮我分析这个CSV文件的销售趋势，生成可视化图表",
    "tools": ["file_reader", "data_analyzer", "chart_generator"]
  }'
```

### 场景2: 自动化工作流
```bash
# 启动复杂工作流
curl -X POST http://localhost:8080/v1/agents/workflow/execute \
  -H "Content-Type: application/json" \
  -d '{
    "workflow": [
      {"tool": "web_scraper", "params": {"url": "https://api.example.com/data"}},
      {"tool": "data_processor", "params": {"format": "json"}},
      {"tool": "report_generator", "params": {"template": "quarterly"}}
    ]
  }'
```

### 场景3: 实时监控演示
- 访问 http://localhost:3000 查看Grafana面板
- 观察实时QPS、延迟分布、错误率
- 触发故障，观察系统恢复过程

## 📈 性能指标

### 基准测试结果
- **并发会话**: 1000+ concurrent sessions
- **响应延迟**: P95 < 500ms (不含LLM推理)
- **吞吐量**: 100+ requests/second
- **资源使用**: 单机4C8G稳定运行

### 压力测试
```bash
# 并发创建会话
make benchmark-sessions

# 工具执行压测
make benchmark-tools

# 流式响应测试
make benchmark-streaming
```

## 🔧 开发指南

### 本地开发环境
```bash
# 安装开发依赖
make install-deps

# 启动开发服务
make dev

# 运行测试套件
make test

# 代码格式化
make fmt

# 静态检查
make lint
```

### 贡献代码
1. Fork本项目
2. 创建特性分支: `git checkout -b feature/amazing-feature`
3. 提交更改: `git commit -m 'Add amazing feature'`
4. 推送分支: `git push origin feature/amazing-feature`
5. 提交Pull Request

## 🎤 面试展示

这个项目专为技术面试设计，展示了以下能力：

### 系统架构能力
- 微服务拆分与服务治理
- 分布式系统设计模式
- 可扩展性与高可用性设计

### 编程语言精通
- **Go**: 并发编程、错误处理、接口设计
- **Rust**: 内存安全、异步编程、性能优化
- **Python**: 异步编程、ML工程、设计模式

### 工程实践
- 测试驱动开发 (TDD)
- 持续集成/持续部署 (CI/CD)
- 代码质量保证
- 文档驱动开发

### 运维能力
- 容器化部署与编排
- 监控告警体系建设
- 性能调优与故障排除
- 安全策略实施

## 📄 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

## 🤝 联系方式

- **作者**: mrblind
- **邮箱**: [your.email@example.com](mailto:your.email@example.com)
- **GitHub**: [@mrblind](https://github.com/mrblind)
- **LinkedIn**: [Your LinkedIn](https://linkedin.com/in/yourprofile)

---

*本项目用于展示现代软件工程能力，欢迎Star ⭐ 和Fork 🍴*
