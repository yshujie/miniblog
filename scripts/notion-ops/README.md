# Notion 生产维护入口

GitHub Actions 的 **Notion read operations (manual)** 仅允许 `main`，输入只开放 `schema_check`、`dry_run` 和 `bootstrap_preview`。使用已运行 backend 的不可变 image ID，不拉镜像、不重建服务、不迁移、不启用来源、不写文章或 Notion。`dry_run` 和预览会保存本地映射、运行审计与历史基线，属于数据库维护读取流程，不能把它们理解成完全不写 MySQL。

## 凭据与报告

仓库 Secret `MINIBLOG_NOTION_TOKEN` 是专用只读内部连接，必须授权固定五库。仅部署和手动维护步骤读取它；不会进入 build/test 环境、构建参数、浏览器或 SSH export 命令。runner 将 token 写入临时 0600 文件，通过原生 SCP 传至服务器 0700 独立目录；SSH 参数只含固定脚本、公开运行 ID 和路径。脚本只接受单行字母、数字、`_`、`-`，拒绝 CR/LF、引号和 Compose `$` 插值。临时文件在正常完成或失败时清理；workflow 的 always 清理步骤再回收当前运行的上传目录。SIGKILL 或服务器失联后，操作者只核验该 run_id 的私有残留，不进行全局清理。

`SVRD_HOST/USERNAME（或 USER）/SSH_KEY/SSH_PORT` 沿用现有部署连接。可选 `SVRD_HOST_FINGERPRINT` 为 `SHA256:…`，配置后严格核验主机密钥；未配置沿用首次信任主机的现有契约。服务器需 Python 3.9+、Docker、flock，deploy 用户需可读取当前 backend 元数据、执行任务并写 `/opt/miniblog`。

普通 CI 在部署步骤原子生成 `/opt/miniblog/.env`（0600），保留已有同步开关、间隔、作者与手工登记开关。只读 Secret 非空时替换 token，未配置时保留旧 token；不自动开启同步。备份 `.env.previous` 也是 0600。维护任务不改 `.env`，任务环境只含只读 token；需要数据库时追加当前运行容器的 MySQL 字段。优先核对实际 `MINIBLOG_DATABASE_*` 映射，两套别名不一致即阻止，不复制 JWT、Redis 或其他运行凭据。

`MINIBLOG_NOTION_BOOTSTRAP_TOKEN` 不被普通 CI 或只读维护 workflow 读取。它只用于下述独立维护 workflow 中已审核确认清单的一次性回填任务，不写入 `.env` 或常驻容器；本入口没有 `bootstrap_apply`、`catalog_prepare` 或 `sync`。

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


## 审批清单维护入口

**Notion reviewed maintenance (manual)** 与原只读 workflow 分开，仅 `main` 可运行，与发布/读取共用 `miniblog-prod` concurrency 和服务器 `.operations.lock`。它不接受任意命令或文件路径。输入为 action、manifest_id、原始文件 SHA256、当前 backend 镜像完整 40 位 SHA、来源配置版本；服务器读取唯一固定位置 `/opt/miniblog/ops/notion/reviews/<manifest_id>.json`。reviews 及上级 ops/notion 目录必须为当前操作者所有、0700、非 symlink；清单是所有者匹配、0600、非 symlink 的普通文件。拒绝重复 JSON 键、未知 envelope/input 字段、超限文件、错误 hash、来源和版本。私有清单/报告不能上传 Actions artifact。

清单 v1 精确字段为 `version:1`、`mode`（与 action 相同）、`source_id`、`expected_config_revision`、`expected_image_sha`、`input`。全局 `pause/resume/scheduler_off` 的 source_id 为空字符串、revision 为 0、input 为 `{}`。其余动作必须为固定五库之一、正整数 revision；包括 `scheduler_on`。SHA 对应当前运行容器的 `ghcr.io/yshujie/miniblog-backend:<fullSHA>`；脚本再次比较该 tag 的本地 image ID 和运行 image ID，不拉取任何镜像。

