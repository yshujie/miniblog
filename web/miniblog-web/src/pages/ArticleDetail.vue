<template>
  <div v-if="article">
    <h1>{{ article.title }}</h1>
    <div style="color:#888;margin-bottom:10px;">
      {{ article.author }} | {{ article.createdAt }}
      <span style="margin-left:18px;">
        <el-tag v-for="tag in article.tags" :key="tag" size="small">
          <router-link :to="`/tag/${tag}`">{{ tag }}</router-link>
        </el-tag>
      </span>
    </div>
    <div v-html="article.content"></div>
    <CommentList />
  </div>
  <div v-else>文章不存在</div>
</template>
<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { fetchArticleDetail } from '../api/article'
import CommentList from '../components/CommentList.vue'

const route = useRoute()
const article = ref<any>(null)

onMounted(async () => {
  article.value = await fetchArticleDetail(Number(route.params.id))
})
</script>
