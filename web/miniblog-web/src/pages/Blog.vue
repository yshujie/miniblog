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
