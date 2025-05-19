<template>
  <div class="blog-layout">
    <aside class="sidebar">
      <slot name="sidebar" />
    </aside>
    <main class="main">
      <slot name="main" />
    </main>
  </div>
</template>
<script setup lang="ts">
</script>
<style lang="less" scoped >
.blog-layout {
  position: relative;
  min-height: calc(100vh - 120px);
  display: flex;
  flex-direction: row;
  align-items: stretch;
  justify-content: flex-start;
  padding-top: 72px;
  padding-bottom: 48px;

  .sidebar {
    position: fixed;
    top: 64px;
    bottom: 60px;
    left: 0;
    width: 20%;
    max-width: 300px;
    height: auto;
    overflow-y: auto;
    z-index: 10;
    font-size: 16px;
    background-color: #fff;
    box-sizing: border-box;
    border-right: 1px solid #eaecef;
  }

  .main {
    flex: 1;
    width: 80%;
    margin-left: 20%;
    padding: 20px;
    box-sizing: border-box;
  }
}

</style>