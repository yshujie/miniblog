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
import { onBeforeMount, ref } from 'vue'
import { fetchModules } from '../api/module'
import type { Module } from '@/types/module'

const modules = ref<Module[]>([])

// onBeforeMount 生命周期钩子，在组件挂载前执行
onBeforeMount(async () => { 
  console.log(`Header component is now before mounted.`)

  // 获取模块列表
  modules.value = await fetchModules()
  console.log("modules", modules.value)
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
