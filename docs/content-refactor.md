# 内容管理与阅读维护指南

Notion 托管工作流见 [自动同步维护指南](notion-sync.md)。手工内容继续使用目录工作台。

后台目录工作台支持选择章节或子章节、粘贴外链、确认标题并一次发布，随后继续收录。保留 Go/Gin/GORM、Vue/Element Plus/Pinia、历史文章 ID、目录 code 和原始外链。Notion 标题只作建议，飞书继续手填。

生产数据与 schema 尚未核验。以下验证来自本地测试库和浏览器夹具；生产部署、重复取舍和实际平台阅读验收由维护者完成。

## 职责与契约

| 模块 | 责任 |
| --- | --- |
| biz/catalog | 目录关系、启用、排序、删除依赖及模块锁 |
| biz/article | 登记、编辑、状态、移动、完整分组重排与导入 |
| biz/reading | 公开目录、详情与可见性；旧 blog 入口转接 |
| source | 保守来源身份、固定官方 Notion 标题预览 |
| 后台 api/content 与 composables | API、大 ID、收录、冻结重试、预览及目录上下文 |
| 阅读站点 reading | 共享目录、首篇选择及路由加载协调 |

文章必属于一个章节，子章节须属于该章节，模块由章节推导。作者、标签可空，外链无需正文。草稿 1、已发布 2、已下架 3、归档 4；兼容接口归档仍返回 Deleted。编辑保持状态，归档先独立恢复为草稿。发布和移动已发布文章要求目录整条路径启用；公开读取同时要求 Published、关系完整及全部祖先启用。

内容写入先锁模块，跨模块按模块 ID 顺序锁两个模块，再锁文章并复核归属。事务采用 READ COMMITTED，避免 MySQL 等待锁前的旧快照影响位置。章节直属和子章节各有完整排序组；读取按 pos,id，重排提交全部未归档成员，拒绝跨组、遗漏或重复，归档位置保留。

标准 Notion 页面按规范 ID 生成 SHA256(notion:pageID)；不可靠页面链接、飞书及其他来源保留路径、参数与 fragment，以保守规范 URL 生成 SHA256(url:canonicalURL)。来源键可空，使用 ASCII 二进制比较的 64 位十六进制摘要。原外链保留，归档仍占用身份。

登记重新解析身份，新增返回 created，重复返回 already_registered 和已有记录，不覆盖标题、目录、状态。网络重试复用冻结内容。旧数字 id 保留，新增 id_text；新命令和公开详情使用字符串 ID。禁止前端把文章 ID 转为 Number，服务端拒绝超出 signed BIGINT 范围。

接口仍在 /v1，字段见 [内容 OpenAPI](../api/openapi/openapi.yaml)：

| 操作 | 路径 |
| --- | --- |
| 预览 | POST /v1/admin/article-sources/preview |
| 登记并可直接发布 | POST /v1/admin/articles/register |
| 移动、归档、恢复 | PUT /v1/admin/articles/:id/move、archive、restore |
| 文章完整组排序 | PUT /v1/admin/articles/reorder |
| 目录完整组排序 | POST /v1/admin/catalog/reorder |
| 兼容入口 | 旧文章创建、编辑、发布、下架及目录管理 |
| 公开读取 | GET /v1/blog/modules、moduleDetail、articleDetail |

响应仍为 code、msg、payload。参数 400，不存在或不可见 404，冲突 409，尚未切换 503 ContentRegistrationUnavailable。数据库查询失败返回错误，不能当作空目录。

## 配置

- MINIBLOG_NOTION_TOKEN：内部连接凭据，仅注入后端。固定官方页面接口，版本 2026-03-11，总超时 5 秒，不自动重试；从 title 属性取标题。未配置、无权限、404、限流、超时返回可手填结果。不要放入前端或仓库。
- MINIBLOG_CONTENT_REGISTER_ENABLED：默认关闭。开启仍需唯一索引存在及历史身份回填；兼容创建和来源变更遵循同一门槛。预览、读取及纯信息维护独立可用。
- VITE_CONTENT_REGISTER_ENABLED：后台构建开关，开发测试开启、生产默认关闭；完成切换后显式开启并重新构建。关闭时隐藏工作台，保留兼容页面。
- 后台 Docker 构建通过 --build-arg VITE_CONTENT_REGISTER_ENABLED=true 开启，通过 --build-arg VITE_API_ROOT 指定 API；默认关闭。凭据只作为后端运行环境，不作为构建参数。
- 两个前端 API 配置见各自 .env.example，开发默认本地 8080。

标题取得成功不代表原文可匿名阅读或嵌入。前台始终提供打开原文，iframe 等待 10 秒提示可在新窗口阅读，不断言文档失效；外链隐藏无法准确计算的阅读进度。

## 分阶段切换

先在备份的本地或预发布库演练。脚本需明确连接参数，禁止使用默认参数直接操作生产。

