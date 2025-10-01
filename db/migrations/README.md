# 数据库迁移和种子数据管理

本文档说明如何管理数据库 schema 迁移和初始数据加载。

## 📁 目录结构

```
db/migrations/sql/
├── 000001_init.up.sql           # Schema 迁移：创建表结构
├── 000001_init.down.sql         # Schema 回滚
├── 000002_seed_data.up.sql      # 种子数据迁移（可选）
├── 000002_seed_data.down.sql    # 种子数据回滚
├── user.sql                     # 初始用户数据
├── module.sql                   # 模块数据
├── section.sql                  # 章节数据
├── article.sql                  # 文章数据（11,715行）
└── casbin_rule.sql              # 权限规则数据
```

## 🔄 数据库初始化流程

### 1. DB Init（数据库初始化）
创建数据库和用户：
```bash
make db-init
```

对应 Jenkins 参数：`SKIP_DB_INIT`（默认 false，即执行）

### 2. DB Migrate（Schema 迁移）
执行所有迁移文件，创建表结构：
```bash
make db-migrate
```

对应 Jenkins 参数：`SKIP_DB_MIGRATE`（默认 false，即执行）

### 3. DB Seed（种子数据加载）
加载初始数据（用户、模块、文章等）：
```bash
make db-seed
```

对应 Jenkins 参数：`SKIP_DB_SEED`（默认 **true**，即跳过）

## 📝 如何添加新的迁移

### 方法 1：使用 golang-migrate 命名规范

如果是 **schema 变更**（创建/修改表结构），应该创建新的迁移版本：

```bash
# 创建新迁移文件
touch db/migrations/sql/000003_add_comment_table.up.sql
touch db/migrations/sql/000003_add_comment_table.down.sql
```

**000003_add_comment_table.up.sql**:
```sql
USE `miniblog`;

CREATE TABLE IF NOT EXISTS `comment` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `article_id` BIGINT NOT NULL,
  `content` TEXT NOT NULL,
  `author` VARCHAR(255) NOT NULL,
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_article_id` (`article_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
```

**000003_add_comment_table.down.sql**:
```sql
USE `miniblog`;

DROP TABLE IF EXISTS `comment`;
```

### 方法 2：添加种子数据文件

如果是 **初始数据**（INSERT 语句），添加到独立的 SQL 文件：

```bash
# 在 db/migrations/sql/ 目录下创建新文件
touch db/migrations/sql/comment_seed.sql
```

然后在 `scripts/load-seed-data.sh` 中添加：
```bash
for sql_file in user.sql module.sql section.sql article.sql casbin_rule.sql comment_seed.sql; do
    # ...
done
```

## 🚀 Jenkins 构建配置

### 首次部署（全新环境）
1. ✅ `SKIP_DB_INIT` = false（创建数据库）
2. ✅ `SKIP_DB_MIGRATE` = false（创建表）
3. ✅ `SKIP_DB_SEED` = **false**（加载初始数据）

### 日常更新（已有环境）
1. ✅ `SKIP_DB_INIT` = true（数据库已存在）
2. ✅ `SKIP_DB_MIGRATE` = false（执行新迁移）
3. ✅ `SKIP_DB_SEED` = true（不重复加载数据）

### 仅部署代码（无数据库变更）
1. ✅ `SKIP_DB_INIT` = true
2. ✅ `SKIP_DB_MIGRATE` = true
3. ✅ `SKIP_DB_SEED` = true

## 🛠️ 本地开发使用

```bash
# 1. 启动 MySQL 容器
docker compose up -d mysql

# 2. 初始化数据库
make db-init

# 3. 执行迁移
make db-migrate

# 4. 加载种子数据（可选）
make db-seed
```

## ⚠️ 注意事项

1. **迁移文件命名规范**：
   - 必须遵循 `{version}_{description}.up.sql` 格式
   - 版本号必须递增（000001, 000002, 000003...）
   - 每个 `.up.sql` 必须有对应的 `.down.sql`

2. **种子数据文件**：
   - 不需要遵循迁移命名规范
   - 建议使用 `ON DUPLICATE KEY UPDATE` 保证幂等性
   - 大数据文件（如 article.sql）应该单独管理

3. **数据加载顺序**：
   - user.sql → module.sql → section.sql → article.sql → casbin_rule.sql
   - 注意外键依赖关系

4. **生产环境建议**：
   - 首次部署后，将 `SKIP_DB_SEED` 设为 true
   - 新数据通过应用程序添加，而不是重复执行 seed 脚本
   - 定期备份数据库

## 📚 相关文档

- [golang-migrate 官方文档](https://github.com/golang-migrate/migrate)
- [MySQL 迁移最佳实践](https://dev.mysql.com/doc/)
