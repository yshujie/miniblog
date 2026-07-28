# MiniBlog 项目 Makefile - 专注后端和前端服务管理
# 依赖外部 MySQL、Redis、Nginx 服务，不关心服务提供方式
.DEFAULT_GOAL := help

# 项目变量
PROJECT_NAME := miniblog
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "v0.0.0-dev")
OUTPUT_DIR := ./_output

# Go 构建变量
GO_MODULE := $(shell head -1 go.mod | awk '{print $$2}')
LDFLAGS := -X '$(GO_MODULE)/internal/pkg/core.Version=$(VERSION)'

# 前端服务路径
BLOG_FRONTEND_DIR := web/miniblog-web
ADMIN_FRONTEND_DIR := web/miniblog-web-admin

.PHONY: help
help: ## 显示帮助信息
	@echo "MiniBlog 项目管理命令:"
	@echo ""
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_0-9-]+:.*?## / {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: check-deps
check-deps: ## 检查依赖服务
	@echo "检查依赖服务..."
	@echo "检查 Docker..."
	@docker --version > /dev/null 2>&1 || (echo "❌ Docker 未安装或未启动" && exit 1)
	@echo "✅ Docker 正常"
	@echo "检查 MySQL 服务..."
	@docker ps --format '{{.Names}}' | grep -E "(mysql|infra-mysql)" > /dev/null || (echo "⚠️  MySQL 服务未运行，请确保有可用的 MySQL 服务")
	@echo "检查 Redis 服务..."
	@docker ps --format '{{.Names}}' | grep -E "(redis|infra-redis)" > /dev/null || (echo "⚠️  Redis 服务未运行，请确保有可用的 Redis 服务")
	@echo "检查 Nginx 服务..."
	@docker ps --format '{{.Names}}' | grep -E "(nginx|infra-nginx)" > /dev/null || (echo "⚠️  Nginx 服务未运行，请确保有可用的 Nginx 服务")
	@echo "✅ 依赖服务检查完成"

# ==============================================================================
# 后端服务管理
# ==============================================================================

.PHONY: build-backend
build-backend: ## 构建后端服务
	@echo "构建后端服务..."
	@mkdir -p $(OUTPUT_DIR)
	@go mod download
	@go build -ldflags "$(LDFLAGS)" -o $(OUTPUT_DIR)/$(PROJECT_NAME) ./cmd/$(PROJECT_NAME)
	@echo "✅ 后端服务构建完成"

.PHONY: docker-build-backend
docker-build-backend: ## 构建后端 Docker 镜像，需要传入 IMAGE_NAME
	@if [ -z "$(IMAGE_NAME)" ]; then echo "❌ 缺少 IMAGE_NAME 变量，例如 IMAGE_NAME=miniblog-backend:prod"; exit 1; fi
	@echo "准备后端依赖..."
	@go mod download
	@echo "构建后端 Docker 镜像 $(IMAGE_NAME)..."
	@docker build -f build/docker/miniblog/Dockerfile.prod.backend -t $(IMAGE_NAME) .

.PHONY: run-backend
run-backend: build-backend ## 运行后端服务
	@echo "启动后端服务..."
	@$(OUTPUT_DIR)/$(PROJECT_NAME) -c configs/miniblog.yaml

.PHONY: dev-backend
dev-backend: ## 后端开发模式（热重载）
	@echo "启动后端开发模式..."
	@air -c .air.toml

.PHONY: test-backend
test-backend: ## 运行后端测试
	@echo "运行后端测试..."
	@go test -v ./...

.PHONY: format-backend
format-backend: ## 格式化后端代码
	@echo "格式化后端代码..."
	@go fmt ./...

# ==============================================================================
# 博客前端服务管理
# ==============================================================================

.PHONY: build-blog
build-blog: ## 构建博客前端
	@echo "构建博客前端..."
	@if [ -d "$(BLOG_FRONTEND_DIR)" ]; then \
		cd $(BLOG_FRONTEND_DIR) && npm install && npm run build; \
		echo "✅ 博客前端构建完成"; \
	else \
		echo "❌ 博客前端目录不存在: $(BLOG_FRONTEND_DIR)"; \
	fi

