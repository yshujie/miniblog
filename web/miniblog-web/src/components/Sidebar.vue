<template>
  <el-card>
    <h3>站点公告</h3>
    <p>GoodBlog Vue3 Demo，欢迎体验！</p>
    <h3>标签</h3>
    <el-tag v-for="tag in tags" :key="tag" class="mr-2">
      <router-link :to="`/tag/${tag}`">{{ tag }}</router-link>
    </el-tag>
  </el-card>
</template>
<script setup lang="ts">
const tags = ['Vue3', '博客', 'TypeScript']
</script>
<style scoped>
.mr-2 { margin-right:8px; }
</style>
