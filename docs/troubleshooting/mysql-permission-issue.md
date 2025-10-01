# MySQL 数据库创建权限问题排查记录

## 问题描述

在 Jenkins CI/CD 流程中执行 DB Init 时，遇到以下错误：

```
ERROR 3680 (HY000) at line 6: Failed to create schema directory 'miniblog' (errno: 13 - Permission denied)
```

## 问题分析

### 根本原因

MySQL 容器的数据目录权限配置不正确，导致 MySQL 进程无法在数据目录创建新的数据库文件夹。

### 详细调查过程

1. **MySQL 容器配置**

   ```bash
   docker inspect mysql
   ```

   - 数据目录挂载：`/var/lib/docker/volumes/infra_mysql_data/_data` → `/var/lib/mysql`

2. **文件系统权限检查**

   ```bash
   ls -ld /var/lib/docker/volumes/infra_mysql_data/_data
   # 输出：drwxr-xr-x 8 www docker 4096 Oct  1 20:16 /var/lib/docker/volumes/infra_mysql_data/_data
   ```

   - 所有者：`www:docker`
   - 权限：`755` (只有所有者有写权限)

3. **MySQL 进程用户检查**

   ```bash
   docker top mysql
   # 输出：UID 999 运行 mysqld 进程
   ```

   - MySQL 进程以 **UID 999** 运行
   - 但数据目录属于 `www` 用户（不是 UID 999）

4. **权限不匹配**
   - MySQL 进程用户（UID 999）不是目录所有者
   - 目录权限 `755` 意味着非所有者只有读和执行权限，**没有写权限**
   - 因此 MySQL 无法创建新的数据库目录

## 解决方案

### 方案 1：修改数据目录所有权（推荐）

将数据目录的所有权改为 MySQL 进程的 UID：

```bash
chown -R 999:999 /var/lib/docker/volumes/infra_mysql_data/_data
```

**优点：**

- 最安全的方案
- MySQL 完全拥有数据目录
- 符合最小权限原则

**执行后验证：**

```bash
docker exec mysql mysql -uroot -p'xxx' -e "CREATE DATABASE test_db;"
# 应该成功创建
docker exec mysql mysql -uroot -p'xxx' -e "DROP DATABASE test_db;"
```

### 方案 2：修改目录权限（临时方案）

```bash
chmod 775 /var/lib/docker/volumes/infra_mysql_data/_data
```

**缺点：**

- 如果 MySQL 进程用户不在目录的组中，仍然会失败
- 权限过于宽松

### 方案 3：重新创建容器（彻底方案）

如果是新环境，可以重新创建 MySQL 容器，确保正确的用户映射：

```yaml
services:
  mysql:
    image: mysql:8.0
    user: "999:999"  # 明确指定运行用户
    volumes:
      - mysql_data:/var/lib/mysql
```

## 实施记录

**执行时间：** 2025-10-01 20:16

**执行命令：**

```bash
chown -R 999:999 /var/lib/docker/volumes/infra_mysql_data/_data
```

**验证结果：**

```bash
docker exec mysql mysql -uroot -p'dE7ke5Eq2THc' -e "
CREATE DATABASE IF NOT EXISTS miniblog DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;
CREATE USER IF NOT EXISTS 'miniblog'@'%' IDENTIFIED BY '2gy0dCwG';
GRANT ALL PRIVILEGES ON miniblog.* TO 'miniblog'@'%';
FLUSH PRIVILEGES;
SHOW DATABASES;
"
```

**结果：** ✅ 成功创建 miniblog 数据库和用户

## 后续建议

### 1. 文档化容器用户映射

在 `docker-compose.yml` 或部署文档中明确说明：

- MySQL 容器运行用户：UID 999
- 数据卷权限要求：UID 999 需要读写权限

### 2. 自动化权限修复

在部署脚本中添加权限检查和修复：

```bash
#!/bin/bash
MYSQL_DATA_DIR="/var/lib/docker/volumes/infra_mysql_data/_data"

if [ -d "$MYSQL_DATA_DIR" ]; then
  CURRENT_OWNER=$(stat -c '%u' "$MYSQL_DATA_DIR")
  if [ "$CURRENT_OWNER" != "999" ]; then
    echo "Fixing MySQL data directory ownership..."
    chown -R 999:999 "$MYSQL_DATA_DIR"
  fi
fi
```

### 3. 监控和告警

添加健康检查，确保 MySQL 可以正常创建数据库：

```yaml
healthcheck:
  test: ["CMD", "mysqladmin", "ping", "-h", "localhost"]
  interval: 10s
  timeout: 5s
  retries: 5
```

## 相关问题

### Q1: 为什么不直接用 root 用户运行 MySQL？

**A:** 安全考虑。容器内使用非特权用户运行服务是最佳实践：

- 降低容器逃逸风险
- 限制潜在的权限滥用
- 符合最小权限原则

### Q2: 已有数据库的权限会受影响吗？

**A:** 不会。`chown -R` 递归修改所有文件和子目录的所有权，确保 MySQL 对所有数据库文件都有完整权限。

### Q3: 如何避免将来再次出现此问题？

**A:**

1. 使用 Docker Compose 时明确指定 `user` 参数
2. 创建数据卷时预设正确的权限
3. 在 CI/CD 脚本中添加权限验证步骤

## 参考资料

- [MySQL Docker Official Image](https://hub.docker.com/_/mysql)
- [Docker User Namespace Remapping](https://docs.docker.com/engine/security/userns-remap/)
- [MySQL Error 3680](https://dev.mysql.com/doc/mysql-errors/8.0/en/server-error-reference.html#error_er_cannot_create_directory)

---

**文档版本：** 1.0  
**最后更新：** 2025-10-01  
**维护者：** DevOps Team
