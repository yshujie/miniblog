<template>
  <div class="home-root">
   
    <!-- 中心区 -->
    <div class="main-info">
      <el-avatar class="avatar" :src="logo" size="large">
        <img :src="logo" alt="logo" />
      </el-avatar>
      <h1>Node.js全栈技术博客</h1>
      <p class="desc">
        专注于前端、Node.js、数据库及面试题解析的高质量技术博客，<br />
        帮助开发者快速成长的学习平台
      </p>
      <el-button type="success" size="large" class="read-btn">开始阅读</el-button>
    </div>

    <!-- 分割线 -->
    <div class="divider"></div>
    
    <!-- 三栏介绍 -->
    <div class="columns">
      <div class="column">
        <div class="col-title">功不唐捐</div>
        <div class="col-desc">记录每一天的进步，一分耕耘，一分收获。</div>
      </div>
      <div class="column">
        <div class="col-title">程序员成长指北</div>
        <div class="col-desc">
          专注 Node.js 技术栈分享，从前端到 Node.js 再到后端数据库，祝您成为优秀的高效 Node.js 全栈工程师
        </div>
      </div>
      <div class="column">
        <div class="col-title">koala</div>
        <div class="col-desc">
          一个有趣的且乐于分享的人。座右铭：今天未完成的，明天更不会完成。
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onActivated, onBeforeMount, onBeforeUnmount, onBeforeUpdate, onDeactivated, onErrorCaptured, onMounted, onRenderTracked, onRenderTriggered, onUnmounted, onUpdated } from 'vue'

const logo = new URL('@/assets/logo.png', import.meta.url).href

// onBeforeMount 生命周期钩子，在组件挂载前执行
onBeforeMount(async () => { 
  console.log(`Index component is now before mounted.`)
})

// onMounted 生命周期钩子，在组件挂载完成后执行
onMounted(() => {
  console.log(`Index component is now mounted.`)
})

// onBeforeUpdate 生命周期钩子，在组件更新前执行
onBeforeUpdate(() => { 
  console.log(`Index component is now before updated.`)
})

// onUpdated 生命周期钩子，在组件更新后执行
onUpdated(() => { 
  console.log(`Index component is now updated.`)
})

// onBeforeUnmount 生命周期钩子，在组件卸载前执行
onBeforeUnmount(() => { 
  console.log(`Index component is now before unmounted.`)
})

// onUnmounted 生命周期钩子，在组件卸载后执行
onUnmounted(() => { 
  console.log(`Index component is now unmounted.`)
})

// onErrorCaptured 声明周期钩子，在捕获了后代组件传递的错误时调用。
onErrorCaptured((err) => { 
  console.log(`Index component is now error captured.`, err)
})

// onRenderTracked 声明周期钩子，在组件渲染过程中追踪响应式依赖时调用。
// 这个钩子仅在开发模式下可用，且在服务器端渲染期间不会被调用。
onRenderTracked(() => { 
  console.log(`Index component is now render tracked.`)
})

// onRenderTriggered 声明周期钩子，在组件渲染过程中触发渲染时调用。
// 这个钩子仅在开发模式下可用，且在服务器端渲染期间不会被调用。
onRenderTriggered(() => { 
  console.log(`Index component is now render triggered.`)
})

// onActivated 声明周期钩子，在组件激活后调用。
// 若组件实例是 <KeepAlive> 缓存树的一部分，当组件被插入到 DOM 中时调用。
onActivated(() => { 
  console.log(`Index component is now activated.`)
})

// onDeactivated 声明周期钩子，在组件停用后调用。
// 若组件实例是 <KeepAlive> 缓存树的一部分，当组件从 DOM 中被移除时调用。
onDeactivated(() => { 
  console.log(`Index component is now deactivated.`)
})


</script>

<style lang="less" scoped>
.home-root {
  max-width: 900px;
  margin: 48px auto 0 auto;
  padding: 24px 12px;
}
.main-info {
  text-align: center;
  margin-bottom: 48px;

  .avatar {
    width: 96px;
    height: 96px;
    margin-bottom: 16px;
  }
  h1 {
    font-size: 36px;
    font-weight: bold;
    margin: 16px 0 8px 0;
    color: #333;
  }
  .desc {
    color: #888;
    font-size: 18px;
    margin-bottom: 22px;
  }
  .read-btn {
    padding: 10px 32px;
    font-size: 18px;
    font-weight: 500;
  }
}

.divider {
  height: 1px;
  background: #eee;
  margin: 48px 0;
}

.columns {
  display: flex;
  justify-content: space-between;
  gap: 24px;
  margin-top: 40px;

  .column {
    flex: 1 1 0;
    padding: 18px;
    background: #fff;
    border-radius: 10px;
    text-align: center;
    min-width: 180px;
  }
  .col-title {
    font-size: 18px;
    font-weight: bold;
    margin-bottom: 12px;
    color: #222;
  }
  .col-desc {
    color: #888;
    font-size: 15px;
    line-height: 1.8;
  }
}
</style>
