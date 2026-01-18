# miniblog-web

This template should help get you started developing with Vue 3 in Vite.

## Environment & Dependencies

- **Node.js**: ≥ 18.0.0（推荐使用 Node 20 LTS 或 Node 22，便于兼容 Vite 6 与 TypeScript 5.8）。
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

> 开发环境默认通过 Vite 代理把以 `/api` 开头的请求转发到 `https://api.yangshujie.com`，避免 CORS。可在 `.env.development` 修改 `VITE_API_BASE_URL`（默认 `/api/v1`）或在 `vite.config.ts` 调整代理目标。

### Type-Check, Compile and Minify for Production

```sh
npm run build
```
