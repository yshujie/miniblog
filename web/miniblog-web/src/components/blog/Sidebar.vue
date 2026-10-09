<template>
  <aside :id="drawer ? 'drawer-sidebar' : 'sidebar'" class="sidebar-root"
    :class="{ 'sidebar-hidden': !sidebarOpen, 'sidebar-drawer': drawer }"
    :inert="!sidebarOpen ? true : undefined" :aria-hidden="!sidebarOpen ? true : undefined">
    <div class="sidebar-content">
      <div class="sidebar-head">
        <div class="sidebar-top">
          <div class="sidebar-topic">
            <router-link :to="topicURL" @click="uiStore.closeDrawer">{{ moduleLabel }}</router-link>
            <span v-if="catalogReady" class="topic-count">{{ entries.length }} 篇文章</span>
          </div>
          <button v-if="!drawer" type="button" class="icon-button" aria-label="收起文章目录" aria-controls="sidebar" :aria-expanded="sidebarOpen" @click="hideSidebar">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" aria-hidden="true"><rect x="3" y="3" width="18" height="18" rx="2" /><path d="M9 3v18m8-10-3 3 3 3" /></svg>
          </button>
        </div>
        <div class="search-field directory-search">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" aria-hidden="true"><circle cx="10.5" cy="10.5" r="6.5" /><path d="m16 16 4 4" /></svg>
          <input v-model="query" type="search" aria-label="搜索本主题标题" placeholder="搜索文章标题" :disabled="!catalogReady" />
          <button v-if="query" type="button" aria-label="清空标题搜索" @click="query = ''">×</button>
        </div>
      </div>
      <div class="directory-tools">
        <span class="directory-label" aria-live="polite">{{ isSearching ? matches.length + ' 个搜索结果' : (drawer && catalogReady ? entries.length + ' 篇文章' : '章节目录') }}</span>
        <button v-if="currentEntry" type="button" class="locate-current" @click="locateCurrent">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" aria-hidden="true"><circle cx="12" cy="12" r="6" /><circle cx="12" cy="12" r="2" /><path d="M12 2v4m0 12v4M2 12h4m12 0h4" /></svg>定位当前
        </button>
      </div>
      <nav ref="directoryList" class="section-list" aria-label="文章目录">
        <p v-if="isSearching && !matches.length" class="no-results" role="status">没有匹配的文章标题。<button class="text-button" type="button" @click="query = ''">清空搜索</button></p>
        <p v-else-if="catalogReady && !sections.length" class="directory-empty">目录暂无文章</p>
        <div v-for="section in visibleSections" :key="section.id" class="section-item" :class="{ 'section-current': currentEntry?.section.id === section.id }">
          <button type="button" class="section-header" :aria-expanded="isSectionExpanded(section.id)" :disabled="isSearching" @click="toggleSection(section.id)">
            <span class="section-title">{{ section.title }}</span>
            <span class="section-count" aria-hidden="true">{{ sectionArticleCount(section) }}</span>
            <svg v-if="!isSearching" class="icon-arrow" :class="{ 'icon-arrow-expanded': isSectionExpanded(section.id) }" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" aria-hidden="true"><path d="m9 6 6 6-6 6" /></svg>
          </button>
          <div v-show="isSectionExpanded(section.id)" class="section-body">
            <div v-for="subsection in section.subsections" :key="subsection.id" class="nav-group">
              <button type="button" class="group-header" :aria-expanded="isSubsectionExpanded(section.id, subsection.code)" :disabled="isSearching" @click="toggleSubsection(section.id, subsection.code)">
                <span class="group-title">{{ subsection.title }}</span>
                <span class="section-count" aria-hidden="true">{{ subsection.articles.length }}</span>
                <svg v-if="!isSearching" class="icon-arrow" :class="{ 'icon-arrow-expanded': isSubsectionExpanded(section.id, subsection.code) }" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" aria-hidden="true"><path d="m9 6 6 6-6 6" /></svg>
              </button>
              <div v-show="isSubsectionExpanded(section.id, subsection.code)" class="article-list article-list-nested">
                <button v-for="article in subsection.articles" :key="article.id" type="button" class="article-item"
                  :class="{ 'article-item-active': article.id === currentArticleId }" :aria-current="article.id === currentArticleId ? 'page' : undefined"
                  :title="article.title" @click="handleArticleClick(article.id)"><span class="article-title">{{ fullTitle(article.title) }}</span></button>
              </div>
            </div>
            <div v-if="section.articles.length" class="article-list">
              <button v-for="article in section.articles" :key="article.id" type="button" class="article-item"
                :class="{ 'article-item-active': article.id === currentArticleId }" :aria-current="article.id === currentArticleId ? 'page' : undefined"
                :title="article.title" @click="handleArticleClick(article.id)"><span class="article-title">{{ fullTitle(article.title) }}</span></button>
            </div>
          </div>
        </div>
      </nav>
      <div class="directory-footer"><router-link :to="topicURL" @click="uiStore.closeDrawer">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" aria-hidden="true"><path d="M12 5c-3-2-6-2-9-1v15c3-1 6-1 9 1 3-2 6-2 9-1V4c-3-1-6-1-9 1Zm0 0v15" /></svg>
        查看主题总览
      </router-link></div>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useUiStore } from '@/stores/ui'
