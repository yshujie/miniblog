<template>
  <aside id="sidebar" class="sidebar-root" :class="{ 'sidebar-hidden': !sidebarOpen }">
    <div class="sidebar-content">
      <div class="sidebar-doc-title">{{ moduleLabel }}</div>
      <nav class="section-list" aria-label="文章目录">
        <div
          v-for="section in props.sections"
          :key="section.id"
          class="section-item"
        >
          <button
            type="button"
            class="section-header"
            :class="{ 'section-header-expanded': isSectionExpanded(section.id) }"
            @click="toggleSection(section.id)"
          >
            <span class="section-title">{{ section.title }}</span>
            <svg
              class="icon-arrow"
              :class="{ 'icon-arrow-expanded': isSectionExpanded(section.id) }"
              xmlns="http://www.w3.org/2000/svg"
              width="14"
              height="14"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
              aria-hidden="true"
            >
              <path d="m9 18 6-6-6-6" />
            </svg>
          </button>

          <div v-show="isSectionExpanded(section.id)" class="section-body">
            <div
              v-for="subsection in section.subsections"
              :key="subsection.id"
              class="nav-group"
            >
              <button
                type="button"
                class="group-header"
                :class="{ 'group-header-expanded': isSubsectionExpanded(section.id, subsection.code) }"
                @click="toggleSubsection(section.id, subsection.code)"
              >
                <span class="group-title">{{ subsection.title }}</span>
                <svg
                  class="icon-arrow icon-arrow-sm"
                  :class="{ 'icon-arrow-expanded': isSubsectionExpanded(section.id, subsection.code) }"
                  xmlns="http://www.w3.org/2000/svg"
                  width="12"
                  height="12"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  aria-hidden="true"
                >
                  <path d="m9 18 6-6-6-6" />
                </svg>
              </button>

              <div
                v-show="isSubsectionExpanded(section.id, subsection.code)"
                class="article-list article-list-nested"
              >
                <button
                  v-for="article in subsection.articles"
                  :key="article.id"
                  type="button"
                  class="article-item"
                  :class="{ 'article-item-active': article.id === currentArticleId }"
                  :title="article.title"
                  @click="handleArticleClick(article.id)"
                >
                  <span class="article-title">{{ displayTitle(article.title) }}</span>
                </button>
              </div>
            </div>

            <div v-if="section.articles.length" class="article-list">
              <button
                v-for="article in section.articles"
                :key="article.id"
                type="button"
                class="article-item"
                :class="{ 'article-item-active': article.id === currentArticleId }"
                :title="article.title"
                @click="handleArticleClick(article.id)"
              >
                <span class="article-title">{{ displayTitle(article.title) }}</span>
              </button>
            </div>
          </div>
        </div>
      </nav>
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
  moduleTitle?: string
}>()

const router = useRouter()
const uiStore = useUiStore()

const expandedSectionIds = ref<Set<string>>(new Set())
const expandedSubsectionKeys = ref<Set<string>>(new Set())

const sidebarOpen = computed(() => uiStore.sidebarOpen)

const moduleLabel = computed(() => {
  if (props.moduleTitle?.trim()) {
    return `${props.moduleTitle.trim()} 文档`
  }
  return '文档导航'
})

const currentArticleId = computed(() => {
  const articleId = router.currentRoute.value.params.article
  if (articleId) return String(articleId)
  return null
})

function subsectionExpandKey(sectionId: string, subsectionCode: string) {
  return `${sectionId}::${subsectionCode}`
}

function displayTitle(title: string) {
  const trimmed = title.trim()
  if (trimmed.length <= 28) return trimmed
  return `${trimmed.slice(0, 27)}…`
}

function isSectionExpanded(sectionId: string) {
  return expandedSectionIds.value.has(sectionId)
}

function isSubsectionExpanded(sectionId: string, subsectionCode: string) {
  return expandedSubsectionKeys.value.has(subsectionExpandKey(sectionId, subsectionCode))
}

function toggleSection(sectionId: string) {
  const set = new Set(expandedSectionIds.value)
  if (set.has(sectionId)) {
    set.delete(sectionId)
  } else {
    set.add(sectionId)
  }
  expandedSectionIds.value = set
}

function toggleSubsection(sectionId: string, subsectionCode: string) {
  const key = subsectionExpandKey(sectionId, subsectionCode)
  const set = new Set(expandedSubsectionKeys.value)
  if (set.has(key)) {
    set.delete(key)
  } else {
    set.add(key)
  }
  expandedSubsectionKeys.value = set
}

function expandForCurrentArticle(articleId: string | null) {
  if (!articleId || !props.sections.length) return

  for (const section of props.sections) {
    for (const subsection of section.subsections) {
      if (subsection.articles.some(article => article.id === articleId)) {
        expandedSectionIds.value = new Set([...expandedSectionIds.value, section.id])
        expandedSubsectionKeys.value = new Set([
          ...expandedSubsectionKeys.value,
          subsectionExpandKey(section.id, subsection.code),
        ])
        return
      }
    }

    if (section.articles.some(article => article.id === articleId)) {
      expandedSectionIds.value = new Set([...expandedSectionIds.value, section.id])
      return
    }
  }
}

