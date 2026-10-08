import { fileURLToPath, URL } from 'node:url';
import { defineConfig } from 'vitest/config';
import vue from '@vitejs/plugin-vue';
export default defineConfig({
  plugins: [vue()],
  resolve: { alias: { '@': fileURLToPath(new URL('./src', import.meta.url)) }},
  test: { environment: 'jsdom', include: ['src/**/*.test.ts'], clearMocks: true, restoreMocks: true, env: { VITE_API_ROOT: 'http://localhost:8080/v1', VITE_CONTENT_REGISTER_ENABLED: 'true' }}
});