import type { Section } from '@/types/section'
import { filterCatalog, findCatalogEntry, flattenCatalog } from '@/util/catalog'
const props = withDefaults(defineProps<{ sections: Section[]; moduleCode: string; moduleTitle?: string; drawer?: boolean; catalogReady?: boolean }>(), { catalogReady: true })
const router = useRouter()
const uiStore = useUiStore()
const query = ref('')
const directoryList = ref<HTMLElement | null>(null)
const expandedSectionIds = ref(new Set<string>())
const expandedSubsectionKeys = ref(new Set<string>())
const sidebarOpen = computed(() => props.drawer || uiStore.sidebarOpen)
const moduleLabel = computed(() => props.moduleTitle?.trim() || '文档导航')
const topicURL = computed(() => '/topics/' + encodeURIComponent(props.moduleCode))
const currentArticleId = computed(() => typeof router.currentRoute.value.params.article === 'string' ? router.currentRoute.value.params.article : null)
const entries = computed(() => flattenCatalog({ sections: props.sections }))
const currentEntry = computed(() => currentArticleId.value ? findCatalogEntry(entries.value, currentArticleId.value) : undefined)
const isSearching = computed(() => Boolean(query.value.trim()))
const matches = computed(() => filterCatalog(entries.value, query.value))
const visibleSections = computed(() => {
  if (!isSearching.value) return props.sections
  const ids = new Set(matches.value.map(entry => entry.article.id))
  return props.sections.map(section => ({ ...section,
    articles: section.articles.filter(article => ids.has(article.id)),
    subsections: section.subsections.map(subsection => ({ ...subsection, articles: subsection.articles.filter(article => ids.has(article.id)) })).filter(subsection => subsection.articles.length),
  })).filter(section => section.articles.length || section.subsections.length)
})
const subsectionExpandKey = (sectionId: string, code: string) => sectionId + '::' + code
const fullTitle = (title: string) => title.trim() || '未命名文章'
const sectionArticleCount = (section: Section) => flattenCatalog({ sections: [section] }).length
const isSectionExpanded = (id: string) => isSearching.value || expandedSectionIds.value.has(id)
const isSubsectionExpanded = (id: string, code: string) => isSearching.value || expandedSubsectionKeys.value.has(subsectionExpandKey(id, code))
function toggleSection(id: string) {
  if (isSearching.value) return
  const expanded = new Set(expandedSectionIds.value)
  if (expanded.has(id)) expanded.delete(id); else expanded.add(id)
  expandedSectionIds.value = expanded
}
function toggleSubsection(id: string, code: string) {
  if (isSearching.value) return
  const key = subsectionExpandKey(id, code)
  const expanded = new Set(expandedSubsectionKeys.value)
  if (expanded.has(key)) expanded.delete(key); else expanded.add(key)
  expandedSubsectionKeys.value = expanded
}
function expandForCurrentArticle() {
  const entry = currentEntry.value
  if (!entry) return
  expandedSectionIds.value = new Set([...expandedSectionIds.value, entry.section.id])
  if (entry.subsection) expandedSubsectionKeys.value = new Set([...expandedSubsectionKeys.value, subsectionExpandKey(entry.section.id, entry.subsection.code)])
}
async function locateCurrent() {
  query.value = ''
  expandForCurrentArticle()
  await nextTick()
  const current = directoryList.value?.querySelector<HTMLElement>('[aria-current="page"]')
  current?.scrollIntoView({ block: 'nearest', inline: 'nearest' })
  current?.focus({ preventScroll: true })
}
watch([() => props.sections, () => props.moduleCode], ([sections, code], previous) => {
  if (previous[1] !== code) { expandedSectionIds.value = new Set(); expandedSubsectionKeys.value = new Set(); query.value = '' }
  const sectionIds = new Set(sections.map(section => section.id))
  const subsectionKeys = new Set(sections.flatMap(section => section.subsections.map(subsection => subsectionExpandKey(section.id, subsection.code))))
  expandedSectionIds.value = new Set([...expandedSectionIds.value].filter(id => sectionIds.has(id)))
  expandedSubsectionKeys.value = new Set([...expandedSubsectionKeys.value].filter(key => subsectionKeys.has(key)))
  expandForCurrentArticle()
}, { immediate: true })
watch(currentArticleId, expandForCurrentArticle)
watch(isSearching, searching => { if (!searching) expandForCurrentArticle() })
function handleArticleClick(id: string) {
  uiStore.closeDrawer()
  void router.push({ name: 'BlogArticle', params: { module: props.moduleCode, article: id } })
}
async function hideSidebar() {
  uiStore.toggleSidebar()
  await nextTick()
  document.getElementById('reader-sidebar-reopen')?.focus()
}
</script>

