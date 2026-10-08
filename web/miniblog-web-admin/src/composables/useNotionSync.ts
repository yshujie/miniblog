import { onScopeDispose, reactive, ref } from 'vue';
import { getSyncStatus, getSyncSources, getSyncPages, getSyncRuns, getSyncRun, getSyncItems, triggerSyncRun, updateSyncControl, updateSyncSource, bindSyncCatalog } from '@/api/notion-sync';
import type { PageResult, SyncStatus, SyncSource, SyncPage, SyncRun, SyncItem, SourceUpdate, CatalogBinding, PageFilters } from '@/types/notion-sync';
import { errorMessage, isUncertain } from '@/utils/api-error';
const empty = <T>(): PageResult<T> => ({ items: [], total: 0, page: 1, limit: 20 });
export function useNotionSync() {
  const status = ref<SyncStatus>(); const sources = ref(empty<SyncSource>()); const pages = ref(empty<SyncPage>()); const runs = ref(empty<SyncRun>()); const items = ref(empty<SyncItem>()); const selectedRun = ref<SyncRun>(); const selectedRunID = ref('');
  const loading = ref(false); const detailLoading = ref(false); const actionBusy = ref(false); const error = ref(''); const detailError = ref('');
  const pageFilters = reactive<PageFilters>({ page: 1, limit: 20, source_id: '', management_state: '', title: '' });
  const runQuery = reactive({ page: 1, limit: 20 }); const sourceQuery = reactive({ page: 1, limit: 20 }); const itemQuery = reactive({ page: 1, limit: 20 });
  let overviewVersion = 0; let pageVersion = 0; let runVersion = 0; let detailVersion = 0; let alive = true;
  let poll: ReturnType<typeof setTimeout> | undefined;
  const active = (run?: SyncRun) => Boolean(run && !run.finished_at && !['succeeded', 'partial_failed', 'failed', 'cancelled', 'completed'].includes(run.status));
  async function loadPages() {
    const version = ++pageVersion;
    try { const result = await getSyncPages({ ...pageFilters }); if (alive && version === pageVersion) pages.value = result; } catch (cause) { if (alive && version === pageVersion) error.value = errorMessage(cause, '读取同步页面失败'); }
  }
  async function loadRuns() {
    const version = ++runVersion;
    try { const result = await getSyncRuns({ ...runQuery }); if (alive && version === runVersion) runs.value = result; } catch (cause) { if (alive && version === runVersion) error.value = errorMessage(cause, '读取运行历史失败'); }
  }
  async function refresh(silent: unknown = false) {
    const quiet = silent === true;
    const version = ++overviewVersion; if (!quiet) { loading.value = true; error.value = ''; }
    try {
      const [health, sourceList] = await Promise.all([getSyncStatus(), getSyncSources({ ...sourceQuery })]);
      if (!alive || version !== overviewVersion) return;
      status.value = health; sources.value = sourceList;
      await Promise.all([loadPages(), loadRuns()]);
    } catch (cause) { if (alive && version === overviewVersion && (!quiet || !error.value)) error.value = errorMessage(cause, '读取同步状态失败'); } finally { if (alive && version === overviewVersion) loading.value = false; }
  }
  async function selectRun(id: string, reset = true) {
    clearTimeout(poll); selectedRunID.value = id; const version = ++detailVersion; detailLoading.value = true; detailError.value = '';
    if (reset) { itemQuery.page = 1; selectedRun.value = undefined; items.value = empty(); }
    try {
      const [run, rows] = await Promise.all([getSyncRun(id), getSyncItems(id, { ...itemQuery })]);
      if (!alive || version !== detailVersion) return;
      selectedRun.value = run; items.value = rows;
      if (active(run)) poll = setTimeout(() => { void selectRun(id, false); void refresh(true); }, 2000);
    } catch (cause) { if (alive && version === detailVersion) detailError.value = errorMessage(cause, '读取运行详情失败'); } finally { if (alive && version === detailVersion) detailLoading.value = false; }
  }
  async function action(run: () => Promise<unknown>): Promise<boolean> {
    if (actionBusy.value) return false;
    actionBusy.value = true; error.value = '';
    try { await run(); if (alive) await refresh(); return true; } catch (cause) { if (alive) error.value = errorMessage(cause, '操作失败，输入已保留'); return false; } finally { if (alive) actionBusy.value = false; }
  }
  async function start(mode: 'dry_run' | 'sync') {
    return action(async () => {
      try {
        const result = await triggerSyncRun(mode);
        await selectRun(String(result.run_id));
      } catch (cause) {
        if (isUncertain(cause)) throw new Error(errorMessage(cause) + '。运行提交结果尚未确认，请先刷新状态和运行历史，避免重复启动。');
        throw cause;
      }
    });
  }
  const control = (data: { paused?: boolean; source_writes_paused?: boolean }) => action(() => updateSyncControl(data));
  const saveSource = (source: SyncSource, data: SourceUpdate) => action(() => updateSyncSource(source.source_id, data));
  const saveBinding = (binding: CatalogBinding, code: string, revision: number) => action(() => bindSyncCatalog(binding, code, revision));
  function closeRun() { clearTimeout(poll); detailVersion++; selectedRunID.value = ''; selectedRun.value = undefined; items.value = empty(); detailLoading.value = false; }
  const healthTimer = setInterval(() => {
    if (alive && document.visibilityState !== 'hidden' && !loading.value && !actionBusy.value) void refresh(true);
  }, 15000);
  onScopeDispose(() => { clearInterval(healthTimer); alive = false; overviewVersion++; pageVersion++; runVersion++; detailVersion++; clearTimeout(poll); });
  return { status, sources, pages, runs, items, selectedRun, selectedRunID, loading, detailLoading, actionBusy, error, detailError, pageFilters, runQuery, sourceQuery, itemQuery, refresh, loadPages, loadRuns, selectRun, closeRun, start, control, saveSource, saveBinding };
}
