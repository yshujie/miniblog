<template><div class="sync-health"><router-link to="/content/sync">Notion 同步管理</router-link><span v-if="status">{{ status.enabled ? (status.paused ? '已暂停' : '已启用') : '未启用' }} · 待处理 {{ status.pending_count }} · 错误 {{ status.error_count }}</span><span v-else-if="error">{{ error }}</span><span v-else>正在读取同步状态…</span></div></template>
<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue';
import { getSyncStatus } from '@/api/notion-sync';
import type { SyncStatus } from '@/types/notion-sync';
import { errorMessage } from '@/utils/api-error';
const status = ref<SyncStatus>(); const error = ref(''); let alive = true;
onMounted(async () => { try { const value = await getSyncStatus(); if (alive) status.value = value; } catch (cause) { if (alive) error.value = errorMessage(cause, '同步状态暂不可用'); } });
onUnmounted(() => { alive = false; });
</script>
<style scoped>.sync-health { display:flex; flex-wrap:wrap; gap:12px; padding:12px; margin-bottom:16px; background:var(--el-fill-color-light); font-size:13px; }</style>
