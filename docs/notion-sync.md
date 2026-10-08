# Notion 自动同步维护指南

在 Notion 中编写、选择主题和设置「博客状态」，miniblog 定时同步标题、标签、目录、状态及阅读地址。正文、作者、历史 ID 和本地排序保留。飞书与未接管 Notion 文章继续手工收录。同步默认关闭；程序发布与历史接管分别验收。

## 来源与目录

`source` 调用固定官方接口；`notionsync` 扫描、预览、协调与接管；`catalog` 维护主题绑定；`article` 在单个事务中投影；`reading` 统一公开规则。启动入口负责任务取消与停机等待，Notion 故障不阻止博客启动。

| 子库 | Data source ID |
| --- | --- |
| Go | 2bf330bd-ddf1-80a6-aa49-000bbd1e154b |
| DDD | d579f619-4f1c-4e7e-9593-ab6529fc63d0 |
| 数据库 | cfa962f3-12be-40aa-95eb-9122186bd4ba |
| 数据结构&算法 | 03d24108-5172-4e8f-8cfd-e3bd52e437af |
| 项目开发 | d13097e3-b78a-4b6e-8848-979749d4b2f3 |

每库先绑定既有模块 code。字段保存 property ID，状态保存 option ID，名称只用于展示。映射键为 draft/published/unpublished/archived。删除重建、类型变化或未知状态需重新核对。描述、难度和日期仅保存来源快照，参考链接不作为原文地址。

新主题包括空主题可创建章节，code 由稳定身份生成，排序追加。同名既有章节只生成待绑定建议，经后台确认复用原 code。改名不改变 code、排序、启用状态；主题消失保留目录及绑定。IAM、Qlume、数据结构、算法需分别确认章节。删除目录检查绑定依赖。

## 配置和运行

- MINIBLOG_NOTION_TOKEN：运行只读内部连接，仅服务端安全环境注入，并授权五库读取。禁止日志、浏览器、构建参数和仓库存储凭据。
- MINIBLOG_NOTION_SYNC_ENABLED：运行总开关，默认 false。后台暂停与来源启用独立控制。
- MINIBLOG_NOTION_SYNC_INTERVAL：默认 5m，最低 1m；整轮最多四分钟。
- MINIBLOG_NOTION_SYNC_AUTHOR：新文章本地默认作者，可空；已有作者保持原值。

同步版本固定 2026-03-11，串行每秒最多一次请求，单次十秒、最多三次只读尝试。429/529 遵守 Retry-After；超过本轮时限保存冷却退出。标题预览保留原五秒超时及不重试行为。

定时和手动任务共用数据库租约：六十秒有效、十五秒续租，使用数据库时间和递增 epoch。文章事务复核租约、配置及绑定版本，HTTP 在事务外。暂停、停机或失效任务不能继续提交。

每库查询归档与非归档分区、全部状态及全部游标，并检查 request_status。先汇总五库再判断跨库移动，缺席绑定页单独获取。查询不完整禁止按缺席撤回。403、404、限流及网络失败记录问题，保留最近成功状态与内容。成功读取确认原生归档、垃圾箱、移出五库或撤销公开才隐藏。

## 四态与管理权

| Notion 状态 | 已接管文章 | 尚无文章 |
| --- | --- | --- |
| 草稿 | 草稿，停止公开，保留 ID | 保存来源记录 |
| 已发布 | 完整校验后更新并公开 | 校验后创建文章 |
| 已下架 | 停止公开，保留文章 | 保存来源记录 |
| 归档 | 保留文章及来源，可以恢复 | 保存来源记录 |

自动公开需要有效 HTTP(S) public_url、合法标题与标签、有效主题绑定、完整且启用目录、有效来源和无本地下架或待核验标记。成功读取 public_url 为空清除旧值，失败保留上次成功地址。实际匿名阅读及嵌入另行验收。

provider=notion 不代表接管，生效绑定才取得管理权。托管标题、分类、标签、来源、状态只读；历史正文保留只读。旧 PUT 只接受未实际改变的托管字段，变更原子拒绝。移动、状态和导入同样在业务层保护。作者通过 local-fields 保存，归档不可编辑；排序仍由后台维护。

紧急下架是本地独立标记，同步不会清除。解除后进入待重新核验，可靠扫描确认后才公开。暂停冻结当前结果，不自动下架或解除管理权。

## API 和页面

沿用认证、/v1 和 code/msg/payload；新增 ID 是字符串。列表默认 page=1、limit=20，limit 最大 100。