<style scoped>
.sidebar-root { display:none; width:280px; min-width:280px; height:100%; min-height:0; flex:none; overflow:hidden; background:var(--page); border-right:1px solid var(--line); }
.sidebar-hidden { width:0; min-width:0; border-right:0; pointer-events:none; }
.sidebar-content { width:280px; height:100%; min-height:0; display:flex; flex-direction:column; overflow:hidden; }
.sidebar-head { padding:16px 18px 8px; flex:none; }
.sidebar-top { display:flex; align-items:center; justify-content:space-between; gap:12px; margin-bottom:14px; min-height:44px; }
.sidebar-topic { display:flex; flex-direction:column; gap:3px; min-width:0; }
.sidebar-topic a { font-size:17px; line-height:1.5; font-weight:650; min-width:0; overflow-wrap:anywhere; }
.sidebar-topic a:hover { color:var(--green); }
.topic-count { color:var(--secondary); font-size:11px; }
.sidebar-top .icon-button { width:44px; height:44px; flex:none; border:0; background:transparent; color:var(--secondary); }
.sidebar-top .icon-button:hover { background:var(--zone); color:var(--ink); }
.sidebar-top svg { width:18px; height:18px; }
.directory-search { width:100%; min-width:0; min-height:44px; height:44px; background:var(--zone); border-color:transparent; border-radius:7px; }
.directory-search:focus-within { border-color:var(--green); background:var(--page); }
.directory-search input { min-width:0; font-size:12px; }
.directory-search input::-webkit-search-cancel-button, .directory-search input::-webkit-search-decoration { display:none; }
.directory-search > svg { width:15px; height:15px; flex:none; }
.directory-search button { color:var(--secondary); font-size:20px; border:0; background:transparent; width:44px; height:44px; flex:none; cursor:pointer; margin-right:-12px; }
.directory-tools { display:flex; align-items:center; justify-content:space-between; min-height:44px; padding:0 18px; flex:none; gap:8px; }
.directory-label { font-size:11px; color:var(--secondary); }
.locate-current { display:inline-flex; align-items:center; justify-content:center; gap:5px; min-height:44px; border:0; background:transparent; color:var(--secondary); font-size:11px; padding:0 2px; cursor:pointer; }
.locate-current:hover { color:var(--green); }
.locate-current svg { width:14px; height:14px; }
.section-list { min-height:0; flex:1; overflow:auto; overscroll-behavior:contain; padding:0 12px 16px; scrollbar-width:thin; scroll-padding:8px; }
.section-item { margin-bottom:6px; }
.section-header, .group-header { width:100%; display:flex; gap:8px; align-items:flex-start; min-height:44px; padding:11px 8px; text-align:left; border:0; border-radius:5px; background:transparent; cursor:pointer; color:var(--ink); }
.section-header { font-size:13px; font-weight:600; }
.section-header:hover:not(:disabled), .group-header:hover:not(:disabled) { background:var(--zone); }
.section-header:disabled, .group-header:disabled { cursor:default; }
.section-current > .section-header .section-title { color:var(--green-dark); }
.section-body { margin:1px 0 8px 9px; padding-left:8px; border-left:1px solid var(--line); }
.group-header { color:var(--secondary); font-size:12px; font-weight:500; padding-left:9px; }
.section-title, .group-title { min-width:0; flex:1; line-height:1.7; overflow-wrap:anywhere; }
.section-count { font-size:11px; font-weight:400; color:var(--secondary); flex:none; line-height:1.7; margin-top:1px; font-variant-numeric:tabular-nums; }
.icon-arrow { width:12px; height:12px; flex:none; margin-top:5px; color:var(--secondary); transition:transform .15s; }
.icon-arrow-expanded { transform:rotate(90deg); }
.article-list { display:flex; flex-direction:column; gap:2px; }
.article-item { position:relative; width:100%; min-height:44px; display:flex; align-items:center; text-align:left; padding:9px 10px; border:0; border-radius:5px; background:transparent; color:var(--secondary); font-size:13px; line-height:1.65; cursor:pointer; }
.article-list-nested { margin-left:8px; padding-left:8px; border-left:1px solid var(--line); }
.article-title { min-width:0; overflow-wrap:anywhere; }
.article-item:hover { background:var(--zone); color:var(--ink); }
.article-item-active { background:var(--selected); color:var(--green-dark); font-weight:600; }
.article-item-active:hover { background:var(--selected); color:var(--green-dark); }
.article-item-active::before { content:''; position:absolute; left:-9px; top:9px; bottom:9px; width:2px; border-radius:2px; background:var(--green); }
.directory-footer { flex:none; padding:7px 18px max(7px, env(safe-area-inset-bottom)); border-top:1px solid var(--line); }
.directory-footer a { display:flex; align-items:center; gap:9px; min-height:44px; font-size:12px; color:var(--secondary); }
.directory-footer a:hover { color:var(--green); }
.directory-footer svg { width:17px; height:17px; flex:none; }
.directory-empty { padding:12px 8px; font-size:12px; color:var(--secondary); }
.no-results { padding:12px 8px; font-size:13px; line-height:1.8; color:var(--secondary); }
.no-results button { display:block; margin-top:8px; }
.sidebar-drawer { display:block; width:100%; min-width:0; border:0; }
.sidebar-drawer .sidebar-content { width:100%; }
.sidebar-drawer .sidebar-head { padding-top:12px; }
.sidebar-drawer .sidebar-top { display:none; }
.sidebar-drawer .directory-search { height:44px; }
.sidebar-drawer .directory-search input { font-size:16px; }
.sidebar-drawer .article-item { font-size:14px; }
.sidebar-drawer .section-header { font-size:14px; }
.sidebar-drawer .group-header { font-size:13px; }
@media (min-width:901px) { .sidebar-root { display:block; } }
@media (prefers-reduced-motion:reduce) { .icon-arrow { transition:none; } }
</style>
