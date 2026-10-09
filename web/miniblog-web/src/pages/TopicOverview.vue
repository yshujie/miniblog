<template>
  <div class="topic-main">
    <nav class="breadcrumbs" aria-label="当前位置">
      <router-link to="/">首页</router-link>
      <span aria-hidden="true">›</span>
      <span>{{ state.module?.title || '主题目录' }}</span>
    </nav>

    <div v-if="state.refreshError" class="refresh-warning" role="alert">
      <p>{{ state.refreshError }}</p>
      <button type="button" :disabled="state.busy" @click="retry">重试更新</button>
    </div>

    <section v-if="state.module" class="topic-hero" aria-labelledby="topic-title">
      <div>
        <h1 id="topic-title">{{ state.module.title }}</h1>
        <p>{{ topicDescription }}</p>
        <p class="topic-stats">{{ state.module.sections.length }} 个章节，{{ entries.length }} 篇文章</p>
      </div>
      <router-link v-if="entries.length" class="primary from-first"
        :to="{ name: 'BlogModule', params: { module: state.module.code } }">
        从第一篇开始 <span aria-hidden="true">↗</span>
      </router-link>
    </section>

    <div v-if="state.status === 'success' && state.module" class="topic-catalog">
      <nav class="chapter-index" aria-label="本主题章节">
        <h2>本主题章节</h2>
        <router-link v-for="section in state.module.sections" :key="section.id"
          :class="{ active: section.id === activeChapter?.id }"
          :aria-current="section.id === activeChapter?.id ? 'location' : undefined"
          :to="{ name: 'TopicOverview', params: { module: state.module.code }, query: { ...route.query, chapter: section.code }, hash: route.hash }">
          {{ section.title }}
        </router-link>
      </nav>

      <div class="topic-content">
        <div class="catalog-tools">
          <div class="search-field">
            <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="1.8" aria-hidden="true">
              <circle cx="10.5" cy="10.5" r="6.5" /><path d="m16 16 5 5" />
            </svg>
            <label class="sr-only" for="topic-title-search">搜索本主题文章标题</label>
            <input id="topic-title-search" ref="searchInput" v-model="query" type="search"
              placeholder="搜索本主题文章标题" autocomplete="off" />
          </div>
          <p class="search-count" role="status" aria-live="polite">
            {{ searching ? `找到 ${matches.length} 篇文章` : '按目录顺序排列' }}
          </p>
        </div>

        <section v-for="chapter in visibleChapters" :id="chapterID(chapter.section.id)"
          :key="chapter.section.id" class="chapter-section" :aria-labelledby="`title-${chapter.section.id}`">
          <div class="chapter-title">
            <h2 :id="`title-${chapter.section.id}`">{{ chapter.section.title }}</h2>
            <span>{{ chapter.entries.length }} 篇</span>
          </div>
          <template v-for="group in chapter.groups" :key="group.subsection?.id || 'direct'">
            <h3 v-if="group.subsection" class="subsection-title">{{ group.subsection.title }}</h3>
            <router-link v-for="entry in group.entries" :key="entry.article.id" class="article-row"
              :to="{ name: 'BlogArticle', params: { module: state.module.code, article: entry.article.id } }">
              <div class="row-copy">
                <span class="row-title">{{ articleTitle(entry.article) }}</span>
                <div class="row-meta">
                  <span>Notion 文档</span>
                  <span v-for="tag in entry.article.tags.slice(0, 2)" :key="tag">{{ tag }}</span>
                </div>
              </div>
              <span class="row-arrow" aria-hidden="true">›</span>
            </router-link>
          </template>
          <p v-if="!chapter.entries.length" class="empty-chapter">本章节暂无已发布文章</p>
        </section>

        <div v-if="searching && !matches.length" class="no-results" role="status">
          <h2>没有找到匹配的文章标题</h2>
          <p>试试更短的关键词，或清空搜索查看本主题目录。</p>
          <button type="button" class="text-button clear-search" @click="clearSearch">清空搜索</button>
        </div>
      </div>
    </div>

    <ContentState v-else :status="state.status" :title="stateTitle" :message="stateMessage"
      :retry-label="state.status === 'empty' ? '刷新目录' : '重试'" @retry="retry">
      <template #actions><router-link to="/" class="secondary-button">返回首页</router-link></template>
    </ContentState>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import ContentState from '@/components/ContentState.vue'
