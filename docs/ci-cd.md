# MiniBlog CI/CD 流程文档

## 概述

MiniBlog 项目采用 GitHub Actions 实现完全自动化的 CI/CD 流程。主要包括：

- **持续集成（CI）**：自动化测试、构建 Docker 镜像并推送至 GitHub Container Registry（GHCR）
- **持续部署（CD）**：通过 SSH 自动部署到生产服务器，实现零停机更新
- **运行环境**：自托管 macOS ARM64 Runner（macmini）
- **触发方式**：代码推送到 main 分支或手动触发

## 完整流程图

```text
[代码推送] → [测试阶段] → [构建镜像] → [推送到GHCR] → [SSH部署] → [健康检查] → [Nginx重载]
    ↓            ↓            ↓             ↓            ↓           ↓            ↓
  main分支    单元测试    3个镜像      ghcr.io    服务器A    后端API    反向代理更新
```

## 工作流文件

### 主工作流：`.github/workflows/cicd.yml`

触发条件：

- `push` 到 `main` 分支（自动触发）
- `workflow_dispatch`（手动触发，可选跳过测试）

运行环境：`[self-hosted, macmini, prod]`

### 数据库工作流：`.github/workflows/db-ops.yml`

触发条件：仅支持 `workflow_dispatch` 手动触发

用途：执行数据库初始化、迁移、种子数据导入等一次性操作

## 阶段详解

### 1. 测试阶段（test）

**作用**：确保代码质量，防止有问题的代码进入生产环境

**步骤**：

1. 检出代码（`actions/checkout@v4`）
2. 配置 Go 环境（Go 1.24，启用内置缓存）
3. 修复 Go 模块缓存权限（`chmod -R u+w ~/go/pkg/mod`）
4. 下载依赖（`go mod download`）
5. 运行单元测试（`make test-backend`）

**跳过条件**：手动触发时可通过 `skip_tests` 参数跳过

### 2. 构建与推送阶段（build-and-push）

**作用**：构建 Docker 镜像并推送到 GHCR

**前置条件**：测试通过或跳过测试

**镜像列表**：

| 镜像名称 | 用途 | 构建上下文 |
|---------|------|-----------|
| `ghcr.io/yshujie/miniblog-backend` | Go 后端 API | `.` |
| `ghcr.io/yshujie/miniblog-frontend-blog` | 博客前端 | `web/miniblog-web/` |
| `ghcr.io/yshujie/miniblog-frontend-admin` | 管理后台前端 | `web/miniblog-web-admin/` |

**镜像标签**：

- `<镜像名>:${GITHUB_SHA}` - 精确版本（用于部署）
- `<镜像名>:latest` - 最新版本

**关键技术**：

- 使用 Docker Buildx 支持多平台构建（当前 `linux/amd64`）
- GitHub Actions 缓存（`type=gha`）提升构建速度
- 每个镜像独立缓存 scope 避免冲突
- GHCR 认证通过 base64 编码避免 macOS Keychain 问题

**登录方式**：

```yaml
# 直接写入 Docker 配置文件，绕过 Keychain
AUTH=$(echo -n "USERNAME:TOKEN" | base64)
cat > ~/.docker/config.json <<EOF
{
  "auths": {
    "ghcr.io": {
      "auth": "$AUTH"
    }
  }
}
EOF
```

### 3. 部署阶段（deploy）

**作用**：将新镜像部署到生产服务器

**运行条件**：构建成功后自动执行

**部署流程**：

#### 3.1 SSH 密钥配置

```bash
# 创建临时 SSH 密钥文件
echo "$SSH_DEPLOY_KEY" > ~/.ssh/deploy_key
chmod 600 ~/.ssh/deploy_key
```

#### 3.2 连接到服务器并执行部署

使用原生 SSH（带 `-F /dev/null` 避免本地 SSH 配置干扰）：

```bash
ssh -F /dev/null \
  -i ~/.ssh/deploy_key \
  -o StrictHostKeyChecking=no \
  -o UserKnownHostsFile=/dev/null \
  user@host 'bash -s' <<'ENDSSH'
  
  # 部署脚本内容...
  
ENDSSH
```

