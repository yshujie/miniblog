<template>
  <main class="blog-layout">
    <div class="blog-body">
      <slot name="sidebar" />
      <article class="main-content" :class="{ 'main-content-expanded': !uiStore.sidebarOpen }">
        <slot name="main" />
        <button type="button" class="sidebar-toggle-trigger"
          :class="{ 'sidebar-closed': !uiStore.sidebarOpen }"
          :aria-expanded="uiStore.sidebarOpen" aria-label="切换文章目录"
          :title="uiStore.sidebarOpen ? '隐藏侧边栏' : '展开侧边栏'" @click="uiStore.toggleSidebar">
          <span aria-hidden="true">{{ uiStore.sidebarOpen ? '›' : '‹' }}</span>
        </button>
      </article>
    </div>
    <el-drawer v-if="!isDesktop" v-model="uiStore.mobileDrawerOpen" title="文章目录"
      direction="ltr" size="min(86vw, 320px)" append-to-body class="reading-drawer">
      <slot name="drawer" />
    </el-drawer>
  </main>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { useUiStore } from '@/stores/ui'
const uiStore = useUiStore()
const isDesktop = ref(window.innerWidth >= 1280)
let media: MediaQueryList | undefined
const syncViewport = () => {
  isDesktop.value = media?.matches ?? window.innerWidth >= 1280
  if (isDesktop.value) uiStore.closeDrawer()
}
onMounted(() => {
  if (typeof window.matchMedia === 'function') {
    media = window.matchMedia('(min-width: 1280px)')
    media.addEventListener('change', syncViewport)
  }
  window.addEventListener('resize', syncViewport)
  syncViewport()
})
onUnmounted(() => {
  media?.removeEventListener('change', syncViewport)
  window.removeEventListener('resize', syncViewport)
  uiStore.closeDrawer()
})
</script>

<style scoped>
.blog-layout, .blog-body { display: flex; width: 100%; height: 100%; min-height: 0; overflow: hidden; }
.blog-body { flex: 1; }
.main-content { position: relative; flex: 1; min-width: 0; min-height: 0; background: var(--card-bg); overflow: hidden; border-left: 1px solid var(--blog-header-border); }
.main-content-expanded { border-left-color: transparent; }
.sidebar-toggle-trigger {
  display: none; position: absolute; left: 0; top: 50%; transform: translateY(-50%); z-index: 5;
  width: 28px; height: 56px; background: var(--card-bg); border: 1px solid var(--sidebar-divider);
  border-radius: 0 8px 8px 0; cursor: pointer; color: var(--sidebar-text); font-size: 1.25rem;
}
.sidebar-toggle-trigger:focus-visible { outline: 2px solid var(--sidebar-active-color); }
@media (min-width: 1280px) { .sidebar-toggle-trigger { display: block; } }
</style>

<style>
.reading-drawer .el-drawer__body { padding: 0; overflow: hidden; }
</style>
