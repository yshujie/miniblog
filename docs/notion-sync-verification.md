# Notion 同步实施与本地验收

以下至“真实接入与生产启用阶段”为上一阶段的独立验收记录，不代表本轮真实连接或生产启用已完成。

日期：2026-10-08。实施基线 `f8e5bf3`，分支 `codex/miniblog-content-refactor`，独立工作树保留。本文区分代码和本地验证、真实连接、生产与实际使用验收。

## 交付范围

已实现六表迁移、JSON 标签及旧 CSV 兼容、官方只读客户端、五库完整分页扫描、字段/状态 ID 映射、主题绑定、租约与版本 fencing、原子四态投影、接管审核工具、来源别名、托管写保护、本地作者和紧急下架。同步默认关闭，后台暂停独立控制；运行期不反写 Notion。

后台 `/content/sync` 展示来源、健康、问题、预览与运行明细。旧工作台和旧路由保留。阅读端使用有效阅读地址、六十秒核验和请求版本保护，保留历史 URL 与小屏目录。接口契约见 [OpenAPI](../api/openapi/openapi.yaml)，运行及切换见 [维护指南](notion-sync.md)。

## 代码与本地数据库

独立 MySQL 8.0.36，测试库 `miniblog_refactor_test_notion`；未使用用户本机数据库或生产数据。完成验收后，本轮测试容器及其匿名卷已删除，用户数据库和镜像保留。

10 项 MySQL 集成测试通过（含 race），覆盖：

- 000004、000005、000006 增量迁移、数据审计、唯一约束与保留数据的回退。
- 并发收录、同组位置、重排、旧 code、分页和大 ID。
- 租约并发获取、数据库时间续租、失效 fencing、旧任务释放不覆盖新租约。
- 四态、重复扫描无重排和无意义更新时间变化、本地下架及解除后重新核验。
- 真正 SQL 失败后的文章、目录、绑定事务回滚；长标签和含逗号标签无损保存。
- 跨模块移动后历史 ID 链接仍能定位；确认来源别名命中原记录；不含 Page ID 的已知 Notion Site 公开短链接重复收录返回原文章，不能绕开紧急下架。
- 历史接管保留标题、正文、作者、原始链接、子章节、排序和创建/更新时间；维护窗口阻止旧来源写入。

默认 `loc=Local` 连接的租约、四态及历史接管三个重点场景亦通过，未修改现有连接时区。

MySQL 验证发现 `ON UPDATE CURRENT_TIMESTAMP` 会在身份升级时改变历史更新时间，已显式保留旧值并回归通过。单元与接口测试还覆盖全部十六种状态迁移、字段错误与网络故障的不同处理、归档作者约束、旧 PUT 冲突的当前资料、分页参数、严格输入、并发/陈旧版本和同步生命周期。

最终全仓 `go test -race ./...`（设置独立测试 MySQL DSN）、`go vet ./...`、`git diff --check` 均通过。验证包含接管配置变化后拒绝旧确认、重新预览并明确确认后的恢复。OpenAPI YAML 的 181 个内部引用和新增文档相对链接核验通过。远程 CI 配置已更新，远程工作流尚未运行；CI 使用独立 MySQL 与 Node 24。示例环境的生产 Compose 合并配置通过静态校验，未启动服务或部署。

## 前端与真实 Chrome 本地夹具

独立 Node 24；本机默认 Node 16 未修改。阅读端 46 项、管理端 57 项测试通过，两端类型检查、非修复 lint 和构建通过。

真实 Chrome 使用全部本机替身 API 和 iframe 内容：12 项同步场景、7 项收录/开关场景通过，外部请求与页面脚本错误均为 0。包含冷启动刷新、连续发布、失败输入保留、大 ID、托管作者、紧急下架/解除、历史跨模块 URL、query/hash、小屏目录及 iframe 保持。390px 是 Chrome 视口测试，不能代替实际移动设备。

[复现说明、报告和七张截图](../web/miniblog-web-admin/docs/notion-sync-fixtures.md) 保留在仓库；临时前端服务已关闭。该证据验证前端交互，未宣称真实后端、Notion 或飞书 iframe 已联调。

