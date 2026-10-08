<template>
  <div class="sync-health" aria-live="polite">
    <router-link to="/content/sync">Notion 同步管理</router-link>
    <span v-if="status" data-test="health-summary">{{ status.enabled ? (status.paused ? '已暂停' : '已启用') : '未启用' }} · {{ syncStateLabel(status.health) }} · 历史待核对 {{ status.pending_count }} · 公开受限 {{ status.blocked_count ?? '—' }} · 读取 / 执行失败 {{ status.error_count }}</span>
    <span v-else-if="!error">正在读取同步状态…</span>
    <span v-if="lastReadAt" class="last-read">读取于 {{ lastReadAt }}</span>
    <span v-if="error" class="health-error" role="alert">{{ error }} <button type="button" :disabled="loading" @click="refresh">重试</button></span>
  </div>
</template>
<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue';
import { getSyncStatus } from '@/api/notion-sync';
import type { SyncStatus } from '@/types/notion-sync';
import { errorMessage } from '@/utils/api-error';
import { syncStateLabel } from './sync-presentation';
const status = ref<SyncStatus>(); const error = ref(''); const loading = ref(false); const lastReadAt = ref('');
let alive = false; let version = 0; let controller: AbortController | undefined;
let interval: ReturnType<typeof setInterval> | undefined; let resumeTimer: ReturnType<typeof setTimeout> | undefined;
const visible = () => document.visibilityState !== 'hidden';
async function refresh() {
  if (!alive || !visible() || controller) return;
  const request = ++version; const current = new AbortController(); controller = current; loading.value = true;
  try {
    const value = await getSyncStatus(current.signal);
    if (alive && request === version) { status.value = value; error.value = ''; lastReadAt.value = new Date().toLocaleTimeString(); }
  } catch (cause) {
    if (alive && request === version && !current.signal.aborted) error.value = errorMessage(cause, '同步状态暂不可用');
  } finally {
    if (alive && request === version) { controller = undefined; loading.value = false; }
  }
}
function startPolling() {
  if (interval === undefined) interval = setInterval(() => { void refresh(); }, 15000);
}
function suspend() {
  if (interval !== undefined) clearInterval(interval);
  if (resumeTimer !== undefined) clearTimeout(resumeTimer);
  interval = undefined; resumeTimer = undefined; ++version;
  controller?.abort(); controller = undefined; loading.value = false;
}
function resume() {
  if (!alive || !visible()) return;
  startPolling();
  if (resumeTimer === undefined) resumeTimer = setTimeout(() => { resumeTimer = undefined; void refresh(); }, 50);
}
function visibilityChanged() { if (visible()) resume(); else suspend(); }
onMounted(() => {
  alive = true;
  document.addEventListener('visibilitychange', visibilityChanged); window.addEventListener('focus', resume);
  if (visible()) { startPolling(); void refresh(); }
});
onUnmounted(() => {
  alive = false; suspend();
  document.removeEventListener('visibilitychange', visibilityChanged); window.removeEventListener('focus', resume);
});
</script>
<style scoped>
.sync-health { display:flex; flex-wrap:wrap; align-items:center; gap:12px; padding:12px; margin-bottom:16px; background:var(--el-fill-color-light); font-size:13px; }
.sync-health span { min-width:0; overflow-wrap:anywhere; } .last-read { color:var(--el-text-color-secondary); } .health-error { color:var(--el-color-danger); }
</style>