.PHONY: dev-blog
dev-blog: ## 博客前端开发模式
	@echo "启动博客前端开发服务器..."
	@if [ -d "$(BLOG_FRONTEND_DIR)" ]; then \
		echo "博客前端: http://localhost:3000"; \
		cd $(BLOG_FRONTEND_DIR) && npm install && npm run dev; \
	else \
		echo "❌ 博客前端目录不存在: $(BLOG_FRONTEND_DIR)"; \
	fi

.PHONY: install-blog
install-blog: ## 安装博客前端依赖
	@echo "安装博客前端依赖..."
	@if [ -d "$(BLOG_FRONTEND_DIR)" ]; then \
		cd $(BLOG_FRONTEND_DIR) && npm install; \
		echo "✅ 博客前端依赖安装完成"; \
	else \
		echo "❌ 博客前端目录不存在: $(BLOG_FRONTEND_DIR)"; \
	fi

# ==============================================================================
# 管理后台服务管理
# ==============================================================================

.PHONY: build-admin
build-admin: ## 构建管理后台
	@echo "构建管理后台..."
	@if [ -d "$(ADMIN_FRONTEND_DIR)" ]; then \
		cd $(ADMIN_FRONTEND_DIR) && npm install && npm run build:prod; \
		echo "✅ 管理后台构建完成"; \
	else \
		echo "❌ 管理后台目录不存在: $(ADMIN_FRONTEND_DIR)"; \
	fi

.PHONY: docker-build-frontend-blog
docker-build-frontend-blog: ## 构建博客前端 Docker 镜像，需要传入 IMAGE_NAME
	@if [ -z "$(IMAGE_NAME)" ]; then echo "❌ 缺少 IMAGE_NAME 变量，例如 IMAGE_NAME=miniblog-frontend-blog:prod"; exit 1; fi
	@echo "构建博客前端 Docker 镜像 $(IMAGE_NAME)..."
	@FLAGS=""; \
	if [ "$(NO_CACHE)" = "true" ]; then \
	  echo "-> 禁用 Docker 构建缓存 (--no-cache --pull)"; \
	  FLAGS="$$FLAGS --no-cache --pull"; \
	fi; \
	if [ -n "$(DOCKER_BUILD_ARGS)" ]; then \
	  echo "-> 附加 Docker 构建参数: $(DOCKER_BUILD_ARGS)"; \
	  FLAGS="$$FLAGS $(DOCKER_BUILD_ARGS)"; \
	fi; \
	docker build $$FLAGS -f build/docker/miniblog/Dockerfile.prod.frontend.blog -t $(IMAGE_NAME) $(BLOG_FRONTEND_DIR)

.PHONY: docker-build-frontend-admin
docker-build-frontend-admin: ## 构建管理后台 Docker 镜像，需要传入 IMAGE_NAME
	@if [ -z "$(IMAGE_NAME)" ]; then echo "❌ 缺少 IMAGE_NAME 变量，例如 IMAGE_NAME=miniblog-frontend-admin:prod"; exit 1; fi
	@echo "构建管理后台 Docker 镜像 $(IMAGE_NAME)..."
	@FLAGS=""; \
	BUILD_ARGS="$(DOCKER_BUILD_ARGS)"; \
	if [ "$(NO_CACHE)" = "true" ]; then \
	  echo "-> 禁用 Docker 构建缓存 (--no-cache --pull)"; \
	  FLAGS="$$FLAGS --no-cache --pull"; \
	fi; \
	if [ -n "$(DOCKER_BUILD_ARGS)" ]; then \
	  echo "-> 附加 Docker 构建参数: $(DOCKER_BUILD_ARGS)"; \
	fi; \
	if [ -n "$$BUILD_ARGS" ]; then \
	  echo "-> 最终 Docker 构建参数: $$BUILD_ARGS"; \
	fi; \
	docker build $$FLAGS $$BUILD_ARGS -f build/docker/miniblog/Dockerfile.prod.frontend.admin -t $(IMAGE_NAME) $(ADMIN_FRONTEND_DIR)

.PHONY: dev-admin
dev-admin: ## 管理后台开发模式
	@echo "启动管理后台开发服务器..."
	@if [ -d "$(ADMIN_FRONTEND_DIR)" ]; then \
		echo "管理后台: http://localhost:3001"; \
		cd $(ADMIN_FRONTEND_DIR) && npm install && npm run dev; \
	else \
		echo "❌ 管理后台目录不存在: $(ADMIN_FRONTEND_DIR)"; \
	fi