## 依赖公告

按用户确认锁定 Vitest 5.0.3、Vite 6.4.0，独立提交并重跑两端验证。既有 form-data、shell-quote、hasown 的受影响间接锁定版本已在原范围内修复；两端审计 critical 为 0，没有使用强制修复。

批准的 Vite 6.4.0 仍命中开发服务公告：[WebSocket 文件读取](https://github.com/vitejs/vite/security/advisories/GHSA-p9ff-h696-f583)、[Windows 路径绕过](https://github.com/vitejs/vite/security/advisories/GHSA-fx2h-pf6j-xcff)。公告各自的修复版本为 6.4.2 和 6.4.3，当前未擅自更换批准版本。本轮浏览器仅监听本机回环地址；没有运行漏洞利用。审计结果不是生产安全验收。

## Notion 只读复核与切换门槛

已通过现有连接工具只读核验五个子库：标题、主题、知识点及四态字段仍在；项目开发存在 IAM、Qlume，数据结构&算法存在数据结构、算法主题。

本次连接工具的非归档 SQL 计数为 Go 38、DDD 6、数据库 2、数据结构&算法 0、项目开发 1，共 47，博客状态均为空。此计数不代替官方 REST 的双归档分区扫描、`request_status`、原文 `public_url`、生产数据匹配或数据库审计。

用户明确选择“先完成代码与本地验收，真实连接切换时配置”。本机没有运行只读 `MINIBLOG_NOTION_TOKEN` 或一次性写入 `MINIBLOG_NOTION_BOOTSTRAP_TOKEN`，因此没有进行真实 API 同步、状态回填或接管。

实际切换尚需：

1. 核验并备份生产 schema、数据和部署版本，完成身份/目录冲突清单。
2. 注入服务端只读连接并授权五库；配置既有模块 code，运行完整 dry-run 固化基线。
3. 审核历史匹配和未匹配项；仅在维护窗口以独立写入凭据完成逐页回填、回读和本地接管。
4. 测试公开页新增、下架、归档、恢复及撤销公开；浏览器验证匿名访问与嵌入，完成实际手机和使用验收。
5. 先 Go 后逐库启用。回退先暂停、撤销租约、关闭总开关，保留字段、绑定、别名和数据。

生产部署、真实 Notion/飞书公开阅读、iOS/Android/Safari 和业务验收未执行，均由独立切换验收记录确认。


## 2026-10-08 真实接入与生产启用阶段

本阶段基线 `main 2f4c406`、clean schema 6，分支 `codex/miniblog-notion-sync-rollout`，复用隔离工作树。新增 schema_check（五库 schema-only、不连数据库）与 catalog_prepare（双暂停、冻结基线、租约和配置版本），生产镜像包含安全操作工具；不增加数据库迁移。

修复相同配置重复保存导致重新核验、同章多主题绑定、跨库无效目标隔离与分库应用成功时间；历史候选留存运行明细，后台只读核对，真实可见性与故障/限制分别呈现。

本阶段验收层次逐项记录，未执行的层次不标记通过：

| 层次 | 当前记录 |
| --- | --- |
| 代码回归 | 先复现配置 no-op、历史清单未留存、公开状态缺字段、Go 试运行时间错误，再修复；全部回归（含 race）、vet 和差异检查通过；新增 journal 冻结、显式重新审核与 CLI 部分失败门槛通过组合复核，非空未知状态拒绝覆盖 |
| 真实 MySQL | 本任务独立 MySQL 8.0.36，回环端口 63368、测试库 miniblog_refactor_test_rollout；不使用用户或生产数据，全部 13 项集成回归通过（含 race） |
| 前端 | 独立 Node 24，锁定依赖保持；类型/非修复 lint/构建通过；管理端 18 文件/73 测试，阅读端 9 文件/46 测试通过。原生 Chrome 本地替身验证状态展示、只读历史对照、大 ID、配置失败输入保留、作者失败重试、下架不可阅读、解除后待核验；不代替真实 Notion 或手机 |
| CI / 兼容生产发布 | PR #3/#4 的 CI 与完整质量门槛通过；main 735344f 的部署 37781019063 成功。三服务实际 SHA 一致、后端 healthy，schema 6 clean 与唯一约束通过；历史文章和目录字段比对保持。运行只读 Token 已注入，bootstrap Token 未注入，同步 false、control paused、五来源 disabled |
| 真实 Notion REST | schema_check 37781133297 成功：五库 complete=true，官方实际 data source ID 与固定清单一致，标题/主题/知识点类型和四态 ID 有效。运行连接身份为 miniblog_reader。尚未完整扫描页面、冻结基线、回填或接管 |
| 生产定时 / 五库试运行 | 未启用；等待实际历史审核、分库接管与计时运行 |
| 匿名阅读 / 实际手机 / 使用确认 | 待真实公开验收页及用户试读，不用浏览器视口代替手机 |

生产先备份核验，再部署关闭同步的兼容版本。真实凭据就绪后全五库只读联验和基线；每库目录准备后重做最终历史预览，经审核再回填/接管并立即手动同步。计时、运行 ID、生产版本、来源与目录映射及待处理清单在实际执行后记录。

本阶段首次生产只读预检时版本为 2f4c406，content-preflight 通过 clean schema >=6 检查；私有目录 /opt/miniblog/backups/notion-rollout-20261008 保存 pre-notion-rollout-schema6.sql.gz（0600，gzip 校验通过）、legacy-fields-before.json 与 content-audit-before.json。身份审计无阻断项；未改目录、文章、Notion 或同步开关。这是连接配置前的记录；随后专用凭据由 GitHub Actions Secrets 提供，真实 REST schema-only 核验已证明五库可读。此证据不代替完整页面扫描或匿名阅读。


### 专用凭据接入与受控初始化

- [PR #3](https://github.com/yshujie/miniblog/pull/3) 将运行只读 Secret 通过受限临时目录写入服务器私有环境；[PR #4](https://github.com/yshujie/miniblog/pull/4) 修复 SSH Key 换行兼容并增强任务目录所有权及清理守卫。34 项 Python 操作工具测试通过。首次 SSH 传输失败发生在服务替换和 Notion 请求之前；修复后验证成功，失败及成功任务的凭据暂存均已清理。
- [兼容部署 37781019063](https://github.com/yshujie/miniblog/actions/runs/37781019063) 实际生产版本为 `735344f086302454590a8d3e0033285ea99e9423`；运行 Token 存在，回填 Token 不存在。私有环境 0600，生产身份审计无阻断项。公开接口健康验证通过；不代表真实文章阅读或实际手机验收。
- [真实 schema_check 37781133297](https://github.com/yshujie/miniblog/actions/runs/37781133297) 使用当时运行的兼容镜像；其 Go/CLI 实现与 735344f 相同。完整报告留存服务器 `/opt/miniblog/ops/notion/37781133297-1/schema.json`（0600），未上传 GitHub Artifact，未读取回填 Secret。
- [一次性初始化工具](../scripts/notion-rollout-init/README.md) 默认仅读预检，显式执行时在一个事务中保存固定五库映射、改名 database/project、新建停用 algorithm，保留历史目录状态 0、ID/code/排序及全部文章字段。不访问 Notion、不建立章节、不冻结基线、不启用来源或同步。实际生产初始化尚未执行。
- 初始化工具本地独立 MySQL 8.0.36 测试通过（专用回环端口 63369、数据库 `miniblog_rollout_init_test`），含 race、第五库 SQL 失败完整回滚、真实旧状态/正文/作者/位置/时间保持、公开 ID 集合不变、重复执行 no-op、MyISAM 拒绝。12 项顶层测试与 vet 通过；真实五库 schema 报告离线复验通过。CI 为该夹具单独建库，远程结果随后记录。

接下来先审核初始化预检并执行固定配置，再完整 dry_run 冻结本次历史基线，生成历史匹配及公开条件清单。运行连接的页面读取、公开地址、接管回填、定时试运行、匿名阅读和实际手机验收仍分别待执行；当前不能把 schema-only 成功标记为五库自动同步已上线。
