import js from '@eslint/js'
import tsParser from '@typescript-eslint/parser'
import tsPlugin from '@typescript-eslint/eslint-plugin'
import vueParser from 'vue-eslint-parser'
import vuePlugin from 'eslint-plugin-vue'
import globals from 'globals'

// Restrict linting to the reading refactor. Legacy unused Markdown remains outside this batch.
const changedSources = [
  'src/App.vue', 'src/main.ts', 'src/router/index.ts', 'src/types/article.ts', 'src/types/response.ts',
  'src/api/blog.ts', 'src/api/module.ts', 'src/api/reading-contract.ts', 'src/api/__tests__/**/*.ts',
  'src/util/http.ts', 'src/util/reading.ts', 'src/util/__tests__/**/*.ts',
  'src/stores/module.ts', 'src/stores/ui.ts', 'src/stores/__tests__/**/*.ts',
  'src/composables/**/*.ts', 'src/reading/**/*.ts', 'src/__tests__/**/*.ts',
  'src/pages/Blog.vue', 'src/pages/Index.vue', 'src/pages/__tests__/**/*.ts',
  'src/components/Header.vue', 'src/components/blog/BlogHeader.vue',
  'src/components/blog/BlogLayout.vue', 'src/components/blog/ExternalArticleCard.vue',
  'src/components/blog/Sidebar.vue', 'src/components/blog/__tests__/**/*.ts',
]

export default [
  { ignores: ['node_modules/**', 'dist/**'] },
  {
    files: changedSources,
    languageOptions: { parser: tsParser, sourceType: 'module', ecmaVersion: 'latest',
      globals: { ...globals.browser, ...globals.es2021, defineProps: 'readonly' } },
    plugins: { '@typescript-eslint': tsPlugin },
    rules: {
      ...js.configs.recommended.rules,
      'no-undef': 'off',
      'no-unused-vars': 'off',
      'no-empty': ['error', { allowEmptyCatch: true }],
      '@typescript-eslint/no-unused-vars': ['error', { argsIgnorePattern: '^_' }],
    },
  },
  {
    files: changedSources.filter(path => path.endsWith('.vue')),
    languageOptions: { parser: vueParser, parserOptions: { parser: tsParser, extraFileExtensions: ['.vue'] } },
    plugins: { vue: vuePlugin },
    rules: {
      ...vuePlugin.configs['vue3-essential'].rules,
      // Retain the existing public component filenames in this scoped refactor.
      'vue/multi-word-component-names': ['error', { ignores: ['Header', 'Sidebar', 'Blog', 'Index'] }],
    },
  },
]
