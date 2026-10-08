import { fileURLToPath, URL } from 'node:url';
import path from 'node:path';
import fs from 'fs';

import { defineConfig } from 'vite';
import vue from '@vitejs/plugin-vue';

import Inspect from 'vite-plugin-inspect';

// element plus 样式自动按需导入
import AutoImport from 'unplugin-auto-import/vite';
import Components from 'unplugin-vue-components/vite';
import { ElementPlusResolver } from 'unplugin-vue-components/resolvers';

import svgSpritePlugin from '@pivanov/vite-plugin-svg-sprite';
// import svgSprites from 'rollup-plugin-svg-sprites';

// https://vitejs.dev/config/
export default defineConfig(() => {
  // 解决终端 optimized dependencies changed. reloading 问题
  const optimizeDepsElementPlusIncludes = ['element-plus/es'];
  fs.readdirSync('node_modules/element-plus/es/components').map((dirname) => {
    fs.access(
      `node_modules/element-plus/es/components/${dirname}/style/css.mjs`,
      (err) => {
        if (!err) {
          optimizeDepsElementPlusIncludes.push(
            `element-plus/es/components/${dirname}/style/css`
          );
        }
      }
    );
  });

  return {
    base: '/', // 注意，必须以"/"结尾，BASE_URL配置
    resolve: {
      alias: {
        '@': fileURLToPath(new URL('./src', import.meta.url))
      },
      extensions: ['.mjs', '.js', '.ts', '.jsx', '.tsx', '.json', '.vue']
    },
    optimizeDeps: {
      include: optimizeDepsElementPlusIncludes
    },
    plugins: [
      vue(),
      Inspect(),
      AutoImport({
        resolvers: [ElementPlusResolver()]
      }),
      Components({
        resolvers: [ElementPlusResolver()]
      }),
      svgSpritePlugin({
        iconDirs: [path.resolve(process.cwd(), 'src/icons/svg')],

        symbolId: 'icon-[name]',

        inject: 'body-last' // 'body-prepend' | 'body-append' | false
      })

    ],
    server: {
      host: 'localhost',
      port: 8001

    }
  };
});
