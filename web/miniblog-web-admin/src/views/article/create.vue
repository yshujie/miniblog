<template><div class="app-container"><h2>收录外部文章</h2><p>文章在 Notion 或飞书中维护，后台负责目录归属和发布。</p><el-alert v-if="!contentRegistrationEnabled" title="收录功能暂未启用，现有文章仍可管理。" type="info" :closable="false" /><el-button type="primary" :disabled="!contentRegistrationEnabled" @click="open = true">粘贴链接收录</el-button><router-link to="/content/workbench" class="workspace-link">进入目录工作台</router-link><QuickCollectDrawer v-model="open" :context="context" /></div></template>
<script setup lang="ts">
import { contentRegistrationEnabled } from '@/utils/content-flags';
import { ref } from 'vue';
import { useRoute } from 'vue-router';
import QuickCollectDrawer from '@/components/content/QuickCollectDrawer.vue';
import type { DirectoryContext } from '@/types/content';
const route = useRoute(); const open = ref(contentRegistrationEnabled);
const context: DirectoryContext = { module_code: typeof route.query.module_code === 'string' ? route.query.module_code : '', section_code: typeof route.query.section_code === 'string' ? route.query.section_code : '', subsection_code: typeof route.query.subsection_code === 'string' ? route.query.subsection_code : '' };
</script>
<style scoped>.workspace-link { margin-left:16px; }</style>
