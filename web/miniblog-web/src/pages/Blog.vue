<template>
  <BlogLayout>
    <template #sidebar>
      <Sidebar :module="theModule" :sections="theModule.sections" />
    </template>
    <template #main>
      <!-- <ArticleCard :article="theArticle" /> -->
    </template>
  </BlogLayout>
</template>

<script setup lang="ts">
import { onMounted, watch, computed } from 'vue'
import { useRoute } from 'vue-router'
import { useModuleStore } from '@/stores/module'

import BlogLayout from '../components/blog/BlogLayout.vue'
import Sidebar from '../components/blog/Sidebar.vue'
import ArticleCard from '../components/blog/ArticleCard.vue'

// 获取路由对象
const route = useRoute()

// module store
const moduleStore = useModuleStore()

// 当前 module
const theModule = computed(() => { 
  const moduleCode = queryModuleCode()
  const module = moduleStore.getModuleByCode(moduleCode)
  if (!module) {
    throw new Error(`Module with code ${moduleCode} not found`)
  }
  return module
})

// 组件挂载时加载数据
onMounted(loadModuleData)

// 监听路由参数变化
watch(
  () => route.params.module,
  async (newModuleCode) => {
    if (newModuleCode) {
      await loadModuleData()
    }
  }
)

// 获取 moduleCode
function queryModuleCode() {
  const moduleCode = route.params.module as string
  if (!moduleCode) {
    throw new Error('moduleCode is required')
  }
  return moduleCode
}

// 加载模块数据
async function loadModuleData() {
  const moduleCode = queryModuleCode()
  await moduleStore.loadSections(moduleCode)
  await moduleStore.loadArticles(moduleCode)
}

</script>
