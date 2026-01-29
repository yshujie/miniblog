<template>
  <main class="blog-layout">
    <!-- 侧边栏 -->
    <slot name="sidebar" />
    
    <!-- 主内容 -->
    <article class="main-content">
      <slot name="main" />
    </article>
  </main>
</template>

<script setup lang="ts">
</script>

<style lang="less" scoped>
.blog-layout {
  margin-left: auto;
  margin-right: auto;
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
  padding: 1rem 1.5rem;
  width: 100%;
  /* 填满父级可用高度，避免 100vh 超出有 Header 时的视口导致整页滚动、上下窜动 */
  min-height: 100%;
  height: 100%;
  box-sizing: border-box;
  overflow: hidden;

  @media (min-width: 768px) {
    flex-direction: row;
    padding: 1.25rem 1.5rem;
    gap: 1.5rem;
  }
}

.main-content {
  flex: 1 1 0%;
  min-width: 0;
  min-height: 0;
  background: var(--card-bg);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-md);
  border: 1px solid var(--border-light);
  overflow-y: auto;
  overflow-x: hidden;
  animation: fadeIn 0.4s ease-out forwards;

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