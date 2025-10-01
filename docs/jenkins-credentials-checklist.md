# Jenkins 凭据配置检查清单

## 🔐 必需凭据

### 1. `miniblog-dev-env` (Secret File)

**类型：** Secret file  
**用途：** 应用环境变量配置  

**内容示例：**

```bash
# APP
APP_MODE=production

# MYSQL - 应用连接 MySQL 的凭据
MYSQL_HOST=mysql
MYSQL_PORT=3306
MYSQL_USERNAME=miniblog
MYSQL_PASSWORD=2gy0dCwG
MYSQL_DBNAME=miniblog

# REDIS - 应用连接 Redis 的凭据
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=68OTeDXq
REDIS_DB=0

# JWT - 应用 JWT 密钥
JWT_SECRET=Rtg8BPKNEf2mB4mgvKONGPZZQSaJWNLijxR42qRgq0iBb5

# FeiShu - 飞书应用凭据
FEISHU_DOCREADER_APPID=cli_a8a6833e6859501c
FEISHU_DOCREADER_APPSECRET=A87ckTk0iNJRSta5zD1XNgqdnbpSoKNv
```

**✅ 检查步骤：**

1. 进入 Jenkins → Credentials → System → Global credentials
2. 查找 ID 为 `miniblog-dev-env` 的凭据
3. 确认类型为 "Secret file"
4. 确认文件内容包含上述所有变量

---

### 2. `mysql-root-password` (Secret Text)

**类型：** Secret text  
**用途：** MySQL root 用户密码，用于 DB Init 阶段创建数据库和用户  

**值：** `dE7ke5Eq2THc`

**说明：** 这是服务器上 MySQL 容器的 root 密码（从 `docker exec mysql env` 获取的 `MYSQL_ROOT_PASSWORD`）

**✅ 检查步骤：**

1. 进入 Jenkins → Credentials → System → Global credentials
2. 查找 ID 为 `mysql-root-password` 的凭据
3. 确认类型为 "Secret text"
4. 确认 Secret 值为 `dE7ke5Eq2THc`

---

## 🔍 配置路径

### 方式 1：通过 Web UI 配置

1. 访问 Jenkins 主页
2. 点击左侧菜单 **"Manage Jenkins"**
3. 点击 **"Credentials"**
4. 点击 **"System"** → **"Global credentials (unrestricted)"**
5. 点击右上角 **"Add Credentials"**

### 方式 2：通过 Jenkins API 配置

```bash
# 添加 Secret text 凭据
curl -X POST 'http://jenkins-url/credentials/store/system/domain/_/createCredentials' \
  --user 'admin:password' \
  --data-urlencode 'json={
    "": "0",
    "credentials": {
      "scope": "GLOBAL",
      "id": "mysql-root-password",
      "secret": "dE7ke5Eq2THc",
      "description": "MySQL root password for DB initialization",
      "$class": "org.jenkinsci.plugins.plaincredentials.impl.StringCredentialsImpl"
    }
  }'
```

---

## 📊 密码对应关系

| 用途 | 凭据 ID | 变量名 | 值 | 说明 |
|------|---------|--------|-----|------|
| MySQL Root | `mysql-root-password` | `DB_ROOT_PASSWORD` | `dE7ke5Eq2THc` | DB Init 创建数据库/用户 |
| MySQL 应用用户 | `miniblog-dev-env` | `MYSQL_USERNAME` | `miniblog` | 应用连接数据库的用户名 |
| MySQL 应用密码 | `miniblog-dev-env` | `MYSQL_PASSWORD` | `2gy0dCwG` | 应用连接数据库的密码 |
| Redis 密码 | `miniblog-dev-env` | `REDIS_PASSWORD` | `68OTeDXq` | 应用连接 Redis 的密码 |

---

## ⚙️ Jenkinsfile 使用方式

### DB Init 阶段（创建数据库和用户）

```groovy
stage('DB Init') {
  when {
    expression { env.RUN_DB_INIT == 'true' }
  }
  steps {
    withCredentials([
      string(credentialsId: params.DB_ROOT_CREDENTIALS_ID, variable: 'DB_ROOT_PASSWORD')
    ]) {
      dir('.') {
        sh 'scripts/db-init.sh'
      }
    }
  }
}
```