import { useTopicPage } from '@/composables/useTopicPage'
import { filterCatalog, flattenCatalog } from '@/util/catalog'
import { articleTitle } from '@/util/reading'
import { topicDescription as describeTopic } from '@/util/topic'

const route = useRoute()
const { state, retry } = useTopicPage()
const query = ref('')
const searchInput = ref<HTMLInputElement>()
const entries = computed(() => state.module ? flattenCatalog(state.module) : [])
const matches = computed(() => filterCatalog(entries.value, query.value))
const searching = computed(() => Boolean(query.value.trim()))
const chapterQuery = computed(() => typeof route.query.chapter === 'string' ? route.query.chapter : '')
const activeChapter = computed(() => state.module?.sections.find(section => section.code === chapterQuery.value) || state.module?.sections[0])
const topicDescription = computed(() => describeTopic(state.module?.code || ''))
const visibleChapters = computed(() => (state.module?.sections || []).map(section => {
  const chapterEntries = matches.value.filter(entry => entry.section.id === section.id)
  const groups = [
    ...section.subsections.map(subsection => ({
      subsection,
      entries: chapterEntries.filter(entry => entry.subsection?.id === subsection.id),
    })),
    { subsection: undefined, entries: chapterEntries.filter(entry => !entry.subsection) },
  ].filter(group => group.entries.length)
  return { section, entries: chapterEntries, groups }
}).filter(chapter => !searching.value || chapter.entries.length))

const stateTitle = computed(() => ({
  loading: '正在加载主题目录…',
  empty: '暂无已发布文章',
  not_found: '主题不可用',
  error: '加载失败',
  success: '',
})[state.status])
const stateMessage = computed(() => state.status === 'empty'
  ? '可以稍后回来，或浏览其他主题。'
  : state.message || (state.status === 'loading' ? '目录加载完成后，可以按章节选择文章。' : ''))
const chapterID = (id: string) => `chapter-${id}`
function clearSearch() {
  query.value = ''
  searchInput.value?.focus()
}
watch(() => route.params.module, () => { query.value = '' })
watch([() => state.module?.code, () => activeChapter.value?.id, chapterQuery], async () => {
  const id = activeChapter.value?.id
  const code = state.module?.code
  if (!chapterQuery.value || !id) return
  await nextTick()
  if (state.module?.code !== code || activeChapter.value?.id !== id || !chapterQuery.value) return
  document.getElementById(chapterID(id))?.scrollIntoView?.({ block: 'start' })
})
</script>

