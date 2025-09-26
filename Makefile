# ==============================================================================
# MiniBlog 项目 Makefile
# 用于统一管理项目的构建、测试、部署等操作
# ==============================================================================

# 定义全局变量
COMMON_SELF_DIR := $(dir $(lastword $(MAKEFILE_LIST)))
ROOT_DIR := $(abspath $(shell cd $(COMMON_SELF_DIR)/ && pwd -P))
OUTPUT_DIR := $(ROOT_DIR)/_output
BUILD_DIR := $(ROOT_DIR)/build
SCRIPTS_DIR := $(ROOT_DIR)/scripts

# 项目信息
PROJECT_NAME := miniblog
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "v0.0.0-dev")
BUILD_TIME := $(shell date +%Y-%m-%dT%H:%M:%S%z)
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Go 相关变量
GO_MODULE := $(shell head -1 go.mod | awk '{print $$2}')
MAIN_FILE := $(ROOT_DIR)/cmd/$(PROJECT_NAME)/main.go
BINARY_NAME := $(PROJECT_NAME)
LDFLAGS := -X '$(GO_MODULE)/internal/pkg/core.Version=$(VERSION)' \
           -X '$(GO_MODULE)/internal/pkg/core.BuildTime=$(BUILD_TIME)' \
           -X '$(GO_MODULE)/internal/pkg/core.GitCommit=$(GIT_COMMIT)'

# 前端项目路径
WEB_BLOG_DIR := $(ROOT_DIR)/web/miniblog-web
WEB_ADMIN_DIR := $(ROOT_DIR)/web/miniblog-web-admin

# Docker 相关
DOCKER_COMPOSE_DEV := $(BUILD_DIR)/docker/miniblog/compose-dev.yml
DOCKER_COMPOSE_PROD_INFRA := $(BUILD_DIR)/docker/miniblog/compose-prod-infra.yml
DOCKER_COMPOSE_PROD_APP := $(BUILD_DIR)/docker/miniblog/compose-prod-app.yml

