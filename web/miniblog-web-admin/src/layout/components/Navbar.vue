<template>
  <header class="admin-header">
    <button class="navigation-toggle" type="button" aria-label="打开导航" :aria-expanded="navigationOpen" @click="$emit('navigation')"><svg-icon icon-class="list" /></button>
    <div class="admin-location"><span>管理端</span><span aria-hidden="true">/</span><strong>{{ String(route.meta.title || '内容管理') }}</strong></div>
    <a href="https://www.yangshujie.com" target="_blank" rel="noopener noreferrer" class="blog-link">查看博客 <svg-icon icon-class="link" /></a>
  </header>
</template>
<script setup lang="ts">
import { useRoute } from 'vue-router';
defineProps<{ navigationOpen: boolean }>(); defineEmits<{ navigation: [] }>();
const route = useRoute();
</script>
<style scoped>
.admin-header { height:60px; position:sticky; top:0; display:flex; align-items:center; gap:14px; padding:0 32px; background:white; border-bottom:1px solid var(--admin-line); z-index:10; }
.admin-location { display:flex; gap:16px; align-items:center; min-width:0; font-size:13px; }.admin-location span { color:var(--admin-muted); }.admin-location strong { font-weight:600; overflow:hidden; text-overflow:ellipsis; white-space:nowrap; }
.blog-link { margin-left:auto; min-height:44px; display:flex; gap:8px; align-items:center; white-space:nowrap; font-size:13px; color:var(--admin-muted); }.navigation-toggle { display:none; }
@media(max-width:780px) { .admin-header { padding:0 16px; gap:8px; }.navigation-toggle { display:grid; place-items:center; width:44px; height:44px; background:white; border:1px solid var(--admin-line); border-radius:6px; }.admin-location { gap:8px; }.admin-location > span { display:none; }.blog-link { font-size:12px; } }
@media(max-width:360px) { .admin-header { padding:0 12px; } }
</style>
