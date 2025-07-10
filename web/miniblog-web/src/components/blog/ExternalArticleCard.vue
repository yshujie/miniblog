<template>
    <div class="article-container">
      <div v-show="!hasArticle" class="no-article-card">
          <el-empty description=" " />
      </div>

      <div v-show="hasArticle" class="article-card">
        <div class="article-card-content">
          <iframe
            :key="currentArticle?.externalLink"
            :src="currentArticle?.externalLink"
            ref="articleFrame"
            frameborder="0"
            style="width: 100%; height: 100%;"
          ></iframe>
        </div>
      </div>
    </div>
</template>
<script setup lang="ts">
import { computed, defineProps, ref, watch, onUnmounted } from 'vue'
import { Article } from '@/types/article'
import { fetchArticleDetail } from '@/api/blog'
import { ElLoading } from 'element-plus'

// 组件 props
const props = defineProps<{ articleId: string|null }>()

// 当前文章
const currentArticle = ref<Article | null>(null)

// loading 实例
let loadingInstance: any = null

// 计算属性：hasArticle
const hasArticle = computed(() => {
  const hasData = currentArticle.value !== null && 
                  currentArticle.value?.externalLink && 
                  currentArticle.value.externalLink.trim() !== ''
  console.log('🔍 hasArticle 计算:', { 
    currentArticle: currentArticle.value, 
    hasData,
    externalLink: currentArticle.value?.externalLink 
  })
  return hasData
})

// iframe 引用
const articleFrame = ref<HTMLIFrameElement | null>(null)

// 监听外部链接变化
watch(() => currentArticle.value?.externalLink, (val) => {
  console.log('watch currentArticle.value?.externalLink: ', val)
  if (val && articleFrame.value) {
    articleFrame.value.onload = () => {
      console.log('✅ iframe 加载成功')
    }
  }
})

watch(() => props.articleId, async (newId, oldId) => {
  console.log('👀 watch articleId 变化:', { newId, oldId, currentId: currentArticle.value?.id })
  if (newId !== currentArticle.value?.id) {
    await fetchCurrentArticle(newId)
  }
}, { immediate: true })

// 组件卸载时清理 loading
onUnmounted(() => {
  hideLoading()
})

// 获取文章详情
async function fetchCurrentArticle(articleId: string | null) {
  try {
    showLoading()
    
    if (!articleId) {
      console.log('📝 articleId 为空，清空当前文章')
      currentArticle.value = null
      return
    }

    console.log('🔄 开始获取文章详情，articleId:', articleId)
    const article = await fetchArticleDetail(articleId)
    
    if (!article) {
      console.log('❌ 获取文章详情失败')
      currentArticle.value = null
      return
    }

    console.log('✅ 获取文章详情成功:', article)
    currentArticle.value = article
  } catch (error) {
    console.error('❌ 获取文章详情异常:', error)
    currentArticle.value = null
  } finally {
    hideLoading()
  }
}

function showLoading() {
  // 如果已有 loading 实例，先关闭
  if (loadingInstance) {
    loadingInstance.close()
  }
  
  loadingInstance = ElLoading.service({
    lock: true,
    text: '正在加载文章内容...',
  })
}

function hideLoading() {
  if (loadingInstance) {
    loadingInstance.close()
    loadingInstance = null
  }
}

</script>
<style scoped lang="less">
.article-container {
  height: 100%;
}

.no-article-card {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 100%;
  margin-top: 150px;
}

.article-card {
  padding: 32px;
  height: calc(100vh - 60px); // 减去头部和底部的高度

  .article-card-header {
    border-bottom: 1px solid #e0e0e0;
    padding-bottom: 20px;
    margin-bottom: 20px;

    .article-title {
      font-size: 24px;
      font-weight: bold;
      margin: 20px 0;
      text-align: left;
    }

    .article-info {
      font-size: 14px;
      color: #888;
      margin: 20px 0;
      line-height: 1.5;
      text-align: left;
      display: flex;
      flex-direction: row;
      flex-wrap: wrap;
      gap: 10px;
    }
  }

  .article-card-content {
    height: 100vh;
    width: 100%;
    position: relative;
    
    iframe {
      width: 100%;
      height: 100%;
      border: none;
      position: absolute;
      top: -128px;
      left: 0;
      right: 0;
      bottom: 0;
  
    }
  }
}
</style>
