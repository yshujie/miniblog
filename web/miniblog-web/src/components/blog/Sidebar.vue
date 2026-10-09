<template>
  <aside :id="drawer ? 'drawer-sidebar' : 'sidebar'" class="sidebar-root"
    :class="{ 'sidebar-hidden': !sidebarOpen, 'sidebar-drawer': drawer }"
    :inert="!sidebarOpen ? true : undefined" :aria-hidden="!sidebarOpen ? true : undefined">
    <div class="sidebar-content">
      <div class="sidebar-head">
        <div class="sidebar-top">
          <router-link :to="topicURL" @click="uiStore.closeDrawer">{{ moduleLabel }}</router-link>
          <button v-if="!drawer" type="button" class="icon-button" aria-label="收起文章目录" aria-controls="sidebar" :aria-expanded="sidebarOpen" @click="hideSidebar">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" aria-hidden="true"><rect x="3" y="3" width="18" height="18" rx="2" /><path d="M9 3v18m8-10-3 3 3 3" /></svg>
          </button>
        </div>
        <div class="search-field directory-search">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" aria-hidden="true"><circle cx="10.5" cy="10.5" r="6.5" /><path d="m16 16 4 4" /></svg>
          <input v-model="query" type="search" aria-label="搜索本主题标题" placeholder="搜索本主题标题" />
          <button v-if="query" type="button" aria-label="清空标题搜索" @click="query = ''">×</button>
        </div>
      </div>
      <nav class="section-list" aria-label="文章目录">
        <p v-if="isSearching && !matches.length" class="no-results" role="status">没有匹配的文章标题。<button class="text-button" type="button" @click="query = ''">清空搜索</button></p>
        <div v-for="section in visibleSections" :key="section.id" class="section-item">
          <button type="button" class="section-header" :aria-expanded="isSectionExpanded(section.id)" @click="toggleSection(section.id)">
            <svg class="icon-arrow" :class="{ 'icon-arrow-expanded': isSectionExpanded(section.id) }" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" aria-hidden="true"><path d="m9 6 6 6-6 6" /></svg>
            <span class="section-title">{{ section.title }}</span>
          </button>
          <div v-show="isSectionExpanded(section.id)" class="section-body">
            <div v-for="subsection in section.subsections" :key="subsection.id" class="nav-group">
              <button type="button" class="group-header" :aria-expanded="isSubsectionExpanded(section.id, subsection.code)" @click="toggleSubsection(section.id, subsection.code)">
                <svg class="icon-arrow" :class="{ 'icon-arrow-expanded': isSubsectionExpanded(section.id, subsection.code) }" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" aria-hidden="true"><path d="m9 6 6 6-6 6" /></svg>
                <span class="group-title">{{ subsection.title }}</span>
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
      <div class="directory-footer"><router-link class="text-button" :to="topicURL" @click="uiStore.closeDrawer">查看主题总览</router-link></div>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useUiStore } from '@/stores/ui'
