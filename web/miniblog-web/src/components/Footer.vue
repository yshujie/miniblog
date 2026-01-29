<template>
  <el-footer class="footer-bar">
    &copy; 2025 Shujie's Blog · Powered by Golang + Vue3 + Element Plus
  </el-footer>
</template>

<script setup lang="ts">

</script>

<style lang="less" scoped>
.footer-bar {
  position: fixed;
  bottom: 0;
  left: 0;
  right: 0;
  z-index: 40;
  text-align: center;
  color: var(--text-muted);
  font-size: 0.875rem;
  height: 48px;
  line-height: 48px;
  background: var(--header-footer-bg);
  border-top: 1px solid var(--border-color);
  box-shadow: var(--shadow-sm);
}
</style>