<template>
  <div class="app-wrapper">
    <el-container
      direction="vertical"
      class="app-container"
      :class="{ 'blog-layout-page': isBlogPage }"
    >
      <!-- 头部导航栏 -->
      <BlogHeader v-if="isBlogPage" class="header-bar" />
      <Header v-else-if="needHeader" class="header-bar" />

      <!-- 主体内容 -->
      <el-main
        class="main-content"
        :class="{
          'full-screen': fullScreen,
          'no-footer': !needFooter,
          'blog-page': isBlogPage
        }"
      >
        <router-view />
      </el-main>

      <!-- 页脚 -->
      <Footer class="footer-bar" v-if="needFooter" />
    </el-container>
  </div>
</template>

<script setup lang="ts">
import Header from './components/Header.vue'
import BlogHeader from './components/blog/BlogHeader.vue'
import Footer from './components/Footer.vue'
import { computed, watch, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'

const route = useRoute()

const BLOG_PAGE_BODY_CLASS = 'blog-page-body'

const needHeader = computed(() => {
  const paths = ['/404']
  if (route.path.startsWith('/blog')) return false
  return !paths.includes(route.path)
})

const needFooter = computed(() => {
  return route.path !== '/404' && !route.path.startsWith('/blog')
})

const fullScreen = computed(() => {
  if (isBlogPage.value) return false
  return !needHeader.value && !needFooter.value
})

const isBlogPage = computed(() => {
  return route.path.startsWith('/blog')
})

// 博客页时给 body 加 overflow: hidden，避免整页滚动导致上下窜动
watch(
  isBlogPage,
  (onBlog) => {
    if (onBlog) {
      document.body.classList.add(BLOG_PAGE_BODY_CLASS)
    } else {
      document.body.classList.remove(BLOG_PAGE_BODY_CLASS)
    }
  },
  { immediate: true }
)

onUnmounted(() => {
  document.body.classList.remove(BLOG_PAGE_BODY_CLASS)
})

</script>

<style lang="less" scoped>
.app-wrapper {
  position: relative;
  background: var(--page-bg);
  min-height: 100vh;
}

.app-container {
  min-height: 100vh;
  background: transparent;
  display: flex;
  flex-direction: column;

  /* 博客页：固定高度，让 main / BlogLayout 的 height:100% 有参照，避免侧栏和文章区被裁切 */
  &.blog-layout-page {
    height: 100vh;
    overflow: hidden;
  }
}

.header-bar {
  height: auto;
}

.main-content {
  flex: 1;
  padding: 0;
  margin-top: 0;
  width: 100%;
  min-height: calc(100vh - 64px);
  padding-bottom: 56px; /* 为固定页脚留出空间 */
  background: transparent;

  &.full-screen,
  &.no-footer {
    padding-bottom: 0;
  }

  &.full-screen {
    margin-top: 0;
    min-height: 100vh;
  }

  /* 博客页：禁止外层滚动，避免与文章区内滚动叠加导致上下窜动 */
  &.blog-page {
    overflow: hidden;
    min-height: 0;
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
  height: 48px;
  color: var(--text-muted);
  font-size: 0.875rem;
  background: var(--header-footer-bg);
  border-top: 1px solid var(--border-color);
  box-shadow: var(--shadow-sm);
}
</style>
