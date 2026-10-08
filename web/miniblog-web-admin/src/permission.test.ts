import { beforeEach, describe, expect, it, vi } from 'vitest';
interface Location { path: string; fullPath: string; meta: { title?: string }; redirectedFrom?: { fullPath: string } }
const { callbacks, addRoute, getInfo, generateRoutes } = vi.hoisted(() => ({ callbacks: [] as Array<(to: Location, from: unknown, next: ReturnType<typeof vi.fn>) => Promise<void>>, addRoute: vi.fn(), getInfo: vi.fn(), generateRoutes: vi.fn() }));
vi.mock('@/router', () => ({ default: { beforeEach: (value: (typeof callbacks)[number]) => { callbacks.push(value); }, afterEach: vi.fn(), addRoute }}));
vi.mock('@/store/modules/user', () => ({ default: () => ({ roles: [], getInfo, resetToken: vi.fn() }) }));
vi.mock('@/store/modules/permission', () => ({ default: () => ({ generateRoutes }) }));
vi.mock('@/utils/auth', () => ({ getToken: () => 'fixture-token' }));
vi.mock('nprogress', () => ({ default: { configure: vi.fn(), start: vi.fn(), done: vi.fn() }}));
vi.mock('@/utils/get-page-title', () => ({ default: () => 'Test' }));
vi.mock('element-plus', () => ({ ElMessage: { error: vi.fn() }}));
import './permission';
beforeEach(() => { vi.clearAllMocks(); getInfo.mockResolvedValue({ roles: ['admin'] }); generateRoutes.mockResolvedValue([{ path: '/content/sync' }]); });
describe('cold authorized dynamic links', () => {
  it('rematches the original local URL after installing authorized routes', async () => {
    const next = vi.fn(); await callbacks[0]({ path: '/404', fullPath: '/404', meta: {}, redirectedFrom: { fullPath: '/content/sync?source=go#history' }}, {}, next);
    expect(generateRoutes).toHaveBeenCalledWith(['admin']); expect(addRoute).toHaveBeenCalledWith({ path: '/content/sync' });
    expect(next).toHaveBeenCalledWith({ path: '/content/sync?source=go#history', replace: true });
  });
  it('never forwards an external or protocol-relative redirected target', async () => {
    for (const fullPath of ['https://example.invalid', '//example.invalid']) {
      const next = vi.fn(); await callbacks[0]({ path: '/404', fullPath: '/404', meta: {}, redirectedFrom: { fullPath }}, {}, next);
      expect(next).toHaveBeenCalledWith({ path: '/404', replace: true });
    }
  });
});
