# 管理端本机真实接口验收

这套工具仅面向此次管理端重构的本机验收。它使用真实 Go HTTP controller、Authn / Authz、业务服务和 MySQL store；Notion HTTP 由本地替身提供。生产的 `installRouters` 没有修改或调用，后端安全策略、部署与生产数据也没有修改。

## 自动契约验证

在仓库根执行：

```bash
scripts/admin-ui-integration/run.sh /tmp/miniblog-admin-ui-integration
```

脚本只使用本机 Docker Unix socket，创建新的 MySQL 8.0.36 容器，绑定 `127.0.0.1` 随机端口，数据存放于临时内存挂载。它不会重用现有容器、数据库或真实凭据；结束时仅删除本次创建且标签匹配的容器。测试 DSN 还受 `scratchDB` 的 loopback 和专用库名校验保护。`summary.json` 保存通过 / 失败 / 跳过数量；不把原始日志视为适合公开的发布附件。

新增 HTTP 覆盖登录与当前账户、匿名请求拒绝、手工收录去重 / 四态 / 标题失败手填、分页总量与直属 / 子章边界、完整文章 / 目录排序、依赖删除限制、托管字段只读 / 本地作者 / hold 与重新核验、来源停用与版本冲突、同主题绑定、维护门槛、预览不写文章或目录、新鲜同步与运行 ID / 明细。已有内容集成测试同时运行。

## 新 UI 贯穿验证

先把新管理端构建为同源 `/v1` 请求：

```bash
cd web/miniblog-web-admin
VITE_API_ROOT=/v1 VITE_CONTENT_REGISTER_ENABLED=true npm run build
cd ../..
MINIBLOG_ADMIN_REAL_DIST="$PWD/web/miniblog-web-admin/dist" \
  scripts/admin-ui-integration/serve.sh /tmp/miniblog-admin-ui-real
```

服务读取指定 `dist`，同源提供静态页面及 `/v1` API。启动只接受空的 `miniblog_refactor_test_admin_ui_*` 本机数据库；已有表即拒绝初始化。HTTP 也只允许 loopback。所有外出 HTTP 请求受保护：Notion API 请求改送本地替身，其他非 loopback 请求一律拒绝。没有来源正文编辑或真实 Notion 写操作。

本机合成登录账号是 `fixtureadmin` / `LocalFixture12`，仅在本次临时数据库创建。浏览器验收不能输出登录 token 或密码响应。`ready.json` 提供 URL、服务 PID、准确大 ID、托管文章 ID 和目录编码。关闭服务使用当前运行窗口的中断，或只向该 `ready.json` 记录的本机服务 PID 发送 TERM；脚本随后清理自己创建的 MySQL 容器。关闭结果 `ready.json.shutdown.json` 保存替身请求 / 写入次数。

API 数据不是浏览器拦截返回：大 ID 为 `9007199254740993`；目录为 `m1 / s1 / sub1`；手工文章、草稿、子章节文章及托管文章都来自临时 MySQL。浏览器脚本另行负责登录、页面入口、点击动作、API 请求结果和重新打开后的持久结果。它应阻止外部博客 / 文档导航，并分别记录浏览器夹具、此真实本机链路、真实外部服务与生产验收。

## 浏览器贯穿与最终截图复现

保留上面的服务进程，在另一个终端运行。需要 Node.js 24、Playwright 与可用的 Chrome；工具不自动安装依赖。当前 Codex 内置运行环境路径如下，其他机器可将两项改成自己的 Node / Playwright 路径，并用 `MINIBLOG_CHROMIUM_EXECUTABLE` 指向本机浏览器。

```bash
admin_node=/Users/yangshujie/.cache/codex-runtimes/codex-primary-runtime/dependencies/node/bin/node
admin_playwright=/Users/yangshujie/.cache/codex-runtimes/codex-primary-runtime/dependencies/node/node_modules/playwright
MINIBLOG_ADMIN_READY=/tmp/miniblog-admin-ui-real/ready.json \
MINIBLOG_PLAYWRIGHT_MODULE="$admin_playwright" \
MINIBLOG_ADMIN_REAL_OUTPUT=/tmp/miniblog-admin-ui-real-browser \
  "$admin_node" scripts/admin-ui-integration/browser.mjs
```

`browser.mjs` 在全新临时库中完成 11 组贯穿流程：登录与精确大 ID / query / hash 返回、资料保存与未保存离开、文章四态、上下文创建子章节、连续收录与重复保留、真实服务端筛选、托管只读 / 作者 PATCH、下架 / 解除后仍不可读、只读预览、**新鲜同步后公开 HTTP 200**、移动导航 / 当前账户 / 退出与客户端会话清理。页面 API 响应不被拦截替换，外部地址一律阻止；它会写本机合成内容，完整复现应从新服务与新临时库开始。输出 `report.json`、API 请求摘要和截图，认证响应与 token 不落盘。

通过贯穿流程后采集最终视口截图：

```bash
MINIBLOG_ADMIN_READY=/tmp/miniblog-admin-ui-real/ready.json \
MINIBLOG_PLAYWRIGHT_MODULE="$admin_playwright" \
MINIBLOG_ADMIN_CAPTURE_OUTPUT=/tmp/miniblog-admin-review \
  "$admin_node" scripts/admin-ui-integration/capture.mjs
```

`capture.mjs` 读取真实临时数据，记录桌面 / 手机视口及成功反馈，并检查页面不横向溢出和反馈位于可见视口。它只放行同源 GET、登录 POST，以及指定托管文章的本地作者 PATCH；最后一次作者保存用于验证成功反馈，所以截图脚本不是完全只读。它不重放收录、状态、同步或目录命令，也不访问真实外部文档或生产。

最终摘要应区分真实 UI → Go → MySQL、独立 Go / MySQL 契约、替身分流单测、浏览器拦截夹具与真实 Notion。`dry_run` 会写运行 / 来源元信息；只有“没有文章 / 目录业务写入”属于它的约束，不能写成“没有数据库写入”。原始 UI 历史运行没有采集完整数据库前后快照时，应以运行结果、公开状态检查和独立契约测试说明可证明的范围。

## 现有认证限制

这轮不修权限策略。新增表征测试实际确认：Authz 拒绝目前仍继续处理；MyInfo 返回固定 `admin` 角色；Logout 不撤销已签发 JWT。因此测试通过不表示权限安全验收通过。生产创建用户 / 改密路由的注册位置仍只是静态发现，本服务不提供这两个入口，也不执行安全探测。账户编辑、改密与角色权限管理不属于此次交付。
