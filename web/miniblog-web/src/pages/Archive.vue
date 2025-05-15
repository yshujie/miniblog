<template>
  <el-card>
    <h2>文章归档</h2>
    <ul>
      <li v-for="article in articles" :key="article.id">
        <router-link :to="`/article/${article.id}`">
          {{ article.createdAt }} - {{ article.title }}
        </router-link>
      </li>
    </ul>
  </el-card>
</template>
<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { fetchArticleList } from '../api/article'
const articles = ref([])
onMounted(async () => {
  articles.value = await fetchArticleList()
})
</script>
