# 部署账户权限需求分析

## 🔍 当前 CI/CD 流程分析

### 执行的操作

在 deploy job 中,通过 SSH 在远程服务器上执行以下操作:

```bash
# 1. 目录操作
mkdir -p /opt/miniblog
cd /opt/miniblog

# 2. 文件操作
cat > .env <<EOF
...
EOF

# 3. 目录创建
mkdir -p /data/logs/miniblog/backend /data/logs/miniblog/frontend-blog /data/logs/miniblog/frontend-admin

# 4. Docker 网络操作
docker network inspect miniblog_net
docker network create miniblog_net

# 5. Docker 登录
echo "${_GH_TOKEN}" | docker login ghcr.io -u "${_GH_USER}" --password-stdin

# 6. Docker Compose 操作
docker compose -f docker-compose.yml -f docker-compose.prod.yml pull
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d
docker compose ps
```

---

## ✅ 权限需求清单

### 1. SSH 登录权限 - **必需** ✅

**需要原因**:

- CI/CD 通过 SSH 连接到远程服务器执行部署命令
- 使用私钥认证 (`DEPLOY_SSH_KEY`)

**配置方式**:

```bash
# 在远程服务器上
# 1. 创建 deploy 用户(如果不存在)
sudo useradd -m -s /bin/bash deploy

# 2. 配置 SSH 公钥认证
sudo mkdir -p /home/deploy/.ssh
sudo vim /home/deploy/.ssh/authorized_keys
# 粘贴公钥内容(对应 DEPLOY_SSH_KEY 的公钥)

sudo chmod 700 /home/deploy/.ssh
sudo chmod 600 /home/deploy/.ssh/authorized_keys
sudo chown -R deploy:deploy /home/deploy/.ssh
```

**验证**:

```bash
# 在 GitHub Actions runner 上测试
ssh -i ~/.ssh/deploy_key deploy@server "whoami"
# 应输出: deploy
```

---

### 2. Docker 组权限 - **必需** ✅

**需要原因**:

- 执行 `docker network` 命令
- 执行 `docker login` 命令
- 执行 `docker compose` 命令
- **所有这些操作都需要访问 Docker daemon socket (`/var/run/docker.sock`)**

**不加入 docker 组的后果**:

```bash
$ docker network ls
permission denied while trying to connect to the Docker daemon socket at unix:///var/run/docker.sock
```

**配置方式**:

```bash
# 在远程服务器上
# 将 deploy 用户加入 docker 组
sudo usermod -aG docker deploy

# 验证组成员
groups deploy
# 应输出: deploy docker

# ⚠️ 重要: 需要重新登录或刷新组权限
# 方式1: 重新登录 SSH
logout
ssh deploy@server

# 方式2: 使用 newgrp (仅当前会话)
newgrp docker

# 方式3: 重启服务器(最彻底,不推荐生产环境)
sudo reboot
```

**验证**:

```bash
# 无需 sudo 即可执行 docker 命令
docker ps
docker network ls
# 应该正常输出,而不是权限错误
```

---

### 3. sudo 权限 - **不需要** ❌

**分析**:

查看当前 CI/CD 脚本,**完全没有使用 sudo**:

```bash
# ❌ 脚本中没有这些命令:
sudo docker ...
sudo docker compose ...
sudo mkdir ...
sudo chown ...
```

**为什么不需要**:

| 操作 | 是否需要 root | 实际权限 |
|------|---------------|----------|
| `mkdir -p /opt/miniblog` | 取决于父目录权限 | deploy 用户对 /opt 有写权限即可 |
| `cat > .env` | 否 | 用户自己的文件 |
| `mkdir -p /data/logs/...` | 取决于父目录权限 | deploy 用户对 /data 有写权限即可 |
| `docker network create` | 否 | docker 组成员即可 |
| `docker login` | 否 | docker 组成员即可 |
| `docker compose pull/up` | 否 | docker 组成员即可 |

**结论**:

- ✅ deploy 账户 **不需要** sudo 权限
- ✅ 符合最小权限原则(Principle of Least Privilege)
- ✅ 更安全,降低误操作或被攻击的风险

**例外情况**:
如果未来需要执行以下操作,才需要考虑 sudo:

- 安装系统软件包 (`apt install`, `yum install`)
- 修改系统配置 (`/etc/...`)
- 管理系统服务 (`systemctl`)
- 修改其他用户的文件
- 绑定特权端口 (< 1024,但现在用 8090 端口)

---

## 📁 目录权限配置

### 需要 deploy 用户有写权限的目录

