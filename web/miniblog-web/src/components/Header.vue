<template>
  <el-header class="header-bar">
    <div class="left">
      <a href="/">
        <span class="logo">clack 的技术博客</span>
      </a>
    </div>

    <div class="right">
 
    <div class="nav">
      <el-menu mode="horizontal" :default-active="currentFullPath" router>
        <el-menu-item index="/">首页</el-menu-item>
        <el-menu-item v-for="module in moduleStore.modules" :key="module.code" :index="`/blog/${module.code}`" @click="handleModuleClick(module.code)">
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
import { computed, onBeforeMount } from 'vue'
import { useModuleStore } from '@/stores/module'
import { useRoute } from 'vue-router'

// module store
const moduleStore = useModuleStore()

// 获取路由实例
const route = useRoute()

// 计算属性 currentFullPath
const currentFullPath = computed(() => {
  // 如果 route.path 以 /blog/ 开头，则返回 /blog/xxx 前两位
  if (route.path.startsWith('/blog/')) {
    const path = route.path.split('/')
    return '/' + path[1] + '/' + path[2]
  }

  return '/'
})


onBeforeMount(async () => {
  console.log('onBeforeMount')

  // 加载模块数据
  await moduleStore.loadModules()

  // 预热所有模块数据
  await moduleStore.loadAllModuleDetail()
})


// 模块点击事件
const handleModuleClick = (moduleCode: string) => {
  // 加载模块详情
  moduleStore.loadModuleDetail(moduleCode)
}

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

  a {
    color: inherit;
    text-decoration: none;
  }
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