<style scoped>
.topic-main { max-width: 1160px; width: 100%; margin: auto; padding: 42px 32px 60px; }
.breadcrumbs { display: flex; align-items: center; flex-wrap: wrap; gap: 9px; min-height: 28px; font-size: 12px; color: var(--text-secondary); }
.breadcrumbs a { color: inherit; }
.topic-hero { display: flex; align-items: flex-end; justify-content: space-between; gap: 32px; padding: 28px 0 30px; border-bottom: 1px solid var(--border-color); }
.topic-hero > div { min-width: 0; }
.topic-hero h1 { font-family: var(--display); font-size: 40px; line-height: 1.4; letter-spacing: -1px; overflow-wrap: anywhere; }
.topic-hero p { max-width: 650px; margin-top: 14px; color: var(--text-secondary); font-size: 15px; }
.topic-hero .topic-stats { margin-top: 16px; font-size: 13px; }
.from-first { flex-shrink: 0; }
.topic-catalog { display: grid; grid-template-columns: 210px minmax(0, 1fr); gap: 52px; padding-top: 30px; }
.chapter-index { position: sticky; top: 24px; align-self: start; }
.chapter-index h2 { margin-bottom: 12px; font-size: 13px; color: var(--text-secondary); }
.chapter-index a { display: flex; align-items: center; min-height: 44px; padding: 8px 14px; border-left: 2px solid var(--border-color); color: var(--text-primary); font-size: 13px; overflow-wrap: anywhere; }
.chapter-index a.active { border-left-color: var(--accent); color: var(--accent); background: var(--sidebar-active-bg, #eaf5f1); }
.chapter-index a:hover { color: var(--accent); background: var(--zone-bg); }
.topic-content { min-width: 0; }
.catalog-tools { display: flex; justify-content: space-between; align-items: center; gap: 20px; margin-bottom: 26px; }
.search-field { display: flex; align-items: center; gap: 9px; height: 44px; width: 300px; max-width: 100%; padding: 0 12px; border: 1px solid var(--border-color); border-radius: 6px; background: var(--page-bg); }
.search-field svg { flex-shrink: 0; color: var(--text-secondary); }
.search-field input { width: 100%; min-width: 0; border: 0; outline: none; background: transparent; color: var(--text-primary); font: inherit; font-size: 14px; }
.search-field:focus-within { outline: 2px solid var(--accent); outline-offset: 2px; }
.search-field input::placeholder, .search-count { color: var(--text-secondary); }
.search-count { font-size: 13px; flex-shrink: 0; }
.chapter-section { margin-bottom: 30px; scroll-margin-top: 24px; }
.chapter-title { display: flex; align-items: baseline; justify-content: space-between; gap: 16px; margin-bottom: 12px; }
.chapter-title h2 { font-size: 19px; line-height: 1.6; font-weight: 600; overflow-wrap: anywhere; }
.chapter-title > span { flex-shrink: 0; font-size: 12px; color: var(--text-secondary); }
.subsection-title { padding: 10px 0 4px; font-size: 13px; color: var(--text-secondary); font-weight: 500; overflow-wrap: anywhere; }
.article-row { display: flex; align-items: center; justify-content: space-between; gap: 20px; min-height: 76px; padding: 14px 0; border-bottom: 1px solid var(--border-color); color: var(--text-primary); }
.row-copy { min-width: 0; }
.row-title { display: block; font-size: 15px; font-weight: 500; line-height: 1.6; overflow-wrap: anywhere; }
.article-row:hover .row-title { color: var(--accent); }
.row-meta { display: flex; flex-wrap: wrap; align-items: center; gap: 8px 14px; margin-top: 6px; font-size: 12px; color: var(--text-secondary); overflow-wrap: anywhere; }
.row-arrow { flex-shrink: 0; font-size: 22px; color: var(--text-secondary); }
.empty-chapter { padding: 18px 0; font-size: 13px; color: var(--text-secondary); }
.no-results { padding: 32px 0; color: var(--text-secondary); }
.no-results h2 { margin-bottom: 8px; font-size: 18px; color: var(--text-primary); }
.no-results p { margin-bottom: 12px; line-height: 1.8; }
.refresh-warning { display: flex; align-items: center; justify-content: space-between; gap: 16px; margin-top: 18px; padding: 12px 16px; background: var(--warning-bg); color: var(--warning); font-size: 13px; }
.refresh-warning button { flex-shrink: 0; min-height: 44px; background: none; border: 0; color: inherit; text-decoration: underline; font: inherit; cursor: pointer; }
@media (max-width: 1100px) { .topic-catalog { grid-template-columns: 180px minmax(0, 1fr); gap: 34px; } }
@media (max-width: 900px) { .topic-catalog { grid-template-columns: 150px minmax(0, 1fr); gap: 28px; } .catalog-tools { flex-wrap: wrap; gap: 12px; } }
@media (max-width: 650px) {
  .topic-main { padding: 24px 24px 40px; }
  .breadcrumbs { font-size: 11px; }
  .topic-hero { display: block; padding: 20px 0 24px; }
  .topic-hero h1 { font-size: 34px; }
  .topic-hero p { font-size: 14px; line-height: 1.9; margin-top: 12px; }
  .topic-hero .topic-stats { margin: 14px 0 20px; font-size: 12px; }
  .topic-catalog { display: block; padding-top: 24px; }
  .chapter-index { position: static; display: flex; gap: 8px; max-width: 100%; overflow-x: auto; padding-bottom: 4px; margin-bottom: 24px; }
  .chapter-index h2 { display: none; }
  .chapter-index a { flex-shrink: 0; border: 1px solid var(--border-color); border-radius: 6px; min-height: 44px; padding: 8px 12px; max-width: 240px; font-size: 12px; }
  .chapter-index a.active { border-color: var(--accent); }
  .catalog-tools { margin-bottom: 24px; }
  .search-field { width: 100%; }
  .search-field input { font-size: 16px; }
  .chapter-title h2 { font-size: 18px; }
  .row-title { font-size: 14px; }
  .article-row { min-height: 84px; gap: 12px; }
  .row-meta { font-size: 11px; }
  .refresh-warning { align-items: start; }
}
@media (max-width: 360px) { .topic-main { padding-left: 20px; padding-right: 20px; } .topic-hero h1 { font-size: 30px; } }
</style>
