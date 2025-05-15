<template>
  <el-row :gutter="20">
    <el-col :span="16">
      <ArticleList :articles="articles" />
    </el-col>
    <el-col :span="8">
      <Sidebar />
    </el-col>
  </el-row>
</template>
<script setup lang="ts">
import { ref, onMounted } from 'vue'
import ArticleList from '../components/ArticleList.vue'
import Sidebar from '../components/Sidebar.vue'
import { fetchArticleList } from '../api/article'

const articles = ref([])
onMounted(async () => {
  articles.value = await fetchArticleList()
})
</script>
