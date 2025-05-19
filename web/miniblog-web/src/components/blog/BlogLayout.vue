<template>

  <template class="blog-layout">
    <el-row>
      <!-- 侧边栏 -->
      <el-col :span="4">
        <aside class="sidebar"><slot name="sidebar" /></aside>
      </el-col>

      <!-- 主内容 -->
      <el-col :span="20">
        <main class="main"><slot name="main" /></main>
      </el-col>
    </el-row>

  </template>

</template>
<script setup lang="ts">
</script>
<style lang="less" scoped >

.blog-layout {
  width: 100%;
  height: calc(100vh - 64px - 60px);
  overflow: hidden;
  display: flex;
  flex-direction: row;
  align-items: stretch;
  background-color: #fff;

  .sidebar {
    border-right: 1px solid #eaecef;
    height: calc(100vh - 64px - 60px);
    overflow: hidden;
    overflow-y: scroll;
  }

  .sidebar, .main {
    overflow: hidden;
    overflow-y: scroll;
  }
}
</style>