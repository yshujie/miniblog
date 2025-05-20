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
import { computed } from 'vue'
import { useRoute } from 'vue-router'

const route = useRoute()
const needFooter = computed(() => {
  return !route.path.includes('/blog')
})

</script>

<style lang="less" scoped>
.app-container {
  height: 100vh;
  background: #fff;
}

.header-bar {
  height: 64px;
}

.main-content {
  padding: 0;
  margin-top: 64px;
  width: 100%;
  height: calc(100vh - 64px);
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
