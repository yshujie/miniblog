<template>
  <div class="blog-layout">
    <div class="blog-body">
      <slot name="sidebar" />
      <article class="main-content" :class="{ 'main-content-expanded': !uiStore.sidebarOpen }"><slot name="main" /></article>
    </div>
    <el-drawer v-if="!isDesktop" v-model="uiStore.mobileDrawerOpen" title="文章目录"
      direction="ltr" size="min(88vw, 360px)" append-to-body class="reading-drawer">
      <slot name="drawer" />
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { useUiStore } from '@/stores/ui'
const uiStore = useUiStore()
const isDesktop = ref(window.innerWidth >= 901)
let media: MediaQueryList | undefined
const syncViewport = () => {
  isDesktop.value = media?.matches ?? window.innerWidth >= 901
  if (isDesktop.value) uiStore.closeDrawer()
}
onMounted(() => {
  if (typeof window.matchMedia === 'function') {
    media = window.matchMedia('(min-width: 901px)')
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
.blog-layout, .blog-body { display:flex; width:100%; height:100%; min-height:0; overflow:hidden; }
.blog-layout { flex:1; }
.blog-body { flex:1; }
.main-content { display:flex; flex-direction:column; flex:1; min-width:0; min-height:0; background:var(--page); overflow:hidden; }
</style>
<style>
.reading-drawer .el-drawer__header { margin-bottom:0; padding:14px 20px; border-bottom:1px solid var(--line); color:var(--ink); font-size:17px; }
.reading-drawer .el-drawer__close-btn { min-width:44px; min-height:44px; color:var(--secondary); }
.reading-drawer .el-drawer__body { display:flex; flex-direction:column; padding:0; overflow:hidden; min-height:0; }
</style>
