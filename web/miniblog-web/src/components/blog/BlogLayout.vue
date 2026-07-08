<template>
  <main class="blog-layout">
    <div class="blog-body">
      <slot name="sidebar" />
      <article class="main-content" :class="{ 'main-content-expanded': !sidebarOpen }">
        <slot name="main" />
        <div
          class="sidebar-toggle-trigger"
          :class="{ 'sidebar-closed': !sidebarOpen }"
          :title="sidebarOpen ? '隐藏侧边栏' : '展开侧边栏'"
          @click="toggleSidebar"
        >
          <span class="sidebar-toggle-icon">{{ sidebarOpen ? '›' : '‹' }}</span>
        </div>
      </article>
    </div>
  </main>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useUiStore } from '@/stores/ui'

const uiStore = useUiStore()
const sidebarOpen = computed(() => uiStore.sidebarOpen)

const toggleSidebar = () => {
  uiStore.toggleSidebar()
}
</script>

<style lang="less" scoped>
.blog-layout {
  margin-left: auto;
  margin-right: auto;
  display: flex;
  flex-direction: column;
  gap: 0;
  padding: 0;
  width: 100%;
  min-height: 100%;
  height: 100%;
  box-sizing: border-box;
  overflow: hidden;
}

.blog-body {
  flex: 1 1 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;

  @media (min-width: 768px) {
    flex-direction: row;
  }
}

.main-content {
  position: relative;
  flex: 1 1 0%;
  min-width: 0;
  min-height: 0;
  background: var(--card-bg);
  overflow: hidden;
  border-left: 1px solid var(--blog-header-border);
  transition: border-color 0.25s ease;

  &.main-content-expanded {
    border-left-color: transparent;
  }
}

.sidebar-toggle-trigger {
  display: none;
  position: absolute;
  left: 0;
  top: 50%;
  transform: translateY(-50%);
  z-index: 50;
  width: 28px;
  height: 56px;
  align-items: center;
  justify-content: center;
  background: var(--card-bg);
  border: 1px solid var(--sidebar-divider);
  border-left: none;
  border-radius: 0 8px 8px 0;
  cursor: pointer;
  box-shadow: 2px 0 6px rgba(0, 0, 0, 0.04);
  opacity: 0;
  transition: opacity 0.2s ease, background 0.2s ease;

  .sidebar-toggle-icon {
    font-size: 1.25rem;
    font-weight: 600;
    color: var(--sidebar-text);
    line-height: 1;
  }

  .main-content:hover &,
  &.sidebar-closed {
    opacity: 1;
  }

  &:hover {
    background: var(--sidebar-bg);

    .sidebar-toggle-icon {
      color: var(--blog-header-active);
    }
  }

  &.sidebar-closed {
    border-left: 1px solid var(--sidebar-divider);

    .sidebar-toggle-icon {
      color: var(--blog-header-active);
    }
  }

  &:focus-visible {
    opacity: 1;
    outline: 2px solid var(--blog-header-active);
    outline-offset: 2px;
  }

  @media (min-width: 1280px) {
    display: flex;
  }
}
</style>