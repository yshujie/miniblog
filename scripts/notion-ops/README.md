# Notion 生产只读维护入口

GitHub Actions 的 **Notion read operations (manual)** 仅允许 `main`，输入只开放 `schema_check`、`dry_run` 和 `bootstrap_preview`。使用已运行 backend 的不可变 image ID，不拉镜像、不重建服务、不迁移、不启用来源、不写文章或 Notion。`dry_run` 和预览会保存本地映射、运行审计与历史基线，属于数据库维护读取流程，不能把它们理解成完全不写 MySQL。

## 凭据与报告

仓库 Secret `MINIBLOG_NOTION_TOKEN` 是专用只读内部连接，必须授权固定五库。仅部署和手动维护步骤读取它；不会进入 build/test 环境、构建参数、浏览器或 SSH export 命令。runner 将 token 写入临时 0600 文件，通过原生 SCP 传至服务器 0700 独立目录；SSH 参数只含固定脚本、公开运行 ID 和路径。脚本只接受单行字母、数字、`_`、`-`，拒绝 CR/LF、引号和 Compose `$` 插值。临时文件在正常完成或失败时清理；workflow 的 always 清理步骤再回收当前运行的上传目录。SIGKILL 或服务器失联后，操作者只核验该 run_id 的私有残留，不进行全局清理。

`SVRD_HOST/USERNAME（或 USER）/SSH_KEY/SSH_PORT` 沿用现有部署连接。可选 `SVRD_HOST_FINGERPRINT` 为 `SHA256:…`，配置后严格核验主机密钥；未配置沿用首次信任主机的现有契约。服务器需 Python 3.9+、Docker、flock，deploy 用户需可读取当前 backend 元数据、执行任务并写 `/opt/miniblog`。

普通 CI 在部署步骤原子生成 `/opt/miniblog/.env`（0600），保留已有同步开关、间隔、作者与手工登记开关。只读 Secret 非空时替换 token，未配置时保留旧 token；不自动开启同步。备份 `.env.previous` 也是 0600。维护任务不改 `.env`，任务环境只含只读 token；需要数据库时追加当前运行容器的 MySQL 字段。优先核对实际 `MINIBLOG_DATABASE_*` 映射，两套别名不一致即阻止，不复制 JWT、Redis 或其他运行凭据。

`MINIBLOG_NOTION_BOOTSTRAP_TOKEN` 暂不被任何普通 CI 或此维护 workflow 读取。它以后只用于已审核确认清单的独立一次性回填任务，不写入 `.env` 或常驻容器；本入口没有 `bootstrap_apply`、`catalog_prepare` 或 `sync`。

报告路径固定为 `/opt/miniblog/ops/notion/<GitHub run_id>-<run_attempt>/`。目录 0700，JSON、元数据与日志 0600，任务使用目录所有者 UID:GID。`task.env` 和上传 token 会清理；任务输出先在内存中按本次 token 和数据库密码脱敏，再写受限日志，超时输出同样处理。报告保留供受权操作者核对，不上传 GitHub artifact、不打印全文或原始错误。Actions 只显示模式、完成状态及报告位置；失败时也保留服务器证据。`operation.json` 记录 image ID。部署和维护共同持有 `/opt/miniblog/.operations.lock`，Actions 也使用 `miniblog-prod` concurrency group。

## 按顺序运行

1. 当前程序兼容部署后，选择 `schema_check`。此模式只传 token，不读取或传递任何数据库凭据。五库均须返回合法 schema、实际 ID 与固定来源 ID 相符；`schema.json` 仅是建议映射，不意味着已保存配置。
2. 通过后台 `/content/sync` 核对并设置五库模块，初次基线来源全部停用；保持运行总开关关闭，暂停同步，等待当前运行结束。只读联验不要求 `source_writes_paused=true`，也不会修改它，手工登记可以继续使用。
3. 选择 `dry_run`。脚本再次完成五库 schema 检查，通过结构 preflight，读取 `status-before.json` 并确认暂停、无活动任务、五库模块齐备；未冻结时还要求五库全停用。CLI 全量读取、自动固化缺失字段映射并冻结首次完整历史基线。完成后 `status-after.json` 必须证明 `baseline_frozen=true`。失败保持来源停用，不使用部分扫描作为历史清单。
4. 选择 `bootstrap_preview`。仍先核对 schema、运行总开关关闭与暂停配置；必须已经冻结基线。`result.json` 含完整候选、状态建议、目录变化和审核指纹。正常 pending/frozen 与 `public_url_missing` 候选允许生成；CLI 要求实际预览运行 completed 且真实 failed/blocked 为零。预览不代表批准接管或发布，历史匹配、外链别名与新页判定逐项人工核对。申请 Published 接管前必须核验公开 URL、目录绑定和启用条件。

之后仍按[维护指南](../../docs/notion-sync.md)分库准备目录、解决冲突、重新生成最终预览及确认清单，在双暂停窗口审核回填并接管，紧接新鲜同步。本 workflow 不执行这些写入步骤。任何手工编辑或配置变化后，旧预览仅作线索，实际接管需重新读取并通过指纹和版本检查。

任务有独立容器名称、随机 owner label 和 CLI 超时；取消或异常结束只清理本次拥有的容器，不复用或删除其他同名任务。每次执行使用新报告目录，重试 workflow 由 run_attempt 隔离；不覆盖上次证据。

## 本地验证

`PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s scripts/notion-ops -p 'test_*.py'` 仅使用模拟 Docker/SSH。覆盖 token 不进入参数/日志、未配置保留、重复键、插值拒绝、原子替换、私有权限、数据库别名冲突、schema 无 DB、暂停/基线门槛、完整候选、失败证据保留与取消清理。workflow 另用 actionlint 验证。没有调用真实 Notion 或服务器。
