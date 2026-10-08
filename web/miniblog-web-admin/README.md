# Miniblog 内容管理后台

基于现有 Vue 3、Element Plus、Pinia 和 Vue Router。文章正文在 Notion/飞书维护，后台管理来源链接、目录、标题、作者、标签和发布状态。

`/content/workbench` 提供目录树、当前目录文章、文章/目录上下排序及快速收录。旧模块、章节、子章节和文章路由保留；章节和子章节列表每行可直接在对应目录收录。同一来源仅有一条文章，重复收录只显示已有记录，移动/恢复需显式操作。

快速收录优先当前有效目录，其次最近有效目录；作者默认为昵称且可留空，标签也可留空；连续收录保留目录和作者。Notion 标题建议允许手改，迟到结果不会覆盖手改标题。飞书或预览失败可直接手填。失败保留输入，结果不确定时锁定原请求供原样重试。已发布文章保存资料保留发布状态，状态操作前先保存未保存的修改；状态成功但随后读取失败时可单独重新读取。

## 本地开发与开关

使用 Node 24，在本目录运行：

```sh
npm ci
npm run dev:test
```

本机后台页面默认 `http://localhost:8001/`，开发/测试 API 默认 `http://localhost:8080/v1`；通过 `VITE_API_ROOT` 可覆盖。不会自动加载旧模板 mock 或外部测试代理。

`VITE_CONTENT_REGISTER_ENABLED` 本地默认开启。生产未显式开启时禁用收录入口；工作台与同步管理菜单仍保留，已有文章仍可管理。`.env.build_prod` 保留原生产 API 地址及关闭的收录开关；生产开启前须完成后端来源回填、唯一索引和 `MINIBLOG_CONTENT_REGISTER_ENABLED=true` 切换。后端返回 `ContentRegistrationUnavailable` 时，页面说明切换尚未完成并保留输入。

```sh
npm run type-check
npm run lint:check
npm run test:unit
npm run build:test
npm run build
```

`lint:check` 仅覆盖本轮内容改造文件，避免格式化遗留模板。单元测试使用独立 Vitest 配置、jsdom 和本机 API 地址；store 自动注册明确排除测试文件。

## 真实浏览器 fixture 验收

`npm run test:browser:fixture` 使用 Playwright 驱动隔离的 Chrome/Chromium。全部 API 请求均被截获为测试数据，非本机页面网络被阻断，不使用真实账户或数据库。

需要已安装的 Playwright 和 Chrome/Chromium。可用 `MINIBLOG_PLAYWRIGHT_MODULE` 指向工作区工具包内的 Playwright；`MINIBLOG_CHROMIUM_EXECUTABLE` 可指定浏览器路径。页面需事先运行，默认检查 `http://127.0.0.1:8001`，可用 `MINIBLOG_ADMIN_URL` 覆盖。

生产开关验收另构建本机 API、关闭登记的 production mode 输出：

```sh
VITE_API_ROOT=http://localhost:8080/v1 VITE_CONTENT_REGISTER_ENABLED=false \
  npx vite build --mode build_prod --outDir /tmp/miniblog-admin-gate
npx vite preview --host 127.0.0.1 --port 8003 --strictPort --outDir /tmp/miniblog-admin-gate
```

在另一个终端运行：

```sh
MINIBLOG_ADMIN_GATE_URL=http://127.0.0.1:8003 npm run test:browser:fixture
```

2026-10-08 本地验证：57 个单元/组件用例通过；真实 Chrome fixture 通过以下场景：目录/昵称默认及连续发布、Notion 迟到建议保留手改标题、空作者发布、503 保留输入、不确定结果原请求重试、重复归档文章显式恢复、production mode 收录开关禁用（工作台菜单保留）。它证明本地 UI 与截获协议行为；实际 API/数据库联调、真实 Notion 接入和生产发布是另外的验收边界。

运行证据输出至 `test-results/browser-fixture/`（已忽略）：

- `report.json`：场景与请求统计。
- `01-publish-and-continue.png`：继续收录保留目录/作者。
- `02-late-title-keeps-manual.png`：手改标题保留。
- `03-failure-retains-input.png`：失败输入保留。
- `04-duplicate-explicit-restore.png`：重复文档恢复草稿。
- `05-production-gate-disabled.png`：生产开关未开启。

## Notion 同步与托管文章

`/content/sync` 显示同步健康、来源配置、主题绑定、页面问题、预览与同步运行及逐项变更。页面每 15 秒刷新可见页面的健康与列表；打开运行明细后，每 2 秒读取尚未结束的运行。分页默认每页 20 项。运行请求取得 ID 后只重试读取；响应不确定时先检查运行历史，不自动重发。

服务总开关由后端配置控制。预览不写入文章或目录；手动同步重新扫描当前来源，不重放旧预览。历史文章的状态审核与回填接管通过受控命令完成，网页只展示待审核情况。配置与主题绑定带版本号提交，冲突或失败保留本次输入。

托管文章以 `allowed_actions` 为准，来源管理字段只读；作者通过单字段 PATCH 保存，历史本地正文保留，目录内排序仍可操作。旧页面在编辑中途被接管时，409 后重新读取管理状态，并提供之前输入的副本；不再提交来源字段。紧急下架覆盖来源发布状态；解除后等待下一次同步重新核验，不立即恢复前台。手工飞书收录、编辑与状态操作仍使用原流程。

[同步浏览器夹具说明与截图](docs/notion-sync-fixtures.md)记录本机复现步骤及验收边界。
