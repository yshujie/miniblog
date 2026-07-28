# scripts 使用说明

## 目录定位

`scripts/` 只保留两类脚本：

- 人工运维工具：批量导入文章、重置用户密码。
- Jenkins/Makefile 兼容入口：推送镜像、Compose 部署、数据库初始化、迁移和种子数据加载。

当前生产 CI/CD 的主要编排在 `.github/workflows/cicd.yml`，使用 GitHub 托管 Runner，
不会调用本目录中的 Docker 代理、Buildx 安装或网络迁移脚本。仓库中仍有少量手动
workflow 使用自托管 Runner，它们的诊断步骤保留在 workflow 内。基础设施网络由独立的
infra 配置维护，本目录不负责修改宿主机 Docker 网络拓扑。

## 脚本清单

| 脚本 | 用途 | 主要调用方 | 是否写数据/外部状态 |
| --- | --- | --- | --- |
| `batch-upsert-articles.sh` | 按 JSON 批量新增或更新文章 | 人工执行 | 是，写业务数据库 |
| `reset-user-password.sh` | 按用户 ID 重置密码 | 人工执行 | 是，写业务数据库 |
| `db-init.sh` | 创建数据库、应用用户和基础表 | Jenkins 或人工执行 | 是，需要管理员权限 |
| `db-migrate.sh` | 执行数据库 `up` 迁移 | Jenkins 或人工执行 | 是，修改数据库结构/数据 |
| `load-seed-data.sh` | 按顺序加载种子 SQL | `make db-seed` | 是，写业务数据库 |
| `deploy.sh` | 包装 `make compose-up` | Jenkins 或人工执行 | 是，更新容器 |
| `push-images.sh` | 推送本次构建产生的镜像 | Jenkins | 是，写镜像仓库 |

## 通用准备

所有命令都应在仓库根目录执行。

### 1. 构建与静态预检

Go 运维工具要求 Go 1.24：

```bash
go version
go test ./scripts/...
```

检查 Shell 语法：

```bash
for script in scripts/*.sh; do
  bash -n "$script"
done
```

涉及 Docker 的脚本还应先检查：

```bash
docker version
docker compose version
docker network inspect miniblog-network >/dev/null
docker network inspect infra-network >/dev/null
```

### 2. 准备环境变量

数据库和 Jenkins 兼容脚本可通过受限权限的环境文件读取配置：

```bash
export PIPELINE_ENV_FILE=/secure/path/miniblog.env
test -r "$PIPELINE_ENV_FILE"
```

建议将文件权限设为 `0600`，且不要把真实密码提交到仓库。数据库脚本常用变量如下：

```dotenv
MYSQL_HOST=db.example.internal
MYSQL_PORT=3306
MYSQL_USERNAME=miniblog
MYSQL_PASSWORD=replace-me
MYSQL_DBNAME=miniblog
```

也可使用同义的 `DB_HOST`、`DB_PORT`、`DB_USER`、`DB_PASSWORD`、`DB_NAME`。
`batch-upsert-articles.sh` 和 `reset-user-password.sh` 不会自动读取
`PIPELINE_ENV_FILE`；执行它们前需要把变量导入当前 shell。这两个 Go 工具的数据库名
变量是 `MYSQL_DATABASE`（不是 `MYSQL_DBNAME`），且不读取 `DB_*` 同义变量。

### 3. 生产变更前检查

在执行任何写操作前，至少确认：

```bash
printf 'host=%s port=%s database=%s user=%s\n' \
  "${MYSQL_HOST:-${DB_HOST:-unset}}" \
  "${MYSQL_PORT:-${DB_PORT:-3306}}" \
  "${MYSQL_DBNAME:-${DB_NAME:-unset}}" \
  "${MYSQL_USERNAME:-${DB_USER:-unset}}"
```

- 输出的主机、数据库和用户属于预期环境。
- 已创建可恢复的数据库备份。
- 当前账号只具有完成本次操作所需的权限。
- 没有另一个迁移、发布或人工数据修复正在同时执行。

## 人工运维工具

### 批量新增或更新文章

先复制示例并填写内容：

```bash
mkdir -p /tmp/miniblog-articles
cp scripts/batch-upsert-articles/articles.example.json /tmp/miniblog-articles/articles.json
cp scripts/batch-upsert-articles/sample-content.md /tmp/miniblog-articles/
```

先做只读校验：

```bash
scripts/batch-upsert-articles.sh \
  -file /tmp/miniblog-articles/articles.json \
  -dry-run
```

确认校验结果后再写入：

```bash
scripts/batch-upsert-articles.sh -file /tmp/miniblog-articles/articles.json
```

必要时可以显式传入 `-host`、`-port`、`-user`、`-db-password` 和
`-database`。建议优先使用环境变量，避免密码进入 shell 历史和进程参数。

结果说明：

- `created`：新增文章数。
- `updated`：按输入中的既有标识更新的文章数。
- `failed`：校验或写入失败数；只要大于 0，就需要检查日志和数据库实际状态。

该工具逐条处理文章，不提供整个文件级别的事务回滚。发生部分失败时，不要直接盲目重跑；
先核对成功记录，再修复输入或连接问题。

### 重置用户密码

先确认目标用户 ID，然后以隐藏输入方式提供新密码：

