<template>
  <el-container direction="vertical" class="app-container">
    <!-- 头部导航栏 -->
    <Header class="header-bar" />

    <!-- 主体内容 -->
    <el-main class="main-content">
      <router-view />
    </el-main>

    <!-- 页脚 -->
    <Footer class="footer-bar" v-if="needFooter" />
  </el-container>
</template>

<script setup lang="ts">
import Header from './components/Header.vue'
import Footer from './components/Footer.vue'
import { onActivated, onBeforeMount, onBeforeUpdate, onBeforeUnmount, onDeactivated, onErrorCaptured, onMounted, onRenderTracked, onRenderTriggered, onUnmounted, onUpdated, computed } from 'vue'
import { useRoute } from 'vue-router'

const route = useRoute()
const needFooter = computed(() => {
  return !route.path.includes('/blog')
})

// onBeforeMount 生命周期钩子，在组件挂载前执行
onBeforeMount(async () => { 
  console.log(`App component is now before mounted.`)
})

// onMounted 生命周期钩子，在组件挂载完成后执行
onMounted(() => {
  console.log(`App component is now mounted.`)
})

// onBeforeUpdate 生命周期钩子，在组件更新前执行
onBeforeUpdate(() => { 
  console.log(`App component is now before updated.`)
})

// onUpdated 生命周期钩子，在组件更新后执行
onUpdated(() => { 
  console.log(`App component is now updated.`)
})

// onBeforeUnmount 生命周期钩子，在组件卸载前执行
onBeforeUnmount(() => { 
  console.log(`App component is now before unmounted.`)
})

// onUnmounted 生命周期钩子，在组件卸载后执行
onUnmounted(() => { 
  console.log(`App component is now unmounted.`)
})

// onErrorCaptured 声明周期钩子，在捕获了后代组件传递的错误时调用。
onErrorCaptured((err) => { 
  console.log(`App component is now error captured.`, err)
})

// onRenderTracked 声明周期钩子，在组件渲染过程中追踪响应式依赖时调用。
// 这个钩子仅在开发模式下可用，且在服务器端渲染期间不会被调用。
onRenderTracked(() => { 
  console.log(`App component is now render tracked.`)
})

// onRenderTriggered 声明周期钩子，在组件渲染过程中触发渲染时调用。
// 这个钩子仅在开发模式下可用，且在服务器端渲染期间不会被调用。
onRenderTriggered(() => { 
  console.log(`App component is now render triggered.`)
})

// onActivated 声明周期钩子，在组件激活后调用。
// 若组件实例是 <KeepAlive> 缓存树的一部分，当组件被插入到 DOM 中时调用。
onActivated(() => { 
  console.log(`App component is now activated.`)
})

// onDeactivated 声明周期钩子，在组件停用后调用。
// 若组件实例是 <KeepAlive> 缓存树的一部分，当组件从 DOM 中被移除时调用。
onDeactivated(() => { 
  console.log(`App component is now deactivated.`)
})
</script>

<style lang="less" scoped>
.app-container {
  min-height: 100vh;
  background: #fff;
}

.header-bar {
  height: 64px;
}

.main-content {
  padding: 0;
  margin-top: 64px;
  height: calc(100vh - 64px);
  overflow: hidden;
  background: #fff;
}

.footer-bar {
  position: fixed;
  bottom: 0;
  left: 0;
  right: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  text-align: center;
  height: 40px;
  color: #888;
  font-size: 15px;
  background: #fff;
  border-top: 1px solid #ececec;
}
</style>
