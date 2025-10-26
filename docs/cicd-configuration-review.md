# CI/CD 配置审查报告

## 审查时间

2025年

## 概述

对 miniblog 项目的 CI/CD 配置进行全面审查,发现并修复了 5 个关键问题。

---

## 发现的问题及修复方案

### ❌ 问题 1: 环境变量在 SSH heredoc 中无法正确展开

**问题描述:**
原始部署脚本使用 `<<'EOSSH'` (单引号) heredoc,导致 GitHub secrets 变量无法在远程服务器上展开:

```yaml
ssh -i ~/.ssh/deploy_key "$REMOTE" 'bash -s' <<'EOSSH'
  # ... 在此处的 ${{ secrets.MYSQL_HOST }} 会被当作字面量传递
  MYSQL_HOST=${{ secrets.MYSQL_HOST }}
EOSSH
```

**根本原因:**

- 单引号 heredoc (`<<'EOF'`) 会阻止变量展开
- GitHub Actions 的 `${{ }}` 表达式只在 YAML 解析阶段展开,不会在 shell heredoc 内部展开

**修复方案:**

1. 将所有 GitHub secrets 先设置为环境变量 (`env:`)
2. 在本地构建 `.env` 文件,使用 `sed` 替换所有变量
3. 通过 `scp` 上传已展开的 `.env` 文件到远程服务器
4. 远程脚本直接使用 `.env` 文件,不再依赖变量传递

**修复后代码:**

```yaml
- name: Deploy on server
  env:
    BACKEND_TAG: ${{ env.BACKEND_IMAGE }}:${{ env.TAG }}
    MYSQL_HOST: ${{ secrets.MYSQL_HOST }}
    # ... 其他所有 secrets
  run: |
    # 本地生成 .env
    cat > /tmp/miniblog.env << 'ENVEOF'
    MYSQL_HOST=${MYSQL_HOST}
    ENVEOF
    
    # 用实际值替换
    sed -i '' -e "s|\${MYSQL_HOST}|${MYSQL_HOST}|g" /tmp/miniblog.env
    
    # 上传到远程
    scp -i ~/.ssh/deploy_key /tmp/miniblog.env "$REMOTE:/opt/miniblog/.env"
```

---

### ❌ 问题 2: docker-compose.yml 硬编码数据库主机名

**问题描述:**
`docker-compose.yml` 中直接硬编码了 `MYSQL_HOST=mysql` 和 `REDIS_HOST=redis`:

```yaml
environment:
  - MYSQL_HOST=mysql        # ❌ 硬编码,无法覆盖
  - REDIS_HOST=redis        # ❌ 硬编码,无法覆盖
```

**影响:**

- 即使 `.env` 文件设置了 `MYSQL_HOST=rm-xxx.mysql.rds.aliyuncs.com`,也会被覆盖
- 生产环境无法连接到 Aliyun RDS,只能连接名为 `mysql` 的容器
- 同样的问题影响 Redis 连接

**修复方案:**
使用 `${VARIABLE:-default}` 语法,允许环境变量覆盖:

```yaml
environment:
  - MYSQL_HOST=${MYSQL_HOST:-mysql}      # ✅ 优先使用 .env,默认 mysql
  - MYSQL_PORT=${MYSQL_PORT:-3306}
  - REDIS_HOST=${REDIS_HOST:-redis}      # ✅ 优先使用 .env,默认 redis
  - REDIS_PORT=${REDIS_PORT:-6379}
```

**设计理念:**

- 本地开发: 不提供 `.env`,使用默认值 `mysql`/`redis` (本地容器)
- 生产部署: 提供 `.env`,使用外部 RDS/Redis 地址

---

### ❌ 问题 3: 外部网络未在 CI/CD 中创建

**问题描述:**
`docker-compose.yml` 声明使用外部网络:

```yaml
networks:
  miniblog_net:
    external: true    # 假设网络已存在
```

但 CI/CD 脚本中没有创建该网络的步骤,导致首次部署失败:

```
Error: network miniblog_net declared as external, but could not be found
```

**修复方案:**
在部署脚本中添加网络创建步骤:

```bash
# 创建必要的网络 (如果已存在则跳过)
docker network inspect miniblog_net >/dev/null 2>&1 || docker network create miniblog_net
```

**说明:**

- `docker network inspect` 检查网络是否存在
- `||` 逻辑或: 如果检查失败(网络不存在),则执行创建
- 这样可以保证幂等性,重复执行不会出错

---

### ❌ 问题 4: 缺少日志目录创建步骤

**问题描述:**
容器挂载了宿主机日志目录:

```yaml
volumes:
  - /data/logs/miniblog/backend:/data/logs/miniblog
  - /data/logs/miniblog/frontend-blog:/var/log/nginx
  - /data/logs/miniblog/frontend-admin:/var/log/nginx
```

但 CI/CD 脚本未创建这些目录,首次部署时可能失败或权限错误。

**修复方案:**
在部署脚本中添加:

```bash
# 创建必要的目录
mkdir -p /data/logs/miniblog/backend \
         /data/logs/miniblog/frontend-blog \
         /data/logs/miniblog/frontend-admin
```

**最佳实践:**

- 始终在启动容器前确保挂载点存在
- 使用 `mkdir -p` 确保父目录也会创建
- 可选: 设置正确的权限 `chown -R 1000:1000 /data/logs/miniblog`

---