**使用的凭据：**

- `mysql-root-password` → `DB_ROOT_PASSWORD` → 执行 `CREATE DATABASE`、`CREATE USER`

**从 miniblog-dev-env 读取：**

- `MYSQL_USERNAME` → 创建的用户名
- `MYSQL_PASSWORD` → 创建的用户密码
- `MYSQL_DBNAME` → 创建的数据库名

### DB Migrate 阶段（执行迁移）

```groovy
stage('DB Migrate') {
  steps {
    dir('.') {
      sh 'scripts/db-migrate.sh'
    }
  }
}
```

**使用的凭据（从 miniblog-dev-env 读取）：**

- `MYSQL_HOST=mysql`
- `MYSQL_PORT=3306`
- `MYSQL_USERNAME=miniblog`
- `MYSQL_PASSWORD=2gy0dCwG`
- `MYSQL_DBNAME=miniblog`

---

## 🧪 验证凭据配置

### 验证 miniblog-dev-env

在 Jenkins Pipeline 中添加调试输出（Setup 阶段已有）：

```groovy
echo "Loaded environment file from credentials 'miniblog-dev-env' (keys: ...)"
```

**预期输出：**

```
Loaded environment file from credentials 'miniblog-dev-env' 
(keys: APP_MODE, MYSQL_HOST, MYSQL_PORT, MYSQL_USERNAME, MYSQL_PASSWORD, 
       MYSQL_DBNAME, REDIS_HOST, REDIS_PORT, REDIS_PASSWORD, REDIS_DB, 
       JWT_SECRET, FEISHU_DOCREADER_APPID, FEISHU_DOCREADER_APPSECRET)
```

### 验证 mysql-root-password

DB Init 阶段会使用这个凭据。如果配置错误，会看到：

```
error: Access denied for user 'root'@'172.22.0.x' (using password: YES)
```

或

```
error: Access denied for user 'root'@'172.22.0.x' (using password: NO)
```

---

## ❓ 常见问题

### Q1: 为什么需要两个不同的密码？

**A:**

- **Root 密码** (`dE7ke5Eq2THc`)：管理员权限，仅用于创建数据库/用户，不应该暴露给应用
- **应用密码** (`2gy0dCwG`)：应用专用，只有 `miniblog` 数据库的权限，遵循最小权限原则

### Q2: 如果 MySQL root 密码不对怎么办？

**A:** 从服务器上获取正确的密码：

```bash
ssh root@47.94.204.124
docker exec mysql env | grep MYSQL_ROOT_PASSWORD
# 输出: MYSQL_ROOT_PASSWORD=dE7ke5Eq2THc
```

### Q3: 如何更新已有的凭据？

**A:**

1. Jenkins → Credentials → 找到对应凭据
2. 点击凭据 ID → 左侧菜单 "Update"
3. 修改 Secret 值
4. 点击 "Save"

### Q4: 应用密码和 Makefile 默认密码不一致会怎样？

**A:** 不会有问题。Makefile 的默认值 `miniblog123` 是兜底值，实际会优先使用：

1. 环境变量 `MYSQL_PASSWORD`（从 Jenkins 凭据加载）
2. 如果没有环境变量，才使用默认值

当前配置会使用 `2gy0dCwG`（正确）✅

---

## 📝 配置完成后

确认以下凭据都已配置：

- [ ] `miniblog-dev-env` (Secret File) - 包含应用环境变量
- [ ] `mysql-root-password` (Secret Text) - MySQL root 密码 `dE7ke5Eq2THc`

然后触发 Jenkins 构建，应该会看到：

```
✅ Run db init: true
✅ Running DB initialization...
✅ CREATE DATABASE IF NOT EXISTS miniblog
✅ CREATE USER IF NOT EXISTS 'miniblog'@'%' IDENTIFIED BY '2gy0dCwG'
✅ GRANT ALL PRIVILEGES ON miniblog.* TO 'miniblog'@'%'
✅ Database initialized successfully
```

---

**更新时间：** 2025-10-01  
**维护者：** DevOps Team
