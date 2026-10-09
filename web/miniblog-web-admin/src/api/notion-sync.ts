import request from '@/utils/request';
import type { AxiosRequestConfig } from 'axios';
import type { PageResult, SyncStatus, SyncSource, SyncPage, SyncRun, SyncItem, SourceUpdate, CatalogBinding, PageFilters } from '@/types/notion-sync';
const root = '/notion-sync';
const call = <T>(config: AxiosRequestConfig) => request(config) as unknown as Promise<T>;
// IDs remain strings. Never coerce a rounded numeric ID back into a plausible identifier.
function exactID(value: string | number | undefined | null): string | undefined {
  if (value == null) return undefined;
  if (typeof value === 'number' && !Number.isSafeInteger(value)) throw new Error('同步接口返回的 ID 无法准确读取');
  return String(value);
}
export function normalizePage<T>(data: { items?: T[]; entries?: T[]; total?: number; page?: number; limit?: number }, query = { page: 1, limit: 20 }): PageResult<T> {
  const items = data.items || data.entries || [];
  return { items, total: data.total ?? items.length, page: data.page || query.page, limit: data.limit || query.limit };
}
async function list<T>(path: string, query: { page: number; limit: number }, map: (item: T) => T = item => item, signal?: AbortSignal): Promise<PageResult<T>> {
  const data = await call<{ items?: T[]; entries?: T[]; total?: number; page?: number; limit?: number }>({ url: root + path, method: 'get', params: query, ...(signal ? { signal } : {}) });
  const result = normalizePage(data, query); result.items = result.items.map(map); return result;
}
export const getSyncStatus = (signal?: AbortSignal) => call<SyncStatus>({ url: root + '/status', method: 'get', ...(signal ? { signal } : {}) });
export const getSyncSources = (query = { page: 1, limit: 20 }, signal?: AbortSignal) => list<SyncSource>('/sources', query, item => ({ ...item, catalog_bindings: item.catalog_bindings || [] }), signal);
export const getSyncPages = (query: PageFilters, signal?: AbortSignal) => list<SyncPage>('/pages', query, item => ({ ...item, article_id: exactID(item.article_id) }), signal);
export const getSyncRuns = (query = { page: 1, limit: 20 }, signal?: AbortSignal) => list<SyncRun>('/runs', query, item => ({ ...item, run_id: exactID(item.run_id) || '' }), signal);
export const getSyncRun = (id: string, signal?: AbortSignal) => call<SyncRun>({ url: root + '/runs/' + encodeURIComponent(id), method: 'get', ...(signal ? { signal } : {}) });
function normalizeItem(item: SyncItem): SyncItem {
  const result = { ...item, article_id: exactID(item.article_id) };
  if (item.after && typeof item.after === 'object' && 'bootstrap_preview' in item.after) {
    const candidate = item.after.bootstrap_preview;
    if (candidate && typeof candidate === 'object') {
      const fields = candidate as { article_id?: string | number; candidate_article_ids?: (string | number)[]; title_hint_article_ids?: (string | number)[] };
      result.after = { ...item.after, bootstrap_preview: {
        ...candidate,
        article_id: exactID(fields.article_id),
        candidate_article_ids: (fields.candidate_article_ids || []).map(id => exactID(id) || ''),
        title_hint_article_ids: (fields.title_hint_article_ids || []).map(id => exactID(id) || '')
      }};
    }
  }
  return result;
}
export const getSyncItems = (id: string, query = { page: 1, limit: 20 }, signal?: AbortSignal) => list<SyncItem>('/runs/' + encodeURIComponent(id) + '/items', query, normalizeItem, signal);
export const updateSyncControl = (data: { paused?: boolean; source_writes_paused?: boolean }) => call<SyncStatus>({ url: root + '/control', method: 'patch', data });
export const updateSyncSource = (id: string, data: SourceUpdate) => call<SyncSource>({ url: root + '/sources/' + encodeURIComponent(id), method: 'patch', data });
export const bindSyncCatalog = (binding: CatalogBinding, section_code: string, expected_config_revision: number) => call<CatalogBinding>({ url: root + '/catalog-bindings/' + encodeURIComponent(binding.id), method: 'put', data: { source_id: binding.source_id, option_id: binding.option_id, section_code, expected_config_revision }});
export const triggerSyncRun = (mode: 'dry_run' | 'sync') => call<{ run_id: string }>({ url: root + '/runs', method: 'post', data: { mode }, timeout: 10000 });