| action | 私有 input | 执行门槛及结果 |
| --- | --- | --- |
| pause | `{}` | 先设置双暂停，等待至少 20 秒再确认无 current run、lease 和独立 CLI/维护容器；不要求基线已冻结。已暂停而排空失败仍保留暂停。 |
| resume | `{}` | 双暂停、冻结基线、排空、无未知回填 journal；仅设置两个暂停字段为 false。 |
| catalog_prepare | `{}` 或 `reuse_map` | 双暂停、冻结基线、排空、无未知 journal；复用原事务 PrepareCatalog，主题/章节审核复用映射含前值，可能递增版本。 |
| catalog_bind | `binding_id,option_id,section_code` | 同上，固定来源与精确配置版本。 |
| catalog_activate | `items,expected_newly_public_article_ids` | 同上，原事务核对目录 title/status/sort 前值及完整新增公开 ID 集；不能以启用祖先隐式扩大发布范围。 |
| source_update | `label/module_code/enabled/config/expected_config_revision` 的明确修改 | 同上；内部 revision 必须等于 envelope，业务 CAS 成功后版本更新。 |
| bootstrap_preview | `{}` 或 `manual_matches` | 双暂停、冻结基线、排空；允许未知 journal 的显式重新审阅。当前 CLI 仍返回五库候选，envelope 来源用于当前审核版本核验。 |
| bootstrap_apply | `confirmations` | 双暂停、冻结基线、排空；每页包含最新 fingerprint、文章/新页身份、expected_state、confirmed_by 等既有确认字段。当前存在未知 journal 时仅允许该未决 PageID 集合内的确认，不接受扩展范围；业务层还会 fresh GET 校验旧指纹、状态和本地快照。 |
| sync | `{}` | 控制已恢复、冻结基线、来源已启用、无当前租约/任务和未知 journal。显式 `--enable-sync`，作者由唯一有效管理员昵称解析；任务仍扫描固定五库，只有来源配置允许的投影会应用。 |
| scheduler_on | `validated_sync_run_id` | 重新双暂停并排空、无未知 journal、当前来源已启用；完成且无 Failed/Blocked 的近期即时 sync、来源 LastSuccessAt 和不可覆盖的版本/image 成功 receipt 全部匹配。 |
| scheduler_off | `{}` | 双暂停、冻结基线、排空；允许未决 journal，便于回退运行开关。 |

审批文件改变后必须重新核对 SHA256，来源配置改变后必须重新核对 revision。没有成功证明时绝不自动重试。每次 Actions run_attempt 创建新的 `/opt/miniblog/ops/notion/rollout-<run_id>-<attempt>/`，拒绝复用目录；每个 CLI 调用都有独立 report/log。CLI 非零即终止，但完整私有 JSON 和已有数据库 journal 保留，不能据进程失败推断事务/Notion 请求完全未生效。部分回填或结果未知须先阅读该报告，再用同一确认清单的新运行受限恢复；不自动回放其他页面。普通目录/配置修改及 resume/on 会被未决 journal 阻止。

即时 sync 完成后还会重新核验来源配置版本，把成功证明写入 `/opt/miniblog/ops/notion/sync-receipts/<sync-run-UUID>.json`（0600、不可覆盖）。scheduler_on 要求该 receipt 的 run/source/revision/image/起止时间与当前数据库及审批一致；私有 reader SHA256 也必须匹配当次只读 Secret，实际常驻 Docker reader 必须完全相同，完成时间不超过 30 分钟。配置或 reader 轮换后必须重新即时 sync；旧配置上的成功运行不能用来开启新配置；不接受任意手工编造的 run 时间作为证明。

首次放行顺序：双暂停 → 目录复用/绑定和受审核祖先启用 → 修改来源配置/启用 → fresh 最终预览 → 人工核对带指纹确认 → 分库 bootstrap_apply → 确认所有未知 journal 已解决 → resume → 一次 sync 并核对文章/公开阅读效果 → pause 排空 → 以该 sync 证明 scheduler_on → 再 resume。scheduler_on 本身保持数据库暂停，不能把健康重启视作定时任务已经开始；最后 resume 后才允许下一轮定时同步。

开关动作只原子修改 `.env` 的 `MINIBLOG_NOTION_SYNC_ENABLED` 和 `MINIBLOG_NOTION_SYNC_AUTHOR`（由当前唯一有效管理员昵称解析），保留其余原行；去除 writer。备份、恢复文件和 `.env` 均 0600。只对当前审核镜像执行 `compose up --no-deps --no-build --pull never miniblog-backend`；重启后核验 image/tag、实际 Docker env、Docker healthy 和本机/公网 health。开启核验失败时仅恢复总开关 false 并尝试重启，数据库保持暂停，不重试 Notion、不自动 resume；若恢复重启也失败，按受限报告人工处理。关闭失败保持 false 目标配置，不重新打开开关。

writer Secret 只出现在 bootstrap-apply 独立 job 的 apply 步骤，进入私有传输文件及该次 apply 容器的临时 env；status/drain/preview 等任务都不拿 writer，常驻 `.env`、镜像、备份无 writer。普通操作只在确需 Notion 读取的动作注入 reader。任务 `.env` 在 finally 删除。取消 workflow 的 always cleanup 依私有 run 容器 receipt，核验 exact 随机 owner label 后只删除本次容器/临时 env，不删除审计报告或其他任务；SSH/服务器失联或 SIGKILL 仍须人工核验当前 run 遗留，禁止全局 prune。

## 本地维护回归

`PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s scripts/notion-ops -p 'test_*.py'` 包含原只读契约及新维护 mock。新测试覆盖精确审批字节/owner/symlink/重复键、固定范围与版本、真实运行镜像绑定、租约和独立任务排空、未知 journal 只限原页恢复、writer 独立传输及取消清理、配置变化不接受旧 receipt、重启健康失败回退和保持双暂停。`actionlint .github/workflows/notion-rollout.yml` 校验 workflow。测试不连接 Docker daemon、生产或 Notion；这些本地结果不等于生产验收。
