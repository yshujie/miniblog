<template>
  <el-card>
    <h2>标签：{{ $route.params.name }}</h2>
    <ul>
      <li v-for="article in filtered" :key="article.id">
        <router-link :to="`/article/${article.id}`">{{ article.title }}</router-link>
      </li>
    </ul>
  </el-card>
</template>
<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRoute } from 'vue-router'
import { fetchArticleList } from '../api/article'

const route = useRoute()
const articles = ref([])
onMounted(async () => {
  articles.value = await fetchArticleList()
})
const filtered = computed(() =>
  articles.value.filter((a: any) => a.tags.includes(route.params.name))
)
</script>
