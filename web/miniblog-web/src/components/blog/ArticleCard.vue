<template>
    <div class="article-card">
      <el-row>
        <!-- 文章预览 -->
        <el-col :span="19">
          <div class="article-card-preview"  ref="previewScrollDom">
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
              />
            </div>
          </div>          
        </el-col>

        <!-- 文章目录 -->
        <el-col :span="5">
          <div class="article-card-catalog">
            <MdCatalog
              :editorId="editorId"
              :modelValue="article.content"
              :scrollElement="scrollElement"
            />
          </div>
        </el-col>
      </el-row>
    </div>
</template>
<script setup lang="ts">
import { computed, defineProps, ref, onMounted, nextTick } from 'vue'
import type { Article } from '../../types/article'
import { MdPreview, MdCatalog } from 'md-editor-v3'
import 'md-editor-v3/lib/style.css'

const props = defineProps<{ article: Article }>()
const scrollElement = ref<HTMLElement>()

const previewScrollDom = ref<HTMLElement | null>(null)
const editorId = `article-${props.article.id || props.article.title}`

</script>
<style scoped lang="less">
.article-card {
  padding: 0;
  margin: 0;
  height: calc(100vh - 60px); // 减去头部和底部的高度

  .article-card-preview {
    margin: 0;
    padding: 0 32px 32px 64px;
    height: calc(100vh - 60px);
    overflow-y: scroll;
    border-right: 1px solid #e0e0e0;
    
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
    padding: 20px 8px;
  }
}
</style>

<style>
.md-editor-catalog-indicator {
  background-color: #409EFF !important;
}
.md-editor-catalog-active > span {
  color: #409EFF;
}
.md-editor-catalog-link > span:hover {
  color: #409EFF;
}
.md-editor-catalog-link > span:active {
  color: #409EFF;
}

</style>