<template>
  <div class="overview-page">
    <div class="admin-page-heading"><div><h1>工作概览</h1><p>查看同步情况，处理影响阅读的问题，继续整理内容。</p></div><el-button type="primary" :disabled="!contentRegistrationEnabled" @click="goCollect">＋ 收录文章</el-button></div>
    <el-alert v-if="!contentRegistrationEnabled" title="收录功能暂未启用，目录和已有文章仍可管理。" type="info" :closable="false" />
    <el-alert v-if="statusError" :title="statusError" type="error" :closable="false"><el-button link :loading="loading" @click="refresh">重新读取</el-button></el-alert>
    <section class="overview-health" :class="{ healthy: status && ['healthy', 'ok'].includes(status.health) }" aria-live="polite"><div><h2>{{ healthTitle }}</h2><p v-if="status">最近完整扫描 {{ formatTime(status.last_complete_scan_at) }} · 最近成功 {{ formatTime(status.last_success_at) }}</p><p v-else>{{ loading ? '正在读取同步状态…' : '同步状态暂未读取' }}</p><small v-if="statusError && status">保留上次读取资料 · {{ lastReadAt }}</small></div><router-link to="/content/sync">查看同步 ›</router-link></section>
    <div class="overview-grid">
      <section class="admin-surface"><div class="surface-title"><h2>需要核对</h2><span>各项可能重叠</span></div><router-link v-for="item in tasks" :key="item.title" to="/content/sync?tab=pages" class="overview-task"><span class="task-count" :class="item.tone">{{ item.count ?? '—' }}</span><div><strong>{{ item.title }}</strong><p>{{ item.description }}</p></div><span aria-hidden="true">›</span></router-link></section>
      <section class="admin-surface"><div class="surface-title"><h2>继续整理</h2><router-link to="/content/workbench">打开目录</router-link></div><router-link v-if="recent" class="recent-directory" :to="{ path: '/content/workbench', query: { ...recent }}"><span>目录</span><strong>{{ catalog.label(recent) }}</strong><span aria-hidden="true">›</span></router-link><p v-else>{{ catalogLoading ? '正在读取目录…' : '选择目录，开始整理主题和文章。' }}</p><p v-if="catalogError" role="alert">{{ catalogError }} <el-button link @click="loadCatalog">重试</el-button></p><p class="overview-description">{{ recent ? '回到最近使用的有效目录。' : '主题、章节与子章节在同一工作区维护。' }}收录时沿用目录与作者。</p><div class="overview-shortcuts"><router-link to="/article/list">检索全站文章</router-link><el-button link :disabled="!contentRegistrationEnabled" @click="goCollect">粘贴外部文档链接</el-button></div></section>
    </div>
    <section class="admin-surface overview-runs"><div class="surface-title"><div><h2>最近运行</h2><p>预览与实际同步分别记录。</p></div><router-link to="/content/sync?tab=runs">全部运行记录</router-link></div><el-alert v-if="runsError" :title="runsError" type="error" :closable="false"><el-button link :loading="loading" @click="refresh">重新读取</el-button></el-alert><p v-if="!runs.length">{{ loading ? '正在读取运行记录…' : runsError ? '记录暂不可用' : '暂无运行记录' }}</p><router-link v-for="run in runs" :key="run.run_id" :to="{ path: '/content/sync', query: { tab: 'runs', run_id: run.run_id }}" class="overview-run"><span class="run-state">{{ syncStateLabel(run.status) }}</span><div><strong>{{ syncStateLabel(run.mode) }} · {{ run.run_id }}</strong><p>{{ formatTime(run.started_at) }} · 扫描 {{ run.counts.seen ?? '—' }} 项，{{ run.mode === 'dry_run' ? '拟更新' : '更新' }} {{ run.counts.updated ?? '—' }} 项，受限 {{ run.counts.blocked ?? '—' }} 项</p></div><span aria-hidden="true">›</span></router-link></section>
  </div>
