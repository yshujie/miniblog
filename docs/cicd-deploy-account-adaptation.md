# CI/CD 适配 deploy 账户安全设计

## 📋 背景

根据服务器的安全设计,deploy 账户有以下特性:

- ✅ 可 SSH 登录（仅密钥认证）
- ✅ 受限 sudo 权限（白名单命令）
- ❌ **不在 docker 组**（避免 root 等价权限）
- ✅ 审计日志记录

因此,CI/CD 脚本需要在所有 Docker 命令前添加 `sudo`。

---

## 🔧 修改内容

### 修改位置

`.github/workflows/cicd.yml` 中的 `deploy` job

### 修改对比

#### ❌ 修改前（不适配安全设计）

```bash
# 创建必要的网络和目录
docker network inspect miniblog_net >/dev/null 2>&1 || docker network create miniblog_net
mkdir -p /data/logs/miniblog/backend /data/logs/miniblog/frontend-blog /data/logs/miniblog/frontend-admin

# GHCR 登录（部署账户已在 docker 组，无需 sudo）
echo "${_GH_TOKEN}" | docker login ghcr.io -u "${_GH_USER}" --password-stdin

# 拉取并滚动更新
docker compose -f docker-compose.yml -f docker-compose.prod.yml pull
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d

# 查看状态
docker compose ps
```

**问题**:

- deploy 不在 docker 组,所有 `docker` 命令都会失败
- 报错: `permission denied while trying to connect to the Docker daemon socket`

#### ✅ 修改后（适配安全设计）

```bash
# 创建必要的网络和目录
sudo docker network inspect miniblog_net >/dev/null 2>&1 || sudo docker network create miniblog_net
sudo mkdir -p /data/logs/miniblog/backend /data/logs/miniblog/frontend-blog /data/logs/miniblog/frontend-admin

# GHCR 登录（使用 sudo，deploy 不在 docker 组）
echo "${_GH_TOKEN}" | sudo docker login ghcr.io -u "${_GH_USER}" --password-stdin

# 拉取并滚动更新
sudo docker compose -f docker-compose.yml -f docker-compose.prod.yml pull
sudo docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d

# 查看状态
sudo docker compose ps
```

**改进**:

- ✅ 所有 Docker 命令都使用 `sudo`
- ✅ 符合 deploy 账户的白名单设计
- ✅ `mkdir -p /data/logs/...` 也使用 `sudo`（需要 root 权限创建 /data 下的目录）

---

## 🔒 安全性分析

### 为什么 deploy 不在 docker 组?

**docker 组 = 准 root 权限**:

```bash
# 如果在 docker 组,可以这样获得 root shell:
docker run -v /:/host -it alpine chroot /host /bin/bash
```

### 为什么使用 sudo + 白名单更安全?

| 对比项 | docker 组方案 ❌ | sudo 白名单方案 ✅ |
|--------|-----------------|-------------------|
| **Docker 命令** | 无限制 | 仅白名单命令 |
| **获取 root** | 可以（挂载根目录） | 不可以 |
| **审计日志** | 无 | 有（/var/log/deployctl.log） |
| **命令限制** | 无 | 有（仅 docker compose 等） |
| **交互式 Shell** | 可以 | 不可以（/usr/sbin/nologin） |

### deploy 账户的白名单命令

根据您的设计,deploy 可以无密码执行:

```bash
# ✅ 允许的 Docker 命令
sudo docker compose ...
sudo docker network ...
sudo docker login ...
sudo docker image prune ...

# ✅ 允许的 systemctl 命令
sudo systemctl start|stop|restart|reload|status|enable|disable <service>

# ✅ 允许的文件操作
sudo mkdir -p /data/...
sudo chown ...
sudo chmod ...

# ✅ 允许的其他命令
sudo rsync ...
```

---

## ✅ 修改验证

### 1. 验证 sudo 无密码执行

在服务器上测试:

```bash
# SSH 登录到服务器（使用 deploy 账户）
ssh -i deploy_key deploy@server

# 测试 Docker 命令（应该无需输入密码）
sudo docker compose version
sudo docker network ls
sudo docker ps

# 如果提示输入密码,说明 sudoers 配置有问题
```

### 2. 验证 CI/CD 部署

触发一次 GitHub Actions workflow:

```bash
git commit --allow-empty -m "test: verify deploy account sudo"
git push origin main
```

观察日志,应该看到:

```log
✅ sudo docker network create miniblog_net
✅ sudo docker login ghcr.io
✅ sudo docker compose pull
✅ sudo docker compose up -d
```

### 3. 查看审计日志

