import { effectScope } from 'vue';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { useNotionSync } from './useNotionSync';
import * as api from '@/api/notion-sync';
vi.mock('@/api/notion-sync', () => ({ getSyncStatus: vi.fn(), getSyncSources: vi.fn(), getSyncPages: vi.fn(), getSyncRuns: vi.fn(), getSyncRun: vi.fn(), getSyncItems: vi.fn(), triggerSyncRun: vi.fn(), updateSyncControl: vi.fn(), updateSyncSource: vi.fn(), bindSyncCatalog: vi.fn() }));
const scopes: ReturnType<typeof effectScope>[] = [];
const setup = () => { const scope = effectScope(); scopes.push(scope); return scope.run(useNotionSync)!; };
const empty = { items: [], total: 0, page: 1, limit: 20 };
const run = (id: string) => ({ run_id: id, status: 'completed', finished_at: '2026-10-08', counts: {}, mode: 'dry_run', phase: 'completed' });
function pending<T>() { let resolve!: (value: T) => void; const promise = new Promise<T>(done => { resolve = done; }); return { promise, resolve }; }
beforeEach(() => {
  vi.resetAllMocks(); vi.mocked(api.getSyncStatus).mockResolvedValue({ enabled: true, sources: [] } as never); vi.mocked(api.getSyncSources).mockResolvedValue(empty);
  vi.mocked(api.getSyncPages).mockResolvedValue(empty); vi.mocked(api.getSyncRuns).mockResolvedValue(empty); vi.mocked(api.getSyncItems).mockResolvedValue(empty);
});
afterEach(() => { scopes.splice(0).forEach(scope => scope.stop()); vi.useRealTimers(); });
describe('sync workbench state', () => {
  it('refreshes idle health while visible without erasing failed form messages, and stops on dispose', async () => {
    vi.useFakeTimers(); Object.defineProperty(document, 'visibilityState', { value: 'visible', configurable: true });
    const sync = setup(); sync.error.value = '配置保存失败，输入仍保留';
    await vi.advanceTimersByTimeAsync(15000); expect(api.getSyncStatus).toHaveBeenCalledOnce(); expect(sync.error.value).toContain('配置保存失败');
    scopes[0].stop(); await vi.advanceTimersByTimeAsync(15000); expect(api.getSyncStatus).toHaveBeenCalledOnce();
  });
  it('does not let a late previous run replace the selected result', async () => {
    const sync = setup(); const old = pending<never>(); vi.mocked(api.getSyncRun).mockReturnValueOnce(old.promise).mockResolvedValueOnce(run('new') as never);
    const first = sync.selectRun('old'); await sync.selectRun('new'); old.resolve(run('old') as never); await first;
    expect(sync.selectedRun.value?.run_id).toBe('new'); expect(sync.detailLoading.value).toBe(false);
  });
  it('keeps the confirmed run ID if its detail read fails without posting another run', async () => {
    const sync = setup(); vi.mocked(api.triggerSyncRun).mockResolvedValue({ run_id: 'confirmed' }); vi.mocked(api.getSyncRun).mockRejectedValue(new Error('read failed'));
    expect(await sync.start('dry_run')).toBe(true); expect(sync.selectedRunID.value).toBe('confirmed'); expect(sync.detailError.value).toContain('read failed'); expect(api.triggerSyncRun).toHaveBeenCalledOnce();
  });
  it('prevents double submits and retains source form state on a revision conflict', async () => {
    const sync = setup(); const first = pending<{ run_id: string }>(); vi.mocked(api.triggerSyncRun).mockReturnValue(first.promise); vi.mocked(api.getSyncRun).mockResolvedValue(run('run') as never);
    const loading = sync.start('dry_run'); expect(await sync.start('dry_run')).toBe(false); first.resolve({ run_id: 'run' }); await loading; expect(api.triggerSyncRun).toHaveBeenCalledOnce();
    const source = { source_id: 's', config_revision: 1 } as never; const form = { label: 'New', module_code: 'go', expected_config_revision: 1 }; vi.mocked(api.updateSyncSource).mockRejectedValue(new Error('配置已变化'));
    expect(await sync.saveSource(source, form)).toBe(false); expect(form.label).toBe('New'); expect(sync.error.value).toContain('配置已变化');
  });
  it('does not publish obsolete filtered page results', async () => {
    const sync = setup(); const old = pending<never>(); vi.mocked(api.getSyncPages).mockReturnValueOnce(old.promise).mockResolvedValueOnce({ ...empty, items: [{ page_id: 'new' }] } as never);
    const first = sync.loadPages(); sync.pageFilters.title = 'New'; await sync.loadPages(); old.resolve({ ...empty, items: [{ page_id: 'old' }] } as never); await first;
    expect(sync.pages.value.items[0].page_id).toBe('new');
  });
});
