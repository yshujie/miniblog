# Notion 同步本机浏览器夹具

此夹具验证真实 Chrome 中的前端交互，所有 API 与 iframe 正文均由脚本截获。只允许 localhost / 127.0.0.1 页面，其他网络请求被阻断；不使用真实账户、数据库、Notion token 或生产 API。它不能证明真实平台权限、公开链接或嵌入阅读可用，也不替代真实 iOS / Android / Safari 验收。

## 复现

使用独立 Node 24，在两个 web 目录各执行一次 `npm ci`。在仓库根目录分别启动两个本机页面：

```sh
VITE_API_ROOT=http://127.0.0.1:8099/v1 VITE_CONTENT_REGISTER_ENABLED=true \
  npm --prefix web/miniblog-web-admin run dev:test -- --host 127.0.0.1 --port 5189 --strictPort
```

```sh
VITE_API_BASE_URL=http://127.0.0.1:8099/v1 \
  npm --prefix web/miniblog-web run dev -- --host 127.0.0.1 --port 5188 --strictPort
```

8099 无须后端服务；夹具会截获所有 `/v1` 与 `/api/v1` 请求。阅读 iframe 演示也使用同一个本机地址，不导航真实外链。

运行前需可用的 Playwright 与 Chrome / Chromium。在管理端目录执行：

```sh
MINIBLOG_PLAYWRIGHT_MODULE=/absolute/path/to/playwright \
MINIBLOG_CHROMIUM_EXECUTABLE=/absolute/path/to/chrome \
  npm run test:browser:sync
```

默认页面地址为 `http://127.0.0.1:5189`（后台）与 `http://127.0.0.1:5188`（阅读）。需要其他本机端口时设置 `MINIBLOG_ADMIN_URL` / `MINIBLOG_READER_URL`。输出目录默认 `/private/tmp/miniblog-sync-browser`，可通过 `MINIBLOG_FIXTURE_OUTPUT` 指定。

脚本位于 [sync-browser-fixture.mjs](../scripts/sync-browser-fixture.mjs)。它包含假登录态，仅用于截获的本地测试数据；冷启动直达同步页面会完整执行路由权限加载。

## 验证范围

- 同步健康与待基线审核、一次预览创建及运行明细、两个没有文章的主题计划。
- 来源配置版本冲突保留输入，主题冲突显式绑定到正常章节。
- 超过 JavaScript 安全整数范围的文章 ID 保持字符串；托管文章标题只读，作者仅 PATCH，保存失败保留作者输入。
- 旧模块已不存在时，历史文章 URL 仍按当前模块替换路径，并保留 query / hash。
- `reading_url` 用于 iframe 与打开原文；小屏阅读每 60 秒刷新标题时保留相同 iframe。
- 紧急下架在下次刷新移除正文；解除后仍不可读，重新同步核验成功后恢复。
- 390 px 管理页布局无页面横向溢出；无页面脚本错误或外部网络请求。

阅读时间由浏览器时钟推进。目录与文章数据、同步结果、核验成功均来自本地替身，不是实际后端或 Notion 状态。

旧快速收录夹具另运行 `npm run test:browser:fixture`，默认后台端口为 8001，可用 `MINIBLOG_ADMIN_URL=http://127.0.0.1:5189` 覆盖。它检查连续收录、迟到标题保留手改、空作者、失败输入保留、不确定结果原样重试及显式恢复重复归档文章。可选 production mode 收录开关复现步骤见 [README](../README.md)。

## 本轮证据

2026-10-08：Node 24；阅读端 46 项、管理端 57 项单元/组件测试通过，两个项目的类型检查、静态检查、构建通过。Chrome 运行证据随本说明保存在 [fixtures/notion-sync/2026-10-08](fixtures/notion-sync/2026-10-08/report.json)，对应 12 项同步场景与 7 项收录及开关场景。收录报告为 [registration-report.json](fixtures/notion-sync/2026-10-08/registration-report.json)。截图只包含本地虚构数据。

- [同步概览](fixtures/notion-sync/2026-10-08/sync-overview.png)
- [预览逐项结果](fixtures/notion-sync/2026-10-08/sync-run-items.png)
- [托管文章作者编辑](fixtures/notion-sync/2026-10-08/managed-article.png)
- [小屏阅读刷新](fixtures/notion-sync/2026-10-08/reader-refreshed-mobile.png)
- [紧急下架](fixtures/notion-sync/2026-10-08/reader-held-mobile.png)
- [小屏管理页](fixtures/notion-sync/2026-10-08/sync-small-screen.png)
- [production mode 收录关闭](fixtures/notion-sync/2026-10-08/registration-gate-disabled.png)

依赖已按批准版本使用 Vitest 5.0.3、Vite 6.4.0；两个锁文件审计均无 critical。Vite 6.4.0 仍有已知开发服务公告，当前保持用户指定版本；可单独批准升级至修复版本后重新验证。审计结果不等于生产安全验收，CI、真实后端联调、真实平台和发布由各自验收记录说明。
