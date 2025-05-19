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
            <el-link underline="never" target="_self" :href="`/blog/${props.module.code}/${article.id}`">
              <span class="article-title">{{ article.title }}</span>
            </el-link>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
<script setup lang="ts">
import { defineProps, onActivated, onBeforeMount, onBeforeUnmount, onBeforeUpdate, onDeactivated, onErrorCaptured, onMounted, onRenderTracked, onRenderTriggered, onUnmounted, onUpdated } from 'vue'
import type { Section } from '@/types/section';
import type { Module } from '@/types/module';

const props = defineProps<{ module: Module, sections: Section[] }>()

// onBeforeMount 生命周期钩子，在组件挂载前执行
onBeforeMount(async () => { 
  console.log(`Sidebar component is now before mounted.`)
})

// onMounted 生命周期钩子，在组件挂载完成后执行
onMounted(() => {
  console.log(`Sidebar component is now mounted.`)
})

// onBeforeUpdate 生命周期钩子，在组件更新前执行
onBeforeUpdate(() => { 
  console.log(`Sidebar component is now before updated.`)
})

// onUpdated 生命周期钩子，在组件更新后执行
onUpdated(() => { 
  console.log(`Sidebar component is now updated.`)
})

// onBeforeUnmount 生命周期钩子，在组件卸载前执行
onBeforeUnmount(() => { 
  console.log(`Sidebar component is now before unmounted.`)
})

// onUnmounted 生命周期钩子，在组件卸载后执行
onUnmounted(() => { 
  console.log(`Sidebar component is now unmounted.`)
})

// onErrorCaptured 声明周期钩子，在捕获了后代组件传递的错误时调用。
onErrorCaptured((err) => { 
  console.log(`Sidebar component is now error captured.`, err)
})

// onRenderTracked 声明周期钩子，在组件渲染过程中追踪响应式依赖时调用。
// 这个钩子仅在开发模式下可用，且在服务器端渲染期间不会被调用。
onRenderTracked(() => { 
  console.log(`Sidebar component is now render tracked.`)
})

// onRenderTriggered 声明周期钩子，在组件渲染过程中触发渲染时调用。
// 这个钩子仅在开发模式下可用，且在服务器端渲染期间不会被调用。
onRenderTriggered(() => { 
  console.log(`Sidebar component is now render triggered.`)
})

// onActivated 声明周期钩子，在组件激活后调用。
// 若组件实例是 <KeepAlive> 缓存树的一部分，当组件被插入到 DOM 中时调用。
onActivated(() => { 
  console.log(`Sidebar component is now activated.`)
})

// onDeactivated 声明周期钩子，在组件停用后调用。
// 若组件实例是 <KeepAlive> 缓存树的一部分，当组件从 DOM 中被移除时调用。
onDeactivated(() => { 
  console.log(`Sidebar component is now deactivated.`)
})


</script>
<style scoped lang="less">
.sidebar-root {
  border-right: 1px solid #eaecef;
  width: 100%;
  padding: 0 32px 32px;

  .section-list {
    display: flex;
    flex-direction: column;
    gap: 10px;

    .section-item {
      padding: 10px;
      border-radius: 5px;

        .section-title {
          line-height: 20px;
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
            line-height: 20px;
            font-size: 13px;
            font-weight: 500;
            color: rgba(60, 60, 60, .7);
            transition: color .5s;
          }
        }
    }   
  }


}
</style>
