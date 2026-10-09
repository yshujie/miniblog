<template>
  <BlogLayout>
    <template #sidebar>
      <Sidebar :sections="state.module?.sections || []" :module-code="moduleCode" :module-title="state.module?.title" />
    </template>
    <template #drawer>
      <Sidebar drawer :sections="state.module?.sections || []" :module-code="moduleCode" :module-title="state.module?.title" />
    </template>
    <template #main>
      <ContentState v-if="state.status === 'loading'" status="loading" :title="'正在加载' + (state.resource === 'article' ? '文章' : '目录') + '…'" />
      <ContentState v-else-if="state.status === 'empty'" status="empty" :title="state.module?.title" message="暂无已发布文章" retry-label="刷新目录" @retry="retry">
        <template #actions><router-link class="secondary-button" :to="topicURL">查看主题总览</router-link></template>
      </ContentState>
      <ContentState v-else-if="state.status === 'not_found' || state.status === 'error'" :status="state.status"
        :title="state.status === 'not_found' ? (state.resource === 'article' ? '文章不可用' : '模块不可用') : '加载失败'"
        :message="state.message" retry-label="重试" @retry="retry">
        <template #actions><router-link class="secondary-button" to="/">返回首页</router-link></template>
      </ContentState>
      <div v-else-if="state.article" class="reading-content">
        <div v-if="state.refreshError" class="refresh-warning" role="alert">{{ state.refreshError }}<button type="button" class="text-button" @click="retry">重试更新</button></div>
        <ExternalArticleCard :article="state.article" :module="state.module || undefined" :placement="placement" :previous="neighbors.previous" :next="neighbors.next" />
      </div>
    </template>
  </BlogLayout>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useReaderPage } from '@/composables/useReaderPage'
import { catalogNeighbors, findCatalogEntry, flattenCatalog } from '@/util/catalog'
import BlogLayout from '@/components/blog/BlogLayout.vue'
import Sidebar from '@/components/blog/Sidebar.vue'
import ExternalArticleCard from '@/components/blog/ExternalArticleCard.vue'
import ContentState from '@/components/ContentState.vue'
const route = useRoute()
const { state, retry } = useReaderPage()
const moduleCode = computed(() => state.module?.code || String(route.params.module || ''))
const topicURL = computed(() => '/topics/' + encodeURIComponent(moduleCode.value))
const entries = computed(() => flattenCatalog({ sections: state.module?.sections || [] }))
const placement = computed(() => state.article ? findCatalogEntry(entries.value, state.article.id) : undefined)
const neighbors = computed(() => catalogNeighbors(entries.value, state.article?.id || ''))
</script>

<style scoped>
.reading-content { display:flex; flex-direction:column; flex:1; height:100%; min-height:0; min-width:0; }
.reading-content :deep(.article-container) { flex:1; height:auto; }
.refresh-warning { display:flex; flex-wrap:wrap; align-items:center; justify-content:space-between; gap:8px; padding:10px 32px; background:#fff8e8; color:var(--secondary); font-size:12px; flex:none; }
.refresh-warning button { flex:none; min-height:36px; font-size:12px; }
@media (max-width:650px) { .refresh-warning { padding:10px 20px; } }
</style>
