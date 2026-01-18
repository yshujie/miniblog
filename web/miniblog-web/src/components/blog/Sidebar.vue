<template>
  <aside id="sidebar" class="sidebar-root" :class="{ 'sidebar-hidden': !sidebarOpen }">
    <div class="sidebar-content">
      <div
        v-for="item in props.sections"
        :key="item.title"
        :index="item.code"
        class="section-item"
      >
        <div class="section-header" :class="{ 'section-header-active': isSectionActive(item) }">
          <h2 class="section-title">{{ item.title }}</h2>
        </div>
        <ul class="article-list">
          <li 
            v-for="article in item.articles" 
            :key="article.id"
            class="article-item"
            :class="{ 'article-item-active': article.id === currentArticleId }"
            @click="handleArticleClick(article.id)"
          >
            <span v-if="article.id === currentArticleId" class="article-icon">
              <svg xmlns="http://www.w3.org/2000/svg" aria-hidden="true" role="img" width="1em" height="1em" viewBox="0 0 24 24">
                <path fill="currentColor" d="M12 2c5.523 0 10 4.477 10 10a10 10 0 0 1-20 0C2 6.477 6.477 2 12 2m-.293 6.293a1 1 0 0 0-1.414 0l-.083.094a1 1 0 0 0 .083 1.32L12.585 12l-2.292 2.293a1 1 0 0 0 1.414 1.414l3-3a1 1 0 0 0 0-1.414z"></path>
              </svg>
            </span>
            <span class="article-title">{{ article.title }}</span>
          </li>
        </ul>
      </div>
    </div>
    <!-- 侧边指示器 -->
    <div class="sidebar-indicator"></div>
  </aside>
</template>
<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { useUiStore } from '@/stores/ui'
import type { Section } from '@/types/section';

const props = defineProps<{ sections: Section[], moduleCode: string }>()

const router = useRouter()
const uiStore = useUiStore()

// 侧边栏打开状态
const sidebarOpen = computed(() => uiStore.sidebarOpen)

// 当前选中的文章 ID
const currentArticleId = computed(() => {
  const articleId = router.currentRoute.value.params.article
  if (articleId) {
    return String(articleId)
  }
  return null
})

// 判断 section 是否激活（包含当前文章）
const isSectionActive = (section: Section) => {
  return section.articles.some(article => article.id === currentArticleId.value)
}

// 文章点击事件
const handleArticleClick = (articleId: string) => {
  router.push(`/blog/${props.moduleCode}/article/${articleId}`)
}
</script>
<style scoped lang="less">
.sidebar-root {
  display: none;
  width: 18rem;
  flex-shrink: 0;
  position: relative;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);

  @media (min-width: 1280px) {
    display: block;
  }

  &.sidebar-hidden {
    width: 0;
    opacity: 0;
    margin-right: -2rem;
    pointer-events: none;
  }

  .sidebar-content {
    position: sticky;
    top: 6rem;
    overflow-y: auto;
    max-height: calc(100vh - 120px);
    padding-right: 1rem;
    transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);

    // 隐藏滚动条
    &::-webkit-scrollbar {
      display: none;
    }
    -ms-overflow-style: none;
    scrollbar-width: none;
  }

  .section-item {
    margin-bottom: 2rem;

    .section-header {
      display: flex;
      align-items: center;
      color: #111827;
      border-left: 4px solid #d1d5db;
      padding-left: 0.75rem;
      margin-bottom: 1rem;

      &.section-header-active {
        border-left-color: #2563eb;
      }

      .section-title {
        font-weight: 700;
        font-size: 1rem;
        margin: 0;
      }
    }

    .article-list {
      list-style: none;
      padding: 0;
      margin: 0;
      display: flex;
      flex-direction: column;
      gap: 0.25rem;

      .article-item {
        padding: 0.5rem 0.75rem;
        border-radius: 0.5rem;
        cursor: pointer;
        transition: all 0.15s;
        display: flex;
        align-items: center;
        gap: 0.5rem;
        font-size: 0.875rem;
        color: #4b5563;

        &:hover {
          background: #eff6ff;
          color: #2563eb;
        }

        &.article-item-active {
          background: #2563eb;
          color: #ffffff;
          font-weight: 500;
          box-shadow: 0 4px 6px -1px rgba(37, 99, 235, 0.2), 0 2px 4px -2px rgba(37, 99, 235, 0.2);

          .article-icon {
            display: flex;
            align-items: center;
          }

          .article-title {
            color: #ffffff;
          }
        }

        .article-icon {
          display: none;
          width: 1em;
          height: 1em;
        }

        .article-title {
          line-height: 1.25rem;
          transition: color 0.15s;
        }
      }
    }
  }

  .sidebar-indicator {
    position: absolute;
    top: 0;
    right: 0;
    height: 100%;
    width: 0.25rem;
    background: #e5e7eb;
    border-radius: 9999px;
    transition: background-color 0.15s;

    .sidebar-root:hover & {
      background: #60a5fa;
    }
  }
}
</style>
