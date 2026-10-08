<template>
  <header class="blog-header">
    <div class="blog-header-inner">
      <button type="button" class="mobile-directory-button" aria-label="打开文章目录"
        :aria-expanded="uiStore.mobileDrawerOpen" @click="uiStore.openDrawer">目录</button>
      <router-link to="/" class="blog-logo">Shujie's Blog</router-link>
      <nav class="blog-nav" aria-label="模块导航">
        <button v-for="module in store.modules" :key="module.code" type="button"
          :class="['blog-nav-item', { active: route.params.module === module.code }]"
          @click="openModule(module.code)">{{ module.title }}</button>
        <button v-if="store.listStatus === 'error'" type="button" @click="reloadModules">重试模块</button>
      </nav>
      <a href="https://github.com/yshujie" target="_blank" rel="noopener noreferrer" class="github-link">GitHub</a>
    </div>
  </header>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router'
import { useReadingNavigation } from '@/composables/useReadingNavigation'
import { useUiStore } from '@/stores/ui'
const route = useRoute()
const uiStore = useUiStore()
const { store, openModule, reloadModules } = useReadingNavigation()
</script>

<style scoped>
.blog-header { flex-shrink: 0; min-height: var(--blog-header-height); background: var(--blog-header-bg); border-bottom: 1px solid var(--blog-header-border); }
.blog-header-inner { display: flex; align-items: center; gap: 1.25rem; height: var(--blog-header-height); padding: 0 1rem; }
.blog-logo { flex-shrink: 0; font-weight: 700; color: var(--text-primary); text-decoration: none; white-space: nowrap; }
.blog-nav { display: flex; flex: 1; min-width: 0; gap: 1.25rem; overflow-x: auto; }
.blog-nav-item { flex-shrink: 0; padding: .5rem 0; background: none; border: 0; color: var(--blog-header-text); cursor: pointer; white-space: nowrap; }
.blog-nav-item.active { color: var(--blog-header-active); font-weight: 600; border-bottom: 2px solid var(--blog-header-active); }
.github-link { color: var(--blog-header-text); text-decoration: none; }
.mobile-directory-button { border: 1px solid var(--sidebar-divider); border-radius: .25rem; background: var(--card-bg); color: var(--text-primary); padding: .35rem; cursor: pointer; }
button:focus-visible, a:focus-visible { outline: 2px solid var(--sidebar-active-color); }
@media (min-width: 1280px) { .mobile-directory-button { display: none; } }
@media (max-width: 600px) {
  .blog-header-inner { gap: .6rem; padding: 0 .5rem; }
  .blog-logo { font-size: .8rem; }
  .github-link { display: none; }
}
</style>
