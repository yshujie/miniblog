<template>
  <header class="blog-header">
    <div class="blog-header-inner">
      <div class="blog-header-left">
        <a href="/" class="blog-logo">Shujie's Blog</a>
        <nav class="blog-nav" aria-label="模块导航">
          <button
            v-for="module in moduleStore.modules"
            :key="module.code"
            type="button"
            class="blog-nav-item"
            :class="{ 'blog-nav-item-active': activeModuleCode === module.code }"
            @click="handleModuleClick(module.code)"
          >
            {{ module.title }}
          </button>
        </nav>
      </div>
      <div class="blog-header-right">
        <a
          href="https://github.com/yshujie"
          target="_blank"
          rel="noopener noreferrer"
          class="blog-github-link"
        >
          GitHub
        </a>
      </div>
    </div>
  </header>
</template>

<script setup lang="ts">
import { computed, onBeforeMount } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useModuleStore } from '@/stores/module'

const moduleStore = useModuleStore()
const router = useRouter()
const route = useRoute()

const activeModuleCode = computed(() => {
  const moduleCode = route.params.module
  return typeof moduleCode === 'string' ? moduleCode : ''
})

onBeforeMount(async () => {
  if (moduleStore.modules.length === 0) {
    await moduleStore.loadModules()
  }
})

function findFirstArticleId(moduleCode: string): string | null {
  const module = moduleStore.getModuleByCode(moduleCode)
  if (!module?.sections?.length) {
    return null
  }

  for (const section of module.sections) {
    for (const subsection of section.subsections) {
      if (subsection.articles.length > 0) {
        return subsection.articles[0].id
      }
    }
    if (section.articles.length > 0) {
      return section.articles[0].id
    }
  }

  return null
}

async function handleModuleClick(moduleCode: string) {
  if (activeModuleCode.value === moduleCode && route.params.article) {
    return
  }

  await moduleStore.loadModuleDetail(moduleCode)

  const firstArticleId = findFirstArticleId(moduleCode)
  if (firstArticleId) {
    router.push(`/blog/${moduleCode}/article/${firstArticleId}`)
    return
  }

  router.push(`/blog/${moduleCode}`)
}
</script>

<style scoped lang="less">
.blog-header {
  flex-shrink: 0;
  height: var(--blog-header-height);
  background: var(--blog-header-bg);
  border-bottom: 1px solid var(--blog-header-border);
  z-index: 50;
}

.blog-header-inner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 100%;
  padding: 0 1.5rem;
  max-width: 100%;
}

.blog-header-left {
  display: flex;
  align-items: center;
  min-width: 0;
  gap: 2rem;
}

.blog-logo {
  flex-shrink: 0;
  font-size: 1rem;
  font-weight: 700;
  letter-spacing: -0.02em;
  color: var(--text-primary);
  text-decoration: none;
  white-space: nowrap;

  &:hover {
    color: var(--blog-header-active);
  }
}

.blog-nav {
  display: flex;
  align-items: stretch;
  gap: 1.75rem;
  min-width: 0;
  overflow-x: auto;

  &::-webkit-scrollbar {
    display: none;
  }
}

.blog-nav-item {
  position: relative;
  flex-shrink: 0;
  padding: 0 0 0.125rem;
  border: none;
  background: none;
  font-size: 0.875rem;
  font-weight: 500;
  line-height: var(--blog-header-height);
  color: var(--blog-header-text);
  cursor: pointer;
  white-space: nowrap;
  transition: color 0.15s ease;

  &::after {
    content: '';
    position: absolute;
    left: 0;
    right: 0;
    bottom: 0;
    height: 2px;
    background: transparent;
    transition: background 0.15s ease;
  }

  &:hover {
    color: var(--text-primary);
  }

  &.blog-nav-item-active {
    color: var(--blog-header-active);
    font-weight: 600;

    &::after {
      background: var(--blog-header-active);
    }
  }
}

.blog-header-right {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  margin-left: 1rem;
}

.blog-github-link {
  font-size: 0.875rem;
  font-weight: 500;
  color: var(--blog-header-text);
  text-decoration: none;
  transition: color 0.15s ease;

  &:hover {
    color: var(--blog-header-active);
  }
}
</style>
