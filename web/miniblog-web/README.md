# miniblog-web

This template should help get you started developing with Vue 3 in Vite.

## Environment & Dependencies

- **Node.js**: 推荐 Node 24；CI 使用 Node 24。请使用项目独立运行时，不修改系统默认 Node。
- **npm**: 随 Node 自带的 npm ≥ 9（仓库使用 `package-lock.json`，默认包管理器为 npm）。
- **核心依赖**：
 	- `vue@^3.5.13`
 	- `vite@^6.2.4` 与 `@vitejs/plugin-vue`
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

阅读路由继续支持 `/blog/:module` 和 `/blog/:module/article/:article`。所有模块入口按目录显示顺序选择首篇（子章节文章先于章节直属文章）；空模块展示空态。文章深链接先按 ID 获取当前模块，跨模块移动后替换到当前路径，并保留 query/hash；指定文章不存在时不会跳到其他文章。公开 v1 文章响应可新增 `module_code`，旧字段及 ID 字符串契约保持不变。

文章正文继续通过外链 iframe 阅读，始终提供打开原文。iframe 的 load 事件仅代表收到加载事件，十秒提示也只提供原文出口，不判断平台页面是否真正可读。小于 1280px 的屏幕通过目录抽屉选择文章。

单元与组件测试使用本地替身，覆盖 v1 映射、大 ID、首篇、空目录、错误重试、历史链接、请求竞态、目录交互和 iframe 状态。它们不证明 Notion/飞书真实页面可访问；真实平台与 iOS/Android/Safari 的阅读验收需要单独记录。
