# 网络架构设计文档

## 概述

本文档描述了服务器基础设施的网络架构设计，采用分层隔离的方式确保安全性和可维护性。

## 网络层次

### 1. infra-frontend (172.19.0.0/16) - DMZ 边界网络

**用途**: 对外暴露的边界网络，接收公网流量

**特点**:

- `enable_icc: false` - 容器间通信禁用（安全隔离）
- `enable_ip_masquerade: true` - 启用 NAT
- 标签: `network.zone=public`, `network.access=external`

**服务**:

- nginx (172.19.0.2) - 反向代理和负载均衡

**职责**:

- SSL/TLS 终止
- 接收外部 HTTP/HTTPS 请求
- 将流量转发到后端网络

---

### 2. infra-backend (172.20.0.0/16) - 基础设施网络

**用途**: 共享的基础设施服务网络

**特点**:

- `enable_icc: true` - 容器间通信开启
- 标签: `network.zone=private`, `network.access=internal`

**服务**:

- mysql (172.20.0.2) - 数据库服务
- redis (172.20.0.3) - 缓存服务
- mongo (172.20.0.4) - 文档数据库
- nginx (172.20.0.6) - 内部代理（同时在 frontend 网络）
- jenkins (172.20.0.5) - CI/CD（同时在 jenkins_net）

**职责**:

- 提供共享的数据存储服务
- 所有应用可访问的基础设施
- 内部服务发现

---

### 3. miniblog_net (172.21.0.0/16) - MiniBlog 应用网络

**用途**: MiniBlog 应用专用隔离网络

**服务**:

- miniblog-backend (172.21.0.5)
- miniblog-frontend-blog (172.21.0.4)
- miniblog-frontend-admin (172.21.0.3)
- mysql (172.21.0.2) - 桥接自 infra-backend
- redis (172.21.0.6) - 桥接自 infra-backend
- nginx (172.21.0.7) - 桥接自 infra-backend

**访问路径**:

```
Internet → nginx(frontend) → nginx(backend+miniblog_net) → miniblog-backend
```

**配置示例**:

```yaml
networks:
  miniblog_net:
    name: infra_shared  # 当前名称，可改为 miniblog_net
    driver: bridge
```

---

### 4. qs_net (172.22.0.0/16) - QS 应用网络

**用途**: QS (Question System) 应用专用隔离网络

**服务**:

- qs-apiserver (172.22.0.2)
- qs-collectionserver (172.22.0.3)
- mysql (172.22.0.4) - 桥接自 infra-backend
- redis (172.22.0.5) - 桥接自 infra-backend
- mongo (172.22.0.6) - 桥接自 infra-backend
- nginx (172.22.0.7) - 桥接自 infra-backend

**访问路径**:

```
Internet → nginx(frontend) → nginx(backend+qs_net) → qs-apiserver/qs-collectionserver
```

---

### 5. jenkins_net (172.23.0.0/16) - CI/CD 网络

**用途**: Jenkins 及构建代理的隔离网络

**服务**:

- jenkins (172.23.0.2) - 同时在 infra-backend
- jenkins-agent-* (如需要) - 构建代理

**特点**:

- 完全隔离，仅 Jenkins 相关服务
- Jenkins 通过 infra-backend 访问基础设施
- 构建过程中创建的临时容器在此网络

---

## 服务网络映射表

| 服务名称                    | infra-frontend | infra-backend | miniblog_net | qs_net | jenkins_net |
|----------------------------|----------------|---------------|--------------|--------|-------------|
| nginx                      | ✅ (主)        | ✅            | ✅           | ✅     | ❌          |
| mysql                      | ❌             | ✅ (主)       | ✅           | ✅     | ❌          |
| redis                      | ❌             | ✅ (主)       | ✅           | ✅     | ❌          |
| mongo                      | ❌             | ✅ (主)       | ❌           | ✅     | ❌          |
| jenkins                    | ❌             | ✅            | ❌           | ❌     | ✅ (主)     |
| miniblog-backend           | ❌             | ❌            | ✅           | ❌     | ❌          |
| miniblog-frontend-blog     | ❌             | ❌            | ✅           | ❌     | ❌          |
| miniblog-frontend-admin    | ❌             | ❌            | ✅           | ❌     | ❌          |
| qs-apiserver               | ❌             | ❌            | ❌           | ✅     | ❌          |
| qs-collectionserver        | ❌             | ❌            | ❌           | ✅     | ❌          |

## 网络创建命令

```bash
# 1. 边界网络（已存在）
docker network create \
  --driver bridge \
  --subnet 172.19.0.0/16 \
  --gateway 172.19.0.1 \
  --opt com.docker.network.bridge.enable_icc=false \
  --opt com.docker.network.bridge.enable_ip_masquerade=true \
  --opt com.docker.network.bridge.name=infra-frontend \
  --label network.zone=public \
  --label network.access=external \
  infra-frontend

# 2. 基础设施网络（已存在）
docker network create \
  --driver bridge \
  --subnet 172.20.0.0/16 \
  --gateway 172.20.0.1 \
  --opt com.docker.network.bridge.enable_icc=true \
  --opt com.docker.network.bridge.name=infra-backend \
  --label network.zone=private \
  --label network.access=internal \
  infra-backend

# 3. MiniBlog 应用网络（重命名 infra_shared）
docker network create \
  --driver bridge \
  --subnet 172.21.0.0/16 \
  --gateway 172.21.0.1 \
  --label app.name=miniblog \
  --label network.type=application \
  miniblog_net

# 4. QS 应用网络（新建）
docker network create \
  --driver bridge \
  --subnet 172.22.0.0/16 \
  --gateway 172.22.0.1 \
  --label app.name=qs \
  --label network.type=application \
  qs_net

# 5. Jenkins CI/CD 网络（新建）
docker network create \
  --driver bridge \
  --subnet 172.23.0.0/16 \
  --gateway 172.23.0.1 \
  --label service.name=jenkins \
  --label network.type=cicd \
  jenkins_net
```

