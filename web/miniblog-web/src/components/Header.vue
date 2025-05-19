<template>
  <el-header class="header-bar">
    <div class="left">
      <span class="logo">Node.js全栈技术博客</span>
    </div>

    <div class="right">
 
    <div class="nav">
      <el-menu mode="horizontal" :default-active="'/'" router>
        <el-menu-item index="/">首页</el-menu-item>
        <el-menu-item v-for="module in modules" :key="module.code" :index="`/blog/${module.code}`">
          {{ module.title }}
        </el-menu-item>
        <el-menu-item key="github">
          <a href="https://github.com/yshujie" target="_blank" rel="noopener">GitHub</a>
        </el-menu-item>
      </el-menu>
    </div>
  </div>

  </el-header>
</template>

<script setup lang="ts">
import { onActivated, onBeforeMount, onBeforeUnmount, onBeforeUpdate, onDeactivated, onErrorCaptured, onMounted, onRenderTracked, onRenderTriggered, onUnmounted, onUpdated, ref } from 'vue'
import { fetchModules } from '../api/module'
import type { Module } from '@/types/module'

const modules = ref<Module[]>([])

// onBeforeMount 生命周期钩子，在组件挂载前执行
onBeforeMount(async () => { 
  console.log(`Header component is now before mounted.`)

  // 获取模块列表
  modules.value = await fetchModules()
})

// onMounted 生命周期钩子，在组件挂载完成后执行
onMounted(() => {
  console.log(`Header component is now mounted.`)
})

// onBeforeUpdate 生命周期钩子，在组件更新前执行
onBeforeUpdate(() => { 
  console.log(`Header component is now before updated.`)
})

// onUpdated 生命周期钩子，在组件更新后执行
onUpdated(() => { 
  console.log(`Header component is now updated.`)
})

// onBeforeUnmount 生命周期钩子，在组件卸载前执行
onBeforeUnmount(() => { 
  console.log(`Header component is now before unmounted.`)
})

// onUnmounted 生命周期钩子，在组件卸载后执行
onUnmounted(() => { 
  console.log(`Header component is now unmounted.`)
})

// onErrorCaptured 声明周期钩子，在捕获了后代组件传递的错误时调用。
onErrorCaptured((err) => { 
  console.log(`Header component is now error captured.`, err)
})

// onRenderTracked 声明周期钩子，在组件渲染过程中追踪响应式依赖时调用。
// 这个钩子仅在开发模式下可用，且在服务器端渲染期间不会被调用。
onRenderTracked(() => { 
  console.log(`Header component is now render tracked.`)
})

// onRenderTriggered 声明周期钩子，在组件渲染过程中触发渲染时调用。
// 这个钩子仅在开发模式下可用，且在服务器端渲染期间不会被调用。
onRenderTriggered(() => { 
  console.log(`Header component is now render triggered.`)
})

// onActivated 声明周期钩子，在组件激活后调用。
// 若组件实例是 <KeepAlive> 缓存树的一部分，当组件被插入到 DOM 中时调用。
onActivated(() => { 
  console.log(`Header component is now activated.`)
})

// onDeactivated 声明周期钩子，在组件停用后调用。
// 若组件实例是 <KeepAlive> 缓存树的一部分，当组件从 DOM 中被移除时调用。
onDeactivated(() => { 
  console.log(`Header component is now deactivated.`)
})
</script>

<style lang="less" scoped>
.header-bar {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  z-index: 100;
  display: flex;
  align-items: center;
  height: 64px;
  padding: 0 32px;
  background: #fff;
  border-bottom: 1px solid #eee;
  box-shadow: 0 2px 8px 0 rgba(0,0,0,0.03);
}

.left {
  flex: 0 0 auto;
  font-weight: bold;
  font-size: 22px;
  color: #333;
}

// 右侧，靠屏幕右方
.right {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  flex: 1 1 auto;

  .search-box {
    width: 320px;
    margin: 0 40px;
  }
  .nav {
    flex: 1 1 auto;

    .el-menu {
      display: flex;
      justify-content: flex-end;
      border-bottom: none;
      background: transparent;
      font-size: 16px;
    }

    .el-menu-item a {
      color: inherit;
      text-decoration: none;
    }
  }
}
</style>
