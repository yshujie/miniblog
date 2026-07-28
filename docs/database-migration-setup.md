# 数据库迁移配置总结

## ✅ 已完成的配置

### 1. 创建了种子数据加载脚本

- 📄 `scripts/load-seed-data.sh`
  - 按顺序加载：user.sql → module.sql → section.sql → article.sql → casbin_rule.sql
  - 支持 Docker 和本地 MySQL 客户端
  - 自动检测运行环境

### 2. 更新了 Makefile

- 新增 `db-seed` target：执行种子数据加载
- 位置：在 `db-migrate` 之后

### 3. 提供 GitHub Actions 手动数据库 workflow

- `.github/workflows/db-ops.yml` 可分别执行 DB Init、DB Migrate 和 DB Seed
- 三个 `skip_*` 输入默认都是 `true`，避免误写生产数据库
- workflow 通过 `SVRD_*` 凭据在 ServerD 执行，并读取 `/opt/miniblog/.env`

### 4. 创建了示例迁移文件

- `000002_seed_data.up.sql`：包含部分种子数据
- `000002_seed_data.down.sql`：回滚种子数据

## 📋 使用方法

### 方法 1：通过 Makefile（推荐）

```bash
# 加载所有种子数据
make db-seed
```

### 方法 2：通过 GitHub Actions

进入 **Actions → DB Operations (manual) → Run workflow**：

1. 首次需要种子数据时，将 `skip_db_seed` 设为 `false`
2. 日常保持 `skip_db_seed=true`，避免重复加载

### 方法 3：直接执行脚本

```bash
bash scripts/load-seed-data.sh
```

## 🔄 完整的数据库初始化流程

```bash
# 1. 创建数据库和用户
make db-init

# 2. 执行 schema 迁移（创建表）
make db-migrate

# 3. 加载种子数据（可选）
make db-seed
```

## 📝 如何添加新的 SQL 文件

### 如果是 Schema 变更（CREATE/ALTER TABLE）

创建新的迁移文件：

```bash
# 文件名格式：{版本号}_{描述}.up.sql
db/migrations/sql/000003_add_new_feature.up.sql
db/migrations/sql/000003_add_new_feature.down.sql
```

然后运行 `make db-migrate` 会自动执行。

### 如果是初始数据（INSERT）

1. 在 `db/migrations/sql/` 目录创建 SQL 文件
2. 编辑 `scripts/load-seed-data.sh`，在 `for` 循环中添加新文件名
3. 运行 `make db-seed` 加载数据

## ⚠️ 重要提示

1. **种子数据只在首次部署时加载**：
   - 首次：`skip_db_seed=false`
   - 之后：`skip_db_seed=true`

2. **迁移文件自动执行**：
   - 所有 `{version}_{desc}.up.sql` 文件会按版本号顺序执行
   - migrate 工具会记录已执行的版本，不会重复执行

3. **数据加载顺序很重要**：
   - 注意外键依赖关系
   - user → module → section → article → casbin_rule

## 🎯 下一步

现在你可以：

1. 提交这些更改到 Git
2. 在 GitHub Actions 中手动触发 `DB Operations (manual)`
3. 将 `skip_db_seed=false` 来加载种子数据
4. 后续运行保持 `skip_db_seed=true`