#### 3.3 服务器端操作序列

1. **创建必要目录**

   ```bash
   # 尝试普通创建，失败则使用非交互 sudo
   mkdir -p /opt/miniblog /data/logs/miniblog/{backend,frontend-blog,frontend-admin} 2>/dev/null || \
     sudo -n mkdir -p /opt/miniblog /data/logs/miniblog/{backend,frontend-blog,frontend-admin}
   ```

2. **更新代码**

   ```bash
   cd /opt/miniblog
   if [ -d .git ]; then
     git fetch --all -p && git reset --hard origin/main
   else
     git clone github-miniblog:yshujie/miniblog .
   fi
   ```

3. **登录 GHCR**

   ```bash
   echo "$GHCR_TOKEN" | docker login ghcr.io -u "yshujie" --password-stdin
   ```

4. **生成环境配置**

   写入 `.env` 文件，包含：
   - 镜像标签（使用本次构建的 SHA）
   - 数据库连接信息
   - Redis 连接信息
   - JWT 密钥
   - 第三方服务配置（飞书等）

5. **拉取并更新容器**

   ```bash
   docker compose -f docker-compose.yml -f docker-compose.prod.yml pull
   docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d
   ```

6. **网络配置与健康检查**

   ```bash
   # 确保 nginx 在应用网络中
   docker network inspect miniblog_net >/dev/null 2>&1 || docker network create miniblog_net
   docker network connect miniblog_net nginx || true
   
   # 轮询后端健康检查（最多60秒）
   echo "Polling backend health (up to 60s)..."
   for i in $(seq 1 30); do
     if docker run --rm --network=miniblog_net curlimages/curl:8.1.2 \
        -fsS http://miniblog-backend:8080/health >/dev/null 2>&1; then
       echo "Backend healthy after $((i*2))s"
       break
     fi
     sleep 2
   done
   
   # 重载 nginx 配置
   docker exec nginx nginx -t && docker exec nginx nginx -s reload
   ```

7. **最终验证**

   ```bash
   # 等待服务完全启动
   sleep 2
   
   # 验证健康端点
   curl -fsS http://127.0.0.1:8090/health && echo " HEALTH=OK"
   
   # 显示容器状态
   docker ps --format 'table {{.Names}}\t{{.Image}}\t{{.Status}}'
   ```

#### 3.4 清理

```bash
# 无论成功失败都删除临时 SSH 密钥
rm -f ~/.ssh/deploy_key
```

## 镜像存储：为什么使用 GHCR？

### 选择 GitHub Container Registry 的原因

1. **统一认证**：与 GitHub 仓库共享权限体系，使用同一个 Personal Access Token
2. **无缝集成**：GitHub Actions 原生支持，配置简单
3. **无速率限制**：私有镜像拉取无限制（Docker Hub 有严格的速率限制）
4. **安全性更好**：支持细粒度的权限控制
5. **成本更低**：私有镜像存储免费

### 部署时如何拉取镜像？

**关键理解**：服务器不需要访问 Docker Hub，只需要：

1. 能够访问 `ghcr.io`（GHCR 的域名）
2. 拥有有效的 GHCR 认证令牌（`GHCR_TOKEN`）
3. 执行 `docker login ghcr.io` 登录
4. 使用 `docker compose pull` 拉取镜像

**流程示意**：

```
CI/CD Pipeline                      生产服务器
    |                                  |
    | 1. 构建镜像                       |
    |---------------------->          |
    | 2. 推送到 ghcr.io                |
    |                                  |
    |                                  | 3. 登录 ghcr.io
    |                                  | 4. 拉取镜像
    |                          <-------|
    |                                  | 5. 启动容器
```

## 必需的 Secrets 配置

在 GitHub 仓库的 Settings > Secrets and variables > Actions 中配置：

### GHCR 相关

| Secret 名称 | 说明 | 获取方式 |
|-----------|------|---------|
| `GHCR_TOKEN` | GitHub Personal Access Token | Settings > Developer settings > Personal access tokens > Generate new token (classic)，勾选 `write:packages` 权限 |

### 服务器 SSH 相关