watch(
  () => props.sections,
  (newSections) => {
    if (!newSections?.length) {
      expandedSectionIds.value = new Set()
      expandedSubsectionKeys.value = new Set()
      return
    }

    expandedSectionIds.value = new Set(newSections.map(section => section.id))
    expandedSubsectionKeys.value = new Set(
      newSections.flatMap(section =>
        section.subsections.map(subsection => subsectionExpandKey(section.id, subsection.code))
      )
    )
    expandForCurrentArticle(currentArticleId.value)
  },
  { immediate: true }
)

watch(currentArticleId, expandForCurrentArticle)

function handleArticleClick(articleId: string) {
  router.push(`/blog/${props.moduleCode}/article/${articleId}`)
}
</script>

<style scoped lang="less">
.sidebar-root {
  display: none;
  width: var(--sidebar-width);
  min-width: var(--sidebar-width);
  height: 100%;
  flex-shrink: 0;
  min-height: 0;
  overflow: hidden;
  background: var(--sidebar-bg);
  border-right: 1px solid var(--sidebar-divider);
  transition: width 0.25s ease, min-width 0.25s ease, border-color 0.25s ease;

  @media (min-width: 1280px) {
    display: block;
  }

  &.sidebar-hidden {
    width: 0;
    min-width: 0;
    border-right-color: transparent;
    pointer-events: none;
  }

  .sidebar-content {
    width: var(--sidebar-width);
    min-width: var(--sidebar-width);
    height: 100%;
    display: flex;
    flex-direction: column;
    min-height: 0;
    overflow-y: auto;
    overflow-x: hidden;
    padding: 1.75rem 0 2.5rem;

    &::-webkit-scrollbar {
      width: 4px;
    }
    &::-webkit-scrollbar-track {
      background: transparent;
    }
    &::-webkit-scrollbar-thumb {
      background: rgba(0, 0, 0, 0.12);
      border-radius: 4px;
    }
  }

  .sidebar-doc-title {
    display: block;
    font-size: 0.875rem;
    font-weight: 600;
    color: var(--sidebar-section-text);
    margin-bottom: 1.25rem;
    padding: 0 1.5rem 1rem;
    letter-spacing: -0.01em;
    border-bottom: 1px solid var(--sidebar-divider);
  }

  .section-list {
    display: flex;
    flex-direction: column;
    gap: 0;
    padding-top: 0.5rem;
  }

  .section-item {
    padding: 0.875rem 0;

    &:not(:last-child) {
      margin-bottom: 0.5rem;
      border-bottom: 1px solid var(--sidebar-divider);
    }
    .section-header,
    .group-header {
      width: 100%;
      display: flex;
      align-items: center;
      justify-content: space-between;
      background: none;
      border: none;
      cursor: pointer;
      text-align: left;
      gap: 0.5rem;
      transition: color 0.15s ease;

      &:focus-visible {
        outline: 2px solid var(--sidebar-active-color);
        outline-offset: -2px;
      }
    }

    .section-header {
      padding: 0.625rem 1.5rem;
      font-size: 0.875rem;
      font-weight: 600;
      color: var(--sidebar-section-text);

      &:hover {
        color: var(--sidebar-text-hover);
      }
    }

    .section-title,
    .group-title {
      flex: 1;
      min-width: 0;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
      line-height: 1.5;
    }

    .icon-arrow {
      flex-shrink: 0;
      color: var(--sidebar-text);
      opacity: 0.6;
      transition: transform 0.2s ease;

      &.icon-arrow-expanded {
        transform: rotate(90deg);
      }
    }

    .section-body {
      padding: 0.375rem 0 0.25rem;
    }

    .nav-group {
      margin-top: 0.25rem;

      &:not(:last-child) {
        margin-bottom: 0.375rem;
      }
    }

    .group-header {
      padding: 0.5rem 1.5rem 0.5rem 2rem;
      font-size: 0.8125rem;
      font-weight: 500;
      color: var(--sidebar-text);

      &:hover {
        color: var(--sidebar-text-hover);
      }
    }

    .article-list {
      display: flex;
      flex-direction: column;
      gap: 0.25rem;
      padding: 0.25rem 0 0.5rem;
    }
  }

  .article-item {
    display: flex;
    align-items: center;
    width: 100%;
    padding: 0.4375rem 1.5rem 0.4375rem 2.75rem;
    font-size: 0.8125rem;
    line-height: 1.5;
    color: var(--sidebar-text);
    background: none;
    border: none;
    cursor: pointer;
    text-align: left;
    transition: color 0.15s ease;

    .article-list-nested & {
      padding-left: 3rem;
    }

    &:hover {
      color: var(--sidebar-text-hover);
    }
    &:focus-visible {
      outline: 2px solid var(--sidebar-active-color);
      outline-offset: -2px;
    }
    &.article-item-active {
      color: var(--sidebar-active-color);
      font-weight: 500;
    }

    .article-title {
      flex: 1;
      min-width: 0;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }
  }
}
</style>
