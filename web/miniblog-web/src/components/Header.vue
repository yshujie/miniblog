<template>
  <header class="header-bar">
    <div class="header-container">
      <div class="header-left">
        <div class="logo-section">
          <a href="/" class="logo-link">
            <span class="logo-text">Shujie's Blog</span>
          </a>
        </div>
      </div>
      <div class="header-right">
        <nav class="nav-section">
          <a 
            v-for="module in moduleStore.modules" 
            :key="module.code" 
            :class="['nav-link', { 'nav-link-active': currentFullPath === `/blog/${module.code}` }]"
            @click="handleModuleClick(module.code)"
          >
            <!-- AI 图标 (芯片/智能) -->
            <component :is="getModuleIcon(module.code)" class="nav-icon" />
            <span>{{ module.title }}</span>
          </a>
          <a 
            href="https://github.com/yshujie" 
            target="_blank" 
            rel="noopener" 
            class="nav-link nav-link-github"
          >
            <svg xmlns="http://www.w3.org/2000/svg" aria-hidden="true" role="img" width="1em" height="1em" viewBox="0 0 24 24">
              <path fill="currentColor" d="M12 2A10 10 0 0 0 2 12c0 4.42 2.87 8.17 6.84 9.5c.5.08.66-.23.66-.5v-1.69c-2.77.6-3.36-1.34-3.36-1.34c-.46-1.16-1.11-1.47-1.11-1.47c-.91-.62.07-.6.07-.6c1 .07 1.53 1.03 1.53 1.03c.87 1.52 2.34 1.07 2.91.83c.09-.65.35-1.09.63-1.34c-2.22-.25-4.55-1.11-4.55-4.92c0-1.11.38-2 1.03-2.71c-.1-.25-.45-1.29.1-2.64c0 0 .84-.27 2.75 1.02c.79-.22 1.65-.33 2.5-.33s1.71.11 2.5.33c1.91-1.29 2.75-1.02 2.75-1.02c.55 1.35.2 2.39.1 2.64c.65.71 1.03 1.6 1.03 2.71c0 3.82-2.34 4.66-4.57 4.91c.36.31.69.92.69 1.85V21c0 .27.16.59.67.5C19.14 20.16 22 16.42 22 12A10 10 0 0 0 12 2"></path>
            </svg>
            GitHub
          </a>
        </nav>
      </div>
    </div>
  </header>
</template>

<script setup lang="ts">
import { computed, onBeforeMount, type Component } from 'vue'
import { useModuleStore } from '@/stores/module'
import { useRoute, useRouter } from 'vue-router'
import { useUiStore } from '@/stores/ui'
import { MagicStick, Compass, Grid, DataBoard, Briefcase, Menu as MenuIcon } from '@element-plus/icons-vue'

// module store
const moduleStore = useModuleStore()
const uiStore = useUiStore()

// 模块图标映射
const moduleIcons: Record<string, Component> = {
  ai: MagicStick,
  go: Compass,
  ddd: Grid,
  database: DataBoard,
  project: Briefcase
}

// 获取路由实例
const router = useRouter()

// 获取路由实例
const route = useRoute()

// 计算属性 currentFullPath
const currentFullPath = computed(() => {
  // 如果 route.path 以 /blog/ 开头，则返回 /blog/xxx 前两位
  if (route.path.startsWith('/blog/')) {
    const path = route.path.split('/')
    return '/' + path[1] + '/' + path[2]
  }

  return '/'
})

// 判断是否是博客页面
const isBlogPage = computed(() => {
  return route.path.startsWith('/blog/')
})

const sidebarOpen = computed(() => uiStore.sidebarOpen)

onBeforeMount(async () => {
  console.log('onBeforeMount')

  // 加载模块数据
  await moduleStore.loadModules()

  // 预热所有模块数据
  await moduleStore.loadAllModuleDetail()
})

// 切换侧边栏
const toggleSidebar = () => {
  uiStore.toggleSidebar()
}

// 模块点击事件
const handleModuleClick = (moduleCode: string) => {
  // 加载模块详情
  moduleStore.loadModuleDetail(moduleCode)

  // 尝试选择第一篇文章
  const firstArticle = moduleStore.modules.find(module => module.code === moduleCode)?.sections[0].articles[0]
  if (firstArticle) {
    router.push(`/blog/${moduleCode}/article/${firstArticle.id}`)
  }
}

const getModuleIcon = (code: string) => moduleIcons[code] || MenuIcon

</script>

<style lang="less" scoped>
.header-bar {
  position: sticky;
  top: 0;
  z-index: 50;
  width: 100%;
  background: var(--zone-bg);
  border-bottom: 1px solid var(--border-divider);
  padding: 0.875rem 1.5rem;
  box-shadow: var(--shadow-sm);
}

.header-container {
  display: flex;
  flex-direction: row;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  max-width: 1400px;
  margin: 0 auto;
}

.header-left {
  flex-shrink: 0;
  display: flex;
  align-items: center;
}

.logo-section {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.logo-link {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  text-decoration: none;
  color: inherit;
}

.logo-icon {
  color: #2563eb;
  font-size: 1.875rem;
}

.logo-text {
  font-size: 1.375rem;
  font-weight: 700;
  letter-spacing: -0.025em;
  color: var(--text-primary);
}

.logo-text-blue {
  color: #2563eb;
}

.header-right {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: flex-end;
}

.nav-section {
  display: flex;
  align-items: center;
  gap: 2rem;
}

.sidebar-toggle-btn {
  display: flex;
  padding: 0.35rem 0.6rem;
  margin-right: 0.5rem;
  background: transparent;
  border: 1px solid var(--border-color);
  border-radius: 0.5rem;
  color: var(--accent);
  cursor: pointer;
  transition: all 0.2s;
  align-items: center;

  &:hover {
    background: var(--sidebar-hover-bg);
  }

  &.sidebar-hidden {
    color: var(--text-muted);
  }
}

.nav-link {
  color: var(--text-secondary);
  font-weight: 500;
  text-decoration: none;
  transition: color 0.2s;
  display: flex;
  align-items: center;
  gap: 0.375rem;

  &:hover {
    color: var(--accent);
  }

  .nav-icon {
    width: 18px;
    height: 18px;
  }

  &.nav-link-active {
    color: var(--accent);
    font-weight: 600;
    border-bottom: 2px solid var(--accent);
    padding-bottom: 0.125rem;
  }

  &.nav-link-github {
    display: flex;
    align-items: center;
  }
}

// 隐藏滚动条
.hide-scrollbar {
  &::-webkit-scrollbar {
    display: none;
  }
  -ms-overflow-style: none;
  scrollbar-width: none;
}
</style>
