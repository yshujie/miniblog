import { beforeEach, describe, expect, it, vi } from 'vitest';
import { getSyncStatus, getSyncRuns } from '@/api/notion-sync';
import { useWorkOverview } from './useWorkOverview';
import type { SyncStatus } from '@/types/notion-sync';
const { lifecycle } = vi.hoisted(() => ({ lifecycle: { visible: false, suspend: () => {}, resume: () => {} }}));
vi.mock('@/api/notion-sync', () => ({ getSyncStatus: vi.fn(), getSyncRuns: vi.fn() }));
vi.mock('./useVisiblePolling', () => ({ useVisiblePolling: (options: { suspend: () => void; refresh: () => void }) => { lifecycle.suspend = options.suspend; lifecycle.resume = options.refresh; return { visible: () => lifecycle.visible }; } }));
const fixture: SyncStatus = { enabled: true, paused: false, health: 'healthy', pending_count: 2, error_count: 1, sources: [] };
beforeEach(() => { vi.clearAllMocks(); lifecycle.visible = false; vi.mocked(getSyncStatus).mockResolvedValue(fixture); vi.mocked(getSyncRuns).mockResolvedValue({ items: [], total: 0, page: 1, limit: 3 }); });
describe('work overview real response lifecycle', () => {
  it('does not request while hidden and loads the two independent summaries when visible', async () => {
    const model = useWorkOverview(); await model.refresh(); expect(getSyncStatus).not.toHaveBeenCalled();
    lifecycle.visible = true; await model.refresh(); expect(model.status.value?.pending_count).toBe(2); expect(model.runsError.value).toBe(''); expect(getSyncRuns).toHaveBeenCalledWith({ page: 1, limit: 3 }, expect.any(AbortSignal));
  });
  it('retains previous status when refresh fails without blocking successful records', async () => {
    const model = useWorkOverview(); lifecycle.visible = true; await model.refresh();
    vi.mocked(getSyncStatus).mockRejectedValueOnce(new Error('status unavailable')); await model.refresh();
    expect(model.status.value).toEqual(fixture); expect(model.statusError.value).toBe('status unavailable'); expect(model.runsError.value).toBe('');
  });
  it('ignores a late reply after the visible lifecycle is suspended', async () => {
    let resolve!: (value: SyncStatus) => void;
    vi.mocked(getSyncStatus).mockReturnValueOnce(new Promise(value => { resolve = value; }));
    const model = useWorkOverview(); lifecycle.visible = true; const pending = model.refresh(); lifecycle.suspend(); resolve(fixture); await pending;
    expect(model.status.value).toBeUndefined(); expect(model.loading.value).toBe(false);
  });
});
