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
          class="article-iframe"
        ></iframe>
      </div>
      
      <!-- 右下角操作按钮组 -->
      <div class="article-actions">
        <!-- <button 
          class="action-btn" 
          @click="scrollToTop"
          title="滚动到顶部"
        >
          ⬆️
        </button> -->
        <button 
          class="action-btn" 
          @click="openSidebar"
          title="打开左侧边栏"
        >
          ⬅️
        </button>
        <button 
          class="action-btn" 
          @click="closeSidebar"
          title="隐藏左侧边栏"
        >
          ➡️
        </button>
      </div>
    </div>
  </div>
</template>
<script setup lang="ts">
import { computed, ref, watch, onUnmounted } from 'vue'
import { Article } from '@/types/article'
import { fetchArticleDetail } from '@/api/blog'
import { ElLoading } from 'element-plus'
import { useUiStore } from '@/stores/ui'

// 组件 props
const props = defineProps<{ articleId: string|null }>()

// UI Store
const uiStore = useUiStore()

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

// 滚动到顶部
// 注意：由于跨域限制，无法直接控制 iframe 内部滚动
// 我们尝试多种方法，如果都失败则滚动主窗口
const scrollToTop = () => {
  if (!articleFrame.value) {
    return
  }

  // 方法1: 尝试使用 postMessage 与 iframe 通信（适用于跨域场景）
  // 注意：这需要 iframe 内容支持监听 message 事件
  try {
    const iframeWindow = articleFrame.value.contentWindow
    if (iframeWindow) {
      iframeWindow.postMessage({ 
        type: 'scrollToTop',
        behavior: 'smooth'
      }, '*')
    }
  } catch (e) {
    // postMessage 失败，静默处理
  }

  // 方法2: 尝试直接访问 iframe 内部（仅同源时可用）
  // 使用安全的访问方式，避免抛出未捕获的错误
  let canAccessIframe = false
  try {
    const iframeWindow = articleFrame.value.contentWindow
    if (iframeWindow) {
      // 尝试访问 contentDocument，跨域时会返回 null 或抛出错误
      const iframeDoc = articleFrame.value.contentDocument
      if (iframeDoc) {
        canAccessIframe = true
        // 同源，可以直接控制滚动
        if (iframeWindow.scrollTo) {
          iframeWindow.scrollTo({ top: 0, behavior: 'smooth' })
          return
        }
        
        // 备用方法：直接设置 scrollTop
        const iframeHtml = iframeDoc.documentElement
        const iframeBody = iframeDoc.body
        if (iframeHtml) iframeHtml.scrollTop = 0
        if (iframeBody) iframeBody.scrollTop = 0
        if (iframeWindow.scroll) iframeWindow.scroll(0, 0)
        return
      }
    }
  } catch (e) {
    // 跨域限制，无法访问 iframe 内部
    // 这是预期的行为，不需要处理
    canAccessIframe = false
  }

  // 方法3: 作为备选，滚动主窗口
  // 对于跨域 iframe，这是唯一可行的方式
  // 虽然不能滚动 iframe 内部，但至少可以滚动页面本身
  window.scrollTo({ top: 0, behavior: 'smooth' })
}

// 打开左侧边栏
const openSidebar = () => {
  if (!uiStore.sidebarOpen) {
    uiStore.setSidebar(true)
  }
}

// 隐藏左侧边栏
const closeSidebar = () => {
  if (uiStore.sidebarOpen) {
    uiStore.setSidebar(false)
  }
}

</script>
<style scoped lang="less">
.article-container {
  width: 100%;
  height: 100%;
}

.no-article-card {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 400px;
  padding: 2rem;
}

.article-card {
  height: 100%;
  padding: 2rem;

  @media (min-width: 768px) {
    padding: 1rem 2em;
  }

  .article-card-content {
    width: 100%;
    height: 100%;
    position: relative;
    overflow: hidden;

    // 增加 40px 高的遮蔽，用于防止
    &::before {
      content: '';
      position: absolute;
      top: 0;
      left: 0;
      width: 50%;
      height: 40px;
      background: rgba(0, 0, 0, 0);
    }
    &::after {
      content: '';
      position: absolute;
      top: 0;
      right: 0;
      width: 50%;
      height: 40px;
      background: #ffffff;
    }


    .article-iframe {
      width: 100%;
      height: 100%;
      border: none;
      display: block;
    }
  }

  // 右下角操作按钮组
  .article-actions {
    position: fixed;
    right: 2rem;
    bottom: 2rem;
    z-index: 50;
    display: flex;
    flex-direction: inherit;
    gap: 0.75rem;
    opacity: 0.6;
    transition: all 0.3s ease;

    &:hover {
      opacity: 1;
    }
  }

  .action-btn {
    width: 2.75rem;
    height: 2.75rem;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--card-bg);
    border: 1px solid var(--border-color);
    border-radius: 50%;
    color: var(--text-secondary);
    font-size: 1.125rem;
    cursor: pointer;
    box-shadow: var(--shadow-md);
    transition: border-color 0.2s, color 0.2s, transform 0.2s;

    &:hover {
      border-color: var(--accent);
      color: var(--accent);
      transform: scale(1.05);
    }

    &:active {
      transform: scale(0.98);
    }
  }
}
</style>
