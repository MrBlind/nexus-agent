# Nexus Agent 项目管理 Makefile
# ====================================

# 项目信息
PROJECT_NAME := nexus-agent
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# 日志相关变量
LOG_DATE := $(shell date '+%Y%m%d%H')
LOG_DIR := logs/$(LOG_DATE)
GO_LOG_FILE := $(LOG_DIR)/nexus-agent-go.log
PYTHON_LOG_FILE := $(LOG_DIR)/nexus-agent-python.log

# 目录定义
GO_DIR := go
PYTHON_DIR := python
PROTO_DIR := proto
DOCS_DIR := docs
DEPLOY_DIR := deploy

# Go 相关变量
GO_MODULE := github.com/mrblind/nexus-agent
GO_MAIN := $(GO_DIR)/cmd/server
GO_BINARY := $(GO_DIR)/bin/server
GO_LDFLAGS := -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)

# Docker 相关变量
DOCKER_IMAGE := $(PROJECT_NAME)
DOCKER_TAG := $(VERSION)
DOCKER_REGISTRY := # 设置你的 Docker registry

# 颜色定义
RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[0;33m
BLUE := \033[0;34m
PURPLE := \033[0;35m
CYAN := \033[0;36m
NC := \033[0m # No Color

# 默认目标
.DEFAULT_GOAL := help

# ====================================
# 开发环境管理
# ====================================

.PHONY: setup
setup: ## 初始化开发环境
	@echo "$(CYAN)🚀 初始化开发环境...$(NC)"
	@$(MAKE) install-deps
	@$(MAKE) proto-gen
	@$(MAKE) db-up
	@echo "$(GREEN)✅ 开发环境初始化完成$(NC)"

.PHONY: install-deps
install-deps: ## 安装所有依赖
	@echo "$(YELLOW)📦 安装依赖...$(NC)"
	@echo "$(BLUE)安装 Go 依赖...$(NC)"
	cd $(GO_DIR) && go mod download
	cd $(GO_DIR) && go mod tidy
	@echo "$(BLUE)安装 Python 依赖...$(NC)"
	cd $(PYTHON_DIR) && pip install -r requirements.txt
	@echo "$(BLUE)安装 Proto 工具...$(NC)"
	cd $(PROTO_DIR) && $(MAKE) install-deps
	@echo "$(GREEN)✅ 依赖安装完成$(NC)"

# ====================================
# 构建相关
# ====================================

.PHONY: build
build: build-go ## 构建所有组件

.PHONY: build-go
build-go: ## 构建 Go 服务
	@echo "$(YELLOW)🔨 构建 Go 服务...$(NC)"
	cd $(GO_DIR) && mkdir -p bin
	cd $(GO_DIR) && go build -ldflags "$(GO_LDFLAGS)" -o bin/server ./cmd/server
	@echo "$(GREEN)✅ Go 服务构建完成: $(GO_BINARY)$(NC)"

.PHONY: build-docker
build-docker: ## 构建 Docker 镜像
	@echo "$(YELLOW)🐳 构建 Docker 镜像...$(NC)"
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) -f $(GO_DIR)/Dockerfile $(GO_DIR)
	docker tag $(DOCKER_IMAGE):$(DOCKER_TAG) $(DOCKER_IMAGE):latest
	@echo "$(GREEN)✅ Docker 镜像构建完成: $(DOCKER_IMAGE):$(DOCKER_TAG)$(NC)"

# ====================================
# 运行相关
# ====================================

.PHONY: run
run: setup-logs ## 同时运行 Python gRPC 服务和 Go HTTP 服务
	@echo "$(CYAN)🚀 启动 Nexus Agent 服务集群...$(NC)"
	@echo "$(BLUE)📁 日志目录: $(LOG_DIR)$(NC)"
	@echo "$(BLUE)📝 Go 日志: $(GO_LOG_FILE)$(NC)"
	@echo "$(BLUE)📝 Python 日志: $(PYTHON_LOG_FILE)$(NC)"
	@echo "$(YELLOW)⚠️  使用 Ctrl+C 停止所有服务$(NC)"
	@$(MAKE) run-services

.PHONY: run-services
run-services: ## 内部命令：并行运行服务
	@echo "$(YELLOW)🧹 清理已存在的服务进程...$(NC)"
	@-pkill -f "python start_grpc_server.py" 2>/dev/null
	@-pkill -f "go run ./cmd/server" 2>/dev/null
	@-fuser -k 8080/tcp 2>/dev/null
	@-fuser -k 50051/tcp 2>/dev/null
	@sleep 2
	@trap 'echo "$(RED)🛑 正在停止所有服务...$(NC)"; kill 0; exit 0' INT; \
	( \
		echo "$(CYAN)🐍 启动 Python gRPC 服务...$(NC)"; \
		cd $(PYTHON_DIR) && PYTHONUNBUFFERED=1 stdbuf -oL -eL python -u start_grpc_server.py 2>&1 | stdbuf -oL -eL sed 's/^/[PYTHON] /' | tee $(PWD)/$(PYTHON_LOG_FILE) & \
		PYTHON_PID=$$!; \
		sleep 3; \
		echo "$(CYAN)🚀 启动 Go HTTP 服务...$(NC)"; \
		cd $(GO_DIR) && stdbuf -oL -eL go run ./cmd/server 2>&1 | stdbuf -oL -eL sed 's/^/[GO] /' | tee $(PWD)/$(GO_LOG_FILE) & \
		GO_PID=$$!; \
		echo "$(GREEN)✅ 服务启动完成$(NC)"; \
		echo "$(BLUE)📊 Python PID: $$PYTHON_PID$(NC)"; \
		echo "$(BLUE)📊 Go PID: $$GO_PID$(NC)"; \
		echo "$(YELLOW)💡 使用 Ctrl+C 停止服务$(NC)"; \
		wait \
	)

.PHONY: run-go
run-go: setup-logs ## 只运行 Go 服务
	@echo "$(CYAN)🚀 启动 Go 服务...$(NC)"
	@echo "$(BLUE)📝 日志文件: $(GO_LOG_FILE)$(NC)"
	cd $(GO_DIR) && go run ./cmd/server 2>&1 | tee ../$(GO_LOG_FILE)

.PHONY: run-python
run-python: setup-logs ## 只运行 Python gRPC 服务
	@echo "$(CYAN)🐍 启动 Python gRPC 服务...$(NC)"
	@echo "$(BLUE)📝 日志文件: $(PYTHON_LOG_FILE)$(NC)"
	cd $(PYTHON_DIR) && PYTHONUNBUFFERED=1 python -u start_grpc_server.py 2>&1 | tee ../$(PYTHON_LOG_FILE)

.PHONY: run-docker
run-docker: build-docker ## 使用 Docker 运行服务
	@echo "$(CYAN)🐳 使用 Docker 启动服务...$(NC)"
	docker run --rm -p 8080:8080 -p 50051:50051 $(DOCKER_IMAGE):$(DOCKER_TAG)

.PHONY: run-bg
run-bg: setup-logs ## 在后台运行服务集群
	@echo "$(CYAN)🚀 在后台启动 Nexus Agent 服务集群...$(NC)"
	@echo "$(BLUE)📁 日志目录: $(LOG_DIR)$(NC)"
	@echo "$(BLUE)📝 Go 日志: $(GO_LOG_FILE)$(NC)"
	@echo "$(BLUE)📝 Python 日志: $(PYTHON_LOG_FILE)$(NC)"
	@echo "$(YELLOW)💡 使用 'make stop' 停止服务$(NC)"
	@echo "$(YELLOW)💡 使用 'make logs' 查看日志$(NC)"
	@echo "$(YELLOW)💡 使用 'make status' 查看状态$(NC)"
	@$(MAKE) run-services-bg

.PHONY: run-services-bg
run-services-bg: ## 内部命令：后台运行服务
	@echo "$(CYAN)🐍 启动 Python gRPC 服务...$(NC)"
	@cd $(PYTHON_DIR) && nohup env PYTHONUNBUFFERED=1 python -u start_grpc_server.py > $(PWD)/$(PYTHON_LOG_FILE) 2>&1 & echo $$! > $(PWD)/$(LOG_DIR)/python.pid
	@sleep 3
	@echo "$(CYAN)🚀 启动 Go HTTP 服务...$(NC)"
	@cd $(GO_DIR) && nohup go run ./cmd/server > $(PWD)/$(GO_LOG_FILE) 2>&1 & echo $$! > $(PWD)/$(LOG_DIR)/go.pid
	@sleep 2
	@echo "$(GREEN)✅ 服务已在后台启动$(NC)"
	@if [ -f "$(LOG_DIR)/python.pid" ]; then \
		echo "$(BLUE)📊 Python PID: $$(cat $(LOG_DIR)/python.pid)$(NC)"; \
	fi
	@if [ -f "$(LOG_DIR)/go.pid" ]; then \
		echo "$(BLUE)📊 Go PID: $$(cat $(LOG_DIR)/go.pid)$(NC)"; \
	fi

.PHONY: stop
stop: ## 停止后台运行的服务
	@echo "$(YELLOW)🛑 停止后台服务...$(NC)"
	@if [ -f "$(LOG_DIR)/python.pid" ]; then \
		PID=$$(cat $(LOG_DIR)/python.pid); \
		if kill -0 $$PID 2>/dev/null; then \
			kill $$PID && echo "$(GREEN)✅ Python 服务已停止 (PID: $$PID)$(NC)"; \
		else \
			echo "$(YELLOW)⚠️  Python 服务进程不存在$(NC)"; \
		fi; \
		rm -f $(LOG_DIR)/python.pid; \
	else \
		echo "$(YELLOW)⚠️  未找到 Python 服务 PID 文件$(NC)"; \
	fi
	@if [ -f "$(LOG_DIR)/go.pid" ]; then \
		PID=$$(cat $(LOG_DIR)/go.pid); \
		if kill -0 $$PID 2>/dev/null; then \
			kill $$PID && echo "$(GREEN)✅ Go 服务已停止 (PID: $$PID)$(NC)"; \
		else \
			echo "$(YELLOW)⚠️  Go 服务进程不存在$(NC)"; \
		fi; \
		rm -f $(LOG_DIR)/go.pid; \
	else \
		echo "$(YELLOW)⚠️  未找到 Go 服务 PID 文件$(NC)"; \
	fi
	@echo "$(GREEN)✅ 所有服务已停止$(NC)"

.PHONY: restart
restart: stop run-bg ## 重启后台服务

.PHONY: setup-logs
setup-logs: ## 创建日志目录
	@mkdir -p $(LOG_DIR)
	@touch $(GO_LOG_FILE) $(PYTHON_LOG_FILE)
	@echo "$(GREEN)📁 日志目录已创建: $(LOG_DIR)$(NC)"
	@echo "$(BLUE)📝 日志文件已创建:$(NC)"
	@echo "  - $(GO_LOG_FILE)"
	@echo "  - $(PYTHON_LOG_FILE)"

# ====================================
# 数据库管理
# ====================================

.PHONY: db-up
db-up: ## 启动开发数据库 (PostgreSQL + Redis)
	@echo "$(CYAN)🗄️  启动数据库服务...$(NC)"
	docker-compose -f docker-compose.dev.yaml up -d
	@echo "$(GREEN)✅ 数据库服务已启动$(NC)"

.PHONY: db-down
db-down: ## 停止数据库服务
	@echo "$(YELLOW)🛑 停止数据库服务...$(NC)"
	docker-compose -f docker-compose.dev.yaml down
	@echo "$(GREEN)✅ 数据库服务已停止$(NC)"

.PHONY: db-logs
db-logs: ## 查看数据库日志
	docker-compose -f docker-compose.dev.yaml logs -f

.PHONY: db-clean
db-clean: ## 清理数据库数据
	@echo "$(RED)⚠️  清理数据库数据...$(NC)"
	docker-compose -f docker-compose.dev.yaml down -v
	@echo "$(GREEN)✅ 数据库数据已清理$(NC)"

# ====================================
# Proto 文件管理
# ====================================

.PHONY: proto-gen
proto-gen: ## 生成 Proto 代码 (Go + Python)
	@echo "$(YELLOW)⚙️  生成 Proto 代码...$(NC)"
	cd $(PROTO_DIR) && $(MAKE) all
	@echo "$(GREEN)✅ Proto 代码生成完成$(NC)"

.PHONY: proto-go
proto-go: ## 生成 Go Proto 代码
	@echo "$(YELLOW)⚙️  生成 Go Proto 代码...$(NC)"
	cd $(PROTO_DIR) && $(MAKE) go

.PHONY: proto-python
proto-python: ## 生成 Python Proto 代码
	@echo "$(YELLOW)⚙️  生成 Python Proto 代码...$(NC)"
	cd $(PROTO_DIR) && $(MAKE) python

.PHONY: proto-clean
proto-clean: ## 清理生成的 Proto 代码
	@echo "$(YELLOW)🧹 清理 Proto 代码...$(NC)"
	cd $(PROTO_DIR) && $(MAKE) clean

.PHONY: proto-validate
proto-validate: ## 验证 Proto 文件
	@echo "$(YELLOW)✅ 验证 Proto 文件...$(NC)"
	cd $(PROTO_DIR) && $(MAKE) validate

# ====================================
# 测试相关
# ====================================

.PHONY: test
test: test-go test-python ## 运行所有测试

.PHONY: test-go
test-go: ## 运行 Go 测试
	@echo "$(YELLOW)🧪 运行 Go 测试...$(NC)"
	cd $(GO_DIR) && go test -v ./...

.PHONY: test-python
test-python: ## 运行 Python 测试
	@echo "$(YELLOW)🧪 运行 Python 测试...$(NC)"
	cd $(PYTHON_DIR) && python -m pytest tests/ -v

.PHONY: test-integration
test-integration: ## 运行集成测试
	@echo "$(YELLOW)🧪 运行集成测试...$(NC)"
	@echo "$(BLUE)启动测试环境...$(NC)"
	@$(MAKE) db-up
	@sleep 5
	cd $(GO_DIR) && go test -tags=integration -v ./...
	cd $(PYTHON_DIR) && python test_grpc_communication.py
	@echo "$(GREEN)✅ 集成测试完成$(NC)"

.PHONY: test-coverage
test-coverage: ## 生成测试覆盖率报告
	@echo "$(YELLOW)📊 生成测试覆盖率报告...$(NC)"
	cd $(GO_DIR) && go test -coverprofile=coverage.out ./...
	cd $(GO_DIR) && go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)✅ 覆盖率报告生成: $(GO_DIR)/coverage.html$(NC)"

# ====================================
# 代码质量
# ====================================

.PHONY: lint
lint: lint-go lint-python ## 运行所有代码检查

.PHONY: lint-go
lint-go: ## 运行 Go 代码检查
	@echo "$(YELLOW)🔍 运行 Go 代码检查...$(NC)"
	cd $(GO_DIR) && go vet ./...
	cd $(GO_DIR) && go fmt ./...
	@if command -v golangci-lint >/dev/null 2>&1; then \
		cd $(GO_DIR) && golangci-lint run; \
	else \
		echo "$(YELLOW)⚠️  golangci-lint 未安装，跳过高级检查$(NC)"; \
	fi

.PHONY: lint-python
lint-python: ## 运行 Python 代码检查
	@echo "$(YELLOW)🔍 运行 Python 代码检查...$(NC)"
	cd $(PYTHON_DIR) && python -m flake8 src/ --max-line-length=88
	cd $(PYTHON_DIR) && python -m black --check src/

.PHONY: format
format: format-go format-python ## 格式化所有代码

.PHONY: format-go
format-go: ## 格式化 Go 代码
	@echo "$(YELLOW)✨ 格式化 Go 代码...$(NC)"
	cd $(GO_DIR) && go fmt ./...
	cd $(GO_DIR) && go mod tidy

.PHONY: format-python
format-python: ## 格式化 Python 代码
	@echo "$(YELLOW)✨ 格式化 Python 代码...$(NC)"
	cd $(PYTHON_DIR) && python -m black src/
	cd $(PYTHON_DIR) && python -m isort src/

# ====================================
# 部署相关
# ====================================

.PHONY: deploy-dev
deploy-dev: ## 部署到开发环境
	@echo "$(CYAN)🚀 部署到开发环境...$(NC)"
	docker-compose -f $(DEPLOY_DIR)/compose/docker-compose.yaml up -d
	@echo "$(GREEN)✅ 开发环境部署完成$(NC)"

.PHONY: deploy-prod
deploy-prod: build-docker ## 部署到生产环境
	@echo "$(CYAN)🚀 部署到生产环境...$(NC)"
	@echo "$(RED)⚠️  请确保已配置生产环境变量$(NC)"
	# 这里添加你的生产部署逻辑
	@echo "$(GREEN)✅ 生产环境部署完成$(NC)"

# ====================================
# 清理相关
# ====================================

.PHONY: clean
clean: ## 清理构建文件
	@echo "$(YELLOW)🧹 清理构建文件...$(NC)"
	rm -rf $(GO_DIR)/bin/
	rm -f $(GO_DIR)/coverage.out $(GO_DIR)/coverage.html
	cd $(PYTHON_DIR) && find . -type d -name __pycache__ -exec rm -rf {} + 2>/dev/null || true
	cd $(PYTHON_DIR) && find . -name "*.pyc" -delete 2>/dev/null || true
	@$(MAKE) proto-clean
	@echo "$(GREEN)✅ 清理完成$(NC)"

.PHONY: clean-docker
clean-docker: ## 清理 Docker 资源
	@echo "$(YELLOW)🧹 清理 Docker 资源...$(NC)"
	docker system prune -f
	docker image prune -f
	@echo "$(GREEN)✅ Docker 清理完成$(NC)"

# ====================================
# 工具相关
# ====================================

.PHONY: logs
logs: ## 查看最新的服务日志
	@echo "$(CYAN)📋 查看最新服务日志...$(NC)"
	@if [ -f "$(GO_LOG_FILE)" ] && [ -f "$(PYTHON_LOG_FILE)" ]; then \
		echo "$(BLUE)📝 同时显示 Go 和 Python 日志$(NC)"; \
		tail -f $(GO_LOG_FILE) $(PYTHON_LOG_FILE); \
	elif [ -f "$(GO_LOG_FILE)" ]; then \
		echo "$(BLUE)📝 显示 Go 日志$(NC)"; \
		tail -f $(GO_LOG_FILE); \
	elif [ -f "$(PYTHON_LOG_FILE)" ]; then \
		echo "$(BLUE)📝 显示 Python 日志$(NC)"; \
		tail -f $(PYTHON_LOG_FILE); \
	else \
		echo "$(YELLOW)⚠️  没有找到日志文件，请先启动服务$(NC)"; \
	fi

.PHONY: logs-live
logs-live: ## 查看带前缀的实时日志 (推荐)
	@echo "$(CYAN)📋 查看带前缀的实时日志...$(NC)"
	@if [ -f "$(GO_LOG_FILE)" ] && [ -f "$(PYTHON_LOG_FILE)" ]; then \
		echo "$(BLUE)📝 显示格式: [GO] 和 [PYTHON] 前缀$(NC)"; \
		echo "$(YELLOW)💡 使用 Ctrl+C 停止查看$(NC)"; \
		( tail -f $(GO_LOG_FILE) | sed 's/^/[GO] /' & \
		  tail -f $(PYTHON_LOG_FILE) | sed 's/^/[PYTHON] /' & \
		  wait ); \
	elif [ -f "$(GO_LOG_FILE)" ]; then \
		echo "$(BLUE)📝 显示 Go 日志$(NC)"; \
		tail -f $(GO_LOG_FILE) | sed 's/^/[GO] /'; \
	elif [ -f "$(PYTHON_LOG_FILE)" ]; then \
		echo "$(BLUE)📝 显示 Python 日志$(NC)"; \
		tail -f $(PYTHON_LOG_FILE) | sed 's/^/[PYTHON] /'; \
	else \
		echo "$(YELLOW)⚠️  没有找到日志文件，请先启动服务$(NC)"; \
	fi

.PHONY: logs-go
logs-go: ## 查看 Go 服务日志
	@echo "$(CYAN)📋 查看 Go 服务日志...$(NC)"
	@if [ -f "$(GO_LOG_FILE)" ]; then \
		tail -f $(GO_LOG_FILE); \
	else \
		echo "$(YELLOW)⚠️  Go 日志文件不存在: $(GO_LOG_FILE)$(NC)"; \
	fi

.PHONY: logs-python
logs-python: ## 查看 Python 服务日志
	@echo "$(CYAN)📋 查看 Python 服务日志...$(NC)"
	@if [ -f "$(PYTHON_LOG_FILE)" ]; then \
		tail -f $(PYTHON_LOG_FILE); \
	else \
		echo "$(YELLOW)⚠️  Python 日志文件不存在: $(PYTHON_LOG_FILE)$(NC)"; \
	fi

.PHONY: logs-list
logs-list: ## 列出所有日志文件
	@echo "$(CYAN)📋 日志文件列表:$(NC)"
	@find logs -name "*.log" -type f 2>/dev/null | sort -r | head -20 || echo "$(YELLOW)⚠️  没有找到日志文件$(NC)"

.PHONY: logs-clean
logs-clean: ## 清理旧日志文件 (保留最近7天)
	@echo "$(YELLOW)🧹 清理旧日志文件...$(NC)"
	@find logs -name "*.log" -type f -mtime +7 -delete 2>/dev/null || true
	@find logs -type d -empty -delete 2>/dev/null || true
	@echo "$(GREEN)✅ 日志清理完成$(NC)"

.PHONY: status
status: ## 查看服务运行状态
	@echo "$(CYAN)📊 服务运行状态:$(NC)"
	@echo "$(BLUE)端口占用情况:$(NC)"
	@lsof -i :8080 2>/dev/null | grep LISTEN || echo "  8080: 未占用"
	@lsof -i :50051 2>/dev/null | grep LISTEN || echo "  50051: 未占用"
	@echo "$(BLUE)进程状态:$(NC)"
	@ps aux | grep -E "(go run|python.*start_grpc_server)" | grep -v grep || echo "  没有找到相关进程"
	@echo "$(BLUE)最新日志:$(NC)"
	@if [ -f "$(GO_LOG_FILE)" ]; then \
		echo "  Go 日志: $(GO_LOG_FILE) ($(shell wc -l < $(GO_LOG_FILE) 2>/dev/null || echo 0) 行)"; \
	fi
	@if [ -f "$(PYTHON_LOG_FILE)" ]; then \
		echo "  Python 日志: $(PYTHON_LOG_FILE) ($(shell wc -l < $(PYTHON_LOG_FILE) 2>/dev/null || echo 0) 行)"; \
	fi

.PHONY: ps
ps: ## 查看运行中的服务
	@echo "$(CYAN)📋 运行中的服务:$(NC)"
	@docker-compose -f docker-compose.dev.yaml ps 2>/dev/null || echo "$(YELLOW)⚠️  Docker Compose 服务未运行$(NC)"

.PHONY: version
version: ## 显示版本信息
	@echo "$(CYAN)📋 版本信息:$(NC)"
	@echo "Project: $(PROJECT_NAME)"
	@echo "Version: $(VERSION)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Git Commit: $(GIT_COMMIT)"

.PHONY: env
env: ## 显示环境变量
	@echo "$(CYAN)📋 环境变量:$(NC)"
	@echo "GO_MODULE: $(GO_MODULE)"
	@echo "GO_BINARY: $(GO_BINARY)"
	@echo "DOCKER_IMAGE: $(DOCKER_IMAGE):$(DOCKER_TAG)"

# ====================================
# 帮助信息
# ====================================

.PHONY: help
help: ## 显示帮助信息
	@echo "$(CYAN)Nexus Agent 项目管理工具$(NC)"
	@echo "$(CYAN)========================$(NC)"
	@echo ""
	@echo "$(YELLOW)开发环境:$(NC)"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  $(GREEN)%-15s$(NC) %s\n", $$1, $$2}' $(MAKEFILE_LIST) | grep -E "(setup|install-deps|db-up|db-down)"
	@echo ""
	@echo "$(YELLOW)构建和运行:$(NC)"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  $(GREEN)%-15s$(NC) %s\n", $$1, $$2}' $(MAKEFILE_LIST) | grep -E "(build|run|stop|restart)"
	@echo ""
	@echo "$(YELLOW)Proto 管理:$(NC)"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  $(GREEN)%-15s$(NC) %s\n", $$1, $$2}' $(MAKEFILE_LIST) | grep -E "proto-"
	@echo ""
	@echo "$(YELLOW)测试和质量:$(NC)"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  $(GREEN)%-15s$(NC) %s\n", $$1, $$2}' $(MAKEFILE_LIST) | grep -E "(test|lint|format)"
	@echo ""
	@echo "$(YELLOW)部署和清理:$(NC)"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  $(GREEN)%-15s$(NC) %s\n", $$1, $$2}' $(MAKEFILE_LIST) | grep -E "(deploy|clean)"
	@echo ""
	@echo "$(YELLOW)日志管理:$(NC)"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  $(GREEN)%-15s$(NC) %s\n", $$1, $$2}' $(MAKEFILE_LIST) | grep -E "logs"
	@echo ""
	@echo "$(YELLOW)工具:$(NC)"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  $(GREEN)%-15s$(NC) %s\n", $$1, $$2}' $(MAKEFILE_LIST) | grep -E "(status|ps|version|env|help)"
	@echo ""
	@echo "$(CYAN)快速开始:$(NC)"
	@echo "  1. $(GREEN)make setup$(NC)      - 初始化开发环境"
	@echo "  2. $(GREEN)make run$(NC)        - 前台启动服务集群 (Python + Go)"
	@echo "  3. $(GREEN)make run-bg$(NC)     - 后台启动服务集群"
	@echo "  4. $(GREEN)make status$(NC)     - 查看服务状态"
	@echo "  5. $(GREEN)make logs$(NC)       - 查看实时日志"
	@echo ""
	@echo "$(CYAN)服务管理:$(NC)"
	@echo "  • $(GREEN)make stop$(NC)       - 停止后台服务"
	@echo "  • $(GREEN)make restart$(NC)    - 重启后台服务"
	@echo "  • $(GREEN)make run-go$(NC)     - 只启动 Go HTTP 服务"
	@echo "  • $(GREEN)make run-python$(NC) - 只启动 Python gRPC 服务"
	@echo ""