| Secret 名称 | 说明 |
|-----------|------|
| `SVRA_HOST` | 生产服务器 IP 或域名 |
| `SVRA_USER` | SSH 登录用户名 |
| `SVRA_SSH_KEY` | SSH 私钥内容（需要提前在服务器添加对应公钥） |

### 应用配置

| Secret 名称 | 说明 |
|-----------|------|
| `MYSQL_HOST` | MySQL 服务器地址 |
| `MYSQL_PORT` | MySQL 端口 |
| `MYSQL_DBNAME` | 数据库名 |
| `MYSQL_USERNAME` | 数据库用户名 |
| `MYSQL_PASSWORD` | 数据库密码 |
| `MYSQL_ROOT_PASSWORD` | MySQL root 密码（仅 db-ops 使用） |
| `REDIS_HOST` | Redis 服务器地址 |
| `REDIS_PORT` | Redis 端口 |
| `REDIS_DB` | Redis 数据库编号 |
| `REDIS_PASSWORD` | Redis 密码 |
| `JWT_SECRET` | JWT 签名密钥 |
| `FEISHU_DOCREADER_APPID` | 飞书应用 ID |
| `FEISHU_DOCREADER_APPSECRET` | 飞书应用密钥 |

## 网络架构与端口映射

### 容器网络拓扑

```
Internet (80/443)
       ↓
[nginx容器] ←─────┐
   ↓ ↓ ↓          │ miniblog_net (Docker网络)
   │ │ │          │
   │ │ └──────────┼→ [miniblog-frontend-admin:8080]
   │ └────────────┼→ [miniblog-frontend-blog:8080]
   └──────────────┼→ [miniblog-backend:8080]
                  │
                  └→ [mysql:3306] [redis:6379]
```

### 端口映射说明

| 容器 | 内部端口 | 宿主端口 | 说明 |
|-----|---------|---------|------|
| nginx | 80/443 | 80/443 | 对外提供 HTTPS 服务 |
| miniblog-backend | 8080 | 8090 | 用于直接健康检查和调试 |
| miniblog-frontend-blog | 8080 | - | 通过 nginx 反向代理 |
| miniblog-frontend-admin | 8080 | - | 通过 nginx 反向代理 |

**为什么前端不直接暴露端口？**

- 所有外部流量通过 nginx 统一入口（TLS 终止、安全头、CORS 等）
- nginx 通过容器名（DNS）反向代理到前端容器
- 减少暴露的攻击面，更安全

## 常见问题与故障排查

### 1. nginx 找不到后端服务（502 Bad Gateway）

**原因**：nginx 容器未连接到 `miniblog_net` 网络，或 upstream 缓存未刷新

**解决方案**（已在部署脚本中自动执行）：

```bash
# 连接 nginx 到应用网络
docker network connect miniblog_net nginx

# 重载 nginx 配置
docker exec nginx nginx -s reload
```

### 2. 部署时 sudo 需要密码导致挂起

**原因**：部署用户没有 NOPASSWD sudo 权限

**解决方案**：

选项 A（推荐）：配置免密 sudo

```bash
# 在服务器上编辑 sudoers
sudo visudo

# 添加以下行（替换 deployuser 为实际用户名）
deployuser ALL=(ALL) NOPASSWD: /bin/mkdir
```

选项 B：提前手动创建目录

```bash
sudo mkdir -p /opt/miniblog /data/logs/miniblog/{backend,frontend-blog,frontend-admin}
sudo chown -R deployuser:deployuser /opt/miniblog /data/logs/miniblog
```

### 3. 镜像拉取失败

**可能原因**：

- GHCR_TOKEN 过期或权限不足
- 服务器无法访问 ghcr.io
- 镜像标签不存在

**排查步骤**：

```bash
# 1. 测试 GHCR 连接
curl -I https://ghcr.io

# 2. 手动测试登录
echo "$GHCR_TOKEN" | docker login ghcr.io -u "yshujie" --password-stdin

# 3. 检查镜像是否存在
docker manifest inspect ghcr.io/yshujie/miniblog-backend:latest

# 4. 查看 GitHub Packages
# 访问：https://github.com/yshujie?tab=packages
```

