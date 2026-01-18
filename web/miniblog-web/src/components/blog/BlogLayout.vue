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
  gap: 2rem;
  padding: 0.5rem 1.5rem;
  height: 100%;

  @media (min-width: 768px) {
    flex-direction: row;
  }
}

.main-content {
  flex: 1 1 0%;
  min-width: 0;
  background: #ffffff;
  border-radius: 1.5rem;
  box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.1), 0 8px 10px -6px rgba(0, 0, 0, 0.1);
  border: 1px solid #f3f4f6;
  overflow: hidden;
  animation: fadeIn 0.6s ease-out forwards;

  @keyframes fadeIn {
    from {
      opacity: 0;
      transform: translateY(10px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }
}
</style>