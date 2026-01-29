.PHONY: all build run clean test help swagger

# 变量定义
APP_NAME=mom
BUILD_DIR=bin
CONFIG_FILE=config/config.yaml
GO_FILES=$(shell find . -name '*.go' -type f)
LDFLAGS=-ldflags "-X main.Version=1.0.0 -X main.GitCommit=$(shell git rev-parse --short HEAD 2>/dev/null || echo 'unknown') -X main.BuildTime=$(shell date -u '+%Y-%m-%d_%H:%M:%S')"

# 默认目标
all: swagger build

# 编译
build:
	@echo "编译中..."
	@mkdir -p $(BUILD_DIR)
	@go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME) main.go
	@echo "编译完成: $(BUILD_DIR)/$(APP_NAME)"

# 运行服务
run:
	@echo "运行服务..."
	@go run main.go server -c $(CONFIG_FILE)

# 清理
clean:
	@echo "清理中..."
	@rm -rf $(BUILD_DIR)
	@rm -rf logs/*.log
	@echo "清理完成"

# 安装依赖
deps:
	@echo "安装依赖..."
	@go mod tidy
	@go mod verify
	@echo "依赖安装完成"

# 生成 Swagger 文档
swagger:
	@echo "生成 Swagger 文档..."
	@swag init -g main.go -o docs
	@echo "Swagger 文档生成完成"

# 运行测试
test:
	@echo "运行测试..."
	@go test -v ./...

# 格式化代码
fmt:
	@echo "格式化代码..."
	@go fmt ./...
	@echo "格式化完成"

# 代码检查
lint:
	@echo "代码检查..."
	@golangci-lint run ./...

# 帮助
help:
	@echo "可用命令:"
	@echo "  make all       - 生成 Swagger 并编译项目"
	@echo "  make build     - 编译项目"
	@echo "  make run       - 运行服务"
	@echo "  make clean     - 清理编译文件和日志"
	@echo "  make deps      - 安装依赖"
	@echo "  make swagger   - 生成 Swagger 文档"
	@echo "  make test      - 运行测试"
	@echo "  make fmt       - 格式化代码"
	@echo "  make lint      - 代码检查"