.PHONY: install-admin
install-admin: ## 安装管理后台依赖
	@echo "安装管理后台依赖..."
	@if [ -d "$(ADMIN_FRONTEND_DIR)" ]; then \
		cd $(ADMIN_FRONTEND_DIR) && npm install; \
		echo "✅ 管理后台依赖安装完成"; \
	else \
		echo "❌ 管理后台目录不存在: $(ADMIN_FRONTEND_DIR)"; \
	fi

# ==============================================================================
# 组合命令 - 兼容性和便利性
# ==============================================================================

.PHONY: build
build: build-backend ## 构建后端服务（默认）

.PHONY: build-all
build-all: build-backend build-blog build-admin ## 构建所有服务

.PHONY: install-all
install-all: install-blog install-admin ## 安装所有前端依赖

.PHONY: dev-all
dev-all: ## 启动所有服务的开发模式
	@echo "启动所有服务的开发模式..."
	@echo "后端服务: http://localhost:8081"
	@echo "博客前端: http://localhost:3000"
	@echo "管理后台: http://localhost:3001"
	@echo ""
	@echo "请在不同终端中运行："
	@echo "  make dev-backend  # 后端服务"
	@echo "  make dev-blog     # 博客前端"
	@echo "  make dev-admin    # 管理后台"

.PHONY: test
test: test-backend ## 运行测试（默认后端）

.PHONY: format
format: format-backend ## 格式化代码（默认后端）

# 保持向后兼容
.PHONY: build-frontend
build-frontend: build-blog build-admin ## 构建前端（兼容命令）

.PHONY: dev
dev: dev-backend ## 开发模式（默认后端）

.PHONY: run
run: run-backend ## 运行服务（默认后端）

# ==============================================================================
# Docker 部署管理
# ==============================================================================

.PHONY: compose-up
compose-up: ## 使用 docker compose 启动服务，需要传入 FILES（空格分隔），可选 PULL=true
	@set -e; \
	FILES="$(strip $(FILES))"; \
	if [ -z "$$FILES" ]; then FILES="docker-compose.yml"; fi; \
	CMD="docker compose"; \
	for file in $$FILES; do \
		CMD="$$CMD -f $$file"; \
	done; \
	echo "使用 $$CMD"; \
	if [ "$(PULL)" = "true" ]; then \
		echo "拉取最新镜像..."; \
		$$CMD pull --ignore-pull-failures; \
	else \
		echo "跳过 docker compose pull"; \
	fi; \
	$$CMD up -d

.PHONY: deploy
deploy: ## 部署所有服务
	@echo "部署 MiniBlog 所有服务..."
	@$(MAKE) compose-up FILES="docker-compose.yml" PULL=true
	@echo "✅ 部署完成"
	@echo "服务地址："
	@echo "  后端API: http://localhost:8081"
	@echo "  博客前端: http://localhost:3000"
	@echo "  管理后台: http://localhost:3001"

.PHONY: deploy-dev
deploy-dev: ## 部署开发环境
	@echo "部署开发环境..."
	@$(MAKE) compose-up FILES="docker-compose.yml docker-compose.dev.yml" PULL=true
	@echo "✅ 开发环境部署完成"

.PHONY: deploy-prod
deploy-prod: ## 部署生产环境  
	@echo "部署生产环境..."
	@$(MAKE) compose-up FILES="docker-compose.yml docker-compose.prod.yml" PULL=true
	@echo "✅ 生产环境部署完成"

