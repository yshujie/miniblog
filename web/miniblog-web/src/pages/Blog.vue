<template>
  <BlogLayout>
    <template #sidebar>
      <Sidebar :sections="state.module?.sections || []"
        :module-code="state.module?.code || String(route.params.module)"
        :module-title="state.module?.title" />
    </template>
    <template #drawer>
      <Sidebar drawer :sections="state.module?.sections || []"
        :module-code="state.module?.code || String(route.params.module)"
        :module-title="state.module?.title" />
    </template>
    <template #main>
      <div v-if="state.status === 'loading'" class="reading-state" role="status" aria-live="polite">
        正在加载{{ state.resource === 'article' ? '文章' : '目录' }}…
      </div>
      <div v-else-if="state.status === 'empty'" class="reading-state" role="status">
        <h1>{{ state.module?.title }}</h1>
        <p>暂无已发布文章</p>
        <el-button @click="retry">刷新目录</el-button>
      </div>
      <div v-else-if="state.status === 'not_found' || state.status === 'error'"
        class="reading-state" role="alert">
        <h1>{{ state.status === 'not_found' ? (state.resource === 'article' ? '文章不可用' : '模块不可用') : '加载失败' }}</h1>
        <p>{{ state.message }}</p>
        <el-button type="primary" @click="retry">重试</el-button>
        <router-link to="/">返回首页</router-link>
      </div>
      <div v-else-if="state.article" class="reading-content">
        <div v-if="state.refreshError" class="refresh-warning" role="alert">{{ state.refreshError }} <button @click="retry">重试更新</button></div>
        <ExternalArticleCard :article="state.article" />
      </div>
    </template>
  </BlogLayout>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router'
import { useReaderPage } from '@/composables/useReaderPage'
import BlogLayout from '@/components/blog/BlogLayout.vue'
import Sidebar from '@/components/blog/Sidebar.vue'
import ExternalArticleCard from '@/components/blog/ExternalArticleCard.vue'

const route = useRoute()
const { state, retry } = useReaderPage()
</script>

<style scoped>
.reading-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 1rem;
  height: 100%;
  min-height: 320px;
  padding: 2rem;
  text-align: center;
  background: var(--card-bg);
}
.reading-content { display:flex; flex-direction:column; height:100%; min-height:0; }
.reading-content :deep(.article-container) { flex:1; height:auto; }
.refresh-warning { padding: .5rem 1rem; background: var(--card-bg); color: var(--text-secondary); }
.reading-state h1 { font-size: 1.5rem; font-weight: 600; }
.reading-state a { color: var(--sidebar-active-color); }
</style>
