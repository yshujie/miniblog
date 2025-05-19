<template>
    <div class="article-card">
      <el-row>
        <!-- 文章预览 -->
        <el-col :span="20" class="article-card-preview">

          <!-- 文章头部 -->
          <div class="article-card-header">
            <h2 class="article-title">{{ article.title }}</h2>
            <div class="article-info">
              <el-tag class="article-tag" v-for="tag in article.tags" :key="tag" type="primary" effect="dark">{{ tag }}</el-tag>
            </div>
          </div>

          <!-- 文章内容 -->
          <div class="article-card-content">
            <MdPreview 
              :editorId="editorId" 
              :modelValue="article.content" 
              class="article-content-preview"
              :scrollElement="scrollElement"
            />
          </div>
        </el-col>

        <!-- 文章目录 -->
        <el-col :span="4" class="article-card-catalog">
          <div class="article-content-catalog-container">
            <MdCatalog
              :editorId="editorId"
              :modelValue="article.content"
              class="article-content-catalog"
            />
          </div>
        </el-col>
      
      </el-row>

    </div>
</template>
<script setup lang="ts">
import { computed, defineProps, ref, onMounted } from 'vue'
import type { Article } from '../../types/article'
import { MdPreview, MdCatalog } from 'md-editor-v3'
import 'md-editor-v3/lib/style.css'

const props = defineProps<{ article: Article }>()
const scrollElement = ref<HTMLElement>()

const editorId = computed(() => {
  return `article-${props.article.id || props.article.title}`
})

onMounted(() => {
  const element = document.querySelector('.article-content')
  if (element instanceof HTMLElement) {
    scrollElement.value = element
  }
})
</script>
<style scoped lang="less">
.article-card {
  display: flex;
  flex-direction: column;
  gap: 20px;
  height: calc(100vh - 120px); // 减去头部和底部的高度

  .article-card-preview {
    border-right: 1px solid #e0e0e0;
    height: 100%;
    overflow-y: auto;
    padding-right: 20px;

    &::-webkit-scrollbar {
      width: 6px;
    }

    &::-webkit-scrollbar-thumb {
      background-color: #ddd;
      border-radius: 3px;
    }
  }

  .article-card-header {
    border-bottom: 1px solid #e0e0e0;
    padding-bottom: 20px;
    margin-bottom: 20px;

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
  }

  .article-card-content {
    margin: 0 40px 0 0;
    padding: 20px 0;
  }

  .article-card-catalog {
    height: 100%;
    overflow-y: auto;
    padding-left: 20px;

    &::-webkit-scrollbar {
      width: 6px;
    }

    &::-webkit-scrollbar-thumb {
      background-color: #ddd;
      border-radius: 3px;
    }

    .article-content-catalog-container {
      position: sticky;
      top: 0;
    }
  }
}
</style>