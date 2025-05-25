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
import { defineProps, ref } from 'vue'
import { useRouter } from 'vue-router'
import type { Section } from '@/types/section';

const props = defineProps<{ sections: Section[], moduleCode: string }>()

const router = useRouter()

const currentArticleId = ref<number | null>(null)

const handleArticleClick = (articleId: number) => {
  currentArticleId.value = articleId
  router.push(`/blog/${props.moduleCode}/article/${articleId}`)
}
</script>
<style scoped lang="less">
.sidebar-root {
  width: 100%;
  height: 100%;
  overflow-y: scroll;

  .section-list {

    .section-item {
      padding: 8px;
      border-radius: 5px;

        .section-title {
          line-height: 16px;
          font-size: 14px;
          font-weight: 600;
          color: #213547;
          padding: 4px 0;
        }

        .article-list {
          .article-item {
            cursor: pointer;
            padding: 4px 0;

            &:hover {
              .article-title {
                color: #409eff;
              }
            }

            &:active {
              .article-title {
                color: #409eff;
              }
            }
          }

          .article-title {
            line-height: 16px;
            font-size: 14px;
            font-weight: 500;
            color: rgba(60, 60, 60, .7);
            transition: color .5s;
          }

          .article-title-active {
            color: #409eff;
          }
        }
    }   
  }
}
</style>
