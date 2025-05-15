<template>
  <div>
    <router-link :to="`/article/${article.id}`">
      <h2>{{ article.title }}</h2>
    </router-link>
    <p>{{ article.summary }}</p>
    <div style="font-size:12px;color:#888;">
      {{ article.author }} | {{ article.createdAt }} | 标签：
      <el-tag v-for="tag in article.tags" :key="tag" size="small" type="info">
        <router-link :to="`/tag/${tag}`">{{ tag }}</router-link>
      </el-tag>
    </div>
  </div>
</template>
<script setup lang="ts">
import { defineProps } from 'vue'
import { Article } from '../types/article'
defineProps<{ article: Article }>()
</script>
