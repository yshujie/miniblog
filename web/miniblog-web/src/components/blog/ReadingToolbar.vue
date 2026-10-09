<template>
  <header class="reader-toolbar">
    <div class="reader-topline">
      <button id="reader-sidebar-reopen" type="button" class="icon-button sidebar-reopen"
        :class="{ 'is-visible': !ui.sidebarOpen }" aria-label="展开文章目录" aria-controls="sidebar"
        :aria-expanded="ui.sidebarOpen" @click="ui.toggleSidebar">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" aria-hidden="true"><rect x="3" y="3" width="18" height="18" rx="2" /><path d="M9 3v18m4-10 3 3-3 3" /></svg>
      </button>
      <nav v-if="module" class="breadcrumbs" aria-label="文章位置">
        <router-link :to="topicURL">{{ module.title }}</router-link>
        <template v-if="placement">
          <span aria-hidden="true">›</span>
          <router-link :to="{ path: topicURL, query: { chapter: placement.section.code } }">{{ placement.section.title }}</router-link>
          <template v-if="placement.subsection"><span aria-hidden="true">›</span><span>{{ placement.subsection.title }}</span></template>
        </template>
      </nav>
      <button type="button" class="mobile-directory" aria-label="打开文章目录" aria-haspopup="dialog" :aria-expanded="ui.mobileDrawerOpen" @click="ui.openDrawer">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" aria-hidden="true"><path d="M9 5h12M9 12h12M9 19h12M3 5h.01M3 12h.01M3 19h.01" /></svg>目录
      </button>
    </div>
    <div class="reader-title-row" tabindex="0" aria-label="完整文章标题"><h1>{{ title }}</h1></div>
    <div class="reader-bottomline">
      <div class="reader-info">
        <span v-if="article.author.trim()" class="reader-author">{{ article.author }}</span>
        <span class="source-mark"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" aria-hidden="true"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" /><path d="M14 2v6h6M8 13h8M8 17h6" /></svg>Notion 文档</span>
        <span v-for="tag in visibleTags" :key="tag" class="reader-tag">{{ tag }}</span>
      </div>
      <a v-if="sourceURL" class="primary original-link" :href="sourceURL" target="_blank" rel="noopener noreferrer">
        打开原文<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" aria-hidden="true"><path d="M15 3h6v6m0-6L10 14" /><path d="M21 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h6" /></svg>
      </a>
    </div>
  </header>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { Article } from '@/types/article'
import type { Module } from '@/types/module'
import type { CatalogEntry } from '@/util/catalog'
import { useUiStore } from '@/stores/ui'
const props = defineProps<{ article: Article; title: string; sourceURL: string | null; module?: Module; placement?: CatalogEntry }>()
const ui = useUiStore()
const topicURL = computed(() => '/topics/' + encodeURIComponent(props.module?.code || ''))
const visibleTags = computed(() => props.article.tags.filter(tag => tag.trim()))
</script>

<style scoped>
.reader-toolbar { position:relative; flex:none; padding:20px 32px; border-bottom:1px solid var(--line); background:var(--page); }
.reader-topline { display:flex; align-items:center; gap:12px; margin-bottom:12px; min-width:0; }
.breadcrumbs { flex:1; min-width:0; overflow-wrap:anywhere; }
.breadcrumbs a { overflow-wrap:anywhere; }
.reader-title-row h1 { font-size:23px; line-height:1.5; letter-spacing:-.4px; font-weight:700; color:var(--ink); overflow-wrap:anywhere; padding-right:132px; }
.reader-bottomline { display:flex; align-items:center; gap:20px; margin-top:12px; }
.reader-info { display:flex; flex-wrap:wrap; align-items:center; gap:10px; color:var(--secondary); font-size:12px; min-width:0; flex:1; overflow-wrap:anywhere; }
.source-mark { display:inline-flex; align-items:center; gap:5px; }
.source-mark svg { width:14px; height:14px; flex:none; }
.reader-tag { background:var(--zone); border-radius:3px; padding:3px 7px; }
.original-link { position:absolute; right:32px; top:58px; flex:none; min-height:44px; padding:0 15px; font-size:12px; gap:8px; }
.original-link svg { width:16px; height:16px; }
.sidebar-reopen { display:none; width:44px; height:44px; flex:none; }
.sidebar-reopen.is-visible { display:inline-flex; }
.sidebar-reopen svg { width:20px; height:20px; }
.mobile-directory { display:none; }
@media (max-width:1100px) { .reader-toolbar { padding-left:24px; padding-right:24px; } .reader-title-row h1 { font-size:21px; } .original-link { right:24px; } }
@media (max-width:900px) {
  .sidebar-reopen.is-visible { display:none; }
  .reader-topline { align-items:flex-start; }
  .breadcrumbs { min-height:44px; align-content:center; }
  .mobile-directory { display:inline-flex; align-items:center; justify-content:center; gap:7px; min-height:44px; padding:0 12px; flex:none; border:1px solid var(--line); border-radius:6px; background:var(--page); color:var(--ink); font-size:13px; cursor:pointer; }
  .mobile-directory svg { width:16px; height:16px; }
}
@media (max-width:650px) {
  .reader-toolbar { padding:14px 20px 16px; }
  .reader-topline { gap:12px; margin-bottom:12px; }
  .breadcrumbs { font-size:11px; line-height:1.7; gap:6px; }
  .reader-title-row h1 { font-size:20px; padding-right:0; }
  .reader-bottomline { gap:12px; margin-top:8px; align-items:center; }
  .reader-info { font-size:11px; gap:8px; }
  .reader-tag { display:none; }
  .original-link { position:static; min-height:44px; padding:0 12px; }
  .mobile-directory { padding:0 10px; font-size:12px; }
}
@media (max-width:360px) { .reader-toolbar { padding-left:16px; padding-right:16px; } .reader-title-row h1 { font-size:18px; } }
@media (max-height:560px) {
  .reader-toolbar { padding-top:6px; padding-bottom:6px; }
  .reader-topline { margin-bottom:4px; }
  .reader-title-row { max-height:2.8em; overflow:auto; font-size:18px; scrollbar-width:thin; }
  .reader-title-row h1 { font-size:18px; line-height:1.4; }
  .reader-bottomline { margin-top:4px; }
  .reader-info { max-height:36px; overflow:auto; scrollbar-width:thin; }
  .original-link { top:38px; }
}
@media (max-height:560px) and (min-width:651px) and (max-width:900px) { .original-link { top:54px; } }
@media (max-height:560px) and (max-width:650px) {
  .reader-toolbar { padding-top:6px; padding-bottom:6px; }
  .original-link { position:static; }
  .reader-title-row h1 { padding-right:0; }
}
@media (max-height:420px) { .reader-title-row { max-height:1.4em; } }
@media (max-height:420px) and (min-width:651px) { .reader-info { padding-right:124px; } }
</style>