import type { Section } from '@/types/section'
import { filterCatalog, findCatalogEntry, flattenCatalog } from '@/util/catalog'
const props = defineProps<{ sections: Section[]; moduleCode: string; moduleTitle?: string; drawer?: boolean }>()
const router = useRouter()
const uiStore = useUiStore()
const query = ref('')
const expandedSectionIds = ref(new Set<string>())
const expandedSubsectionKeys = ref(new Set<string>())
const sidebarOpen = computed(() => props.drawer || uiStore.sidebarOpen)
const moduleLabel = computed(() => props.moduleTitle?.trim() || '文档导航')
const topicURL = computed(() => '/topics/' + encodeURIComponent(props.moduleCode))
const currentArticleId = computed(() => typeof router.currentRoute.value.params.article === 'string' ? router.currentRoute.value.params.article : null)
const entries = computed(() => flattenCatalog({ sections: props.sections }))
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
  const entry = currentArticleId.value ? findCatalogEntry(entries.value, currentArticleId.value) : undefined
  if (!entry) return
  expandedSectionIds.value = new Set([...expandedSectionIds.value, entry.section.id])
  if (entry.subsection) expandedSubsectionKeys.value = new Set([...expandedSubsectionKeys.value, subsectionExpandKey(entry.section.id, entry.subsection.code)])
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
.sidebar-root { display:none; width:280px; min-width:280px; height:100%; min-height:0; flex:none; overflow:hidden; background:var(--zone); border-right:1px solid var(--line); }
.sidebar-hidden { width:0; min-width:0; border-right:0; pointer-events:none; }
.sidebar-content { width:280px; height:100%; min-height:0; display:flex; flex-direction:column; overflow:hidden; }
.sidebar-head { padding:22px 20px 18px; flex:none; }
.sidebar-top { display:flex; align-items:center; justify-content:space-between; gap:12px; margin-bottom:14px; font-size:16px; font-weight:700; }
.sidebar-top a { min-width:0; overflow-wrap:anywhere; }
.sidebar-top .icon-button { width:36px; height:36px; min-height:36px; flex:none; border:0; }
.sidebar-top svg { width:18px; height:18px; }
.directory-search { width:100%; min-width:0; height:40px; }
.directory-search input { min-width:0; font-size:12px; }
.directory-search > svg { width:15px; height:15px; flex:none; }
.directory-search button { color:var(--secondary); font-size:20px; border:0; background:transparent; width:28px; height:32px; flex:none; cursor:pointer; }
.section-list { min-height:0; flex:1; overflow:auto; padding:0 12px 24px; scrollbar-width:thin; }
.section-item { margin-bottom:15px; }
.section-header, .group-header { width:100%; display:flex; gap:7px; align-items:flex-start; min-height:44px; padding:12px 7px; text-align:left; border:0; background:transparent; cursor:pointer; color:var(--ink); }
.section-header { font-size:13px; font-weight:600; }
.group-header { padding-left:18px; color:var(--secondary); font-size:12px; font-weight:500; }
.section-title, .group-title { min-width:0; flex:1; line-height:1.7; overflow-wrap:anywhere; }
.icon-arrow { width:13px; height:13px; flex:none; margin-top:5px; transition:transform .15s; }
.icon-arrow-expanded { transform:rotate(90deg); }
.article-list { display:flex; flex-direction:column; gap:3px; }
.article-item { position:relative; width:100%; min-height:44px; display:flex; text-align:left; padding:12px 12px 12px 28px; border:0; border-radius:5px; background:transparent; color:var(--ink); font-size:13px; line-height:1.65; cursor:pointer; }
.article-list-nested .article-item { padding-left:36px; }
.article-title { min-width:0; overflow-wrap:anywhere; }
.article-item:hover { background:var(--selected); color:var(--green); }
.article-item-active { background:var(--selected); color:var(--green); font-weight:600; }
.article-item-active::before { content:''; position:absolute; left:0; top:10px; bottom:10px; width:3px; border-radius:2px; background:var(--green); }
.directory-footer { flex:none; padding:14px 24px max(14px, env(safe-area-inset-bottom)); border-top:1px solid var(--line); font-size:13px; }
.no-results { padding:12px 8px; font-size:13px; line-height:1.8; color:var(--secondary); }
.no-results button { display:block; margin-top:8px; }
.sidebar-drawer { display:block; width:100%; min-width:0; border:0; }
.sidebar-drawer .sidebar-content { width:100%; }
.sidebar-drawer .sidebar-head { padding-top:14px; }
.sidebar-drawer .sidebar-top { display:none; }
.sidebar-drawer .directory-search { height:44px; }
.sidebar-drawer .directory-search input { font-size:16px; }
.sidebar-drawer .article-item { font-size:14px; }
@media (min-width:901px) { .sidebar-root { display:block; } }
@media (prefers-reduced-motion:reduce) { .icon-arrow { transition:none; } }
</style>
