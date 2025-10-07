# Proto 文件编译 Makefile

# 变量定义
PROTO_DIR = .
GO_OUT_DIR = ../go/proto
PYTHON_OUT_DIR = ../python/src/proto

# 确保输出目录存在
$(GO_OUT_DIR):
	mkdir -p $(GO_OUT_DIR)

$(PYTHON_OUT_DIR):
	mkdir -p $(PYTHON_OUT_DIR)

# 生成 Go 代码
.PHONY: go
go: $(GO_OUT_DIR)
	protoc --proto_path=$(PROTO_DIR) \
		--go_out=$(GO_OUT_DIR) \
		--go_opt=paths=source_relative \
		--go-grpc_out=$(GO_OUT_DIR) \
		--go-grpc_opt=paths=source_relative \
		$(PROTO_DIR)/*.proto

# 生成 Python 代码
.PHONY: python
python: $(PYTHON_OUT_DIR)
	python -m grpc_tools.protoc --proto_path=$(PROTO_DIR) \
		--python_out=$(PYTHON_OUT_DIR) \
		--grpc_python_out=$(PYTHON_OUT_DIR) \
		$(PROTO_DIR)/*.proto

# 生成所有代码
.PHONY: all
all: go python

# 清理生成的文件
.PHONY: clean
clean:
	rm -rf $(GO_OUT_DIR)
	rm -rf $(PYTHON_OUT_DIR)

# 安装依赖
.PHONY: install-deps
install-deps:
	# Go 依赖
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	
	# Python 依赖
	pip install grpcio grpcio-tools

# 验证 proto 文件
.PHONY: validate
validate:
	protoc --proto_path=$(PROTO_DIR) --descriptor_set_out=/dev/null $(PROTO_DIR)/*.proto

.PHONY: help
help:
	@echo "可用的命令:"
	@echo "  make go          - 生成 Go gRPC 代码"
	@echo "  make python      - 生成 Python gRPC 代码"
	@echo "  make all         - 生成所有 gRPC 代码"
	@echo "  make clean       - 清理生成的文件"
	@echo "  make install-deps - 安装必要的依赖"
	@echo "  make validate    - 验证 proto 文件语法"
