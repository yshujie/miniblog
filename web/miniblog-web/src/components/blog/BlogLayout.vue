<template>

    <el-row class="blog-layout">
      <!-- 侧边栏 -->
      <el-col :span="4" class="sidebar">
        <aside class="sidebar"><slot name="sidebar" /></aside>
      </el-col>

      <!-- 主内容 -->
      <el-col :span="20" class="main">
        <main class="main"><slot name="main" /></main>
      </el-col>
    </el-row>

</template>
<script setup lang="ts">
</script>
<style lang="less" scoped >

.blog-layout {
  width: 100%;
  height: calc(100vh - 64px - 48px);
  overflow: hidden;

  .sidebar {
    width: 100%;
    height: calc(100vh - 64px - 48px);
    border-right: 1px solid #e2e8f0;
    overflow: hidden;
    box-shadow: inset 0 0 0 1px #e2e8f0;
  }  
}

</style>