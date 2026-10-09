<template><section class="collect-entry"><p class="eyebrow">目录与收录</p><h1>收录外部文章</h1><p>正文继续在外部文档中维护，在收录面板选择目录、核对标题并发布。</p><el-alert v-if="!contentRegistrationEnabled" title="收录功能暂未启用，现有文章仍可管理。" type="info" :closable="false" /><div class="entry-actions"><el-button type="primary" :disabled="!contentRegistrationEnabled" @click="open = true">粘贴链接收录</el-button><router-link to="/content/workbench">进入目录工作台 →</router-link></div><QuickCollectDrawer v-model="open" :context="context" /></section></template>
<script setup lang="ts">
import { computed, ref } from 'vue';
import { useRoute } from 'vue-router';
import QuickCollectDrawer from '@/components/content/QuickCollectDrawer.vue';
import { contentRegistrationEnabled } from '@/utils/content-flags';
import type { DirectoryContext } from '@/types/content';
const route = useRoute(); const open = ref(contentRegistrationEnabled);
const context = computed<DirectoryContext>(() => ({ module_code: typeof route.query.module_code === 'string' ? route.query.module_code : '', section_code: typeof route.query.section_code === 'string' ? route.query.section_code : '', subsection_code: typeof route.query.subsection_code === 'string' ? route.query.subsection_code : '' }));
</script>
<style scoped>.collect-entry { max-width:720px; } .eyebrow { font-size:12px; color:var(--admin-muted); } h1 { font-size:26px; color:var(--admin-ink); } p { color:var(--admin-muted); line-height:1.8; } .entry-actions { display:flex; flex-wrap:wrap; align-items:center; gap:20px; margin-top:24px; } .entry-actions a { color:var(--admin-green); font-size:13px; min-height:44px; display:flex; align-items:center; }@media(max-width:780px) { h1 { font-size:24px; } }</style>