```bash
read -r -s RESET_PASSWORD
export RESET_PASSWORD
printf '\n'
scripts/reset-user-password.sh -id 123
unset RESET_PASSWORD
```

密码长度必须为 6 到 18 个字符。也支持 `-password`，但命令行参数可能进入
shell 历史和进程列表，不建议在服务器上使用。

成功时脚本会输出目标用户 ID 的更新结果。失败时先确认用户是否存在、数据库目标是否正确，
再检查数据库连接；不要通过日志或聊天工具发送明文密码。

## 数据库脚本

### 初始化数据库

这是高权限操作，默认不会执行。先完成备份和目标确认，再显式打开保护开关：

```bash
export PIPELINE_ENV_FILE=/secure/path/miniblog.env
export ENABLE_DB_INIT=true
export DB_ROOT_PASSWORD='replace-me'
scripts/db-init.sh
unset DB_ROOT_PASSWORD ENABLE_DB_INIT
```

如果 `ENABLE_DB_INIT` 不是 `true`，输出 `Skipping DB init` 且退出码为 0，表示
保护机制生效，并不表示数据库已经初始化。成功执行时会创建或校正数据库、应用用户和基础表。

失败处理：保留完整错误信息，确认管理员账号、网络和 `db/migrations/mysql/init_db.sql`
后再决定是否重试。不要在不清楚已执行到哪一步时手工拼接 SQL。

### 执行数据库迁移

先查看迁移状态（本机已安装 `migrate` 时）：

```bash
migrate \
  -path "$(pwd)/db/migrations/sql" \
  -database 'mysql://USER:PASSWORD@tcp(HOST:3306)/DATABASE?multiStatements=true' \
  version
```

再执行仓库封装：

```bash
export PIPELINE_ENV_FILE=/secure/path/miniblog.env
scripts/db-migrate.sh
```

脚本优先使用本机 `migrate`；未安装时使用 `migrate/migrate` 容器。使用容器且需
访问 Docker 内部服务时，应指定实际网络，例如：

```bash
export DOCKER_NETWORK=infra-network
scripts/db-migrate.sh
```

输出 `no change` 表示当前已是最新版本；输出具体版本并成功退出表示迁移已应用。
若出现 dirty version、SQL 执行失败或连接中断，立即停止发布，先核对数据库实际版本和
迁移副作用。不要在未调查原因前执行 `force`。

### 加载种子数据

默认不会写入。确认目标和备份后执行：

```bash
export PIPELINE_ENV_FILE=/secure/path/miniblog.env
export ENABLE_DB_SEED=true
scripts/load-seed-data.sh
unset ENABLE_DB_SEED
```

也可以使用 `make db-seed`，两者都会依次尝试加载 `user.sql`、`module.sql`、
`section.sql`、`article.sql` 和 `casbin_rule.sql`。

如果输出 `Skipping seed load`，表示保护机制阻止了写入。若中途失败，前面已成功加载的
文件不会自动回滚；应先核对各表状态和 SQL 的幂等性，再决定从哪里继续。

## Jenkins/发布兼容脚本

### Compose 部署

`deploy.sh` 仍由 `Jenkinsfile` 调用。GitHub Actions 的生产部署直接在 workflow 中编排，
不会调用该脚本。

预检 Compose 配置：

```bash
export DEPLOY_COMPOSE_FILES='docker-compose.yml docker-compose.prod.yml'
docker compose -f docker-compose.yml -f docker-compose.prod.yml config --quiet
docker network inspect miniblog-network >/dev/null
docker network inspect infra-network >/dev/null
```

确认后执行：

```bash
export PULL_IMAGES=true
scripts/deploy.sh
```

成功后检查：

```bash
docker compose -f docker-compose.yml -f docker-compose.prod.yml ps
docker compose -f docker-compose.yml -f docker-compose.prod.yml logs --tail=100
```

失败时使用同一组 Compose 文件查看容器状态和日志。不要通过创建旧的
`infra_shared`、`qs_net` 或 `jenkins_net` 网络来绕过错误；当前 Compose 契约是
`miniblog-network` 和 `infra-network`。

### 推送镜像

`push-images.sh` 根据 Jenkins 设置的构建标志选择镜像：

- `RUN_FRONTEND_BUILD=true` 时读取 `FRONTEND_BLOG_IMAGE_TAG` 和
  `FRONTEND_ADMIN_IMAGE_TAG`。
- `RUN_BACKEND_BUILD=true` 时读取 `BACKEND_IMAGE_TAG`。

人工预检示例：

```bash
docker login ghcr.io
docker image inspect "$BACKEND_IMAGE_TAG" >/dev/null
```

然后执行：

```bash
scripts/push-images.sh
```

输出 `Images were not built in this run, skipping push.` 且退出码为 0，表示没有任何
构建标志与镜像标签组成有效推送任务。推送失败时先检查登录状态、标签是否存在和仓库权限，
不要重新构建或覆盖标签，除非已经确认流水线产物不正确。

## 维护规则

- 新脚本必须说明调用方、输入、写入范围、保护开关、成功判据和失败恢复方式。
- 能放在 Makefile 或 GitHub Actions 中清晰表达的流水线逻辑，不再复制为一次性脚本。
- 宿主机 Docker 代理、Runner 安装和基础设施网络迁移由对应基础设施仓库维护。
- 删除脚本前必须先检查 `Jenkinsfile`、Makefile、workflow 和文档引用。
