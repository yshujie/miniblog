<template>
  <header class="reader-toolbar">
    <h1 class="sr-only">{{ title }}</h1>
    <div class="reader-topline">
      <button id="reader-sidebar-reopen" type="button" class="icon-button sidebar-reopen"
        :class="{ 'is-visible': !ui.sidebarOpen }" aria-label="展开文章目录" aria-controls="sidebar"
        :aria-expanded="ui.sidebarOpen" @click="ui.toggleSidebar">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" aria-hidden="true"><rect x="3" y="3" width="18" height="18" rx="2" /><path d="M9 3v18m4-10 3 3-3 3" /></svg>
      </button>
      <nav v-if="module" class="breadcrumbs" aria-label="文章位置">
        <router-link :to="topicURL" :title="module.title">{{ module.title }}</router-link>
        <template v-if="placement">
          <span class="breadcrumb-separator" aria-hidden="true">›</span>
          <router-link :to="{ path: topicURL, query: { chapter: placement.section.code } }" :title="placement.section.title">{{ placement.section.title }}</router-link>
          <template v-if="placement.subsection"><span class="breadcrumb-separator" aria-hidden="true">›</span><span class="breadcrumb-label" :title="placement.subsection.title">{{ placement.subsection.title }}</span></template>
        </template>
      </nav>
      <button type="button" class="mobile-directory" aria-label="打开文章目录" aria-haspopup="dialog" :aria-expanded="ui.mobileDrawerOpen" @click="ui.openDrawer">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" aria-hidden="true"><path d="M9 5h12M9 12h12M9 19h12M3 5h.01M3 12h.01M3 19h.01" /></svg>目录
      </button>
      <a v-if="originalURL" class="text-button original-link" :href="originalURL" target="_blank" rel="noopener noreferrer">
        打开原文<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" aria-hidden="true"><path d="M15 3h6v6m0-6L10 14" /><path d="M21 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h6" /></svg>
      </a>
    </div>
  </header>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { Module } from '@/types/module'
import type { CatalogEntry } from '@/util/catalog'
import { useUiStore } from '@/stores/ui'
const props = defineProps<{ title: string; originalURL: string | null; module?: Module; placement?: CatalogEntry }>()
const ui = useUiStore()
const topicURL = computed(() => '/topics/' + encodeURIComponent(props.module?.code || ''))
</script>

<style scoped>
.reader-toolbar { position:relative; flex:none; height:56px; min-height:56px; padding:0 32px; border-bottom:1px solid var(--line); background:var(--page); }
.reader-topline { display:flex; align-items:center; gap:12px; height:100%; min-width:0; }
.breadcrumbs { flex:1; min-width:0; min-height:44px; flex-wrap:nowrap; overflow:hidden; white-space:nowrap; overflow-wrap:normal; }
.breadcrumbs a, .breadcrumb-label { min-width:0; overflow:hidden; text-overflow:ellipsis; white-space:nowrap; overflow-wrap:normal; }
.breadcrumbs a:first-child { flex-shrink:2; }
.breadcrumb-separator { flex:none; }
.original-link { flex:none; margin-left:auto; min-height:44px; padding:0 4px; font-size:12px; gap:7px; color:var(--secondary); white-space:nowrap; }
.original-link:hover { color:var(--green); }
.original-link svg { width:16px; height:16px; }
.sidebar-reopen { display:none; width:44px; height:44px; flex:none; }
.sidebar-reopen.is-visible { display:inline-flex; }
.sidebar-reopen svg { width:20px; height:20px; }
.mobile-directory { display:none; }
@media (max-width:1100px) { .reader-toolbar { padding-left:24px; padding-right:24px; } }
@media (max-width:900px) {
  .sidebar-reopen.is-visible { display:none; }
  .mobile-directory { display:inline-flex; align-items:center; justify-content:center; gap:7px; min-height:44px; padding:0 10px; flex:none; border:1px solid var(--line); border-radius:6px; background:var(--page); color:var(--ink); font-size:12px; cursor:pointer; }
  .mobile-directory svg { width:16px; height:16px; }
}
@media (max-width:650px) {
  .reader-toolbar { padding-left:20px; padding-right:20px; }
  .reader-topline { gap:8px; }
  .breadcrumbs { font-size:11px; gap:6px; }
  .original-link { gap:5px; padding:0 2px; }
}
@media (max-width:360px) { .reader-toolbar { padding-left:16px; padding-right:16px; } }
@media (max-height:560px) { .reader-toolbar { height:52px; min-height:52px; } }
</style>
