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
            <el-link underline="never" target="_self" :href="`/blog/${props.moduleCode}/article/${article.id}`">
              <span class="article-title">{{ article.title }}</span>
            </el-link>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
<script setup lang="ts">
import { defineProps } from 'vue'
import type { Section } from '@/types/section';

const props = defineProps<{ sections: Section[], moduleCode: string }>()

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
                color: #213547;
              }
            }

            &:active {
              .article-title {
                color: #213547;
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
        }
    }   
  }
}
</style>
