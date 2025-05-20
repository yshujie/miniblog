<template>
  <BlogLayout>
    <template #sidebar>
      <Sidebar :sections="sections" :moduleCode="moduleCode" />
    </template>
    <template #main>
      <ArticleCard :articleId="chosenArticleId" />
    </template>
  </BlogLayout>
</template>

<script setup lang="ts">
import { onMounted, watch, computed, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'
import { useModuleStore } from '@/stores/module'

import BlogLayout from '../components/blog/BlogLayout.vue'
import Sidebar from '../components/blog/Sidebar.vue'
import ArticleCard from '../components/blog/ArticleCard.vue'

// 获取路由对象
const route = useRoute()

// module store
const moduleStore = useModuleStore()

// 计算属性 sections
const sections = computed(() => {
  return moduleStore.currentModule?.sections || []
})

// 计算属性 moduleCode
const moduleCode = computed(() => {
  return moduleStore.currentModule?.code || ''
})

// 计算属性 chosenArticleId
const chosenArticleId = computed(() => {
  return queryArticleId()
})

// 组件挂载时，设置当前模块
onMounted(() => {
  const moduleCode = queryModuleCode()
  const module = moduleStore.getModuleByCode(moduleCode)
  if (!module) {
    throw new Error(`Module with code ${moduleCode} not found`)
  }

  // 设置当前模块
  moduleStore.setCurrentModule(module)
})

// 组件卸载时，清除当前模块
onUnmounted(() => {
  moduleStore.clearCurrentModule()
})

// 获取 moduleCode
function queryModuleCode() {
  const moduleCode = route.params.module as string
  if (!moduleCode) {
    throw new Error('moduleCode is required')
  }
  return moduleCode
}


// 获取 articleId
function queryArticleId(): number | null {
  const articleId = route.params.article as string
  if (!articleId) {
    console.log('articleId is not found')
    return null
  }
  return parseInt(articleId)
}

</script>
