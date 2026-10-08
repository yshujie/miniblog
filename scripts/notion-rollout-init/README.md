# 固定五库的一次性初始化

此工具只保存已批准的模块名称和 Notion 字段映射。默认预检只读取数据库、生成私有计划；只有 `--apply` 才写数据库。它不访问 Notion，不读取正文，不创建文章或章节，不启用模块、来源或同步，也不执行迁移、历史状态回填或接管。

固定对应关系无法从参数或文件改变：

| Data source ID | 模块 code | 名称 |
| --- | --- | --- |
| `2bf330bd-ddf1-80a6-aa49-000bbd1e154b` | `go` | Go |
| `d579f619-4f1c-4e7e-9593-ab6529fc63d0` | `ddd` | DDD |
| `cfa962f3-12be-40aa-95eb-9122186bd4ba` | `database` | 数据库 |
| `03d24108-5172-4e8f-8cfd-e3bd52e437af` | `algorithm` | 数据结构&算法 |
| `d13097e3-b78a-4b6e-8848-979749d4b2f3` | `project` | 项目开发 |

`go/ddd/database/project` 必须已存在。仅更新 database/project 的标题，保留 ID、code、状态（含历史停用值 0）、sort 和创建时间。algorithm 不存在时以状态 2 在当前最大 sort 后创建；已存在时必须名称一致、状态 2、没有章节，否则拒绝并等待审核。不会将既有 algorithm 下架或改为其他用途。

## 输入与预检

先用当前审核版本的 `scripts/notion-sync --mode schema_check` 取得本轮完整报告。报告必须 `complete=true`、版本 `2026-03-11`、恰好五个不同白名单 ID，每库 `status=valid` 且实际返回 ID 与请求 ID 一致。工具还原报告中的字段和选项，通过现有 CheckSchemas 重新验证，并要求重新发现的配置与 suggested_config 完全一致；属性 ID 保留原始百分号编码，额外博客状态选项允许存在。

这个离线检查能证明报告内部一致，无法鉴别一个完全伪造的报告，也无法证明读取后 Notion 没有发生变更。操作人必须审核本轮真实读取结果、保留原报告及 SHA256，字段配置完成后再按正常流程完整 dry_run 核对实际数据。失败或不完整的 schema_check 报告不能用于初始化。

另准备五库审核版本文件。当前初始版本为 0 时示例如下；必须以实际只读查询结果为准：

```json
{
  "expected_config_revisions": {
    "2bf330bd-ddf1-80a6-aa49-000bbd1e154b": 0,
    "d579f619-4f1c-4e7e-9593-ab6529fc63d0": 0,
    "cfa962f3-12be-40aa-95eb-9122186bd4ba": 0,
    "03d24108-5172-4e8f-8cfd-e3bd52e437af": 0,
    "d13097e3-b78a-4b6e-8848-979749d4b2f3": 0
  }
}
```

两个输入必须为操作人所有的私有常规文件（例如权限 0600），禁止符号链接；拒绝未知 JSON 字段、重复键、尾随 JSON 和超出 4 MiB 的输入。只支持固定配置初始化，不接受任意 module code、URL、Notion 写模式或命令行密码。

数据库凭据仅从环境读取：`MYSQL_DSN`，或者明确设置的 `MYSQL_HOST/PORT/DATABASE/USERNAME/PASSWORD`。也兼容服务的 `MINIBLOG_DATABASE_HOST/PORT/DBNAME/USERNAME/PASSWORD`；两组并存而值不同会拒绝。密码可显式为空，工具不提供默认账号或密码。时间按 UTC 解析，数据库日志关闭。不要把凭据写入命令参数、报告或开启 shell 跟踪。

运行容器的 `MINIBLOG_NOTION_SYNC_ENABLED` 必须已明确为 `false` 或 `0`。工具同样要求自己的环境值明确关闭，但这不能替代对正在运行的服务配置的核验。操作窗口内使用现有部署维护锁，避免同时进行目录管理或部署；工具不自动修改服务环境、暂停开关或租约。

```sh
go build -o /private/tmp/notion-rollout-init ./scripts/notion-rollout-init

/private/tmp/notion-rollout-init \
  --schema-report /private/path/schema.json \
  --expected-revisions /private/path/revisions.json \
  --report /private/path/init-preflight.json
```