```bash
# 1. 应用目录
/opt/miniblog/          # 存放 docker-compose.yml 和 .env

# 2. 日志目录
/data/logs/miniblog/    # 容器日志挂载
├── backend/
├── frontend-blog/
└── frontend-admin/

# 3. 数据持久化目录 (如果有)
/data/miniblog/         # 可选,如果需要持久化数据
```

### 配置方式

**方案 1: 改变目录所有者 (推荐)**

```bash
# 在远程服务器上,以 root 身份执行
sudo mkdir -p /opt/miniblog
sudo chown -R deploy:deploy /opt/miniblog

sudo mkdir -p /data/logs/miniblog
sudo chown -R deploy:deploy /data/logs/miniblog
```

**方案 2: 使用 ACL (更精细)**

```bash
sudo mkdir -p /opt/miniblog
sudo setfacl -R -m u:deploy:rwx /opt/miniblog
sudo setfacl -R -d -m u:deploy:rwx /opt/miniblog  # 默认权限
```

**方案 3: 使用共享组**

```bash
# 创建专用组
sudo groupadd miniblog-app

# 添加 deploy 到组
sudo usermod -aG miniblog-app deploy

# 设置目录权限
sudo mkdir -p /opt/miniblog
sudo chown root:miniblog-app /opt/miniblog
sudo chmod 775 /opt/miniblog
```

**验证**:

```bash
# 以 deploy 用户身份测试
ssh deploy@server "touch /opt/miniblog/test.txt"
ssh deploy@server "mkdir -p /data/logs/miniblog/test"
# 应该成功,无权限错误
```

---

## 🔒 安全考虑

### 1. Docker 组的安全风险

**⚠️ 警告**: docker 组成员 = 准 root 权限

**原因**:

```bash
# docker 组成员可以这样获得 root shell:
docker run -v /:/host -it alpine chroot /host /bin/bash
```

**缓解措施**:

✅ **推荐做法**:

1. 专用部署账户,不用于日常操作
2. 禁用密码登录,仅允许密钥认证
3. 限制 SSH 来源 IP (在 `sshd_config` 或防火墙)
4. 启用审计日志,监控 docker 命令执行
5. 定期轮换 SSH 密钥

```bash
# /etc/ssh/sshd_config
Match User deploy
    PasswordAuthentication no
    PubkeyAuthentication yes
    AllowUsers deploy@192.168.1.0/24  # 限制来源 IP
```

❌ **不要做**:

1. 不要用 deploy 账户做日常登录
2. 不要共享 deploy 账户的密钥
3. 不要给 deploy 账户 sudo 权限(除非真的需要)

---

### 2. SSH 密钥管理

**最佳实践**:

```bash
# 1. 生成专用密钥对(在本地)
ssh-keygen -t ed25519 -C "github-actions-deploy" -f deploy_key
# 或使用 RSA 4096 位
ssh-keygen -t rsa -b 4096 -C "github-actions-deploy" -f deploy_key

# 2. 限制私钥使用(可选,高级)
# 在服务器的 authorized_keys 中添加限制
cat >> /home/deploy/.ssh/authorized_keys <<EOF
command="/opt/scripts/deploy-only.sh",no-port-forwarding,no-X11-forwarding,no-agent-forwarding ssh-ed25519 AAAA...
EOF

# 3. deploy-only.sh 脚本仅允许特定命令
#!/bin/bash
case "$SSH_ORIGINAL_COMMAND" in
    "docker compose -f"*) eval "$SSH_ORIGINAL_COMMAND" ;;
    "docker network"*) eval "$SSH_ORIGINAL_COMMAND" ;;
    "mkdir -p /opt/miniblog") eval "$SSH_ORIGINAL_COMMAND" ;;
    *) echo "Command not allowed"; exit 1 ;;
esac
```

---

## ✅ 完整配置清单

### 在远程服务器上执行(一次性配置)

```bash
#!/bin/bash
# deploy-account-setup.sh

# 1. 创建 deploy 用户
sudo useradd -m -s /bin/bash deploy

# 2. 创建必要目录并设置权限
sudo mkdir -p /opt/miniblog
sudo mkdir -p /data/logs/miniblog/{backend,frontend-blog,frontend-admin}
sudo chown -R deploy:deploy /opt/miniblog /data/logs/miniblog

# 3. 将 deploy 加入 docker 组
sudo usermod -aG docker deploy

# 4. 配置 SSH 公钥
sudo mkdir -p /home/deploy/.ssh
sudo tee /home/deploy/.ssh/authorized_keys > /dev/null <<EOF
# 粘贴你的公钥(对应 DEPLOY_SSH_KEY 的公钥)
ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIxxx... github-actions-deploy
EOF

sudo chmod 700 /home/deploy/.ssh
sudo chmod 600 /home/deploy/.ssh/authorized_keys
sudo chown -R deploy:deploy /home/deploy/.ssh

# 5. 禁用 deploy 账户的密码登录
sudo passwd -l deploy

# 6. 验证
echo "验证 deploy 用户配置:"
id deploy
groups deploy
ls -ld /opt/miniblog /data/logs/miniblog
ls -la /home/deploy/.ssh/

echo "配置完成! 请重新登录以使组权限生效"
echo "验证命令: ssh -i deploy_key deploy@server 'docker ps'"
```

