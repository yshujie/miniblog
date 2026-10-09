# miniblog-web

This template should help get you started developing with Vue 3 in Vite.

## Environment & Dependencies

- **Node.js**: 推荐 Node 24；CI 使用 Node 24。请使用项目独立运行时，不修改系统默认 Node。
- **npm**: 随 Node 自带的 npm ≥ 9（仓库使用 `package-lock.json`，默认包管理器为 npm）。
- **核心依赖**：
 	- `vue@^3.5.13`
  - `vite@6.4.0` 与 `@vitejs/plugin-vue`
 	- `pinia@^3.0.2`
 	- `element-plus@^2.9.10`
 	- `axios@^1.9.0`
 	- `md-editor-v3@^5.5.1`
 	- `vue3-markdown-it@^1.0.10`
- **开发工具**：
 	- `typescript@~5.8.0` & `vue-tsc@^2.2.8`
 	- `@tsconfig/node22`（TypeScript 编译目标配置）
 	- `npm-run-all2@^7.0.2`（用于并行执行 `npm run build` 内部脚本）

> 提示：如需在本地或 CI 中切换 Node 版本，建议使用 nvm/volta 等工具固定版本；请勿混用 yarn/pnpm 以免破坏锁文件。

## Recommended IDE Setup

[VSCode](https://code.visualstudio.com/) + [Volar](https://marketplace.visualstudio.com/items?itemName=Vue.volar) (and disable Vetur).

## Type Support for `.vue` Imports in TS

TypeScript cannot handle type information for `.vue` imports by default, so we replace the `tsc` CLI with `vue-tsc` for type checking. In editors, we need [Volar](https://marketplace.visualstudio.com/items?itemName=Vue.volar) to make the TypeScript language service aware of `.vue` types.

## Customize configuration

See [Vite Configuration Reference](https://vite.dev/config/).

## Project Setup

```sh
npm install
```

### Compile and Hot-Reload for Development

```sh
npm run dev
```

> 开发环境默认通过 Vite 代理把以 `/api` 开头的请求转发到 `http://127.0.0.1:8080`。可在 `.env.local` 修改 `VITE_API_PROXY_TARGET` 或 `VITE_API_BASE_URL`（默认 `/api/v1`）。

- 生产环境默认 `VITE_API_BASE_URL=https://api.yangshujie.com/v1`（见 `.env.production`）；如需自定义后端域名，请在打包前覆盖该环境变量。

### Type-Check, Compile and Minify for Production

```sh
npm run build
```

## 阅读功能与本地验证

开发和测试默认使用 `/api/v1`，开发代理指向 `http://127.0.0.1:8080`。需要另一个本地端口时复制 `.env.example` 为 `.env.local` 并设置 `VITE_API_PROXY_TARGET`。生产构建保留原 API 地址，也可用 `VITE_API_BASE_URL` 覆盖。

```sh
npm ci
npm run lint:check
npm run type-check
npm run test:unit
npm run build
```

阅读路由继续支持 `/blog/:module` 和 `/blog/:module/article/:article`。首页“开始阅读”及旧模块地址按目录顺序选择首篇（子章节文章先于章节直属文章）；空模块展示空态。顶部主题导航进入新增主题总览 `/topics/:module`，不会请求正文或跳转首篇。章节入口使用 `?chapter=code` 定位；目录和总览仅搜索本主题文章标题。文章深链接先按 ID 获取当前模块，跨模块移动后替换到当前路径，并保留 query/hash；指定文章不存在时不会跳到其他文章。公开 v1 文章响应可新增 `module_code`，旧字段及 ID 字符串契约保持不变。

文章正文继续通过外链 iframe 阅读，始终提供打开原文。iframe 的 load 事件仅代表收到加载事件，十秒提示也只提供原文出口，不判断平台页面是否真正可读。901px 起目录常驻；900px 及以下通过目录抽屉选择文章；650px 及以下通过独立主题面板切换主题。两个面板互斥。

单元与组件测试使用本地替身，覆盖 v1 映射、大 ID、首篇、空目录、错误重试、历史链接、请求竞态、目录交互和 iframe 状态。它们不证明 Notion/飞书真实页面可访问；真实平台与 iOS/Android/Safari 的阅读验收需要单独记录。


公开文章可提供 `reading_url`，原文入口优先使用该地址，再回退旧 `external_link`；两者只接受无凭据的 HTTP(S)。iframe 与原文入口分开选择地址：可识别的普通 Notion 页面从当前 origin 和 page ID 生成 `/ebd/<pageID>`，已有 `/ebd/` 地址保持不变，不用历史链接覆盖当前页面。含 `p`/`v` 的数据库或视图地址、未知 slug、自定义域名和其他平台不猜测页面身份。文章 ID 与旧路由保持不变。模块摘要和完整目录缓存有效期为 60 秒；阅读页可见时每 60 秒刷新，重新回到前台或获得焦点时刷新。相同文章 ID 与嵌入地址保留 iframe，原文参数或元信息更新不重载正文；资料刷新失败保留已加载内容并显示重试提示，明确的 404 则移除正文。紧急下架与解除后的来源重新核验由后端判断。

[管理端本机浏览器夹具](../miniblog-web-admin/docs/notion-sync-fixtures.md)同时覆盖阅读地址、跨模块历史链接、60 秒刷新和紧急下架。当前验证使用本机替身正文，仍不证明真实 Notion/飞书允许 iframe 阅读。

## 读者 UI 与独立浏览器验收

首页使用公开主题摘要，视口附近章节预览最多三个并发详情请求。计数只来自完整目录，单个预览失败可以独立重试。主题总览和阅读页在可见时每60秒刷新目录，暂时错误保留已有内容，明确404清除内容。目录刷新保留用户展开选择；正文iframe仅文章ID或实际嵌入地址变化时重建。阅读外框只保留单行章节导航、目录和原文入口，标题、作者、标签由 Notion 正文展示；保留屏幕阅读器可识别的语义标题。

阅读目录采用白底层级导航，主题总篇数从完整目录计算，目录尚未加载时不显示零篇。章节与子章节旁的篇数随标题搜索过滤，搜索期间自动展开并停用折叠动作。长标题完整换行；桌面与移动抽屉共用目录样式。当前文章可通过“定位当前”清除搜索、展开所属路径并在目录中滚动、聚焦，不改变阅读路由或重建 iframe；文章未出现在目录时隐藏此入口。

独立夹具仅使用本机公开API和替身文档，不驱动管理后台、不连接生产服务。先使用Node24按锁文件安装依赖，然后分别启动：

```sh
node scripts/reader-fixture-server.mjs
VITE_API_BASE_URL=/api/v1 VITE_API_PROXY_TARGET=http://127.0.0.1:8771 npm run dev -- --host 127.0.0.1 --port 8770 --strictPort
```

打开 `http://127.0.0.1:8770` 可审阅真实Vue实现与本机示例数据。浏览器验收脚本需要环境可用的Playwright与Chromium；使用已有工作区运行时即可，不需要改变产品依赖：

```sh
MINIBLOG_PLAYWRIGHT_MODULE=/absolute/path/to/playwright \
MINIBLOG_CHROMIUM_EXECUTABLE=/absolute/path/to/chromium \
MINIBLOG_READER_URL=http://127.0.0.1:8770 \
MINIBLOG_READER_OUTPUT=/absolute/path/to/output \
node scripts/reader-browser-fixture.mjs
```

检查包含首页、总览、搜索、旧首篇、历史迁移、iframe稳定、刷新失败/下架、移动面板与焦点、320–1440px适配及状态截图。真实Notion可读性、Safari/iOS/Android和生产新路由硬刷新需要另行验收；替身截图不代表实际来源或部署结果。
