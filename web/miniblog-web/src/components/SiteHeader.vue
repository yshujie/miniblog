<template>
  <header class="site-header">
    <div class="site-header-inner">
      <router-link to="/" class="site-brand" aria-label="Shujie's Blog 首页"><img :src="logo" alt=""><span>Shujie's Blog</span></router-link>
      <nav class="site-nav" aria-label="主题导航">
        <router-link to="/" :class="{ active: route.name === 'Home' || route.path === '/' }" :aria-current="route.path === '/' ? 'page' : undefined">首页</router-link>
        <router-link v-for="topic in store.modules" :key="topic.code" :to="{ name: 'TopicOverview', params: { module: topic.code } }"
          :class="{ active: route.params.module === topic.code }" :aria-current="route.params.module === topic.code ? 'page' : undefined">{{ topic.title }}</router-link>
        <button v-if="store.listStatus === 'error'" type="button" class="text-button" @click="reloadModules">重试主题</button>
      </nav>
      <a class="github-link" href="https://github.com/yshujie" target="_blank" rel="noopener noreferrer">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M9 19c-4 1-4-2-6-2m12 4v-4a3 3 0 0 0-1-2c3-.3 6-1.4 6-5a4 4 0 0 0-1-3 4 4 0 0 0-.1-3S17 3.7 15 5a12 12 0 0 0-6 0C7 3.7 5.1 4 5.1 4A4 4 0 0 0 5 7a4 4 0 0 0-1 3c0 3.6 3 4.7 6 5a3 3 0 0 0-1 2v4"/></svg>GitHub
      </a>
      <button type="button" class="mobile-topic-button" aria-label="打开主题选择" aria-haspopup="dialog" :aria-expanded="ui.topicPickerOpen" @click="ui.openTopics">
        <span>{{ currentTopic?.title || '浏览主题' }}</span><span aria-hidden="true">⌄</span>
      </button>
    </div>
    <TopicPicker />
  </header>
</template>
<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import logo from '@/assets/logo.jpeg'
import { useReadingNavigation } from '@/composables/useReadingNavigation'
import { useUiStore } from '@/stores/ui'
import TopicPicker from './TopicPicker.vue'
const route = useRoute(), ui = useUiStore()
const { store, reloadModules } = useReadingNavigation()
const currentTopic = computed(() => store.modules.find(topic => topic.code === route.params.module))
</script>
<style scoped>
.site-header { height: var(--site-header-height); flex: none; border-bottom: 1px solid var(--line); background: var(--page); position: relative; z-index: 20; }
.site-header-inner { max-width: 1240px; height: 100%; margin: auto; padding: 0 32px; display: flex; align-items: center; gap: 42px; }
.site-brand { display: flex; align-items: center; gap: 12px; flex: none; font-family: var(--display); font-size: 23px; letter-spacing: -.5px; font-weight: 700; }
.site-brand img { width: 34px; height: 34px; object-fit: cover; border-radius: 10px; }
.site-nav { display: flex; gap: 28px; flex: 1; min-width: 0; height: 100%; overflow-x: auto; }
.site-nav a { height: 100%; display: flex; align-items: center; flex: none; position: relative; white-space: nowrap; color: var(--secondary); font-size: 14px; }
.site-nav a.active { color: var(--green); font-weight: 600; } .site-nav a.active::after { content: ''; position: absolute; bottom: 0; height: 3px; left: 0; right: 0; border-radius: 3px 3px 0 0; background: var(--green); }
.site-nav a:hover, .github-link:hover { color: var(--green); } .site-nav button { white-space: nowrap; }
.github-link { display: flex; align-items: center; gap: 7px; flex: none; font-size: 13px; color: var(--secondary); } .github-link svg { width: 18px; height: 18px; }
.mobile-topic-button { display: none; }
@media (max-width: 900px) { .site-header-inner { padding: 0 24px; gap: 24px; } .site-brand { font-size: 20px; gap: 9px; } .site-nav { gap: 18px; } }
@media (max-width: 650px) { .site-header-inner { padding: 0 20px; gap: 14px; justify-content: space-between; } .site-nav, .github-link { display: none; } .site-brand { font-size: 20px; gap: 10px; } .site-brand img { width: 30px; height: 30px; border-radius: 9px; } .mobile-topic-button { display: flex; align-items: center; gap: 7px; min-height: 44px; min-width: 0; max-width: 43%; border: 1px solid var(--line); border-radius: 6px; font-size: 13px; padding: 0 12px; color: var(--ink); background: var(--page); } .mobile-topic-button span:first-child { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; } }
@media (max-width: 360px) { .site-header-inner { padding: 0 16px; gap: 10px; } .site-brand { font-size: 18px; gap: 8px; } .mobile-topic-button { padding: 0 9px; } }
</style>
