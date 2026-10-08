<template>
  <section class="article-container" :aria-label="title">
    <div class="article-toolbar">
      <h1>{{ title }}</h1>
      <a v-if="sourceURL" :href="sourceURL" target="_blank" rel="noopener noreferrer">打开原文</a>
    </div>
    <div v-if="!sourceURL" class="link-message" role="status">
      文章链接不可用，请稍后再试或联系作者。
    </div>
    <template v-else>
      <div v-if="frameState === 'loading'" class="link-message" role="status">正在打开文章，可随时打开原文阅读。</div>
      <div v-else-if="frameState === 'waiting'" class="link-message" role="status">
        文章仍在加载。如无法显示，请打开原文阅读。
      </div>
      <iframe :key="frameKey" :data-frame-key="frameKey" :src="sourceURL" :title="title" class="article-iframe"
        referrerpolicy="no-referrer" @load="onFrameLoad" />
    </template>
  </section>
</template>

<script setup lang="ts">
import { computed, onUnmounted, ref, watch } from 'vue'
import type { Article } from '@/types/article'
import { articleTitle, safeExternalURL } from '@/util/reading'

const props = defineProps<{ article: Article }>()
const title = computed(() => articleTitle(props.article))
const sourceURL = computed(() => safeExternalURL(props.article.readingURL) || safeExternalURL(props.article.externalLink))
const frameKey = computed(() => props.article.id + ':' + sourceURL.value)
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
.article-container { display: flex; flex-direction: column; height: 100%; min-height: 0; background: var(--card-bg); }
.article-toolbar { display: flex; align-items: center; gap: 1rem; padding: .75rem 1rem; border-bottom: 1px solid var(--border-color); }
.article-toolbar h1 { flex: 1; min-width: 0; font-size: 1rem; font-weight: 600; overflow-wrap: anywhere; }
.article-toolbar a { flex-shrink: 0; color: var(--sidebar-active-color); }
.link-message { padding: .75rem 1rem; color: var(--text-secondary); font-size: .875rem; }
.article-iframe { flex: 1; width: 100%; min-height: 0; border: 0; background: var(--card-bg); }
</style>
