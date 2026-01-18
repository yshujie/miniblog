<template>
  <div class="app-wrapper">
    <!-- 阅读进度条 -->
    <div class="reading-progress" v-if="isBlogPage">
      <div 
        class="reading-progress-bar" 
        :style="{ width: readingProgress + '%' }"
      ></div>
    </div>

    <el-container direction="vertical" class="app-container">
      <!-- 头部导航栏 -->
      <Header class="header-bar" v-if="needHeader" />

      <!-- 主体内容 -->
      <el-main class="main-content" :class="{ 'full-screen': fullScreen }">
        <router-view />
      </el-main>

      <!-- 页脚 -->
      <Footer class="footer-bar" v-if="needFooter" />
    </el-container>
  </div>
</template>

<script setup lang="ts">
import Header from './components/Header.vue'
import Footer from './components/Footer.vue'
import { computed, ref, onMounted, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'

const route = useRoute()

const needHeader = computed(() => {
  const paths = ['/404']
  return !paths.includes(route.path)
})

const needFooter = computed(() => {
  const paths = ['/404', '/blog']
  return !paths.includes(route.path)
})

const fullScreen = computed(() => {
  return !needHeader.value && !needFooter.value
})

// 判断是否是博客页面
const isBlogPage = computed(() => {
  return route.path.startsWith('/blog/')
})

// 阅读进度
const readingProgress = ref(0)

// 更新阅读进度
const updateReadingProgress = () => {
  const scrollY = window.scrollY
  const docHeight = document.documentElement.scrollHeight - window.innerHeight
  const progress = docHeight > 0 ? (scrollY / docHeight) * 100 : 0
  readingProgress.value = Math.min(100, Math.max(0, progress))
}

onMounted(() => {
  if (isBlogPage.value) {
    window.addEventListener('scroll', updateReadingProgress)
    updateReadingProgress()
  }
})

onUnmounted(() => {
  window.removeEventListener('scroll', updateReadingProgress)
})

</script>

<style lang="less" scoped>
.app-wrapper {
  position: relative;
  background: #f9fafb;
  min-height: 100vh;
}

.reading-progress {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 0.25rem;
  z-index: 60;
  background: #f3f4f6;

  .reading-progress-bar {
    height: 100%;
    background: #2563eb;
    transition: width 0.2s ease-out;
  }
}

.app-container {
  height: 100vh;
  background: transparent;
}

.header-bar {
  height: auto;
}

.main-content {
  padding: 0;
  margin-top: 0;
  width: 100%;
  min-height: calc(100vh - 64px);
  background: transparent;

  &.full-screen {
    margin-top: 0;
    height: 100vh;
  }
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
