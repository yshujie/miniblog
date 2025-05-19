<template>
  <div>
    <h2 class="article-title">{{ article.title }}</h2>
    <div class="article-info">
      <el-tag class="article-tag" v-for="tag in article.tags" :key="tag" type="primary" effect="dark">
        {{ tag }}
      </el-tag>
    </div>

    <div class="article-content">
      <vue3-markdown-it :source="article.content" />
    </div>
  </div>
</template>
<script setup lang="ts">
import Vue3MarkdownIt from 'vue3-markdown-it'
import { defineProps } from 'vue'
import type { Article } from '../../types/article'

defineProps<{ article: Article }>()

</script>
<style scoped>
.article-title {
  font-size: 24px;
  font-weight: bold;
  margin: 20px 0;
  text-align: left;
}

.article-info {
  font-size: 14px;
  color: #888;
  margin: 20px 0;
  line-height: 1.5;
  text-align: left;
  display: flex;
  flex-direction: row;
  flex-wrap: wrap;
  gap: 10px;
}

.article-content {
  margin: 20px 0;
}
</style>