---

## 📊 权限需求总结表

| 权限类型 | 是否需要 | 用途 | 配置命令 |
|---------|---------|------|---------|
| **SSH 登录** | ✅ 必需 | 远程执行部署命令 | 配置 `authorized_keys` |
| **docker 组** | ✅ 必需 | 执行所有 docker 命令 | `usermod -aG docker deploy` |
| **sudo 权限** | ❌ 不需要 | 当前脚本无 sudo 操作 | 无需配置 |
| **/opt/miniblog 写权限** | ✅ 必需 | 存放 compose 文件和 .env | `chown deploy:deploy /opt/miniblog` |
| **/data/logs 写权限** | ✅ 必需 | 容器日志持久化 | `chown deploy:deploy /data/logs/miniblog` |
| **root 权限** | ❌ 不需要 | 无任何 root 操作 | 无需配置 |

---

## 🔄 验证流程

### 在 GitHub Actions runner 上测试

```bash
# 1. 测试 SSH 连接
ssh -i deploy_key deploy@server "whoami"
# 期望输出: deploy

# 2. 测试 docker 命令
ssh -i deploy_key deploy@server "docker ps"
# 应该正常输出,无权限错误

# 3. 测试目录写入
ssh -i deploy_key deploy@server "touch /opt/miniblog/test.txt"
ssh -i deploy_key deploy@server "ls -l /opt/miniblog/test.txt"
# 应该成功创建文件

# 4. 测试 docker compose
ssh -i deploy_key deploy@server "cd /opt/miniblog && docker compose version"
# 应该输出版本号

# 5. 测试完整部署流程(模拟)
ssh -i deploy_key deploy@server 'bash -s' <<'EOF'
set -euo pipefail
cd /opt/miniblog || exit 1
docker network inspect miniblog_net >/dev/null 2>&1 || docker network create miniblog_net
mkdir -p /data/logs/miniblog/backend
echo "All permissions OK!"
EOF
# 期望输出: All permissions OK!
```

---

## 🎯 最佳实践建议

### 1. 最小权限原则

✅ **当前配置符合最小权限原则**:

- 仅给予必要的 docker 组权限
- 不给予不必要的 sudo 权限
- 专用账户,职责单一

### 2. 目录隔离

```bash
/opt/miniblog/              # 应用配置目录
├── docker-compose.yml      # 部署配置
├── docker-compose.prod.yml # 生产环境覆盖
└── .env                    # 环境变量(部署时生成)

/data/logs/miniblog/        # 日志目录(与应用配置分离)
├── backend/
├── frontend-blog/
└── frontend-admin/

/data/ssl/                  # SSL 证书(与应用配置分离)
├── certs/
└── private/
```

### 3. 审计日志

```bash
# 启用 Docker 日志审计
# /etc/docker/daemon.json
{
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "10m",
    "max-file": "3"
  }
}

# 监控 deploy 用户的操作
sudo auditctl -w /opt/miniblog -p wa -k miniblog_deploy
sudo auditctl -w /var/run/docker.sock -p rw -k docker_access
```

---

## 📚 相关文档

- [Docker 安全最佳实践](https://docs.docker.com/engine/security/)
- [Linux 用户和组管理](https://www.linux.com/training-tutorials/how-manage-users-groups-linux/)
- [SSH 公钥认证](https://www.ssh.com/academy/ssh/public-key-authentication)
- [Docker daemon socket 权限](https://docs.docker.com/engine/install/linux-postinstall/#manage-docker-as-a-non-root-user)

---

## 🔑 结论

对于当前的 CI/CD 流程,部署账户 `deploy` 的权限需求:

| 权限 | 需要 | 说明 |
|------|------|------|
| SSH 登录 | ✅ 是 | 远程执行部署命令的前提 |
| docker 组 | ✅ 是 | 执行所有 docker 和 docker compose 命令 |
| sudo 权限 | ❌ 否 | 当前脚本完全无 sudo 操作 |

**配置步骤**:

1. 创建 deploy 用户
2. 配置 SSH 公钥认证
3. 加入 docker 组 (`usermod -aG docker deploy`)
4. 设置必要目录的写权限
5. **不要**给予 sudo 权限(除非未来真的需要)

这样的配置既满足功能需求,又符合安全最佳实践! 🔒
