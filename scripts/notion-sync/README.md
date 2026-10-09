# Notion 同步命令行

数据库迁移由部署流程完成，本工具不会自动建表。默认模式 `dry_run` 读取固定五个 Notion 数据源并写本地审计清单；首次完整扫描将所有历史 page ID 与基线标记原子保存。它不会改文章、目录或 Notion。

- `--mode schema_check`：不连接 MySQL，只对固定五库执行官方 schema 读取，输出请求与实际 data source ID、字段类型、property/option ID 和建议映射。不查询页面、不写 Notion；五库任一身份、字段或授权不符合要求均返回非零，并保留完整受限报告。
- `--mode catalog_prepare --source-id … --expected-config-revision …`：只准备选定来源当前 schema 的主题及空主题。要求历史基线已冻结、同步和来源写入均暂停，复用租约和版本守卫。创建章节或待审核绑定、返回 binding ID；不查询页面、不处理文章、不写 Notion。可增加 `--catalog-plan` 文件，格式 `{"reuse_map":[{"option_id":"…","expected_option_name":"…","section_code":"…","expected_section_title":"…","expected_section_status":1,"expected_section_sort":0}]}`；状态和排序必须显式提供，0 是有效审核值。目录归属、旧值、占用及 schema 全部核验后，在同一事务绑定和改名；非法复用整批回滚。普通冲突仍保留待处理报告。变化使配置版本增加，重复无变化不增加版本、不重排。之后必须重做历史预览。不能临时解除暂停绕过维护入口。
- `--mode control_update --input …`：严格 JSON 只接受 paused/source_writes_paused，双暂停撤销租约。
- `--mode source_update|catalog_bind|catalog_activate --source-id … --expected-config-revision … --input …`：受限来源配置、绑定和已审核目录启用。目录启用清单须包含旧名称、状态、排序和精确新增公开文章 ID 集合；变化原子拒绝。来源、绑定和目录启用的 CLI 命令在独立租约事务内复核双暂停、冻结基线和无未决回填日志，排空预检不能代替提交时守卫。
- `--mode drain_status`：输出当前任务、租约及未决回填日志数量与 Page ID 集合；仅数据库租约为空不证明 HTTP 或容器已经排空。
- `--mode run_status --run-id …`：读取特定 UUID 运行证据。
- `--mode author_resolve`：只读取唯一账号的有效昵称（最多 128 字符），不按未定义的历史 status 筛选，也不读取密码；多个账号或昵称无效时拒绝自动选择；CLI 默认作者来自 MINIBLOG_NOTION_SYNC_AUTHOR，显式 --author 优先。
- `--mode status`：查看本地运行状态，不请求 Notion。CLI 的 enabled 仅反映本次 --enable-sync 选项，不代表常驻定时器；常驻总开关从后台 HTTP 状态与部署配置核验。
- `--mode dry_run`：重新读取官方 API，生成审计运行。
- `--mode sync --enable-sync`：重新读取并应用；要求基线已冻结、源已配置并启用、暂停已解除。
- `--mode bootstrap_preview`：完整读取后生成历史匹配清单。自动提出的匹配依据为可信 page ID，或本轮读取已证明属于该页的精确公开地址；未知 slug 不猜，标题仅作线索。`requires_legacy_alias:true` 的候选须额外确认 `allow_legacy_alias:true`。清单包含公开 URL 条件、目标主题章节与子章节保留/转为章节直属变化；歧义、未匹配项继续待确认。`--manual-matches /path/matches.json` 可提交已人工核对的历史别名关联（格式 `{"manual_matches":[{"page_id":"…","article_id":"…"}]}`），生成新的审核指纹。
- `--mode bootstrap_apply --confirmations /path/review.json --allow-notion-write`：仅处理逐项核对的清单；生产任务必须同时给出 --source-id 和 --expected-config-revision，全清单范围核验后才写 Notion。已发布在写前和回读后都核验有效公开地址、目录路径、绑定、标题标签及来源启用，草稿/下架/归档不依赖公开地址。要求暂停同步与外部来源写入，并提供独立 `MINIBLOG_NOTION_BOOTSTRAP_TOKEN`。读取仍使用 `MINIBLOG_NOTION_TOKEN`。本模式不修改 Notion 正文。

确认文件格式：`{"confirmations":[{"page_id":"…","article_id":"…","expected_state":"published","expected_fingerprint":"预览返回的指纹","confirmed_by":"操作人"}]}`。四态为 `draft`、`published`、`unpublished`、`archived`。已有文章回填值必须与核对时的本地状态一致；非标准历史别名须额外 `allow_legacy_alias:true`。明确未匹配的新页面使用 `new_page:true` 并省略 `article_id`，同样要求明确四态、审核指纹与确认人，绝不从空状态推导发布。接管后先保存来源绑定，草稿/下架/归档不创建博客文章；后续重新读取并满足已发布、公开 URL 和有效目录等条件时才创建。没有核对项不写入、不接管。写请求超时后先回读；结果未知保留 journal，下一次只在全部核验一致时继续。

