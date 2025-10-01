# 网络架构快速参考

## 当前网络配置

### 已完成的配置

```bash
# ✅ 已将 MySQL 桥接到 infra_shared
docker network connect infra_shared mysql

# ⏳ 待执行: 桥接 Redis 和 Nginx
docker network connect infra_shared redis
docker network connect infra_shared nginx
```

### 网络映射表（简化版）

| 服务 | 主网络 | 桥接网络 | 说明 |
|-----|--------|---------|------|
| nginx | infra-frontend | infra-backend, infra_shared | 反向代理，需访问所有应用 |
| mysql | infra-backend | infra_shared | 被 MiniBlog 使用 |
| redis | infra-backend | infra_shared | 被 MiniBlog 使用 |
| mongo | infra-backend | - | MiniBlog 不使用 |
| jenkins | infra-backend | jenkins_net (可选) | CI/CD 独立网络 |

## 立即执行的命令

### 方案 A：最小改动（推荐用于快速修复）

```bash
# 1. 桥接必要的服务
docker network connect infra_shared redis
docker network connect infra_shared nginx

# 2. 更新 Jenkins 环境变量
# 在 Jenkins → Credentials → miniblog-dev-env 中修改：
# MYSQL_HOST=mysql
# REDIS_HOST=redis

# 3. 重新触发构建
# Jenkins UI → miniblog job → Build Now
```

### 方案 B：完整架构（推荐用于长期规划）

```bash
# 1. 运行迁移脚本
chmod +x scripts/migrate-network.sh
scripts/migrate-network.sh

# 2. 查看网络拓扑
docker network ls --format '{{.Name}}' | while read net; do 
  echo "=== $net ==="; 
  docker network inspect $net --format '{{range .Containers}}  - {{.Name}} ({{.IPv4Address}}){{println}}{{end}}'; 
done
```

## docker-compose.yml 网络配置

### 当前配置（使用 infra_shared）

```yaml
services:
  miniblog-backend:
    networks:
      - infra_shared
    environment:
      - MYSQL_HOST=mysql     # ← 修改为实际容器名
      - REDIS_HOST=redis     # ← 修改为实际容器名

networks:
  infra_shared:
    external: true
```

### 推荐配置（未来扩展）

```yaml
services:
  miniblog-backend:
    networks:
      - miniblog_net
    environment:
      - MYSQL_HOST=mysql
      - REDIS_HOST=redis

networks:
  miniblog_net:
    name: infra_shared  # 使用现有网络作为别名
    external: true
```

## 验证命令

```bash
# 1. 检查容器网络连接
docker inspect mysql --format='{{range $k, $v := .NetworkSettings.Networks}}{{$k}} {{end}}'

# 2. 测试连通性
docker exec miniblog-backend ping -c 3 mysql
docker exec miniblog-backend ping -c 3 redis

# 3. 查看网络详情
docker network inspect infra_shared

# 4. 检查 DNS 解析
docker exec miniblog-backend nslookup mysql
docker exec miniblog-backend getent hosts redis
```

## 故障排查

### 问题: migrate 容器无法连接 MySQL

```bash
# 检查 MySQL 是否在 infra_shared 网络
docker network inspect infra_shared | jq '.[] | .Containers | .[] | select(.Name == "mysql")'

# 如果没有，手动连接
docker network connect infra_shared mysql
```

### 问题: 后端容器无法解析 redis 主机名

```bash
# 检查 redis 是否在 infra_shared 网络
docker network inspect infra_shared | grep redis

# 连接 redis 到网络
docker network connect infra_shared redis
```

### 问题: Nginx 502 Bad Gateway

```bash
# 检查 nginx 是否能访问后端
docker exec nginx ping -c 3 miniblog-backend

# 如果失败，连接 nginx 到应用网络
docker network connect infra_shared nginx

# 检查 nginx 配置
docker exec nginx nginx -t
docker exec nginx cat /etc/nginx/conf.d/miniblog.conf
```

## 下一步行动清单

- [ ] 在服务器上执行 `docker network connect infra_shared redis`
- [ ] 在服务器上执行 `docker network connect infra_shared nginx`
- [ ] 更新 Jenkins credentials 中的 MYSQL_HOST=mysql, REDIS_HOST=redis
- [ ] 重新触发 Jenkins 构建
- [ ] 验证 DB migration 成功
- [ ] 检查应用是否能正常连接 Redis
- [ ] 更新 Nginx 配置使用容器名代理（如需要）
- [ ] 考虑移除 docker-compose.prod.yml 中的端口暴露（8090）

## 参考文档

- 详细架构设计: `docs/infrastructure/network-architecture.md`
- 迁移脚本: `scripts/migrate-network.sh`
