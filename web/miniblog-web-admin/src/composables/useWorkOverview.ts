import { ref } from 'vue';
import { getSyncStatus, getSyncRuns } from '@/api/notion-sync';
import { useVisiblePolling } from './useVisiblePolling';
import { errorMessage } from '@/utils/api-error';
import type { SyncStatus, SyncRun } from '@/types/notion-sync';
export function useWorkOverview() {
  const status = ref<SyncStatus>(); const runs = ref<SyncRun[]>([]); const statusError = ref(''); const runsError = ref(''); const loading = ref(false); const lastReadAt = ref('');
  let version = 0; let controller: AbortController | undefined;
  async function refresh() {
    if (!polling.visible() || controller) return;
    const current = new AbortController(); const request = ++version; controller = current; loading.value = true;
    const results = await Promise.allSettled([getSyncStatus(current.signal), getSyncRuns({ page: 1, limit: 3 }, current.signal)]);
    if (request !== version || current.signal.aborted) return;
    if (results[0].status === 'fulfilled') { status.value = results[0].value; statusError.value = ''; lastReadAt.value = new Date().toLocaleTimeString(); } else statusError.value = errorMessage(results[0].reason, '同步状态暂未读取');
    if (results[1].status === 'fulfilled') { runs.value = results[1].value.items; runsError.value = ''; } else runsError.value = errorMessage(results[1].reason, '运行记录暂未读取');
    controller = undefined; loading.value = false;
  }
  function suspend() { ++version; controller?.abort(); controller = undefined; loading.value = false; }
  const polling = useVisiblePolling({ refresh: () => { void refresh(); }, suspend });
  if (polling.visible()) void refresh();
  return { status, runs, statusError, runsError, loading, lastReadAt, refresh };
}