## 桥接配置

### 将基础设施服务桥接到应用网络

```bash
# MiniBlog 依赖
docker network connect miniblog_net mysql
docker network connect miniblog_net redis
docker network connect miniblog_net nginx

# QS 依赖
docker network connect qs_net mysql
docker network connect qs_net redis
docker network connect qs_net mongo
docker network connect qs_net nginx

# Jenkins 依赖（已在 infra-backend，无需额外桥接）
docker network connect jenkins_net jenkins
```

## 安全考虑

### 1. 网络隔离原则

- **应用间隔离**: miniblog_net 和 qs_net 完全隔离，互不可见
- **数据访问控制**: 仅通过桥接方式允许应用访问基础设施
- **DMZ 隔离**: infra-frontend 禁用容器间通信，仅作流量入口

### 2. 端口暴露策略

**对外暴露（宿主机端口）**:

- nginx: 80, 443 (必需，公网访问)
- jenkins: 8080, 50000 (可选，管理用)

**内部端口（仅容器间）**:

- mysql: 3306 (不对外暴露)
- redis: 6379 (不对外暴露)
- mongo: 27017 (不对外暴露)
- miniblog-backend: 8080 (不对外暴露)
- qs-*: 8080+ (不对外暴露)

### 3. 防火墙规则建议

```bash
# 宿主机防火墙（示例）
# 仅允许 80, 443, 22(SSH), 8080(Jenkins 管理)
iptables -A INPUT -p tcp --dport 80 -j ACCEPT
iptables -A INPUT -p tcp --dport 443 -j ACCEPT
iptables -A INPUT -p tcp --dport 22 -j ACCEPT
iptables -A INPUT -p tcp --dport 8080 -s <管理IP> -j ACCEPT
iptables -A INPUT -j DROP
```

## Nginx 配置示例

### /etc/nginx/conf.d/miniblog.conf

```nginx
upstream miniblog_backend {
    # 通过 Docker 网络直接访问，无需宿主机端口
    server miniblog-backend:8080;
}

server {
    listen 80;
    server_name blog.yangshujie.com admin.blog.yangshujie.com;
    
    # SSL 配置省略...
    
    location /api/ {
        proxy_pass http://miniblog_backend;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
    
    location / {
        # 前端静态文件由 nginx 直接服务
        # 或代理到前端容器
        proxy_pass http://miniblog-frontend-blog:80;
    }
}
```

### /etc/nginx/conf.d/qs.conf

```nginx
upstream qs_api {
    server qs-apiserver:8080;
}

upstream qs_collection {
    server qs-collectionserver:8081;
}

server {
    listen 80;
    server_name qs.yangshujie.com;
    
    location /api/ {
        proxy_pass http://qs_api;
    }
    
    location /collect/ {
        proxy_pass http://qs_collection;
    }
}
```

## 迁移步骤

### 从当前 infra_shared 迁移到新架构

```bash
# 1. 创建新网络
docker network create miniblog_net --subnet 172.21.0.0/16
docker network create qs_net --subnet 172.22.0.0/16
docker network create jenkins_net --subnet 172.23.0.0/16

# 2. 桥接基础设施服务
docker network connect miniblog_net mysql
docker network connect miniblog_net redis
docker network connect miniblog_net nginx

docker network connect qs_net mysql
docker network connect qs_net redis
docker network connect qs_net mongo
docker network connect qs_net nginx

docker network connect jenkins_net jenkins

# 3. 更新 docker-compose.yml
# 将 networks.infra_shared 改为 networks.miniblog_net

# 4. 重新部署应用
docker-compose down
docker-compose up -d

# 5. 验证网络连通性
docker exec miniblog-backend ping -c 3 mysql
docker exec qs-apiserver ping -c 3 mongo

# 6. 清理旧网络（确认无容器使用后）
docker network rm infra_shared
```

## 故障排查

### 检查网络连通性

```bash
# 查看容器所在网络
docker inspect <container_name> | jq '.[0].NetworkSettings.Networks'

# 测试跨网络连通性
docker exec <container> ping -c 3 <target_host>

# 查看网络详情
docker network inspect <network_name>
```

### 常见问题

**Q: 容器无法解析其他容器名称？**
A: 确保容器在同一网络中，Docker 的嵌入式 DNS 仅对同一网络有效。

**Q: Jenkins 构建时无法访问 MySQL？**
A: 确保 jenkins 同时在 infra-backend 网络中，或将 MySQL 桥接到 jenkins_net。

**Q: Nginx 代理后端报 502 错误？**
A: 检查 nginx 是否加入了应用网络，确保能解析后端容器名。

## 监控和维护

### 网络流量监控

```bash
# 查看网络流量统计
docker stats --format "table {{.Container}}\t{{.NetIO}}"

# 监控特定网络的容器
docker ps --filter "network=miniblog_net" --format "table {{.Names}}\t{{.Status}}"
```

### 定期检查

- 每周检查网络配置是否符合设计
- 监控跨网络流量，识别异常访问
- 审计容器网络成员关系

## 总结

此网络架构提供：

✅ **安全隔离**: 应用间完全隔离，降低横向移动风险  
✅ **灵活扩展**: 新应用可独立创建网络，不影响现有服务  
✅ **清晰边界**: 三层网络架构（public/private/app）职责明确  
✅ **便于管理**: 网络命名和标签便于识别和维护  
✅ **高性能**: 容器间直接通信，避免宿主机网络栈开销  

---

**最后更新**: 2025-10-01  
**维护者**: DevOps Team
