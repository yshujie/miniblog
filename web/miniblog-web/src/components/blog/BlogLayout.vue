<template>
  <main class="blog-layout">
    <div class="blog-body">
      <slot name="sidebar" />
      <article class="main-content">
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
  border-radius: 0;
  box-shadow: none;
  border: none;
  overflow-y: auto;
  overflow-x: hidden;
  animation: fadeIn 0.4s ease-out forwards;
}

/* 文章区左侧悬停显隐侧边栏按钮（大屏显示，无间隙） */
.sidebar-toggle-trigger {
  display: none;
  position: absolute;
  left: 0;
  top: 50%;
  transform: translateY(-50%);
  z-index: 50;
  pointer-events: auto;
  width: 40px;
  height: 80px;
  align-items: center;
  justify-content: center;
  background: var(--zone-bg);
  border: 1px solid var(--border-divider);
  border-left: none;
  border-radius: 0 12px 12px 0;
  cursor: pointer;
  box-shadow: 2px 0 8px rgba(0, 0, 0, 0.06);
  opacity: 0.7;
  transition: width 0.25s ease, opacity 0.25s ease, background 0.2s ease, box-shadow 0.25s ease;

  &:hover {
    width: 80px;
    opacity: 1;
    background: var(--card-bg);
    box-shadow: 2px 0 14px rgba(0, 0, 0, 0.12);
  }

  .sidebar-toggle-icon {
    display: block;
    width: 100%;
    text-align: center;
    font-size: 1.5rem;
    font-weight: 600;
    color: var(--text-muted);
    transition: color 0.2s ease;
    white-space: nowrap;
    overflow: hidden;
    opacity: 0;
    transition: opacity 0.2s ease;
  }

  &:hover .sidebar-toggle-icon {
    opacity: 1;
    color: var(--accent);
  }

  /* 侧栏隐藏时：按钮更醒目，方便用户点回 */
  &.sidebar-closed {
    width: 60px;
    opacity: 1;
    background: var(--card-bg);
    border-color: var(--border-divider);
    box-shadow: 2px 0 12px rgba(0, 0, 0, 0.1), 0 0 0 1px rgba(250, 137, 25, 0.2);
    border-left: 3px solid var(--accent);

    .sidebar-toggle-icon {
      opacity: 1;
      color: var(--accent);
      font-size: 1.625rem;
    }

    &:hover {
      width: 80px;
      box-shadow: 2px 0 16px rgba(0, 0, 0, 0.14), 0 0 0 1px rgba(250, 137, 25, 0.3);
    }
  }

  &:focus-visible {
    outline: 2px solid var(--accent);
    outline-offset: 2px;
  }

  @media (min-width: 1280px) {
    display: flex;
  }

  @keyframes fadeIn {
    from {
      opacity: 0;
      transform: translateY(8px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }
}
</style>