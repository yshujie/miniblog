# Notion 同步命令行

数据库迁移由部署流程完成，本工具不会自动建表。默认模式 `dry_run` 读取固定五个 Notion 数据源并写本地审计清单；首次完整扫描将所有历史 page ID 与基线标记原子保存。它不会改文章、目录或 Notion。

- `--mode status`：查看本地运行状态，不请求 Notion。
- `--mode dry_run`：重新读取官方 API，生成审计运行。
- `--mode sync --enable-sync`：重新读取并应用；要求基线已冻结、源已配置并启用、暂停已解除。
- `--mode bootstrap_preview`：完整读取后生成历史匹配清单。自动提出的匹配依据为可信 page ID，或本轮读取已证明属于该页的精确公开地址；未知 slug 不猜，标题仅作线索。`requires_legacy_alias:true` 的候选须额外确认 `allow_legacy_alias:true`。清单包含公开 URL 条件、目标主题章节与子章节保留/转为章节直属变化；歧义、未匹配项继续待确认。`--manual-matches /path/matches.json` 可提交已人工核对的历史别名关联（格式 `{"manual_matches":[{"page_id":"…","article_id":"…"}]}`），生成新的审核指纹。
- `--mode bootstrap_apply --confirmations /path/review.json --allow-notion-write`：仅处理逐项核对的清单。要求暂停同步与外部来源写入，并提供独立 `MINIBLOG_NOTION_BOOTSTRAP_TOKEN`。读取仍使用 `MINIBLOG_NOTION_TOKEN`。本模式不修改 Notion 正文。

确认文件格式：`{"confirmations":[{"page_id":"…","article_id":"…","expected_state":"published","expected_fingerprint":"预览返回的指纹","confirmed_by":"操作人"}]}`。四态为 `draft`、`published`、`unpublished`、`archived`。已有文章回填值必须与核对时的本地状态一致；非标准历史别名须额外 `allow_legacy_alias:true`。明确未匹配的新页面使用 `new_page:true` 并省略 `article_id`，同样要求明确四态、审核指纹与确认人，绝不从空状态推导发布。接管后先保存来源绑定，草稿/下架/归档不创建博客文章；后续重新读取并满足已发布、公开 URL 和有效目录等条件时才创建。没有核对项不写入、不接管。写请求超时后先回读；结果未知保留 journal，下一次只在全部核验一致时继续。

审计报告完整遍历所有运行明细分页；失败或部分失败返回非零退出码，并保留逐项结果。回填前验证最新 schema、页面、来源身份冲突、配置版本和本地完整快照；单次写后必须回读，结果未知保留 journal。配置编辑在回填运行期间暂不可用。

通过环境配置数据库连接和 token，不把凭据放进参数、报告或日志。`--report` 输出文件权限为 0600。暂停或关闭定时同步只冻结当前结果，不自动逆向修改 Notion；回退保留绑定、历史来源 alias 和运行记录。

测试使用内存数据库、模拟 REST 传输及模拟客户端，没有真实 Notion 账号或生产访问。当前尚未配置服务端 token：真实官方 REST 读取权限、公开 URL、原生归档双分区和生产切换都未验收。
