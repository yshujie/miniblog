<template>
  <el-card v-for="article in articles" :key="article.id" class="mb-3">
    <ArticleCard :article="article" />
  </el-card>
</template>
<script setup lang="ts">
import { defineProps } from 'vue'
import { Article } from '../types/article'
import ArticleCard from './ArticleCard.vue'

defineProps<{ articles: Article[] }>()
</script>
<style scoped>
.mb-3 { margin-bottom: 18px; }
</style>