### 4. 健康检查超时

**原因**：后端启动时间过长或端口未正确暴露

**排查方法**：

```bash
# 检查容器日志
docker logs miniblog-backend --tail 100

# 从容器网络内部测试
docker run --rm --network=miniblog_net curlimages/curl:8.1.2 \
  http://miniblog-backend:8080/health

# 检查端口监听
docker exec miniblog-backend netstat -tlnp | grep 8080
```

## 调试命令速查

### 查看运行状态

```bash
# 查看所有容器
docker ps -a

# 格式化输出
docker ps --format 'table {{.Names}}\t{{.Image}}\t{{.Status}}\t{{.Ports}}'

# 查看特定容器日志
docker logs miniblog-backend --tail 100 -f

# 查看网络连接
docker network inspect miniblog_net
```

### 手动部署测试

```bash
# 在服务器上手动执行部署步骤
cd /opt/miniblog
git pull
docker compose -f docker-compose.yml -f docker-compose.prod.yml pull
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d

# 查看健康状态
curl http://127.0.0.1:8090/health
```

### 回滚操作

```bash
# 查看可用的镜像版本
docker images | grep miniblog

# 使用特定 SHA 标签回滚
docker pull ghcr.io/yshujie/miniblog-backend:<OLD_SHA>
docker tag ghcr.io/yshujie/miniblog-backend:<OLD_SHA> \
           ghcr.io/yshujie/miniblog-backend:latest

# 重新部署
docker compose up -d
```

## 数据库操作（db-ops 工作流）

**重要**：数据库操作具有破坏性，仅在必要时手动触发

### 触发方式

1. 访问：`https://github.com/yshujie/miniblog/actions/workflows/db-ops.yml`
2. 点击 "Run workflow"
3. 选择要执行的操作（取消勾选 skip 参数）

### 可用操作

- **DB Init**：初始化数据库结构（仅首次）
- **DB Migrate**：执行数据库迁移
- **DB Seed**：导入种子数据

## 性能优化

### 缓存策略

- **Go 构建缓存**：`actions/setup-go` 内置缓存
- **Docker 层缓存**：GitHub Actions Cache（type=gha）
- **npm 依赖缓存**：RUN mount 缓存

### 构建优化

- 分离依赖和代码层（提高缓存命中率）
- 使用 `npm ci` 而非 `npm install`（可重复构建）
- Go 构建参数：`-ldflags="-s -w" -trimpath`（减小二进制体积）

### 部署优化

- 健康检查确保服务就绪后才重载 nginx
- 使用 docker compose 实现无停机部署（滚动更新）
- 镜像标签使用 SHA 保证版本精确性

## 安全最佳实践

1. **最小权限原则**：Secrets 仅授予必要权限
2. **SSH 密钥隔离**：使用专用部署密钥，工作流结束后立即删除
3. **网络隔离**：应用容器在独立网络中，仅通过 nginx 对外
4. **配置加密**：所有敏感信息通过 GitHub Secrets 管理
5. **镜像签名**：使用精确的 SHA 标签，防止标签覆盖攻击

## 扩展与定制

### 添加 Docker Hub 支持

如需同时推送到 Docker Hub：

```yaml
- name: Login to Docker Hub
  run: |
    echo "${{ secrets.DOCKERHUB_TOKEN }}" | docker login -u "${{ secrets.DOCKERHUB_USERNAME }}" --password-stdin

- name: Build & Push
  uses: docker/build-push-action@v6
  with:
    tags: |
      ghcr.io/yshujie/miniblog-backend:${{ env.TAG }}
      docker.io/yshujie/miniblog-backend:${{ env.TAG }}
```

### 添加通知

在 workflow 末尾添加通知步骤：

```yaml
- name: Notify
  if: always()
  run: |
    curl -X POST ${{ secrets.WEBHOOK_URL }} \
      -d "status=${{ job.status }}" \
      -d "commit=${{ github.sha }}"
```

## 相关文档

- [网络架构说明](./infrastructure/network-architecture.md)
- [数据库迁移指南](./database-migration-setup.md)
- [Jenkins 配置指南](./jenkins-db-init-guide.md)