预检不修改数据库，输出仅包含模式、结果、库数和输入报告摘要。完整审核计划在原子写入的 0600 文件中，包含固定模块变更、五库字段映射与前后版本、现有文章/目录数量和公开文章 ID 集合的 SHA256；不含正文、外链、数据库地址、凭据或完整来源数据库行。`source_record_count` 是实际读到的来源行数，每条计划的 `before_exists` 区分已有行与缺失行；管理接口合成的五库 DTO 不代表已存在数据库行。预检与执行输入报告可以使用同一个文件，但结果报告不能覆盖输入。

## 显式执行与门槛

仅在审核预检计划后，使用相同 schema 报告及版本文件执行：

```sh
/private/tmp/notion-rollout-init \
  --schema-report /private/path/schema.json \
  --expected-revisions /private/path/revisions.json \
  --report /private/path/init-apply.json \
  --apply
```

执行使用唯一外层 READ COMMITTED 事务，先锁 control，再按主键顺序锁既有模块，最后锁来源。锁内重新核验：

- control 已暂停，基线未冻结，current_run_id 为空；按数据库 UTC 时间没有有效租约。
- 数据库可已有 0–5 个固定来源；拒绝未知、重复或 source_id/data_source_id 不一致的来源。已有行全部 disabled，module_code 只能为空或各自固定 code。缺失来源仅当审核版本为 0 才计划创建；输入 schema 和版本文件仍须包含全部五库。
- 每库 config_revision 与输入一致；managed page binding 数量为 0。
- 目录及同步表已存在，module.sort/article.tags_json 扩展已完成；MySQL 十张相关表均须为 InnoDB 常规表。不接受目录 NULL 状态或 NULL 排序。

模块改名复用事务绑定的 catalog.UpdateModule，五库配置复用事务绑定的 Service.UpdateSource；缺失来源不在预检中 seed，显式执行时通过该业务方法按默认初始模型创建，并再次检查版本 0，成功后的版本为 1。子事务使用保存点，任何一步失败都返回外层回滚。已有来源无当前字段变化时不会改来源版本；初始化后各源仍 disabled。工具不会修改 control/source_writes_paused、冻结基线、抢占租约或建立主题绑定。

提交前比较文章、章节、子章节、页面绑定、主题绑定数量，以及使用正式公开过滤规则算出的公开文章 ID 集合摘要；任何变化都回滚。原模块 ID/code/status/sort/created_at 也必须保持。database/project 的 updated_at 因真实改名正常更新；原文章及正文时间不会改变。

返回 0 表示预检通过或事务提交成功。参数/报告/环境无效返回 2，数据库门槛或操作失败返回 1，错误只使用固定类别。旧版本文件在初始化后会因 CAS 失败而拒绝再次执行：重新读取版本，再预检；配置相同且当前版本正确时复跑为 no-op。如果输出 `apply committed; report_write_failed`，数据库已提交，应先只读核验，不能把它当作回滚或盲目重试。

之后的完整扫描、目录准备、历史状态确认和启用仍走已审核的上线流程，初始化工具不会代替这些步骤。

## 本地验证

```sh
go test -race ./scripts/notion-rollout-init
go vet ./scripts/notion-rollout-init
```

回归覆盖完整报告与实际字段不一致、报告五库遗漏/重复、opaque ID、四态外的选项兼容、来源行全部缺失/部分缺失/已有五行、缺失行非零审核版本拒绝、默认预检零写入、初始化门槛拒绝、第五库 SQL 失败全部回滚、历史目录状态 0/文章正文/作者/位置/时间保持、精确重复 no-op、私有报告以及 MySQL 驱动独立日志脱敏。设置 `MINIBLOG_ROLLOUT_SCHEMA_TEST_REPORT` 可对私有真实 schema 报告做离线校验，不会调用 Notion。

真实 MySQL 回归必须显式注入 `MINIBLOG_ROLLOUT_INIT_TEST_DSN`，只接受 loopback TCP 且数据库名严格为 `miniblog_rollout_init_test`。**测试会重建该专用库的十张夹具表，不可指向业务库或共享测试库。**测试文件读取环境，不从参数接收密码，也不打印 DSN：

```sh
# 由本地私有测试环境安全注入变量后运行；不要在命令中展开连接串。
go test -race ./scripts/notion-rollout-init -run TestMySQL -count=1
```

MySQL 测试分别使用 UPDATE 和 INSERT 失败触发器确认第五次来源写入已实际发生，再检查目录改名、新算法模块及前四库来源创建/配置均回滚；随后验证成功提交、公开集合不变、历史状态与文章时间保持、重复执行不变。没有该测试变量时，此回归明确跳过。
