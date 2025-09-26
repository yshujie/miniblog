# MiniBlog

<!-- 一个基于 Gin + Vue 的现代化全栈博客系统，提供完整的内容管理、用户认证和前台展示功能 -->

MiniBlog 是一个企业级的博客管理系统，采用 Go + Vue 技术栈构建，支持模块化内容组织、权限管理和多前端展示。系统设计遵循领域驱动设计（DDD）原则，具备良好的可扩展性和维护性。

## 功能特性

<!-- 描述该项目的核心功能点 -->

### 🚀 核心功能

- **RESTful API 服务**：基于 Gin 框架的 `/v1` REST API，提供完整的 CRUD 操作
- **用户认证与授权**：JWT Token 认证 + Casbin RBAC 权限控制
- **分层内容管理**：模块 → 章节 → 文章的层次化内容组织
- **双前端支持**：博客展示前端（Vue + Vuetify）+ 管理后台（Vue + Ant Design Vue）
- **外部内容集成**：支持飞书文档内容同步，自动转换为 Markdown 格式

### 🔧 技术特性

- **可观测性**：结构化日志、请求追踪、健康检查
- **安全性**：CORS 支持、安全响应头、SQL 注入防护
- **高性能**：Redis 缓存支持、连接池优化
- **容器化部署**：Docker + Docker Compose 一键部署

## 软件架构

<!-- 项目的技术架构设计 -->

### 🏗️ 系统架构

```
┌─────────────────────────────────────────────┐
│                 前端层                      │
├─────────────────┬───────────────────────────┤
│   博客前端      │      管理后台             │
│ (Vue+Vuetify)   │  (Vue+Ant Design Vue)     │
└─────────────────┴───────────────────────────┘
                          │
┌─────────────────────────┼─────────────────────────┐
│                    API 网关                      │
│              (Nginx + SSL)                       │
└─────────────────────────┼─────────────────────────┘
                          │
┌─────────────────────────┼─────────────────────────┐
│                   后端服务                        │
├─────────────────────────┼─────────────────────────┤
│  Controller 层 ← → Biz 层 ← → Store 层 ← → Model 层 │
└─────────────────────────┼─────────────────────────┘
                          │
┌─────────────────────────┼─────────────────────────┐
│                   数据层                         │
├─────────────────────────┼─────────────────────────┤
│     MySQL              │         Redis           │
│   (持久化存储)          │       (缓存)           │
└─────────────────────────┴─────────────────────────┘
```

### 🎯 分层设计

- **Controller 层**：HTTP 请求处理，参数验证，响应格式化
- **Biz 层**：业务逻辑编排，规则验证，事务管理
- **Store 层**：数据访问抽象，GORM 封装
- **Model 层**：领域模型定义，业务实体

### 🔧 核心组件

- **配置管理**：Viper 统一配置，支持文件 + 环境变量
- **权限控制**：Casbin RBAC 模型，gorm-adapter 持久化
- **日志系统**：Zap 结构化日志，请求 ID 追踪
- **中间件**：认证、授权、跨域、安全头、日志记录

## 快速开始

### 依赖检查

<!-- 描述该项目的依赖，比如依赖的包、工具或者其他任何依赖项 -->

**系统要求：**

- **Go 1.23+** - 后端服务开发运行环境
- **MySQL 8.0+** - 主数据库（初始化脚本：`configs/mysql/miniblog.sql`）
- **Redis 6.0+** - 缓存服务（可选）
- **Docker & Docker Compose** - 容器化部署
- **Node.js 18+** - 前端构建（可选）
- **飞书开放平台凭证** - 外部文档同步（可选）

**环境变量配置：**

复制环境变量模板并根据实际情况修改：

```bash
cp .env.example .env
```

主要配置项（均带 `MINIBLOG_` 前缀）：

- **数据库**：`DATABASE_HOST`、`DATABASE_PORT`、`DATABASE_USERNAME`、`DATABASE_PASSWORD`、`DATABASE_DBNAME`
- **Redis**：`REDIS_HOST`、`REDIS_PORT`、`REDIS_PASSWORD`、`REDIS_DB`  
- **JWT**：`JWT_SECRET`
- **飞书**：`FEISHU_DOCREADER_APPID`、`FEISHU_DOCREADER_APPSECRET`

### 构建

<!-- 描述如何构建该项目 -->

**方式一：使用 Makefile（推荐）**

```bash
# 完整构建
make build

# 格式化代码
make format

# 清理构建产物
make clean
```

