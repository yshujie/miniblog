<template>
    <div class="article-container">
      <div v-show="!hasArticle" class="no-article-card">
          <el-empty description=" " />
      </div>

      <div v-show="hasArticle" class="article-card">
        <el-row>
          <!-- 文章预览 -->
          <el-col :span="19">
            <div class="article-card-preview" ref="previewScrollDom">
              <!-- 文章头部 -->
              <div class="article-card-header">
                <h2 class="article-title">{{ currentArticle?.title }}</h2>
                <div class="article-info">
                  <el-tag class="article-tag" v-for="tag in currentArticle?.tags" :key="tag" type="primary" effect="dark">{{ tag }}</el-tag>
                </div>
              </div>

              <!-- 文章内容 -->
              <div class="article-card-content">
                <MdPreview 
                  :editorId="editorId" 
                  :modelValue="currentArticle?.content"
                  :showCodeRowNumber="true"
                  :previewTheme="'default'"
                  :previewOnly="true"
                  :previewOptions="{
                    markdown: {
                      breaks: true,
                      html: true,
                      gfm: true,
                      tasklists: true,
                      tables: true,
                      sanitize: false,
                      smartLists: true,
                      smartypants: true
                    }
                  }"
                />
              </div>
            </div>          
          </el-col>

          <!-- 文章目录 -->
          <el-col :span="5">
            <div class="article-card-catalog">
              <MdCatalog
                :editorId="editorId"
                :modelValue="currentArticle?.content"
                :scrollElement="scrollElement"
                :theme="'light'"
                :scrollElementOffsetTop="80"
                :scrollElementOffsetBottom="20"
                :catalogVisible="true"
                :catalogMaxHeight="'calc(100vh - 300px)'"
              />
            </div>
          </el-col>
        </el-row>
      </div>
    </div>
</template>
<script setup lang="ts">
import 'md-editor-v3/lib/style.css'
import { MdPreview, MdCatalog } from 'md-editor-v3'
import { computed, defineProps, ref, onMounted, nextTick, onUpdated } from 'vue'
import { Article } from '@/types/article'
import { fetchArticleDetail } from '@/api/blog'

// 组件 props
const props = defineProps<{ articleId: string|null }>()

// 组件数据
const editorId = `article-${props.articleId}`
const scrollElement = ref<HTMLElement>()
const previewScrollDom = ref<HTMLElement | null>(null)
// 当前文章
const currentArticle = ref<Article | null>(null)

// 计算属性：hasArticle
const hasArticle = computed(() => {
  return currentArticle.value !== null && currentArticle.value instanceof Article
})

// 组件挂载时，获取文章
onMounted(async () => {
  console.log('onMounted')
  console.log("props.articleId", props.articleId)
  await fetchCurrentArticle(props.articleId)
})

onUpdated(async () => {
  console.log('onUpdated')
  console.log("props.articleId", props.articleId)
  await fetchCurrentArticle(props.articleId)
})

// 获取文章详情
async function fetchCurrentArticle(articleId: string | null) {
  if (!articleId) {
    return null
  }

  const article = await fetchArticleDetail(articleId)
  console.log('article detail:', article)
  if (!article) {
    return
  }

  currentArticle.value = article
}

</script>
<style scoped lang="less">
.article-container {
  height: 100%;
}

.no-article-card {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 100%;
  margin-top: 150px;
}

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
    margin: 0 40px 100px 0;
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