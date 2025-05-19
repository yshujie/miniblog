<template>
  <BlogLayout>
    <template #sidebar>
      <Sidebar :module="theModule" :sections="theModule.sections" />
    </template>
    <template #main>
      <ArticleCard :article="theArticle" />
    </template>
  </BlogLayout>
</template>

<script setup lang="ts">
import { onActivated, onBeforeMount, onBeforeUnmount, onBeforeUpdate, onDeactivated, onErrorCaptured, onMounted, onRenderTracked, onRenderTriggered, onUnmounted, onUpdated, ref } from 'vue'
import { useRoute } from 'vue-router'

import { fetchModuleByCode } from '@/api/module'
import type { Module } from '@/types/module'
import type { Article } from '@/types/article'

import BlogLayout from '../components/blog/BlogLayout.vue'
import Sidebar from '../components/blog/Sidebar.vue'
import ArticleCard from '../components/blog/ArticleCard.vue'
import { fetchArticleById } from '@/api/article'

const theModule = ref<Module>({} as Module)
const theArticle = ref<Article>({} as Article)

// onBeforeMount 生命周期钩子，在组件挂载前执行
onBeforeMount(async () => { 
  console.log(`Blog component is now before mounted.`)
})

// onMounted 生命周期钩子，在组件挂载完成后执行
onMounted(() => {
  console.log(`Blog component is now mounted.`)

  // init module，初始化当前 module、sections
  initModule()

  // init article，初始化当前 article
  initArticle()
})

// onBeforeUpdate 生命周期钩子，在组件更新前执行
onBeforeUpdate(() => { 
  console.log(`Blog component is now before updated.`)

  // init module，初始化当前 module、sections
  initModule()

  // init article，初始化当前 article
  initArticle()
})

// onUpdated 生命周期钩子，在组件更新后执行
onUpdated(() => { 
  console.log(`Blog component is now updated.`)
})

// onBeforeUnmount 生命周期钩子，在组件卸载前执行
onBeforeUnmount(() => { 
  console.log(`Blog component is now before unmounted.`)
})

// onUnmounted 生命周期钩子，在组件卸载后执行
onUnmounted(() => { 
  console.log(`Blog component is now unmounted.`)
})

// onErrorCaptured 声明周期钩子，在捕获了后代组件传递的错误时调用。
onErrorCaptured((err) => { 
  console.log(`Blog component is now error captured.`, err)
})

// onRenderTracked 声明周期钩子，在组件渲染过程中追踪响应式依赖时调用。
// 这个钩子仅在开发模式下可用，且在服务器端渲染期间不会被调用。
onRenderTracked(() => { 
  console.log(`Blog component is now render tracked.`)
})

// onRenderTriggered 声明周期钩子，在组件渲染过程中触发渲染时调用。
// 这个钩子仅在开发模式下可用，且在服务器端渲染期间不会被调用。
onRenderTriggered(() => { 
  console.log(`Blog component is now render triggered.`)
})

// onActivated 声明周期钩子，在组件激活后调用。
// 若组件实例是 <KeepAlive> 缓存树的一部分，当组件被插入到 DOM 中时调用。
onActivated(() => { 
  console.log(`Blog component is now activated.`)
})

// onDeactivated 声明周期钩子，在组件停用后调用。
// 若组件实例是 <KeepAlive> 缓存树的一部分，当组件从 DOM 中被移除时调用。
onDeactivated(() => { 
  console.log(`Blog component is now deactivated.`)
})


// init module 初始化 blog 模块
async function initModule() {
  var moduleCode = useRoute().params.module as string
  if (!moduleCode) {
    throw new Error('moduleCode is required')
  }

  var fetchedModule = await fetchModuleByCode(moduleCode)
  if (!fetchedModule) {
    throw new Error('module not found')
  }

  console.log(`fetchedModule: ${JSON.stringify(fetchedModule)}`)

  theModule.value = fetchedModule
}

// init article 初始化当前 article
async function initArticle() {
  console.log("in initArticle ... ")
  console.log('route: ', useRoute().params)
  var articleId = useRoute().params.article as string
  if (!articleId) {
    console.error('articleId is required')
    return
  }

  var fetchedArticle = await fetchArticleById(Number(articleId))
  if (!fetchedArticle) {
    console.error('article not found')
    return
  }

  theArticle.value = fetchedArticle
}
</script>
