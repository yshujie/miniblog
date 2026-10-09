import { onScopeDispose, reactive, ref } from 'vue';
import { getSyncStatus, getSyncSources, getSyncPages, getSyncRuns, getSyncRun, getSyncItems, triggerSyncRun, updateSyncControl, updateSyncSource, bindSyncCatalog } from '@/api/notion-sync';
import type { PageResult, SyncStatus, SyncSource, SyncPage, SyncRun, SyncItem, SourceUpdate, CatalogBinding, PageFilters } from '@/types/notion-sync';
import { errorMessage, isUncertain } from '@/utils/api-error';
import { useVisiblePolling } from './useVisiblePolling';
const empty = <T>(): PageResult<T> => ({ items: [], total: 0, page: 1, limit: 20 });
export function useNotionSync() {
  const status = ref<SyncStatus>(); const sources = ref(empty<SyncSource>()); const pages = ref(empty<SyncPage>()); const runs = ref(empty<SyncRun>()); const items = ref(empty<SyncItem>()); const selectedRun = ref<SyncRun>(); const selectedRunID = ref('');
  const loading = ref(false); const detailLoading = ref(false); const latestSourceLoading = ref(false); const actionBusy = ref(false); const error = ref(''); const detailError = ref('');
  const pageFilters = reactive<PageFilters>({ page: 1, limit: 20, source_id: '', management_state: '', title: '' });
  const pageFilterDraft = reactive({ source_id: '', management_state: '', title: '' });
  const runQuery = reactive({ page: 1, limit: 20 }); const sourceQuery = reactive({ page: 1, limit: 20 }); const itemQuery = reactive({ page: 1, limit: 20 });
  let overviewVersion = 0; let pageVersion = 0; let runVersion = 0; let detailVersion = 0; let sourceVersion = 0; let alive = true;
  let overviewController: AbortController | undefined; let pageController: AbortController | undefined; let runController: AbortController | undefined; let detailController: AbortController | undefined; let sourceController: AbortController | undefined;
  let poll: ReturnType<typeof setTimeout> | undefined;
  const active = (run?: SyncRun) => Boolean(run && !run.finished_at && !['succeeded', 'partial_failed', 'failed', 'cancelled', 'completed', 'completed_with_errors', 'abandoned'].includes(run.status));
  async function loadPages() {
    if (!polling.visible()) return;
    pageController?.abort(); const current = new AbortController(); pageController = current; const version = ++pageVersion;
    try { const result = await getSyncPages({ ...pageFilters }, current.signal); if (alive && version === pageVersion) pages.value = result; } catch (cause) { if (alive && version === pageVersion && !current.signal.aborted) error.value = errorMessage(cause, '读取同步页面失败'); } finally { if (alive && version === pageVersion) pageController = undefined; }
  }
  function searchPages() { Object.assign(pageFilters, pageFilterDraft, { page: 1 }); return loadPages(); }
  async function loadRuns() {
    if (!polling.visible()) return;
    runController?.abort(); const current = new AbortController(); runController = current; const version = ++runVersion;
    try { const result = await getSyncRuns({ ...runQuery }, current.signal); if (alive && version === runVersion) runs.value = result; } catch (cause) { if (alive && version === runVersion && !current.signal.aborted) error.value = errorMessage(cause, '读取运行历史失败'); } finally { if (alive && version === runVersion) runController = undefined; }
  }
  async function refresh(silent: unknown = false) {
    if (!polling.visible()) return;
    const quiet = silent === true;
    if (quiet && overviewController) return;
    overviewController?.abort(); const current = new AbortController(); overviewController = current;
    const version = ++overviewVersion; if (!quiet) { loading.value = true; error.value = ''; }
    try {
      const [health, sourceList] = await Promise.all([getSyncStatus(current.signal), getSyncSources({ ...sourceQuery }, current.signal)]);
      if (!alive || version !== overviewVersion) return;
      status.value = health; sources.value = sourceList;
      await Promise.all([loadPages(), loadRuns()]);
    } catch (cause) { if (alive && version === overviewVersion && !current.signal.aborted && (!quiet || !error.value)) error.value = errorMessage(cause, '读取同步状态失败'); } finally { if (alive && version === overviewVersion) { overviewController = undefined; loading.value = false; } }
  }
  async function selectRun(id: string, reset = true) {
    clearTimeout(poll); detailController?.abort(); selectedRunID.value = id; const version = ++detailVersion;
    if (reset) { itemQuery.page = 1; selectedRun.value = undefined; items.value = empty(); }
    if (!polling.visible()) { detailLoading.value = false; return; }
    const current = new AbortController(); detailController = current; detailLoading.value = true; detailError.value = '';
    try {
      const [run, rows] = await Promise.all([getSyncRun(id, current.signal), getSyncItems(id, { ...itemQuery }, current.signal)]);
      if (!alive || version !== detailVersion) return;
      selectedRun.value = run; items.value = rows;
      if (active(run)) poll = setTimeout(() => { if (polling.visible()) { void selectRun(id, false); void refresh(true); } }, 2000);
    } catch (cause) { if (alive && version === detailVersion && !current.signal.aborted) detailError.value = errorMessage(cause, '读取运行详情失败'); } finally { if (alive && version === detailVersion) { detailController = undefined; detailLoading.value = false; } }
  }
  async function readSource(id: string): Promise<SyncSource | undefined> {
    if (!polling.visible()) return undefined;
    sourceController?.abort(); const current = new AbortController(); sourceController = current; const version = ++sourceVersion; latestSourceLoading.value = true;
    try {
      const health = await getSyncStatus(current.signal);
      if (!alive || version !== sourceVersion) return undefined;
      const source = health.sources.find(source => source.source_id === id);
      if (!source) throw new Error('最新版中已找不到该来源，请关闭配置并重新检查来源列表');
      return source;
    } catch (cause) { if (alive && version === sourceVersion && !current.signal.aborted) error.value = errorMessage(cause, '读取最新版来源失败，草稿仍保留'); return undefined; } finally { if (alive && version === sourceVersion) { sourceController = undefined; latestSourceLoading.value = false; } }
  }
  async function action(run: () => Promise<unknown>): Promise<boolean> {
    if (actionBusy.value) return false;
    actionBusy.value = true; error.value = '';
    try { await run(); if (alive) await refresh(); return true; } catch (cause) { if (alive) error.value = errorMessage(cause, '操作失败，输入已保留'); return false; } finally { if (alive) actionBusy.value = false; }
  }
  async function start(mode: 'dry_run' | 'sync') {
    return action(async () => {
      try { const result = await triggerSyncRun(mode); await selectRun(String(result.run_id)); } catch (cause) { if (isUncertain(cause)) throw new Error(errorMessage(cause) + '。运行提交结果尚未确认，请先刷新状态和运行历史，避免重复启动。'); throw cause; }
    });
  }
  const control = (data: { paused?: boolean; source_writes_paused?: boolean }) => action(() => updateSyncControl(data));
  const saveSource = (source: SyncSource, data: SourceUpdate) => action(() => updateSyncSource(source.source_id, data));
  const saveBinding = (binding: CatalogBinding, code: string, revision: number) => action(() => bindSyncCatalog(binding, code, revision));
  function closeRun() { clearTimeout(poll); detailController?.abort(); detailController = undefined; detailVersion++; selectedRunID.value = ''; selectedRun.value = undefined; items.value = empty(); detailLoading.value = false; }
  function suspendReads() {
    clearTimeout(poll); overviewVersion++; pageVersion++; runVersion++; detailVersion++; sourceVersion++;
    overviewController?.abort(); pageController?.abort(); runController?.abort(); detailController?.abort(); sourceController?.abort();
    overviewController = pageController = runController = detailController = sourceController = undefined;
    loading.value = detailLoading.value = latestSourceLoading.value = false;
  }
  const polling = useVisiblePolling({
    refresh: () => { if (!actionBusy.value) void refresh(true); },
    resume: () => { if (!actionBusy.value) void refresh(true); if (selectedRunID.value) void selectRun(selectedRunID.value, false); },
    suspend: suspendReads
  });
  onScopeDispose(() => { alive = false; suspendReads(); });
  return { status, sources, pages, runs, items, selectedRun, selectedRunID, loading, detailLoading, latestSourceLoading, actionBusy, error, detailError, pageFilters, pageFilterDraft, runQuery, sourceQuery, itemQuery, refresh, searchPages, loadPages, loadRuns, readSource, selectRun, closeRun, start, control, saveSource, saveBinding };
}