**方式二：直接使用 Go**

```bash
# 编译二进制文件
go build -o _output/miniblog ./cmd/miniblog

# 整理依赖
go mod tidy
```

构建产物输出到 `_output/miniblog`。

### 运行

<!-- 描述如何运行该项目 -->

**方式一：Docker Compose 一键部署（推荐）**

```bash
# 完整部署（基础设施 + 应用）
make deploy-all

# 分步部署
make deploy-infra     # 部署 MySQL, Redis, Nginx
make deploy-backend   # 部署后端服务  
make deploy-frontend  # 部署前端服务
```

**方式二：本地开发运行**

1. **启动依赖服务**（MySQL、Redis）
2. **配置环境变量**（编辑 `.env` 文件）
3. **启动后端服务**：

```bash
# 使用配置文件启动
./_output/miniblog --config configs/miniblog.yaml

# 或直接运行
go run ./cmd/miniblog -c configs/miniblog.yaml
```

4. **验证服务**：

```bash
# 健康检查
curl http://localhost:8081/health

# 查看服务状态
make status
```

**服务地址：**

- **后端 API**：<http://localhost:8081>
- **博客前端**：<http://localhost:3000>
- **管理后台**：<http://localhost:3001>

## 使用指南

<!-- 描述如何使用该项目 -->

### 🔐 用户认证

**登录获取 Token：**

```bash
curl -X POST http://localhost:8081/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"your-password"}'
```

**使用 Token 访问受保护接口：**

```bash
curl -H "Authorization: Bearer <your-jwt-token>" \
  http://localhost:8081/v1/admin/users/myinfo
```

### 🛠️ 管理端 API（需认证）

**用户管理：**

- 创建用户：`POST /v1/admin/users`
- 修改密码：`GET /v1/admin/users/:name/change-password`
- 获取当前用户信息：`GET /v1/admin/users/myinfo`

**内容管理：**

- **模块管理**：`/v1/admin/modules`（创建、查询模块）
- **章节管理**：`/v1/admin/sections`（创建章节、按模块查询、获取详情）
- **文章管理**：`/v1/admin/articles`（创建、列表、详情、更新、发布/下架）

### 🌐 前台展示 API（公开）

- **模块列表**：`GET /v1/blog/modules`
- **模块详情**：`GET /v1/blog/moduleDetail?moduleCode=<code>`
- **文章详情**：`GET /v1/blog/articleDetail?articleID=<id>`

### 📖 部署管理

**查看服务状态：**

```bash
make status              # 查看所有服务状态
make logs-backend        # 查看后端日志
make logs-infra          # 查看基础设施日志
```

更多 API 详情请参考 `api/openapi/openapi.yaml` 文档。

## 如何贡献

<!-- 告诉其他开发者如何给该项目贡献源码 -->

我们欢迎所有形式的贡献！请遵循以下步骤：

1. **Fork 项目** 并创建功能分支：

   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **本地开发** 并确保代码质量：

   ```bash
   make format          # 格式化代码
   go vet ./...         # 静态检查
   go test ./...        # 运行测试（如有）
   ```

3. **构建验证** 确保项目正常编译：

   ```bash
   make build           # 编译项目
   make deploy-backend  # 测试部署
   ```

4. **提交 PR** 并详细说明：
   - 功能背景和目的
   - 实现方案和关键变更
   - 测试验证方法

5. **代码审查** 后合并到主分支

## 社区

<!-- 如果有需要可以介绍一些社区相关的内容 -->

- **问题反馈**：[GitHub Issues](https://github.com/yshujie/miniblog/issues)
- **功能建议**：欢迎提交 Feature Request
- **技术交流**：可通过 Issues 或 Discussions 参与讨论

## 关于作者

<!-- 这里写上项目作者 -->

- **作者**：杨书杰 (Yang Shujie)
- **邮箱**：<yshujie@163.com>
- **GitHub**：[@yshujie](https://github.com/yshujie)

## 谁在用

<!-- 可以列出使用本项目的其他有影响力的项目，算是给项目打个广告吧 -->

如果您在项目中使用了 MiniBlog，欢迎在 Issues 中告诉我们：

- 您的使用场景
- 遇到的问题和建议
- 希望添加的功能

这将帮助我们更好地改进项目！

## 许可证

本项目目前为个人学习项目，暂未指定开源许可证。

- **学习参考**：欢迎学习和参考代码实现
- **商业使用**：请先联系作者获得授权
- **贡献代码**：提交的代码将遵循项目许可证