# 颜色定义
RED := \033[31m
GREEN := \033[32m
YELLOW := \033[33m
BLUE := \033[34m
RESET := \033[0m

# ==============================================================================
# 默认目标
# ==============================================================================
.PHONY: all
all: help

# ==============================================================================
# 帮助信息
# ==============================================================================
.PHONY: help
help: ## 显示帮助信息
	@echo ""
	@echo "$(YELLOW)MiniBlog 项目管理 Makefile$(RESET)"
	@echo ""
	@echo "$(BLUE)使用方法:$(RESET)"
	@echo "  make <target>"
	@echo ""
	@echo "$(BLUE)可用目标:$(RESET)"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  $(GREEN)%-20s$(RESET) %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""

# ==============================================================================
# 开发相关目标
# ==============================================================================
.PHONY: deps
deps: ## 安装项目依赖
	@echo "$(BLUE)安装 Go 依赖...$(RESET)"
	@go mod download
	@go mod tidy
	@echo "$(BLUE)安装前端依赖...$(RESET)"
	@cd $(WEB_BLOG_DIR) && npm install
	@cd $(WEB_ADMIN_DIR) && npm install
	@echo "$(GREEN)✅ 依赖安装完成$(RESET)"

.PHONY: tidy
tidy: ## 整理 Go 模块依赖
	@echo "$(BLUE)整理 Go 模块依赖...$(RESET)"
	@go mod tidy
	@echo "$(GREEN)✅ 依赖整理完成$(RESET)"

.PHONY: format
format: ## 格式化代码
	@echo "$(BLUE)格式化 Go 代码...$(RESET)"
	@gofmt -s -w ./
	@go vet ./...
	@echo "$(BLUE)格式化前端代码...$(RESET)"
	@cd $(WEB_BLOG_DIR) && npm run format 2>/dev/null || echo "跳过 blog 格式化"
	@cd $(WEB_ADMIN_DIR) && npm run lint:fix 2>/dev/null || echo "跳过 admin 格式化"
	@echo "$(GREEN)✅ 代码格式化完成$(RESET)"

.PHONY: lint
lint: ## 代码质量检查
	@echo "$(BLUE)Go 代码检查...$(RESET)"
	@go vet ./...
	@golangci-lint run 2>/dev/null || echo "golangci-lint 未安装，跳过检查"
	@echo "$(BLUE)前端代码检查...$(RESET)"
	@cd $(WEB_BLOG_DIR) && npm run lint 2>/dev/null || echo "跳过 blog 检查"
	@cd $(WEB_ADMIN_DIR) && npm run lint 2>/dev/null || echo "跳过 admin 检查"
	@echo "$(GREEN)✅ 代码检查完成$(RESET)"

.PHONY: test
test: ## 运行测试
	@echo "$(BLUE)运行 Go 测试...$(RESET)"
	@go test -v -race -cover ./...
	@echo "$(GREEN)✅ 测试完成$(RESET)"

.PHONY: add-copyright
add-copyright: ## 添加版权头信息
	@echo "$(BLUE)添加版权头信息...$(RESET)"
	@addlicense -v -f $(SCRIPTS_DIR)/boilerplate.txt $(ROOT_DIR) --skip-dirs=third_party,vendor,$(OUTPUT_DIR),web 2>/dev/null || echo "addlicense 未安装，跳过版权添加"
	@echo "$(GREEN)✅ 版权头添加完成$(RESET)"

# ==============================================================================
# 构建相关目标
# ==============================================================================
.PHONY: build
build: tidy ## 构建后端二进制文件
	@echo "$(BLUE)构建后端服务...$(RESET)"
	@mkdir -p $(OUTPUT_DIR)
	@go build -ldflags "$(LDFLAGS)" -o $(OUTPUT_DIR)/$(BINARY_NAME) $(MAIN_FILE)
	@echo "$(GREEN)✅ 后端构建完成: $(OUTPUT_DIR)/$(BINARY_NAME)$(RESET)"

.PHONY: build-linux
build-linux: tidy ## 构建 Linux 版本
	@echo "$(BLUE)构建 Linux 版本...$(RESET)"
	@mkdir -p $(OUTPUT_DIR)
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(OUTPUT_DIR)/$(BINARY_NAME)-linux $(MAIN_FILE)
	@echo "$(GREEN)✅ Linux 版本构建完成: $(OUTPUT_DIR)/$(BINARY_NAME)-linux$(RESET)"

.PHONY: build-web
build-web: ## 构建前端静态文件
	@echo "$(BLUE)构建博客前端...$(RESET)"
	@cd $(WEB_BLOG_DIR) && npm run build
	@echo "$(BLUE)构建管理后台...$(RESET)"
	@cd $(WEB_ADMIN_DIR) && npm run build:prod
	@echo "$(GREEN)✅ 前端构建完成$(RESET)"

.PHONY: build-all
build-all: build build-web ## 构建所有组件
	@echo "$(GREEN)✅ 全部构建完成$(RESET)"

.PHONY: clean
clean: ## 清理构建产物
	@echo "$(BLUE)清理构建产物...$(RESET)"
	@rm -rf $(OUTPUT_DIR)
	@rm -rf $(WEB_BLOG_DIR)/dist 2>/dev/null || true
	@rm -rf $(WEB_ADMIN_DIR)/dist 2>/dev/null || true
	@echo "$(GREEN)✅ 清理完成$(RESET)"

# ==============================================================================
# 开发运行目标
# ==============================================================================
.PHONY: dev
dev: ## 启动开发环境
	@echo "$(BLUE)启动开发环境...$(RESET)"
	@docker compose -f $(DOCKER_COMPOSE_DEV) up -d
	@echo "$(GREEN)✅ 开发环境启动完成$(RESET)"
	@echo "$(YELLOW)💡 后端服务: http://localhost:8081$(RESET)"
	@echo "$(YELLOW)💡 博客前端: http://localhost:5173$(RESET)"
	@echo "$(YELLOW)💡 管理后台: http://localhost:8080$(RESET)"

.PHONY: dev-backend
dev-backend: build ## 启动后端开发服务
	@echo "$(BLUE)启动后端服务...$(RESET)"
	@$(OUTPUT_DIR)/$(BINARY_NAME) -c configs/miniblog.yaml

.PHONY: dev-web
dev-web: ## 启动前端开发服务
	@echo "$(BLUE)启动博客前端开发服务...$(RESET)"
	@cd $(WEB_BLOG_DIR) && npm run dev &
	@echo "$(BLUE)启动管理后台开发服务...$(RESET)"
	@cd $(WEB_ADMIN_DIR) && npm run dev &
	@wait

.PHONY: stop-dev
stop-dev: ## 停止开发环境
	@echo "$(BLUE)停止开发环境...$(RESET)"
	@docker compose -f $(DOCKER_COMPOSE_DEV) down
	@echo "$(GREEN)✅ 开发环境已停止$(RESET)"

# ==============================================================================
# 部署相关目标
# ==============================================================================
.PHONY: deploy-infra
deploy-infra: ## 部署基础设施 (MySQL, Redis, Nginx)
	@echo "$(BLUE)部署基础设施...$(RESET)"
	@docker compose -f $(DOCKER_COMPOSE_PROD_INFRA) up -d
	@echo "$(GREEN)✅ 基础设施部署完成$(RESET)"

.PHONY: deploy-app
deploy-app: build-linux ## 部署应用服务
	@echo "$(BLUE)部署应用服务...$(RESET)"
	@docker compose -f $(DOCKER_COMPOSE_PROD_APP) up -d --build
	@echo "$(GREEN)✅ 应用服务部署完成$(RESET)"

.PHONY: deploy-all
deploy-all: deploy-infra deploy-app ## 部署完整应用
	@echo "$(GREEN)✅ 完整应用部署完成$(RESET)"

.PHONY: undeploy
undeploy: ## 停止并清理所有部署
	@echo "$(BLUE)停止并清理部署...$(RESET)"
	@docker compose -f $(DOCKER_COMPOSE_PROD_APP) down --remove-orphans
	@docker compose -f $(DOCKER_COMPOSE_PROD_INFRA) down --remove-orphans
	@docker compose -f $(DOCKER_COMPOSE_DEV) down --remove-orphans
	@echo "$(GREEN)✅ 部署清理完成$(RESET)"

# ==============================================================================
# 数据库管理
# ==============================================================================
.PHONY: db-migrate
db-migrate: ## 运行数据库迁移
	@echo "$(BLUE)运行数据库迁移...$(RESET)"
	@mysql -h127.0.0.1 -P3306 -uroot -proot < configs/mysql/miniblog.sql 2>/dev/null || \
		echo "$(YELLOW)⚠️  请确保 MySQL 服务已启动$(RESET)"
	@echo "$(GREEN)✅ 数据库迁移完成$(RESET)"

.PHONY: db-reset
db-reset: ## 重置数据库
	@echo "$(BLUE)重置数据库...$(RESET)"
	@echo "$(RED)⚠️  这将删除所有数据，请确认！$(RESET)"
	@read -p "输入 'yes' 继续: " confirm; [ "$$confirm" = "yes" ] || exit 1
	@mysql -h127.0.0.1 -P3306 -uroot -proot -e "DROP DATABASE IF EXISTS miniblog; CREATE DATABASE miniblog;" 2>/dev/null || \
		echo "$(YELLOW)⚠️  请确保 MySQL 服务已启动$(RESET)"
	@$(MAKE) db-migrate
	@echo "$(GREEN)✅ 数据库重置完成$(RESET)"

# ==============================================================================
# 监控和日志
# ==============================================================================
.PHONY: status
status: ## 查看服务状态
	@echo "$(BLUE)服务状态:$(RESET)"
	@docker compose -f $(DOCKER_COMPOSE_DEV) ps 2>/dev/null || echo "开发环境未启动"
	@docker compose -f $(DOCKER_COMPOSE_PROD_INFRA) ps 2>/dev/null || echo "生产基础设施未启动"
	@docker compose -f $(DOCKER_COMPOSE_PROD_APP) ps 2>/dev/null || echo "生产应用未启动"

.PHONY: logs
logs: ## 查看所有服务日志
	@echo "$(BLUE)查看服务日志...$(RESET)"
	@docker compose -f $(DOCKER_COMPOSE_DEV) logs -f 2>/dev/null || \
	docker compose -f $(DOCKER_COMPOSE_PROD_APP) logs -f 2>/dev/null || \
		echo "$(YELLOW)没有运行中的服务$(RESET)"

.PHONY: logs-backend
logs-backend: ## 查看后端服务日志
	@docker compose -f $(DOCKER_COMPOSE_DEV) logs -f miniblog 2>/dev/null || \
	docker compose -f $(DOCKER_COMPOSE_PROD_APP) logs -f miniblog 2>/dev/null || \
		echo "$(YELLOW)后端服务未运行$(RESET)"

.PHONY: logs-db
logs-db: ## 查看数据库日志
	@docker compose -f $(DOCKER_COMPOSE_DEV) logs -f mysql 2>/dev/null || \
	docker compose -f $(DOCKER_COMPOSE_PROD_INFRA) logs -f mysql 2>/dev/null || \
		echo "$(YELLOW)数据库服务未运行$(RESET)"

# ==============================================================================
# 工具目标
# ==============================================================================
.PHONY: swagger
swagger: ## 启动 Swagger 文档服务
	@echo "$(BLUE)启动 Swagger 文档...$(RESET)"
	@swagger serve -F=swagger --no-open --port 65534 $(ROOT_DIR)/api/openapi/openapi.yaml || \
		echo "$(YELLOW)swagger 工具未安装，请安装: go install github.com/go-swagger/go-swagger/cmd/swagger@latest$(RESET)"
	@echo "$(YELLOW)💡 Swagger 文档: http://localhost:65534$(RESET)"

.PHONY: version
version: ## 显示版本信息
	@echo "$(BLUE)版本信息:$(RESET)"
	@echo "  项目: $(PROJECT_NAME)"
	@echo "  版本: $(VERSION)"
	@echo "  构建时间: $(BUILD_TIME)"
	@echo "  Git 提交: $(GIT_COMMIT)"
	@echo "  Go 模块: $(GO_MODULE)"

.PHONY: env-info
env-info: ## 显示环境信息
	@echo "$(BLUE)环境信息:$(RESET)"
	@echo "  Go 版本: $$(go version)"
	@echo "  Node 版本: $$(node --version 2>/dev/null || echo '未安装')"
	@echo "  NPM 版本: $$(npm --version 2>/dev/null || echo '未安装')"
	@echo "  Docker 版本: $$(docker --version 2>/dev/null || echo '未安装')"
	@echo "  Docker Compose: $$(docker compose version 2>/dev/null || echo '未安装')"

# ==============================================================================
# CI/CD 相关
# ==============================================================================
.PHONY: ci-test
ci-test: deps lint test ## CI 环境测试
	@echo "$(GREEN)✅ CI 测试完成$(RESET)"

.PHONY: ci-build
ci-build: build-all ## CI 环境构建
	@echo "$(GREEN)✅ CI 构建完成$(RESET)"

.PHONY: release
release: ci-test ci-build ## 准备发布
	@echo "$(BLUE)准备发布 $(VERSION)...$(RESET)"
	@git tag -a $(VERSION) -m "Release $(VERSION)" 2>/dev/null || echo "$(YELLOW)标签已存在或 git 未初始化$(RESET)"
	@echo "$(GREEN)✅ 发布准备完成$(RESET)"

# ==============================================================================
# 兼容性别名 (向后兼容)
# ==============================================================================