1. 备份并记录迁移版本，先扩展到版本 4。历史库不要直接全部 up：

   ```sh
   DB_MIGRATE_TO=4 scripts/db-migrate.sh
   ```

   用 DB_HOST、DB_PORT、DB_USER、DB_PASSWORD、DB_NAME 或已有 MYSQL_* 参数。000004 扩展 TEXT 外链、来源字段、模块排序与普通索引，不重写历史文章内容。

2. 默认只读审计，MYSQL_DSN 通过本地安全环境提供：

   ```sh
   go run ./scripts/audit-content -report /tmp/miniblog-content-audit.json
   ```

   退出码 0 为无阻断，2 为关系、状态、外链、已有身份冲突或重复。报告只有 ID、摘要与原因，不含原外链或凭据。重复只生成清单，由维护者处理；工具不合并、删除或改状态。相同 pos 仅提示，仍按 pos,id 读取，随后可完整重排。

3. 停止全部旧写进程、后台写操作和导入任务，保留读取。处理阻断并重审，再回填：

   ```sh
   go run ./scripts/audit-content -apply -report /tmp/miniblog-content-backfill.json
   ```

   有阻断不回填。单事务复核快照，保留 ID、原外链、正文、状态、位置、历史时间；纯正文来源仍为空。还有旧写进程时不得判定切换完成。

4. 确认全部写入口升级为本分支统一用例，再执行唯一约束：

   ```sh
   DB_MIGRATE_TO=5 scripts/db-migrate.sh
   ```

   000005 再查来源缺失和重复，失败阻止约束生效，新入口保持关闭。migrate 若记录 dirty，维护者先核对 schema、日志和版本 4，只有确认失败在 ALTER 前，才修复迁移标记至 4 后重试；不要自动 force 或绕过审计。

5. 开启后端 MINIBLOG_CONTENT_REGISTER_ENABLED=true；后台 VITE_CONTENT_REGISTER_ENABLED=true 后构建。用测试文档验证新增、重复、连续发布、刷新和移动，再做真实使用验收。Notion 页面须授予内部连接读取能力，原文分享权限另行设置。

## 回退及导入

先关闭前后台收录开关、停止新写。兼容版本须认识新增字段和统一状态，保留新增数据、身份及唯一约束。不要恢复 40673c8 原始写程序，它不遵循新规则。

000004/000005 的 down 刻意保留字段和约束，仅回退迁移标记，重新 up 可重复执行。本地已演练保留数据的 down/up，不代表生产备份恢复或 schema 已核验。

批量导入保留 JSON、content_file 与 dry-run，转接统一事务。ID 支持十进制字符串或 JSON 整数，显式 ID 保留并用于编辑；标题不作身份，无 ID 的重复外链返回已有记录，不覆盖元数据。content 缺省保留正文、显式空字符串清空；status 缺省新建草稿、更新保留现态。归档先独立恢复；作者、标签可空，历史正文仍支持。

```sh
scripts/batch-upsert-articles.sh -file articles.json -dry-run
scripts/batch-upsert-articles.sh -file articles.json
```

dry-run 校验归属、身份、冲突和状态，不写入、不预留 ID 或位置。

## 验证

本轮使用独立 Node 24，未改默认 Node 16。content-checks.yml 增补 PR 的 Go/race/vet、MySQL、类型、非修复 lint、单元及构建门槛，远端 CI 尚未运行。

本轮按确认计划升级 Vitest 5.0.3 与 Vite 6.4.0，构建和 CI 使用 Node 24。原 mocker/tinypool 公告已退出锁文件命中。Vite 6.4.0 仍命中后续公告，按明确版本约定保留并单独记录；其余结果见 [同步验证记录](notion-sync-verification.md)。工具通过不表示依赖审计无告警。

```sh
go test -race ./...
go vet ./...
# 仅本地专用空库；未配置则跳过 MySQL 集成
MINIBLOG_TEST_MYSQL_DSN='root@tcp(127.0.0.1:13316)/miniblog_refactor_test_local?parseTime=true&multiStatements=true' go test -race ./scripts/content-integration
# 两个前端分别执行，使用独立 Node 24
npm ci
npm run type-check
npm run lint:check
npm run test:unit
npm run build
```

MySQL 8.0.36 覆盖迁移、回填、唯一约束、历史时间、down/up、跨模块并发重复收录、位置、移动、状态、隐藏祖先、真实 SQL 失败回滚、105 篇目录、分页及完整排序。测试拒绝远端或非 miniblog_refactor_test_* 数据库。

2026-10-08：Go 全量 race 与 vet 通过；6 项真实 MySQL/Gin 集成通过；后台 39 项、阅读站点 41 项单元/组件测试通过，类型、非修复 lint 与构建通过。实际模块摘要还补查了历史 id=0 与新接口真实 ID 的兼容。

Chrome 本地 API 夹具验证连续收录、预览竞态、失败输入、重复处理、历史链接、前进后退、小屏目录和 iframe 出口。夹具不证明真实 Notion/飞书分享、跨站嵌入、Safari 或真机行为；实际平台、使用验收和生产部署仍待完成。
