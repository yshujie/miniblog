import { mount, flushPromises, type VueWrapper } from '@vue/test-utils';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import SyncHealthBar from './SyncHealthBar.vue';
import { getSyncStatus } from '@/api/notion-sync';
import type { SyncStatus } from '@/types/notion-sync';
vi.mock('@/api/notion-sync', () => ({ getSyncStatus: vi.fn() }));
const originalVisibility = Object.getOwnPropertyDescriptor(document, 'visibilityState');
let visibility = 'visible'; let wrapper: VueWrapper | undefined;
const status = (pending = 1, blocked = 2): SyncStatus => ({ enabled: true, paused: false, health: 'healthy', pending_count: pending, blocked_count: blocked, error_count: 3, sources: [] });
function pending<T>() { let resolve!: (value: T) => void; const promise = new Promise<T>(done => { resolve = done; }); return { promise, resolve }; }
function open() { wrapper = mount(SyncHealthBar, { global: { stubs: { RouterLink: { props: ['to'], template: '<a :href="to"><slot /></a>' }}}}); return wrapper; }
beforeEach(() => {
  vi.useFakeTimers(); vi.mocked(getSyncStatus).mockReset().mockResolvedValue(status()); visibility = 'visible';
  Object.defineProperty(document, 'visibilityState', { get: () => visibility, configurable: true });
});
afterEach(() => {
  wrapper?.unmount(); wrapper = undefined; vi.useRealTimers();
  if (originalVisibility) Object.defineProperty(document, 'visibilityState', originalVisibility);
  else Reflect.deleteProperty(document, 'visibilityState');
});
describe('workbench sync health', () => {
  it('polls every 15 seconds only while visible and merges visibility/focus refreshes', async () => {
    const view = open(); await flushPromises();
    expect(view.text()).toContain('历史待核对 1'); expect(view.text()).toContain('公开受限 2'); expect(view.text()).toContain('读取 / 执行失败 3');
    expect(getSyncStatus).toHaveBeenCalledOnce(); expect(getSyncStatus).toHaveBeenCalledWith(expect.any(AbortSignal));
    await vi.advanceTimersByTimeAsync(15000); expect(getSyncStatus).toHaveBeenCalledTimes(2);
    visibility = 'hidden'; document.dispatchEvent(new Event('visibilitychange')); await vi.advanceTimersByTimeAsync(45000);
    expect(getSyncStatus).toHaveBeenCalledTimes(2);
    visibility = 'visible'; document.dispatchEvent(new Event('visibilitychange')); window.dispatchEvent(new Event('focus')); document.dispatchEvent(new Event('visibilitychange'));
    await vi.advanceTimersByTimeAsync(50); expect(getSyncStatus).toHaveBeenCalledTimes(3);
    view.unmount(); wrapper = undefined; await vi.advanceTimersByTimeAsync(30000); window.dispatchEvent(new Event('focus')); await vi.advanceTimersByTimeAsync(50);
    expect(getSyncStatus).toHaveBeenCalledTimes(3);
  });
  it('cancels hidden requests and ignores a stale response after the resumed request wins', async () => {
    const old = pending<SyncStatus>(); vi.mocked(getSyncStatus).mockReturnValueOnce(old.promise).mockResolvedValueOnce(status(9, 4));
    const view = open(); const signal = vi.mocked(getSyncStatus).mock.calls[0][0]!;
    visibility = 'hidden'; document.dispatchEvent(new Event('visibilitychange')); expect(signal.aborted).toBe(true);
    visibility = 'visible'; document.dispatchEvent(new Event('visibilitychange')); await vi.advanceTimersByTimeAsync(50); await flushPromises();
    old.resolve(status(99, 99)); await flushPromises();
    expect(view.text()).toContain('历史待核对 9'); expect(view.text()).toContain('公开受限 4'); expect(view.text()).not.toContain('99');
  });
  it('keeps the last successful summary on a read failure and recovers with an explicit retry', async () => {
    const view = open(); await flushPromises(); vi.mocked(getSyncStatus).mockRejectedValueOnce(new Error('读取暂时失败'));
    await vi.advanceTimersByTimeAsync(15000); await flushPromises();
    expect(view.text()).toContain('历史待核对 1'); expect(view.get('[role="alert"]').text()).toContain('读取暂时失败');
    vi.mocked(getSyncStatus).mockResolvedValueOnce(status(4, 0)); await view.get('button').trigger('click'); await flushPromises();
    expect(view.text()).toContain('历史待核对 4'); expect(view.text()).toContain('公开受限 0'); expect(view.find('[role="alert"]').exists()).toBe(false);
  });
  it('does not make a request when initially hidden until the page is visible', async () => {
    visibility = 'hidden'; open(); await vi.advanceTimersByTimeAsync(30000); expect(getSyncStatus).not.toHaveBeenCalled();
    visibility = 'visible'; document.dispatchEvent(new Event('visibilitychange')); await vi.advanceTimersByTimeAsync(50); expect(getSyncStatus).toHaveBeenCalledOnce();
  });
});
