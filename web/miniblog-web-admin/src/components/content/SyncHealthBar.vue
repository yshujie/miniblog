<template>
  <div class="sync-health" aria-live="polite">
    <router-link to="/content/sync">Notion 同步管理</router-link>
    <span v-if="status" data-test="health-summary">{{ status.enabled ? '总开关开启' : '总开关关闭' }}{{ status.paused ? ' · 维护暂停' : '' }}{{ status.source_writes_paused ? ' · 来源回填暂停' : '' }} · {{ syncStateLabel(status.health) }} · 历史待核对 {{ status.pending_count }} · 公开受限 {{ status.blocked_count ?? '—' }} · 读取 / 执行失败 {{ status.error_count }}</span>
    <span v-else-if="!error">正在读取同步状态…</span>
    <span v-if="lastReadAt" class="last-read">读取于 {{ lastReadAt }}</span>
    <span v-if="error" class="health-error" role="alert">{{ error }} <button type="button" :disabled="loading" @click="refresh">重试</button></span>
  </div>
</template>
<script setup lang="ts">
import { ref } from 'vue';
import { useVisiblePolling } from '@/composables/useVisiblePolling';
import { getSyncStatus } from '@/api/notion-sync';
import type { SyncStatus } from '@/types/notion-sync';
import { errorMessage } from '@/utils/api-error';
import { syncStateLabel } from './sync-presentation';
const status = ref<SyncStatus>(); const error = ref(''); const loading = ref(false); const lastReadAt = ref('');
let version = 0; let controller: AbortController | undefined;
async function refresh() {
  if (!polling.visible() || controller) return;
  const request = ++version; const current = new AbortController(); controller = current; loading.value = true;
  try {
    const value = await getSyncStatus(current.signal);
    if (request === version) { status.value = value; error.value = ''; lastReadAt.value = new Date().toLocaleTimeString(); }
  } catch (cause) {
    if (request === version && !current.signal.aborted) error.value = errorMessage(cause, '同步状态暂不可用');
  } finally {
    if (request === version) { controller = undefined; loading.value = false; }
  }
}
function suspend() {
  ++version; controller?.abort(); controller = undefined; loading.value = false;
}
const polling = useVisiblePolling({ refresh: () => { void refresh(); }, suspend });
// Start after the polling lifecycle is initialized so refresh can consult visibility.
if (polling.visible()) void refresh();
</script>
<style scoped>
.sync-health { display:flex; flex-wrap:wrap; align-items:center; gap:12px; padding:12px; margin-bottom:16px; background:var(--el-fill-color-light); font-size:13px; }
.sync-health span { min-width:0; overflow-wrap:anywhere; } .last-read { color:var(--el-text-color-secondary); } .health-error { color:var(--el-color-danger); }
</style>