历史预览报告兼容原 run_id/items，同时增加 run_items，完整保存本次读取快照供重新审核受影响项。审计报告完整遍历所有运行明细分页；失败或部分失败返回非零退出码，并保留逐项结果。回填前验证最新 schema、页面、来源身份冲突、配置版本和本地完整快照；单次写后必须回读，结果未知保留 journal。配置编辑在回填运行期间暂不可用。普通扫描冻结 write_requested/verified 的原审核快照，不让旧清单获得新元数据授权。显式重新预览必须重新获取页面，且审核快照与新指纹在同一租约事务中保存；读取失败不替换原审核资料。

通过受限环境配置数据库连接和 token，不把凭据放进参数、报告或日志。帮助和参数错误不读取或打印环境凭据；兼容的 `--db-password` 仅保留旧调用能力，生产任务不用该参数。`--report` 使用同目录 0600 临时文件及原子替换；旧 0644 文件被新 0600 文件替换，拒绝符号链接和非普通目标，不沿用宽松权限。

生产镜像包含 `/app/notion-sync`、`/app/audit-content` 和 `/app/content-preflight`。在服务器受限报告目录中，通过一次性 `docker compose run --rm --no-deps --entrypoint /app/notion-sync miniblog-backend` 任务执行以上模式；挂载目录和私有 env 文件时不得打印完整 Compose 配置或环境。常驻配置沿用服务器 /opt/miniblog/.env，原子更新为 0600；更新时核验无并行 CI 部署，不能另建未被 CI 读取的 runtime.env。CLI 默认读取 DB 环境；报告挂载目录 0700，输入和报告 0600，任务容器使用报告目录所有者的 UID:GID，避免生成宿主无法读取的 root-owned 0600 文件。bootstrap 凭据不能只放进 compose --env-file：常驻 service 未映射该变量，须使用服务器支持的任务级 --env-from-file，或安全读取、export 后通过 --env MINIBLOG_NOTION_BOOTSTRAP_TOKEN 传入变量名，参数不带值。运行只读 token 保留在服务端，bootstrap token 仅向审核回填任务注入，不配置给常驻服务。历史核对清单也固化在运行明细中，后台只读查看，网页没有反写 Notion 的入口。暂停或关闭定时同步只冻结当前结果，不自动逆向修改 Notion；回退保留绑定、历史来源 alias 和运行记录。

单元测试使用内存数据库、模拟 REST 传输及模拟客户端。真实官方 REST、生产启用及实际设备的当前验收状态见[验收记录](../../docs/notion-sync-verification.md)。

分库执行门槛：目录准备、绑定、目标来源启用后读取最终版本，再生成完整 bootstrap_preview；CLI 核验运行完成且 failed/blocked 为零，正常 pending/frozen 允许，但候选仍需逐项审核。每库确认文件只含当期来源的页面，精确核对 ID、新页标记、状态及指纹；Published 还须 public_url_available、无发布阻止原因、已绑定且启用的目标目录。接管会使历史 Published 文章待核验，成功后立即解除双暂停，并执行 --mode sync --enable-sync 全量重新读取；部署总开关 false 时后台 HTTP 同步不能代替该命令。核验本次运行、该来源成功时间与逐项 effective_visibility 后再开定时器。部分接管失败时停止扩大范围并保留日志，先核验成功项、重新审核失败项，再完成即时同步，不能静默留下待核验文章。

GitHub Actions 只读联验及审核文件驱动的分库维护入口见[生产维护入口](../notion-ops/README.md)。原只读流程继续仅开放 schema_check、dry_run、bootstrap_preview；独立 rollout 流程接受固定动作、私有审核文件摘要、实际镜像 SHA 和来源版本，只有 bootstrap_apply 注入一次性写 Secret。

真实五库 schema 检查通过后，固定模块改名与初始字段配置可使用[一次性初始化工具](../notion-rollout-init/README.md)：默认仅读预检，显式执行保留停用状态和历史文章；不代替后续基线、接管审核或公开启用。

未决回填的恢复限制：正常超时保留原审核项，可先回读并按同清单继续；含未决日志时禁止通过维护命令改来源、改绑定或开定时器。若此时 Notion 属性已删除重建，不能盲目重复原写入。须保留日志，经独立人工核对通过现有后台配置界面修正映射，再重新预览、审核受影响页面；不存在自动提升配置版本或自动清除未知写结果的恢复路径。
