import { fileURLToPath, URL } from 'node:url'
import { webcrypto } from 'node:crypto'

import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'

// Provide missing browser-ish globals when Vite loads this config in Node.
const ensureNodeGlobals = () => {
  // Some devtools helpers expect a Web Crypto instance.
  if (typeof globalThis.crypto === 'undefined') {
    globalThis.crypto = webcrypto as any
  }

  // The Vue devtools kit reads localStorage during module init.
  if (typeof (globalThis as any).localStorage?.getItem !== 'function') {
    const store = new Map<string, string>()

    globalThis.localStorage = {
      getItem(key: string) {
        return store.has(key) ? store.get(key)! : null
      },
      setItem(key: string, value: string) {
        store.set(key, String(value))
      },
      removeItem(key: string) {
        store.delete(key)
      },
      clear() {
        store.clear()
      },
      key(index: number) {
        return Array.from(store.keys())[index] ?? null
      },
      get length() {
        return store.size
      },
    } as any
  }
}

// https://vite.dev/config/
export default defineConfig(async ({ mode }) => {
  const env = loadEnv(mode, process.cwd(), 'VITE_')
  ensureNodeGlobals()
  const { default: vueDevTools } = await import('vite-plugin-vue-devtools')

  return {
    plugins: [
      vue(),
      vueDevTools(),
    ],
    resolve: {
      alias: {
        '@': fileURLToPath(new URL('./src', import.meta.url))
      },
    },
    server: {
      proxy: {
        // Forward API calls to the backend to dodge CORS in dev.
        '/api': {
          target: env.VITE_API_PROXY_TARGET || 'http://127.0.0.1:8080',
          changeOrigin: true,
          rewrite: (path: any) => path.replace(/^\/api/, ''),
        },
      },
    },
  }
})