### ❌ 问题 5: 缺少健康检查和部署验证

**问题描述:**
原始脚本部署完成后直接结束,没有:

- 等待服务启动的时间
- 检查服务是否正常运行
- 验证 API 是否可访问

导致即使部署失败,CI/CD 也会显示成功。

**修复方案:**
添加部署验证步骤:

```bash
# 等待服务启动
echo "等待服务启动..."
sleep 10

# 检查后端健康
if curl -fsS http://localhost:8090/health >/dev/null 2>&1; then
  echo "✅ 后端服务健康"
else
  echo "⚠️  后端服务未就绪"
fi

# 查看容器状态
docker compose ps
```

**说明:**

- `sleep 10`: 给服务足够的启动时间
- `curl -fsS`: 静默模式检查健康端点
- `docker compose ps`: 显示所有容器状态,便于排查问题
- 注意: 当前实现不会因健康检查失败而中断部署,可根据需要调整

---

## 其他发现

### ⚠️ 安全隐患: 配置文件中的明文密码

**位置:** `configs/env/env.prod`

```bash
MYSQL_PASSWORD=2gy0dCwG
REDIS_PASSWORD=68OTeDXq
JWT_SECRET=7Bb0dCa
FEISHU_DOCREADER_APPSECRET=wEAC5...
```

**风险:**

- 敏感信息以明文形式存储在 Git 仓库中
- 所有有仓库访问权限的人都能看到生产环境密码
- Git 历史中永久保留这些密码

**建议方案:**

1. **推荐:** 完全移除 `configs/env/env.prod`,改用 GitHub Secrets

   ```bash
   # 不再需要 configs/env/env.prod
   git rm configs/env/env.prod
   ```

   所有环境变量都通过 CI/CD 的 `env:` 注入,已在本次修复中实现。

2. **备选:** 如需保留配置文件模板,创建 `.env.example`:

   ```bash
   # configs/env/env.prod.example
   MYSQL_PASSWORD=<your-password-here>
   REDIS_PASSWORD=<your-password-here>
   ```

3. **加固:** 添加 `.gitignore` 规则:

   ```gitignore
   configs/env/env.prod
   configs/env/env.*.local
   .env
   .env.local
   ```

---

### ✅ SSL 证书假设

**当前配置:**

```yaml
volumes:
  - /data/ssl/certs/api.yangshujie.com.crt:/etc/miniblog/ssl/api.yangshujie.com.crt:ro
  - /data/ssl/private/api.yangshujie.com.key:/etc/miniblog/ssl/api.yangshujie.com.key:ro
```

**建议验证:**
在远程服务器上确认证书文件存在:

```bash
# 在远程服务器执行
ssh deploy@remote-server
ls -la /data/ssl/certs/api.yangshujie.com.crt
ls -la /data/ssl/private/api.yangshujie.com.key
```

如果证书不存在,考虑:

1. 使用 Let's Encrypt 自动获取证书
2. 在 CI/CD 中添加证书上传步骤
3. 或使证书挂载可选 (对于仅 HTTP 部署)

---

## 修复验证清单

在推送代码后,请验证以下内容:

- [ ] CI/CD workflow 成功完成
- [ ] 远程服务器上 `/opt/miniblog/.env` 文件包含正确的环境变量
- [ ] `docker network ls` 显示 `miniblog_net` 存在
- [ ] `docker compose ps` 显示所有容器状态为 `Up`
- [ ] `curl http://remote-server:8090/health` 返回正常响应
- [ ] 后端可以成功连接到 Aliyun RDS MySQL
- [ ] 后端可以成功连接到外部 Redis
- [ ] 检查 `/data/logs/miniblog/backend/` 是否有日志输出

---

## 修复摘要

| 问题 | 严重性 | 状态 | 影响范围 |
|------|--------|------|----------|
| SSH heredoc 变量不展开 | 🔴 Critical | ✅ 已修复 | 所有 secrets 无法传递 |
| 硬编码数据库主机名 | 🔴 Critical | ✅ 已修复 | 无法连接外部数据库 |
| 外部网络未创建 | 🟠 High | ✅ 已修复 | 首次部署失败 |
| 缺少日志目录创建 | 🟡 Medium | ✅ 已修复 | 潜在权限问题 |
| 缺少健康检查 | 🟡 Medium | ✅ 已修复 | 无法验证部署成功 |
| 明文密码存储 | 🔴 Critical | ⚠️  待处理 | 安全风险 |
| SSL 证书假设 | 🟢 Low | ⚠️  需验证 | 可能影响 HTTPS |

---

## 技术债务

以下项目可在后续迭代中优化:

1. **数据库迁移自动化**: 当前未在 CI/CD 中执行 `db-migrate.sh`,需手动迁移
2. **零停机部署**: 考虑使用蓝绿部署或滚动更新策略
3. **监控集成**: 添加部署后的 Prometheus/Grafana 指标检查
4. **回滚机制**: 如果健康检查失败,自动回滚到上一个版本
5. **环境隔离**: 考虑分离 dev/staging/prod 环境的 compose 配置

---

## 参考文档

- [Docker Compose 环境变量优先级](https://docs.docker.com/compose/environment-variables/envvars-precedence/)
- [GitHub Actions 环境变量和 secrets](https://docs.github.com/en/actions/learn-github-actions/variables)
- [Docker 网络管理最佳实践](https://docs.docker.com/network/)
