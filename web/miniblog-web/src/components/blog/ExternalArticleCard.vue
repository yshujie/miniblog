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
import { computed, defineProps, ref, onMounted, onUpdated, watch } from 'vue'
import { Article } from '@/types/article'
import { fetchArticleDetail } from '@/api/blog'
import { ElLoading } from 'element-plus'

// 组件 props
const props = defineProps<{ articleId: number|null }>()

// 当前文章
const currentArticle = ref<Article | null>(null)

// 计算属性：hasArticle
const hasArticle = computed(() => {
  return currentArticle.value !== null && currentArticle.value instanceof Article
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
  if (newId && newId !== currentArticle.value?.id) {
    await fetchCurrentArticle(newId)
  }
}, { immediate: true })


// 组件挂载时，获取文章
onMounted(async () => {
  console.log('onMounted', props.articleId)
  await fetchCurrentArticle(props.articleId)
  console.log('onMounted done', currentArticle.value?.id) 
})

// 获取文章详情
async function fetchCurrentArticle(articleId: number | null) {
  showLoading()
  if (!articleId) {
    return null
  }

  const article = await fetchArticleDetail(articleId)
  if (!article) {
    return
  }

  currentArticle.value = article
  hideLoading()
}

function showLoading() { 
  const loading = ElLoading.service({
    lock: true,
    text: 'Loading...',
  })
}

function hideLoading() {
  ElLoading.service().close()
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
    padding-bottom: 128px;
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
