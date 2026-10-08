import { effectScope } from 'vue';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { useNotionSync } from './useNotionSync';
import * as api from '@/api/notion-sync';
vi.mock('@/api/notion-sync', () => ({ getSyncStatus: vi.fn(), getSyncSources: vi.fn(), getSyncPages: vi.fn(), getSyncRuns: vi.fn(), getSyncRun: vi.fn(), getSyncItems: vi.fn(), triggerSyncRun: vi.fn(), updateSyncControl: vi.fn(), updateSyncSource: vi.fn(), bindSyncCatalog: vi.fn() }));
const originalVisibility = Object.getOwnPropertyDescriptor(document, 'visibilityState');
let visibility = 'visible';
const scopes: ReturnType<typeof effectScope>[] = [];
const setup = () => { const scope = effectScope(); scopes.push(scope); return scope.run(useNotionSync)!; };
const empty = { items: [], total: 0, page: 1, limit: 20 };
const run = (id: string) => ({ run_id: id, status: 'completed', finished_at: '2026-10-08', counts: {}, mode: 'dry_run', phase: 'completed' });
function pending<T>() { let resolve!: (value: T) => void; const promise = new Promise<T>(done => { resolve = done; }); return { promise, resolve }; }
beforeEach(() => {
  visibility = 'visible'; Object.defineProperty(document, 'visibilityState', { get: () => visibility, configurable: true });
  vi.resetAllMocks(); vi.mocked(api.getSyncStatus).mockResolvedValue({ enabled: true, sources: [] } as never); vi.mocked(api.getSyncSources).mockResolvedValue(empty);
  vi.mocked(api.getSyncPages).mockResolvedValue(empty); vi.mocked(api.getSyncRuns).mockResolvedValue(empty); vi.mocked(api.getSyncItems).mockResolvedValue(empty);
});
afterEach(() => { scopes.splice(0).forEach(scope => scope.stop()); vi.useRealTimers(); if (originalVisibility) Object.defineProperty(document, 'visibilityState', originalVisibility); else Reflect.deleteProperty(document, 'visibilityState'); });
describe('sync workbench state', () => {
  it('refreshes idle health while visible without erasing failed form messages, and stops on dispose', async () => {
    vi.useFakeTimers();
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
  it('does not submit filter drafts on health refresh; an explicit search applies all fields and resets pagination', async () => {
    vi.useFakeTimers(); const sync = setup(); sync.pageFilters.page = 3;
    sync.pageFilterDraft.title = '未提交标题'; sync.pageFilterDraft.source_id = 'source'; sync.pageFilterDraft.management_state = 'managed';
    await vi.advanceTimersByTimeAsync(15000);
    expect(api.getSyncPages).toHaveBeenLastCalledWith({ page: 3, limit: 20, title: '', source_id: '', management_state: '' }, expect.any(AbortSignal));
    await sync.searchPages();
    expect(api.getSyncPages).toHaveBeenLastCalledWith({ page: 1, limit: 20, title: '未提交标题', source_id: 'source', management_state: 'managed' }, expect.any(AbortSignal));
    sync.pageFilterDraft.title = '另一份草稿'; await sync.refresh(true);
    expect(vi.mocked(api.getSyncPages).mock.lastCall![0].title).toBe('未提交标题');
  });
  it('aborts hidden overview and detail reads, merges resume events and ignores stale outcomes', async () => {
    vi.useFakeTimers(); const sync = setup(); const oldStatus = pending<never>(); const oldRun = pending<never>();
    vi.mocked(api.getSyncStatus).mockReturnValueOnce(oldStatus.promise).mockResolvedValue({ enabled: false, health: 'new', sources: [] } as never);
    vi.mocked(api.getSyncRun).mockReturnValueOnce(oldRun.promise).mockResolvedValue(run('selected') as never);
    const overview = sync.refresh(); const detail = sync.selectRun('selected');
    const overviewSignal = vi.mocked(api.getSyncStatus).mock.calls[0][0]!; const detailSignal = vi.mocked(api.getSyncRun).mock.calls[0][1]!;
    visibility = 'hidden'; document.dispatchEvent(new Event('visibilitychange'));
    expect(overviewSignal.aborted).toBe(true); expect(detailSignal.aborted).toBe(true);
    await vi.advanceTimersByTimeAsync(45000); expect(api.getSyncStatus).toHaveBeenCalledOnce(); expect(api.getSyncRun).toHaveBeenCalledOnce();
    visibility = 'visible'; document.dispatchEvent(new Event('visibilitychange')); window.dispatchEvent(new Event('focus')); document.dispatchEvent(new Event('visibilitychange'));
    await vi.advanceTimersByTimeAsync(50); expect(api.getSyncStatus).toHaveBeenCalledTimes(2); expect(api.getSyncRun).toHaveBeenCalledTimes(2);
    oldStatus.resolve({ enabled: true, health: 'old', sources: [] } as never); oldRun.resolve(run('stale') as never); await Promise.all([overview, detail]);
    expect(sync.status.value?.health).toBe('new'); expect(sync.selectedRun.value?.run_id).toBe('selected'); expect(sync.error.value).toBe(''); expect(sync.detailLoading.value).toBe(false);
  });
  it('stops active run polling when hidden and resumes the selected page without resetting pagination', async () => {
    vi.useFakeTimers(); const sync = setup(); vi.mocked(api.getSyncRun).mockResolvedValue({ ...run('active'), status: 'running', finished_at: undefined } as never);
    await sync.selectRun('active'); sync.itemQuery.page = 2;
    visibility = 'hidden'; document.dispatchEvent(new Event('visibilitychange')); await vi.advanceTimersByTimeAsync(20000); expect(api.getSyncRun).toHaveBeenCalledOnce();
    visibility = 'visible'; window.dispatchEvent(new Event('focus')); await vi.advanceTimersByTimeAsync(50);
    expect(api.getSyncItems).toHaveBeenLastCalledWith('active', { page: 2, limit: 20 }, expect.any(AbortSignal));
    await vi.advanceTimersByTimeAsync(2000); expect(api.getSyncRun).toHaveBeenCalledTimes(3);
  });
  it('waits for visibility before the initial read and releases listeners on disposal', async () => {
    vi.useFakeTimers(); visibility = 'hidden'; const sync = setup(); await sync.refresh(); await vi.advanceTimersByTimeAsync(30000); expect(api.getSyncStatus).not.toHaveBeenCalled();
    visibility = 'visible'; document.dispatchEvent(new Event('visibilitychange')); await vi.advanceTimersByTimeAsync(50); expect(api.getSyncStatus).toHaveBeenCalledOnce();
    scopes[0].stop(); window.dispatchEvent(new Event('focus')); await vi.advanceTimersByTimeAsync(15000); expect(api.getSyncStatus).toHaveBeenCalledOnce();
  });
  it('reads the requested source from a fresh complete status response rather than a stale list', async () => {
    const sync = setup(); vi.mocked(api.getSyncStatus).mockResolvedValue({ sources: [{ source_id: 's', config_revision: 8 }] } as never);
    expect(await sync.readSource('s')).toEqual({ source_id: 's', config_revision: 8 });
    expect(await sync.readSource('missing')).toBeUndefined(); expect(sync.error.value).toContain('找不到该来源');
  });
});