| 操作 | 接口 |
| --- | --- |
| 状态、来源、页面 | GET /v1/admin/notion-sync/status、sources、pages |
| 暂停、来源配置 | PATCH /v1/admin/notion-sync/control、sources/:source_id |
| 目录冲突 | PUT /v1/admin/notion-sync/catalog-bindings/:binding_id |
| 预览、同步 | POST /v1/admin/notion-sync/runs，dry_run 或 sync，202+run_id |
| 运行和明细 | GET /v1/admin/notion-sync/runs、runs/:run_id、runs/:run_id/items |
| 作者 | PATCH /v1/admin/articles/:id/local-fields |
| 本地下架 | PUT /v1/admin/articles/:id/publication-hold |

配置和绑定提交 expected_config_revision，陈旧配置返回冲突。dry_run 保存审计和基线，不修改 Notion、文章或目录。sync 重新扫描当前状态，不重放旧预览。首轮状态回填只走独立 CLI。

后台 /content/sync 提供来源配置、任务和问题处理。工作台分别展示来源期望、本站状态及阻止原因，提供打开 Notion、本地作者与下架操作。

公开目录和详情共用可见性查询并输出 reading_url。前台优先使用该值，旧服务端回退 external_link；iframe 和原文入口共用有效地址。目录缓存六十秒，可见阅读页定时核验，隐藏后暂停，恢复合并刷新。标题、排序变化保持 iframe；文章或地址变化才重建。404 清除内容，网络失败保留最近成功内容。历史链接按当前归属定位并保留 query/hash。

## 迁移和首轮接管

先备份、核验真实 schema、迁移标记、部署版本和现有文章。此前 47 页仅为审计起点。

1. 完成 [内容维护指南](content-refactor.md) 的 000004、身份审计和 000005。冲突未解决时停止。
2. 部署当前完整程序前执行 000006，仅扩展 tags_json 和六张同步表，总开关保持关闭。CI 的 [只读结构检查](../scripts/content-preflight/README.md) 要求 clean version >= 6 与真实字段约束齐备；关闭同步不能代替迁移。启动不会自动建表。兼容版本必须认识 JSON、别名、管理权和 reading_url。
3. 暂停同步及来源写入，排空任务、后台提交与导入。无损标签回填：`go run ./scripts/audit-content -apply -backfill-tags -report /tmp/miniblog-tags-audit.json`。JSON 非 NULL 时权威，NULL 才读取旧 CSV；上限 64KiB，超限报错。兼容 CSV 仅在无损且容量允许时更新。
4. 配置五源模块，执行全库 dry_run。首个完整基线将已有页固定为 baseline_pending，未确认页以后仍不能自动视为新页。
5. CLI bootstrap_preview 列出匹配、重复、目录/子章节变化、博客状态、公开条件与指纹。标题仅是线索。确认文件逐项明确 Page ID、文章 ID 或新页面、拟回填状态、审核人。
6. bootstrap_apply 用独立写凭据，暂停两开关并完成跨表冲突审计后执行。逐页记录待写入、已请求、回读确认、已接管。非空且不同的 Notion 状态拒绝覆盖，超时先回读。回读后 MySQL 重新核验配置、文章、归属、身份及指纹；不宣称两系统原子提交。
7. 升级身份保存不可变 legacy_source_key。登记、编辑、导入和冲突查询同时查主身份与别名，并对已知页面及公开阅读地址精确匹配，防止不含 Page ID 的 Notion Site slug 重复登记；未知地址不猜测归属。历史外链、正文、ID、code、作者和排序不重写；重复歧义不合并。
8. 测试页验证新增、重复扫描、下架、归档、恢复和撤销公开。先 Go 再逐库启用。未解决身份冲突阻止全量切换。

CLI 参数和确认文件格式见 `go run ./scripts/notion-sync -h`。运行期只读；bootstrap 凭据仅在审核维护窗口注入。

## 回退与验证

先暂停并撤销租约，再关闭总开关。保留文章、绑定、JSON、别名和 schema。000006 down 只回退标记，不删除数据。禁止回到不认识托管规则的原始程序。恢复历史值须核对版本，防止覆盖后续修改。

记录保留九十天，绑定及未解决问题长期保留。健康展示最近尝试、完整扫描及成功应用；超过十五分钟未完整成功显示异常。

结果见 [同步验证记录](notion-sync-verification.md)。代码、本地 MySQL、浏览器夹具、真实 Notion/飞书阅读、使用验收和生产部署分别记录。
