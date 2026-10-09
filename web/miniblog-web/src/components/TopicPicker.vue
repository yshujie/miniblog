<template>
  <el-drawer v-model="opened" title="浏览技术主题" direction="btt" size="auto" :with-header="false" append-to-body class="topic-picker" @opened="focusClose" @closed="restoreFocus">
    <div class="topic-picker-top"><h2 id="topic-picker-title">浏览技术主题</h2><button ref="closeButton" type="button" class="icon-button" aria-label="关闭主题选择" @click="ui.closeTopics">×</button></div>
    <nav aria-label="选择阅读主题">
      <router-link v-for="topic in store.modules" :key="topic.code" class="topic-choice" :class="{ active: route.params.module === topic.code }"
        :aria-current="route.params.module === topic.code ? 'page' : undefined" :to="{ name: 'TopicOverview', params: { module: topic.code } }" @click="ui.closeTopics">
        <div><strong>{{ topic.title }}</strong><p>{{ topicDescription(topic.code) }}</p></div><span aria-hidden="true">›</span>
      </router-link>
    </nav>
    <div v-if="store.listStatus === 'error'" class="picker-status" role="alert"><p>主题加载失败，请重试。</p><button type="button" class="text-button" @click="reloadModules">重试主题</button></div>
    <p v-else-if="store.listStatus === 'loading' && !store.modules.length" class="picker-status" role="status">正在加载主题…</p>
    <p v-else-if="store.listStatus === 'empty'" class="picker-status" role="status">暂无阅读模块</p>
    <p class="topic-picker-note">按主题进入目录，选择你想读的文章。</p>
  </el-drawer>
</template>
<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useUiStore } from '@/stores/ui'
import { useModuleStore } from '@/stores/module'
import { topicDescription } from '@/util/topic'
const ui = useUiStore(), store = useModuleStore(), route = useRoute()
const opened = computed({ get: () => ui.topicPickerOpen, set: (value: boolean) => { if (!value) ui.closeTopics() } })
const closeButton = ref<HTMLButtonElement>()
let trigger: HTMLElement | undefined, media: MediaQueryList | undefined
watch(() => ui.topicPickerOpen, value => { if (value && document.activeElement instanceof HTMLElement) trigger = document.activeElement })
watch(() => route.fullPath, () => ui.closeTopics())
const reloadModules = () => { void store.loadModules(true).catch(() => {}) }
const focusClose = () => { closeButton.value?.focus() }
async function restoreFocus() {
  await nextTick()
  if (ui.mobileDrawerOpen) return
  if (trigger?.isConnected && trigger.getClientRects().length) trigger.focus()
  else document.querySelector<HTMLElement>('.site-brand')?.focus()
}
function syncViewport() { if (media && !media.matches) ui.closeTopics() }
onMounted(() => {
  if (typeof window.matchMedia === 'function') { media = window.matchMedia('(max-width: 650px)'); media.addEventListener('change', syncViewport); syncViewport() }
})
onUnmounted(() => { media?.removeEventListener('change', syncViewport); ui.closeTopics() })
</script>
<style>
.topic-picker.el-drawer { max-height: 80vh; max-height: 80dvh; border-radius: 16px 16px 0 0; padding-bottom: env(safe-area-inset-bottom); }
.topic-picker .el-drawer__body { padding: 0; overflow-y: auto; }
.topic-picker-top { display: flex; justify-content: space-between; align-items: center; gap: 16px; padding: 18px 24px; border-bottom: 1px solid var(--line); }
.topic-picker-top h2 { font-size: 18px; } .topic-picker-top .icon-button { border: 0; font-size: 26px; }
.topic-choice { display: flex; align-items: center; justify-content: space-between; padding: 23px 24px; border-bottom: 1px solid var(--line); gap: 20px; overflow-wrap: anywhere; }
.topic-choice strong { display: block; font-size: 20px; margin-bottom: 8px; font-weight: 600; } .topic-choice p { font-size: 13px; line-height: 1.8; color: var(--secondary); } .topic-choice > span { font-size: 22px; }
.topic-choice.active { color: var(--green); background: var(--selected); } .topic-choice:hover { background: var(--zone); }
.topic-picker-note, .picker-status { font-size: 12px; color: var(--secondary); padding: 18px 24px; }
</style>
