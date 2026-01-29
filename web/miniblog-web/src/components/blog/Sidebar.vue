<template>
  <aside id="sidebar" class="sidebar-root" :class="{ 'sidebar-hidden': !sidebarOpen }">
    <div class="sidebar-content">
      <a href="/" class="sidebar-logo">Shujie's Blog</a>
      <div class="section-list">
        <div
          v-for="section in props.sections"
          :key="section.id"
          class="section-item"
        >
          <button
            type="button"
            class="section-header"
            @click="toggleSection(section.id)"
          >
            <span class="section-title">{{ section.title }}({{ section.articles.length }}讲)</span>
            <svg
              class="icon-arrow"
              :class="{ 'icon-arrow-collapsed': !isSectionExpanded(section.id) }"
              xmlns="http://www.w3.org/2000/svg"
              width="16"
              height="16"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
            >
              <path d="m18 15-6-6-6 6" />
            </svg>
          </button>
          <div class="section-sep" />
          <div v-show="isSectionExpanded(section.id)" class="article-list">
            <button
              v-for="article in section.articles"
              :key="article.id"
              type="button"
              class="article-item"
              :class="{ 'article-item-active': article.id === currentArticleId }"
              @click="handleArticleClick(article.id)"
            >
              <span class="article-title">{{ article.title }}</span>
            </button>
          </div>
        </div>
      </div>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useUiStore } from '@/stores/ui'
import type { Section } from '@/types/section'

const props = defineProps<{
  sections: Section[]
  moduleCode: string
}>()

const router = useRouter()
const uiStore = useUiStore()

const expandedSectionIds = ref<Set<string>>(new Set())

// 侧边栏打开状态
const sidebarOpen = computed(() => uiStore.sidebarOpen)

// 当前选中的文章 ID
const currentArticleId = computed(() => {
  const articleId = router.currentRoute.value.params.article
  if (articleId) return String(articleId)
  return null
})

// 是否展开某 section
function isSectionExpanded(sectionId: string) {
  return expandedSectionIds.value.has(sectionId)
}

// 切换 section 展开/收起
function toggleSection(sectionId: string) {
  const set = new Set(expandedSectionIds.value)
  if (set.has(sectionId)) {
    set.delete(sectionId)
  } else {
    set.add(sectionId)
  }
  expandedSectionIds.value = set
}

// 当 sections 变化时（如切换模块），默认全部展开
watch(
  () => props.sections,
  (newSections) => {
    if (!newSections?.length) {
      expandedSectionIds.value = new Set()
      return
    }
    expandedSectionIds.value = new Set(newSections.map(s => s.id))
  },
  { immediate: true }
)

// 当选中文章变化时，只确保该文章所在 section 展开，不收起其他 section
watch(
  currentArticleId,
  (articleId) => {
    if (!articleId || !props.sections.length) return
    const section = props.sections.find(s => s.articles.some(a => a.id === articleId))
    if (section) {
      expandedSectionIds.value = new Set([...expandedSectionIds.value, section.id])
    }
  }
)

function handleArticleClick(articleId: string) {
  router.push(`/blog/${props.moduleCode}/article/${articleId}`)
}
</script>

<style scoped lang="less">
.sidebar-root {
  display: none;
  width: 24rem;
  flex-shrink: 0;
  min-height: 0; /* 允许在 flex 布局中被压缩，避免底部被裁切 */
  background: var(--zone-bg);
  border-right: 1px solid var(--border-divider);
  border-radius: 0;
  box-shadow: none;
  transition: width 0.3s ease, opacity 0.3s ease;

  @media (min-width: 1280px) {
    display: flex;
    flex-direction: column;
  }

  &.sidebar-hidden {
    width: 0;
    opacity: 0;
    margin-right: -2rem;
    pointer-events: none;
  }

  .sidebar-content {
    flex: 1 1 0;
    min-height: 0;
    overflow-y: auto;
    max-height: 100%; /* 填满侧栏高度，超出在内部滚动，避免底部被裁切 */
    padding: 1.25rem 1rem 1.5rem 1.5rem;

    &::-webkit-scrollbar {
      width: 5px;
    }
    &::-webkit-scrollbar-track {
      background: transparent;
    }
    &::-webkit-scrollbar-thumb {
      background: var(--border-color);
      border-radius: 4px;
    }
  }

  .sidebar-logo {
    display: block;
    font-size: 1.25rem;
    font-weight: 700;
    color: var(--text-primary);
    text-decoration: none;
    margin-bottom: 1rem;
    padding-bottom: 0.75rem;
    border-bottom: 1px solid var(--border-color);
    transition: color 0.2s ease;

    &:hover {
      color: var(--accent);
    }
  }

  .section-list {
    display: flex;
    flex-direction: column;
    gap: 0;
  }

  .section-item {
    .section-header {
      width: 100%;
      display: flex;
      align-items: center;
      justify-content: space-between;
      padding: 0.75rem 0.5rem;
      margin: 0 -0.5rem;
      font-size: 0.9375rem;
      font-weight: 600;
      color: var(--text-primary);
      background: none;
      border: none;
      border-radius: 0.375rem;
      cursor: pointer;
      text-align: left;
      gap: 0.5rem;
      transition: background 0.15s ease, color 0.15s ease;

      &:hover {
        background: var(--sidebar-hover-bg);
        color: var(--text-secondary);
      }
      &:focus-visible {
        outline: 2px solid var(--accent);
        outline-offset: 2px;
      }
    }
    .section-title {
      flex: 1;
      min-width: 0;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
      line-height: 1.4;
      color: var(--text-secondary);
      font-weight: 500;
    }
    .icon-arrow {
      flex-shrink: 0;
      color: var(--text-muted);
      transition: transform 0.2s ease;
      &.icon-arrow-collapsed {
        transform: rotate(180deg);
      }
    }
    .section-sep {
      height: 1px;
      background: var(--border-color);
      margin: 0;
    }
    .article-list {
      display: flex;
      flex-direction: column;
      gap: 0.5rem;
      padding: 0.5rem 0 0.75rem 1rem;
    }
  }

  .article-item {
    display: flex;
    align-items: flex-start;
    width: 100%;
    padding: 0.5rem 0.75rem;
    margin: 0 -0.75rem;
    font-size: 0.8125rem;
    line-height: 1.45;
    color: var(--text-secondary);
    background: none;
    border: none;
    border-radius: 0.375rem;
    cursor: pointer;
    text-align: left;
    transition: background 0.15s ease, color 0.15s ease;

    &:hover {
      background: var(--sidebar-hover-bg);
      color: var(--text-primary);
    }
    &:focus-visible {
      outline: 2px solid var(--accent);
      outline-offset: 2px;
    }
    &.article-item-active {
      background: var(--sidebar-active-bg);
      color: var(--text-primary);
      font-weight: 500;
    }
    .article-title {
      flex: 1;
      min-width: 0;
      overflow: hidden;
      text-overflow: ellipsis;
      display: -webkit-box;
      -webkit-line-clamp: 2;
      line-clamp: 2;
      -webkit-box-orient: vertical;
    }
  }
}
</style>
