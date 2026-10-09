<template>
  <footer class="site-footer"><span>© {{ year }} Shujie. 功不唐捐，玉汝于成。</span><nav aria-label="页脚导航"><router-link to="/">首页</router-link><a href="https://github.com/yshujie" target="_blank" rel="noopener noreferrer">GitHub</a></nav></footer>
</template>
<script setup lang="ts">
const year = new Date().getFullYear()
</script>
<style scoped>
.site-footer { width: calc(100% - 64px); max-width: 1096px; margin: 14px auto 0; padding: 24px 0 30px; display: flex; justify-content: space-between; flex-wrap: wrap; gap: 16px; border-top: 1px solid var(--line); font-size: 12px; color: var(--secondary); }
.site-footer nav { display: flex; gap: 20px; } .site-footer a:hover { color: var(--green); }
@media (max-width: 650px) { .site-footer { width: calc(100% - 48px); padding: 22px 0 max(22px, env(safe-area-inset-bottom)); font-size: 11px; margin-top: 0; } }
</style>
