<template>
  <div class="sidebar-root">
    <div class="section-list">
      <div
        v-for="item in props.sections"
        :key="item.title"
        :index="item.code"
        class="section-item"
      >
        <h3 class="section-title">
          {{ item.title }}
        </h3>
        <div class="article-list">
          <div class="article-item" v-for="article in item.articles" :key="article.title">
            <el-link underline="never" target="_self" @click="handleArticleClick(article.id)">
              <span class="article-title" :class="{ 'article-title-active': article.id === currentArticleId }">{{ article.title }}</span>
            </el-link>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
<script setup lang="ts">
import { computed, defineProps } from 'vue'
import { useRouter } from 'vue-router'
import type { Section } from '@/types/section';

const props = defineProps<{ sections: Section[], moduleCode: string }>()

const router = useRouter()

// 当前选中的文章 ID
const currentArticleId = computed(() => {
  const articleId = router.currentRoute.value.params.article
  if (articleId) {
    return String(articleId)
  }
  return null
})

// 文章点击事件
const handleArticleClick = (articleId: string) => {
  router.push(`/blog/${props.moduleCode}/article/${articleId}`)
}
</script>
<style scoped lang="less">
.sidebar-root {
  overflow-y: auto;
  background: #ffffff;
  padding: 24px 20px;

  .section-list {

    .section-item {
      margin-bottom: 32px;

        .section-title {
          line-height: 1.3;
          font-size: 16px;
          font-weight: 600;
          color: #1e293b;
          padding: 0 0 16px 0;
          margin: 0 0 20px 0;
          position: relative;
          
          &::before {
            content: '';
            position: absolute;
            left: 0;
            bottom: 0;
            width: 40px;
            height: 3px;
            background: linear-gradient(90deg, #3b82f6, #8b5cf6);
            border-radius: 2px;
          }
        }

        .article-list {
          .article-item {
            cursor: pointer;
            padding: 10px 16px;
            margin: 4px 0;
            border-radius: 8px;
            transition: all 0.2s ease;
            position: relative;

            &:hover {
              background: #f8fafc;
              // transform: translateX(4px);
              
              .article-title {
                color: #3b82f6;
              }
            }

            &:active {
              background: #f1f5f9;
              
              .article-title {
                color: #2563eb;
              }
            }
          }

          .article-title {
            line-height: 1.3;
            font-size: 13px;
            font-weight: 500;
            color: #475569;
            transition: all 0.2s ease;
            display: block;
            text-decoration: none;
            position: relative;
          }

          .article-title:hover {
            color: #3b82f6;
          }

          .article-title-active {
            color: #2563eb;
            font-weight: 600;
            
            &::before {
              content: '';
              position: absolute;
              left: -16px;
              top: 50%;
              transform: translateY(-50%);
              width: 4px;
              height: 20px;
              background: linear-gradient(180deg, #3b82f6, #8b5cf6);
              border-radius: 2px;
            }
          }
        }
    }   
  }
}
</style>
