<template>
  <div ref="containerRef" class="article-container">
    <div v-show="!hasArticle" class="no-article-card">
      <el-empty description="请选择一篇文章" />
    </div>

    <div v-show="hasArticle" class="article-card">
      <div class="article-card-content">
        <iframe
          :key="currentArticle?.externalLink"
          :src="currentArticle?.externalLink"
          frameborder="0"
          class="article-iframe"
          title="文章内容"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch, onUnmounted, nextTick } from 'vue'
import { Article } from '@/types/article'
import { fetchArticleDetail } from '@/api/blog'
import { ElLoading } from 'element-plus'

const props = defineProps<{ articleId: string | null }>()

const containerRef = ref<HTMLElement | null>(null)
const currentArticle = ref<Article | null>(null)

let loadingInstance: ReturnType<typeof ElLoading.service> | null = null

const hasArticle = computed(() => {
  return Boolean(
    currentArticle.value?.externalLink &&
    currentArticle.value.externalLink.trim() !== ''
  )
})

watch(
  () => props.articleId,
  async (newId) => {
    if (newId !== currentArticle.value?.id) {
      await fetchCurrentArticle(newId)
    }
  },
  { immediate: true }
)

onUnmounted(() => {
  hideLoading()
})

async function fetchCurrentArticle(articleId: string | null) {
  try {
    await nextTick()
    showLoading()

    if (!articleId) {
      currentArticle.value = null
      return
    }

    const article = await fetchArticleDetail(articleId)
    currentArticle.value = article ?? null
  } catch {
    currentArticle.value = null
  } finally {
    hideLoading()
  }
}

function showLoading() {
  if (loadingInstance) {
    loadingInstance.close()
  }

  loadingInstance = ElLoading.service({
    target: containerRef.value ?? undefined,
    text: '正在加载文章内容...',
  })
}

function hideLoading() {
  loadingInstance?.close()
  loadingInstance = null
}
</script>

<style scoped lang="less">
.article-container {
  width: 100%;
  height: 100%;
  min-height: 0;
}

.no-article-card {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 100%;
  min-height: 320px;
}

.article-card {
  height: 100%;
  padding: 0;

  .article-card-content {
    width: 100%;
    height: 100%;
    position: relative;
    overflow: hidden;
    background: var(--card-bg);

    // 遮挡 Notion 嵌入页右上角操作按钮
    &::before {
      content: '';
      position: absolute;
      top: 0;
      left: 0;
      width: 50%;
      height: 40px;
      background: transparent;
      z-index: 1;
      pointer-events: none;
    }

    &::after {
      content: '';
      position: absolute;
      top: 0;
      right: 0;
      width: 50%;
      height: 40px;
      background: var(--card-bg);
      z-index: 1;
      pointer-events: none;
    }

    .article-iframe {
      width: 100%;
      height: 100%;
      border: none;
      display: block;
    }
  }
}
</style>
