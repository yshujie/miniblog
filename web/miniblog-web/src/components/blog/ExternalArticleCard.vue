<template>
  <section class="article-container" :aria-label="title">
    <ReadingToolbar :article="article" :title="title" :sourceURL="sourceURL" :module="module" :placement="placement" />
    <div v-if="!sourceURL" class="link-message link-unavailable" role="status">
      <h2>文章链接不可用</h2><p>请稍后再试，或从主题目录选择其他文章。</p>
    </div>
    <template v-else>
      <div v-if="frameState === 'loading'" class="link-message" role="status">正在打开文章，可随时打开原文阅读。</div>
      <div v-else-if="frameState === 'waiting'" class="link-message" role="status">文章仍在加载。如无法显示，请打开原文阅读。</div>
      <div class="embed-container">
        <iframe :key="frameKey" :data-frame-key="frameKey" :src="sourceURL" :title="title" class="article-iframe"
          referrerpolicy="no-referrer" @load="onFrameLoad" />
      </div>
    </template>
    <nav v-if="module && placement" class="reading-pagination" aria-label="文章翻页">
      <router-link v-if="previous" class="previous" :to="articleRoute(previous)">
        <span aria-hidden="true">‹</span><span><span class="page-label">上一篇</span>{{ articleTitle(previous.article) }}</span>
      </router-link>
      <router-link v-else class="previous" :to="topicURL"><span aria-hidden="true">‹</span><span><span class="page-label">本主题</span>返回文章目录</span></router-link>
      <router-link v-if="next" class="next" :to="articleRoute(next)">
        <span><span class="page-label">下一篇</span>{{ articleTitle(next.article) }}</span><span aria-hidden="true">›</span>
      </router-link>
      <router-link v-else class="next" :to="topicURL"><span><span class="page-label">本主题</span>查看全部文章</span><span aria-hidden="true">›</span></router-link>
    </nav>
  </section>
</template>

<script setup lang="ts">
import { computed, onUnmounted, ref, watch } from 'vue'
import type { Article } from '@/types/article'
import type { Module } from '@/types/module'
import type { CatalogEntry } from '@/util/catalog'
import { articleTitle, safeExternalURL } from '@/util/reading'
import ReadingToolbar from './ReadingToolbar.vue'
const props = defineProps<{ article: Article; module?: Module; placement?: CatalogEntry; previous?: CatalogEntry; next?: CatalogEntry }>()
const title = computed(() => articleTitle(props.article))
const sourceURL = computed(() => safeExternalURL(props.article.readingURL) || safeExternalURL(props.article.externalLink))
const frameKey = computed(() => props.article.id + ':' + sourceURL.value)
const topicURL = computed(() => '/topics/' + encodeURIComponent(props.module?.code || ''))
const articleRoute = (entry: CatalogEntry) => ({ name: 'BlogArticle', params: { module: props.module?.code, article: entry.article.id } })
const frameState = ref<'loading' | 'load_event' | 'waiting'>('loading')
let timeout: ReturnType<typeof setTimeout> | undefined
function clearTimer() { clearTimeout(timeout); timeout = undefined }
watch(frameKey, () => {
  clearTimer()
  frameState.value = 'loading'
  if (sourceURL.value) timeout = setTimeout(() => { frameState.value = 'waiting' }, 10000)
}, { immediate: true })
// A cross-origin load event does not prove its document is readable.
function onFrameLoad(event: Event) {
  if ((event.target as HTMLIFrameElement).dataset.frameKey !== frameKey.value) return
  clearTimer()
  frameState.value = 'load_event'
}
onUnmounted(clearTimer)
</script>

<style scoped>
.article-container { display:flex; flex-direction:column; height:100%; min-height:0; min-width:0; background:var(--page); overflow:hidden; }
.link-message { padding:10px 32px; color:var(--secondary); font-size:12px; background:var(--zone); border-bottom:1px solid var(--line); flex:none; }
.link-unavailable { flex:1; display:flex; flex-direction:column; justify-content:center; align-items:center; text-align:center; gap:12px; overflow:auto; }
.link-unavailable h2 { font-size:23px; color:var(--ink); }
.link-unavailable p { max-width:440px; line-height:1.8; }
.embed-container { display:flex; flex:1; min-height:0; min-width:0; padding:18px 28px; background:var(--zone); }
.article-iframe { flex:1; width:100%; height:100%; min-height:0; min-width:0; border:1px solid var(--line); border-radius:5px; background:var(--page); }
.reading-pagination { display:flex; flex:none; align-items:center; justify-content:space-between; padding:10px 32px max(10px, env(safe-area-inset-bottom)); min-height:66px; border-top:1px solid var(--line); gap:20px; }
.reading-pagination a { display:flex; align-items:center; gap:12px; min-height:44px; min-width:0; max-width:47%; color:var(--secondary); font-size:13px; line-height:1.7; overflow-wrap:anywhere; }
.reading-pagination a:hover { color:var(--green); }
.reading-pagination .next { text-align:right; margin-left:auto; }
.page-label { display:block; font-size:11px; }
@media (max-width:1100px) { .embed-container { padding:16px; } .link-message { padding-left:24px; padding-right:24px; } }
@media (max-width:650px) {
  .embed-container { padding:12px; }
  .link-message { padding:10px 20px; }
  .reading-pagination { padding:8px 20px max(8px, env(safe-area-inset-bottom)); min-height:64px; gap:12px; }
  .reading-pagination a { font-size:11px; gap:7px; line-height:1.6; }
  .page-label { font-size:10px; }
}
@media (max-width:360px) { .reading-pagination { padding-left:16px; padding-right:16px; } }
@media (max-height:560px) {
  .embed-container { padding:6px; }
  .reading-pagination { align-items:flex-start; min-height:0; max-height:56px; padding-top:4px; padding-bottom:max(4px, env(safe-area-inset-bottom)); overflow:auto; scrollbar-width:thin; }
  .link-message:not(.link-unavailable) { max-height:40px; overflow:auto; padding-top:6px; padding-bottom:6px; }
}
</style>
