<template>
  <!-- 加载状态 -->
  <div v-if="isLoading" class="loading-container">
    <el-loading :loading="true" text="正在加载模块数据..." />
  </div>
  
  <!-- 错误状态 -->
  <div v-else-if="hasError" class="error-container">
    <el-result
      icon="warning"
      title="模块不存在"
      :sub-title="`找不到模块 '${route.params.module}'，请检查URL是否正确`">
      <template #extra>
        <el-button type="primary" @click="$router.push('/')">返回首页</el-button>
      </template>
    </el-result>
  </div>
  
  <!-- 正常内容 -->
  <BlogLayout v-else>
    <template #sidebar>
      <Sidebar :sections="sections" :moduleCode="moduleCode" />
    </template>
    <template #main>
      <ExternalArticleCard :articleId="chosenArticleId" />
    </template>
  </BlogLayout>
</template>

<script setup lang="ts">
import { onMounted, watch, computed, onUnmounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { useModuleStore } from '@/stores/module'

import BlogLayout from '../components/blog/BlogLayout.vue'
import Sidebar from '../components/blog/Sidebar.vue'
import ExternalArticleCard from '../components/blog/ExternalArticleCard.vue'

// 获取路由对象
const route = useRoute()

// module store
const moduleStore = useModuleStore()

// 加载状态
const isLoading = ref(false)
const hasError = ref(false)

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
onMounted(async () => {
  await setCurrentModule(queryModuleCode()) 
})

// 监听路由变化
watch(() => route.params.module, async (newModuleCode) => {
  if (newModuleCode && typeof newModuleCode === 'string') {
    await setCurrentModule(newModuleCode)
  }
}, { immediate: true })

// 组件卸载时，清除当前模块
onUnmounted(() => {
  moduleStore.clearCurrentModule()
})

// 设置当前模块
async function setCurrentModule(moduleCode: string) {
  try {
    isLoading.value = true
    hasError.value = false
    
    // 先确保模块列表已加载
    if (moduleStore.modules.length === 0) {
      console.log('⏳ 加载模块列表...')
      await moduleStore.loadModules()
    }
    
    let module = moduleStore.getModuleByCode(moduleCode)
    if (!module) {
      // 如果还是找不到，可能是模块代码不存在
      console.error(`❌ 模块 "${moduleCode}" 不存在`)
      hasError.value = true
      isLoading.value = false
      return
    }
    
    // 如果模块存在但没有详细信息，加载详细信息
    if (!module.sections || module.sections.length === 0) {
      console.log(`⏳ 加载模块 "${moduleCode}" 的详细信息...`)
      await moduleStore.loadModuleDetail(moduleCode)
      module = moduleStore.getModuleByCode(moduleCode)!
    }
    
    moduleStore.setCurrentModule(module)
    console.log(`✅ 成功设置当前模块: ${module.title} (${module.code})`)
    isLoading.value = false
  } catch (error) {
    console.error(`❌ 设置模块失败:`, error)
    hasError.value = true
    isLoading.value = false
  }
}

// 获取 moduleCode
function queryModuleCode() {
  const moduleCode = route.params.module as string
  if (!moduleCode) {
    throw new Error('moduleCode is required')
  }
  return moduleCode
}

// 获取 articleId
function queryArticleId(): string | null {
  const articleId = route.params.article as string
  if (!articleId) {
    console.log('articleId is not found')
    return null
  }
  return articleId
}

</script>

<style scoped>
.loading-container {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 100vh;
  background-color: #f5f5f5;
}

.error-container {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 100vh;
  background-color: #f5f5f5;
}
</style>