</template>
<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { useRouter } from 'vue-router';
import { contentRegistrationEnabled } from '@/utils/content-flags';
import { useWorkOverview } from '@/composables/useWorkOverview';
import { useCatalog } from '@/composables/useCatalog';
import useWorkspace from '@/store/modules/contentWorkspace';
import { errorMessage } from '@/utils/api-error';
import { syncStateLabel } from '@/components/content/sync-presentation';
const router = useRouter(); const workspace = useWorkspace(); const catalog = useCatalog();
const { status, runs, statusError, runsError, loading, lastReadAt, refresh } = useWorkOverview();
const catalogLoading = ref(false); const catalogError = ref('');
const recent = computed(() => workspace.recentDirectories.find(catalog.valid));
const healthTitle = computed(() => {
  if (!status.value) return '同步状态';
  if (!status.value.enabled) return '自动同步未启用';
  if (status.value.paused) return '同步维护暂停';
  if (status.value.source_writes_paused) return '来源回填暂停';
  if (status.value.current_run_id) return '同步任务运行中';
  return `自动同步 · ${syncStateLabel(status.value.health)}`;
});
const tasks = computed(() => [
  { title: '历史页面待核对', description: '保留现有阅读状态，等待受控审核。', count: status.value?.pending_count, tone: 'waiting' },
  { title: '页面公开受限', description: '查看来源、目录和本站下架状态。', count: status.value?.blocked_count, tone: 'waiting' },
  { title: '读取或执行失败', description: '检查对应来源的最近错误。', count: status.value?.error_count, tone: 'failure' }
]);
function formatTime(value?: string) { if (!value) return '暂无记录'; const time = new Date(value); return Number.isNaN(time.getTime()) ? value : time.toLocaleString(); }
function goCollect() { router.push({ path: '/content/workbench', query: { ...(recent.value || {}), collect: '1' }}); }
async function loadCatalog() { catalogLoading.value = true; catalogError.value = ''; try { await catalog.load(); } catch (cause) { catalogError.value = errorMessage(cause, '目录暂未读取'); } finally { catalogLoading.value = false; } }
onMounted(loadCatalog);
</script>
<style scoped>
.overview-health { display:flex; gap:20px; justify-content:space-between; align-items:center; padding:24px; border:1px solid var(--admin-line); background:white; border-radius:8px; margin-bottom:24px; }.overview-health.healthy { background:var(--admin-pale); border-color:#d3e8e1; }.overview-health h2 { margin:0; font-size:17px; }.overview-health p { font-size:13px; margin:8px 0 0; color:var(--admin-muted); }.overview-health small { display:block; color:var(--admin-muted); margin-top:8px; }.overview-health a { white-space:nowrap; color:var(--admin-green); }
.overview-grid { display:grid; grid-template-columns:1.2fr 1fr; gap:24px; margin-bottom:24px; }.surface-title { display:flex; justify-content:space-between; gap:16px; align-items:flex-start; margin-bottom:18px; }.surface-title h2 { margin:0; font-size:17px; }.surface-title > span { color:var(--admin-muted); font-size:13px; }.surface-title p { margin:8px 0; font-size:13px; }.surface-title a { color:var(--admin-green); font-size:13px; min-height:44px; display:flex; align-items:flex-start; padding-top:3px; }
.overview-task,.overview-run { display:flex; align-items:center; gap:16px; padding:20px 0; border-bottom:1px solid var(--admin-line); }.overview-task:last-child,.overview-run:last-child { border-bottom:0; }.overview-task > div,.overview-run > div { flex:1; min-width:0; }.overview-task p,.overview-run p { margin:6px 0 0; font-size:13px; color:var(--admin-muted); }.overview-task strong,.overview-run strong { font-size:14px; overflow-wrap:anywhere; }.task-count { width:36px; height:36px; display:grid; place-items:center; border-radius:6px; background:var(--admin-canvas); flex-shrink:0; font-size:18px; font-weight:600; }.waiting { background:#fbf4e4; color:#896015; }.failure { background:#fbeef0; color:#b42e41; }.recent-directory { display:flex; gap:16px; align-items:center; min-height:72px; }.recent-directory > span:first-child { background:var(--admin-pale); color:var(--admin-green); border-radius:6px; padding:10px; }.recent-directory strong { flex:1; overflow-wrap:anywhere; }.overview-description { margin:22px 0; font-size:13px; }.overview-shortcuts { display:flex; flex-wrap:wrap; gap:16px; border-top:1px solid var(--admin-line); padding-top:16px; color:var(--admin-green); }.overview-shortcuts a { display:flex; align-items:center; min-height:44px; font-size:13px; }.run-state { padding:5px 8px; border-radius:4px; background:var(--admin-pale); color:var(--admin-green); font-size:12px; white-space:nowrap; }
@media(max-width:1000px) { .overview-grid { grid-template-columns:1fr; } }@media(max-width:600px) { .overview-health { padding:20px; align-items:flex-start; flex-direction:column; }.overview-health a { min-height:44px; display:flex; align-items:center; }.overview-run { gap:12px; align-items:flex-start; }.run-state { white-space:normal; max-width:70px; }.overview-grid { gap:16px; margin-bottom:16px; } }
</style>
