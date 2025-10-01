# Jenkins 数据库初始化指南

## 📋 前置条件检查

在触发带 DB Init 的构建之前，请确认以下配置：

### 1. Jenkins 凭据配置

需要在 Jenkins 中配置以下凭据：

#### 凭据 1：`miniblog-dev-env` (Secret File)

包含应用环境变量的 `.env` 文件，至少需要：

```bash
MYSQL_HOST=mysql
MYSQL_PORT=3306
MYSQL_USERNAME=miniblog
MYSQL_PASSWORD=miniblog123
MYSQL_DBNAME=miniblog
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=1
JWT_SECRET=your-jwt-secret-here
```

**📍 配置路径：** Jenkins → Credentials → System → Global credentials → Add Credentials

- **Kind:** Secret file
- **File:** 上传包含上述内容的文件
- **ID:** `miniblog-dev-env`

#### 凭据 2：`mysql-root-password` (Secret Text)

MySQL root 用户密码，用于创建数据库和用户。

根据您的服务器配置，MySQL root 密码是：`dE7ke5Eq2THc`

**📍 配置路径：** Jenkins → Credentials → System → Global credentials → Add Credentials

- **Kind:** Secret text
- **Secret:** `dE7ke5Eq2THc`
- **ID:** `mysql-root-password`

---

## 🚀 触发带 DB Init 的构建

### 方法 1：通过 Jenkins UI 手动触发（推荐）

1. **进入 Jenkins 项目页面**
   - 访问：<http://your-jenkins-url/job/miniblog/>

2. **点击 "Build with Parameters"**（左侧菜单）

3. **配置构建参数**

   关键参数设置：

   ```
   ✅ SKIP_DB_INIT = false          # 启用数据库初始化
   ✅ SKIP_DB_MIGRATE = false       # 保持数据库迁移启用
   ✅ DEPLOY_AFTER_BUILD = true     # 构建后自动部署
   
   其他参数保持默认：
   IMAGE_REGISTRY = miniblog
   IMAGE_TAG = prod
   ENV_CREDENTIALS_ID = miniblog-dev-env
   DB_ROOT_CREDENTIALS_ID = mysql-root-password
   ```

4. **点击 "Build" 按钮**

---

### 方法 2：修改 Jenkinsfile 默认值

如果希望每次构建都执行 DB Init（不推荐，仅用于初始化阶段）：

```groovy
// 在 Jenkinsfile 的 parameters 部分修改
booleanParam(
  name: 'SKIP_DB_INIT', 
  defaultValue: false,  // 改为 false
  description: 'Skip the database initialisation stage'
)
```

---

## 🔍 验证构建日志

构建过程中，关注以下关键阶段的日志：

### 1. Setup 阶段

确认环境变量加载成功：

```
Loaded environment file from credentials 'miniblog-dev-env' 
(keys: MYSQL_HOST, MYSQL_PORT, MYSQL_USERNAME, ...)
```

### 2. Prepare Network 阶段

确认网络正确：

```
+ NETWORK=miniblog_net make docker-network-ensure
Network miniblog_net already exists
```

### 3. Deploy 阶段

所有容器成功启动：

```
✅ Container miniblog-backend Started
✅ Container miniblog-frontend-blog Started
✅ Container miniblog-frontend-admin Started
```

### 4. DB Init 阶段（首次运行）

数据库和用户创建成功：

```
Running DB initialization...
-> Local migrate binary not found, using dockerized mysql client
CREATE DATABASE IF NOT EXISTS miniblog
CREATE USER IF NOT EXISTS 'miniblog'@'%'
GRANT ALL PRIVILEGES ON miniblog.* TO 'miniblog'@'%'
✅ Database initialized successfully
```

### 5. DB Migrate 阶段

数据库迁移成功：

```
[db-migrate] Resolved DB_HOST=mysql, DB_PORT=3306
-> Using dockerized migrate image
1/u create_users_table (123.456ms)
2/u create_articles_table (234.567ms)
✅ Migrations completed
```

---

## ❌ 常见问题排查

### 问题 1：`Access denied for user 'root'`

**原因：** MySQL root 密码不正确或凭据未配置

**解决：**

1. 检查 Jenkins 凭据 `mysql-root-password` 是否配置
2. 确认密码是否正确：`dE7ke5Eq2THc`

### 问题 2：`dial tcp: lookup mysql: no such host`

**原因：** 网络配置不正确或 MySQL 未加入 miniblog_net

**解决：**

```bash
# 在服务器上执行
docker network connect miniblog_net mysql
docker network inspect miniblog_net | grep -A 5 mysql
```

### 问题 3：`Network miniblog_net not found`

**原因：** miniblog_net 网络不存在

**解决：**

```bash
# 在服务器上执行
docker network create miniblog_net
```

### 问题 4：`Permission denied` 创建数据库目录

**原因：** MySQL 容器文件系统权限问题

**解决：** 使用 dockerized mysql client（Makefile 已自动处理）

---

## ✅ 成功标志

构建成功后，可以验证：

### 1. 检查数据库是否创建

```bash
ssh root@47.94.204.124
docker exec mysql mysql -uroot -p'dE7ke5Eq2THc' -e "SHOW DATABASES;"
# 应该看到 miniblog 数据库
```

### 2. 检查用户权限

```bash
docker exec mysql mysql -uroot -p'dE7ke5Eq2THc' -e "SELECT user, host FROM mysql.user WHERE user='miniblog';"
# 应该看到 miniblog | %
```

### 3. 检查表结构

```bash
docker exec mysql mysql -uminiblog -pminiblog123 miniblog -e "SHOW TABLES;"
# 应该看到迁移创建的表
```

### 4. 检查应用是否正常运行

```bash
docker ps --filter 'name=miniblog'
# 所有容器状态应该是 Up

docker logs miniblog-backend --tail 20
# 应该看到服务启动成功的日志
```

---

## 📝 后续操作

首次 DB Init 成功后：

1. **将 `SKIP_DB_INIT` 改回 `true`**
   - 避免每次构建都重新初始化数据库
   - 数据库和用户已经创建，无需重复执行

2. **配置 Nginx 反向代理**
   - 配置域名指向 `miniblog-backend:8080`
   - 配置静态资源指向前端容器

3. **配置 HTTPS 证书**
   - 上传 SSL 证书到服务器
   - 更新 backend 配置使用证书

4. **监控和日志**
   - 确保日志目录 `/data/logs/miniblog/` 存在
   - 配置日志轮转

---

## 🔗 相关文档

- [网络架构设计](./infrastructure/network-architecture.md)
- [Docker Compose 配置](../docker-compose.yml)
- [Makefile 命令参考](../Makefile)
- [数据库迁移脚本](../db/migrations/)

---

**更新时间：** 2025-10-01  
**维护者：** DevOps Team