.PHONY: db-migrate
db-migrate: ## 运行数据库迁移（优先使用本地 migrate 二进制，否则使用 dockerized migrate 镜像）
	@echo "Running DB migrations..."
	@DB_HOST=$${DB_HOST:-$${MYSQL_HOST:-mysql}}; \
	DB_PORT=$${DB_PORT:-$${MYSQL_PORT:-3306}}; \
	DB_USER=$${DB_USER:-$${MYSQL_USERNAME:-miniblog}}; \
	DB_PASSWORD=$${DB_PASSWORD:-$${MYSQL_PASSWORD:-miniblog123}}; \
	DB_NAME=$${DB_NAME:-$${MYSQL_DBNAME:-$${MYSQL_DATABASE:-miniblog}}}; \
	if ! MIGRATIONS_DIR="$$(cd db/migrations/sql 2>/dev/null && pwd)"; then \
		echo "❌ Migrations directory not found: $$(pwd)/db/migrations/sql"; \
		exit 1; \
	fi; \
	DB_URL="mysql://$${DB_USER}:$${DB_PASSWORD}@tcp($${DB_HOST}:$${DB_PORT})/$${DB_NAME}?multiStatements=true"; \
	DB_URL_REDACTED=$$(echo "$$DB_URL" | sed -E 's#(//[^:]+:)[^@]+@#\1****@#'); \
	echo "[db-migrate] DB_URL=$${DB_URL_REDACTED}"; \
	if command -v migrate >/dev/null 2>&1; then \
		echo "-> Using local migrate binary"; \
		migrate -path "$$MIGRATIONS_DIR" -database "$$DB_URL" up; \
	else \
		echo "-> Local migrate binary not found, using dockerized migrate image"; \
		DOCKER_NET=$${DOCKER_NETWORK:-infra-network}; \
		docker run --rm --network "$$DOCKER_NET" -v "$$MIGRATIONS_DIR:/migrations:ro" migrate/migrate -path /migrations -database "$$DB_URL" up; \
	fi

.PHONY: db-seed
db-seed: ## 加载种子数据（初始化数据：用户、模块、文章等）
	@echo "Loading seed data..."
	@bash scripts/load-seed-data.sh

.PHONY: db-init
db-init: ## 初始化数据库（执行初始 SQL 脚本，幂等）。需要有数据库管理员权限来创建数据库/用户
	@echo "Running DB initialization..."
	@DB_HOST="$${DB_HOST:-$${MYSQL_HOST:-mysql}}"; \
	DB_PORT="$${DB_PORT:-$${MYSQL_PORT:-3306}}"; \
	DB_ROOT_USER="$${DB_ROOT_USER:-root}"; \
	DB_ROOT_PASSWORD="$${DB_ROOT_PASSWORD:-}"; \
	APP_DB_NAME="$${DB_NAME:-$${MYSQL_DBNAME:-$${MYSQL_DATABASE:-miniblog}}}"; \
	APP_DB_USER="$${DB_USER:-$${MYSQL_USERNAME:-miniblog}}"; \
	APP_DB_PASSWORD="$${DB_PASSWORD:-$${MYSQL_PASSWORD:-miniblog123}}"; \
	SCRIPT=./db/migrations/mysql/init_db.sql; \
	echo "-> Debug: DB_HOST=$$DB_HOST, DB_PORT=$$DB_PORT, DB_ROOT_USER=$$DB_ROOT_USER"; \
	echo "-> Debug: APP_DB_NAME=$$APP_DB_NAME, APP_DB_USER=$$APP_DB_USER"; \
	echo "-> Creating database: $$APP_DB_NAME, user: $$APP_DB_USER"; \
	if command -v mysql >/dev/null 2>&1; then \
		echo "-> Using local mysql client to execute init script $$SCRIPT"; \
		sed -e "s/\$${APP_DB_NAME}/$$APP_DB_NAME/g" \
		    -e "s/\$${APP_DB_USER}/$$APP_DB_USER/g" \
		    -e "s/\$${APP_DB_PASSWORD}/$$APP_DB_PASSWORD/g" \
		    "$$SCRIPT" | MYSQL_PWD="$$DB_ROOT_PASSWORD" mysql -h "$$DB_HOST" -P "$$DB_PORT" -u "$$DB_ROOT_USER"; \
	else \
		command -v docker >/dev/null 2>&1 || { echo "❌ mysql client and docker are both unavailable"; exit 1; }; \
		DOCKER_NET=$${DOCKER_NETWORK:-infra-network}; \
		echo "-> Local mysql client not found, using mysql:8.0 on $$DOCKER_NET"; \
		sed -e "s/\$${APP_DB_NAME}/$$APP_DB_NAME/g" \
		    -e "s/\$${APP_DB_USER}/$$APP_DB_USER/g" \
		    -e "s/\$${APP_DB_PASSWORD}/$$APP_DB_PASSWORD/g" \
		    "$$SCRIPT" | MYSQL_PWD="$$DB_ROOT_PASSWORD" docker run --rm -i --network "$$DOCKER_NET" \
		      -e MYSQL_PWD mysql:8.0 \
		      mysql -h "$$DB_HOST" -P "$$DB_PORT" -u "$$DB_ROOT_USER"; \
	fi