在服务器上:

```bash
# 查看 deploy 账户的操作日志
sudo tail -f /var/log/deployctl.log
```

应该看到类似:

```log
2025-10-26 10:30:15 [deploy] EXEC: docker compose -f docker-compose.yml pull
2025-10-26 10:30:15 [deploy] ALLOWED: docker compose
2025-10-26 10:30:20 [deploy] EXEC: docker compose -f docker-compose.yml up -d
2025-10-26 10:30:20 [deploy] ALLOWED: docker compose
```

---

## 🛡️ 额外的安全改进建议

### 1. 限制 sudo docker login 的风险

**问题**: `sudo docker login` 会将凭据保存在 root 用户的配置中 (`/root/.docker/config.json`)

**建议**: 使用临时凭据或环境变量

```bash
# 方式1: 使用环境变量（不保存凭据）
echo "${_GH_TOKEN}" | sudo DOCKER_CONFIG=/tmp/.docker docker login ghcr.io -u "${_GH_USER}" --password-stdin

# 方式2: 登录后立即拉取,然后登出
echo "${_GH_TOKEN}" | sudo docker login ghcr.io -u "${_GH_USER}" --password-stdin
sudo docker compose pull
sudo docker logout ghcr.io
```

### 2. 限制 /data 目录权限

```bash
# 在服务器上一次性创建并设置权限
sudo mkdir -p /data/logs/miniblog/{backend,frontend-blog,frontend-admin}
sudo chown -R deploy:deploy /data/logs/miniblog

# 这样 CI/CD 就不需要 sudo mkdir 了
```

修改 CI/CD:

```bash
# 如果目录权限已正确设置,可以去掉 sudo
mkdir -p /data/logs/miniblog/backend /data/logs/miniblog/frontend-blog /data/logs/miniblog/frontend-admin
```

### 3. 使用专用部署脚本

创建 `/srv/deploy/miniblog-deploy.sh`:

```bash
#!/bin/bash
set -euo pipefail

cd /opt/miniblog

# 读取 .env 文件
source .env

# 登录 GHCR
echo "${GHCR_TOKEN}" | docker login ghcr.io -u "${GHCR_USER}" --password-stdin

# 创建网络
docker network inspect miniblog_net >/dev/null 2>&1 || docker network create miniblog_net

# 部署
docker compose -f docker-compose.yml -f docker-compose.prod.yml pull
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d

# 清理
docker logout ghcr.io
docker image prune -f

echo "✅ 部署完成"
```

CI/CD 简化为:

```yaml
- name: Deploy on server
  env:
    _GH_TOKEN: ${{ secrets.GHCR_TOKEN }}
    _GH_USER: ${{ github.repository_owner }}
    # ... 其他环境变量
  run: |
    REMOTE="${{ secrets.SVRA_USER }}@${{ secrets.SVRA_HOST }}"
    
    # 上传 .env 文件（包含所有环境变量）
    scp -i ~/.ssh/deploy_key .env "$REMOTE:/opt/miniblog/.env"
    
    # 执行部署脚本
    ssh -i ~/.ssh/deploy_key "$REMOTE" "sudo /srv/deploy/miniblog-deploy.sh"
```

---

## 📊 总结

### 核心修改

所有 Docker 命令添加 `sudo`:

| 命令 | 修改前 | 修改后 |
|------|--------|--------|
| 网络操作 | `docker network create` | `sudo docker network create` |
| 目录创建 | `mkdir -p /data/...` | `sudo mkdir -p /data/...` |
| 登录 | `docker login` | `sudo docker login` |
| 拉取 | `docker compose pull` | `sudo docker compose pull` |
| 启动 | `docker compose up -d` | `sudo docker compose up -d` |
| 查看 | `docker compose ps` | `sudo docker compose ps` |

### 安全收益

- ✅ deploy 账户无 root 等价权限
- ✅ 所有操作有审计日志
- ✅ 仅能执行白名单命令
- ✅ 不能交互式登录
- ✅ 符合最小权限原则

### 兼容性

- ✅ 与现有服务器安全设计完全兼容
- ✅ 与 deployctl 白名单机制配合
- ✅ 不需要修改 deploy 账户配置

---

## 🔗 相关文档

- [用户权限安全加固说明](./user-security-hardening.md)
- [部署账户权限需求分析](./deploy-account-permissions.md)
- [GitHub Actions 使用 deploy 账户部署指南](./github-actions-deploy-guide.md)

---

**最后更新**: 2025-10-26  
**修改版本**: CI/CD v3.0 (适配 deploy 账户安全设计)