.PHONY: down
down: ## 停止应用服务
	@docker compose down

.PHONY: status
status: ## 查看服务状态
	@echo "应用服务:"
	@docker compose ps
	@echo "基础设施服务:"
	@docker ps --filter "name=infra-" --format "table {{.Names}}\t{{.Status}}"

.PHONY: logs
logs: ## 查看应用日志
	@docker compose logs -f

.PHONY: health
health: ## 健康检查
	@curl -s http://localhost:8081/health || echo "后端服务未启动"

# ==============================================================================
# 实用工具
# ==============================================================================

.PHONY: clean
clean: ## 清理构建产物
	@echo "清理构建产物..."
	@rm -rf $(OUTPUT_DIR)
	@if [ -d "$(BLOG_FRONTEND_DIR)/dist" ]; then rm -rf $(BLOG_FRONTEND_DIR)/dist; fi
	@if [ -d "$(ADMIN_FRONTEND_DIR)/dist" ]; then rm -rf $(ADMIN_FRONTEND_DIR)/dist; fi
	@echo "✅ 清理完成"

.PHONY: docker-network-ensure
docker-network-ensure: ## 确保 Docker 网络存在，需要传入 NETWORK
	@if [ -z "$(NETWORK)" ]; then echo "❌ 缺少 NETWORK 变量，例如 NETWORK=miniblog-network"; exit 1; fi
	@if ! docker network ls --format '{{.Name}}' | grep -w "$(NETWORK)" >/dev/null 2>&1; then \
		echo "创建 Docker 网络 $(NETWORK)..."; \
		docker network create "$(NETWORK)"; \
	else \
		echo "Docker 网络 $(NETWORK) 已存在"; \
	fi

.PHONY: docker-push-image
docker-push-image: ## 推送 Docker 镜像，需要传入 IMAGE_NAME
	@if [ -z "$(IMAGE_NAME)" ]; then echo "❌ 缺少 IMAGE_NAME 变量，例如 IMAGE_NAME=miniblog-backend:prod"; exit 1; fi
	@echo "推送 Docker 镜像 $(IMAGE_NAME)..."
	@docker push $(IMAGE_NAME)

.PHONY: docker-prune-images
docker-prune-images: ## 清理悬空 Docker 镜像
	@docker image prune -f

.PHONY: clean-deps
clean-deps: ## 清理前端依赖
	@echo "清理前端依赖..."
	@if [ -d "$(BLOG_FRONTEND_DIR)/node_modules" ]; then rm -rf $(BLOG_FRONTEND_DIR)/node_modules; fi
	@if [ -d "$(ADMIN_FRONTEND_DIR)/node_modules" ]; then rm -rf $(ADMIN_FRONTEND_DIR)/node_modules; fi
	@echo "✅ 前端依赖清理完成"

.PHONY: version
version: ## 显示版本
	@echo "$(PROJECT_NAME) $(VERSION)"

.PHONY: env-setup
env-setup: ## 初始化环境配置
	@if [ ! -f .env ]; then cp .env.example .env; echo "已创建 .env"; fi

.PHONY: info
info: ## 显示项目信息
	@echo "MiniBlog 项目信息:"
	@echo "===================="
	@echo "项目名称: $(PROJECT_NAME)"
	@echo "版本: $(VERSION)"
	@echo "后端服务: Go $(shell go version | awk '{print $$3}')"
	@echo "博客前端: $(if $(shell test -d $(BLOG_FRONTEND_DIR) && echo 1),Vue.js,未安装)"
	@echo "管理后台: $(if $(shell test -d $(ADMIN_FRONTEND_DIR) && echo 1),Vue.js + Element UI,未安装)"
	@echo ""
	@echo "服务地址:"
	@echo "  后端API: http://localhost:8081"
	@echo "  博客前端: http://localhost:3000"
	@echo "  管理后台: http://localhost:3001"

# ==============================================================================
# 快速启动命令
# ==============================================================================

.PHONY: start-dev
start-dev: check-deps deploy-dev ## 启动完整开发环境
	@echo "🚀 开发环境启动完成"
	@echo "后端API: http://localhost:8081"
	@echo "博客前端: http://localhost:3000"  
	@echo "管理后台: http://localhost:3001"
	@echo ""
	@echo "💡 提示: 使用 'make logs' 查看服务日志"
