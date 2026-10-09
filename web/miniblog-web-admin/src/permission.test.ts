import { beforeEach, describe, expect, it, vi } from 'vitest';
interface Location { path: string; fullPath: string; meta: { title?: string }; redirectedFrom?: { fullPath: string } }
const { callbacks, addRoute, getInfo, generateRoutes, authState } = vi.hoisted(() => ({ callbacks: [] as Array<(to: Location, from: unknown, next: ReturnType<typeof vi.fn>) => Promise<void>>, addRoute: vi.fn(), getInfo: vi.fn(), generateRoutes: vi.fn(), authState: { token: 'fixture-token' }}));
vi.mock('@/router', () => ({ default: { beforeEach: (value: (typeof callbacks)[number]) => { callbacks.push(value); }, afterEach: vi.fn(), addRoute, resolve: (target: string) => { const url = new URL(target, 'http://miniblog.local'); return { path: url.pathname, query: Object.fromEntries(url.searchParams), hash: url.hash }; } }}));
vi.mock('@/store/modules/user', () => ({ default: () => ({ roles: [], getInfo, resetToken: vi.fn() }) }));
vi.mock('@/store/modules/permission', () => ({ default: () => ({ generateRoutes }) }));
vi.mock('@/utils/auth', () => ({ getToken: () => authState.token }));
vi.mock('nprogress', () => ({ default: { configure: vi.fn(), start: vi.fn(), done: vi.fn() }}));
vi.mock('@/utils/get-page-title', () => ({ default: () => 'Test' }));
vi.mock('element-plus', () => ({ ElMessage: { error: vi.fn() }}));
import './permission';
beforeEach(() => { authState.token = 'fixture-token'; vi.clearAllMocks(); getInfo.mockResolvedValue({ roles: ['admin'] }); generateRoutes.mockResolvedValue([{ path: '/content/sync' }]); });
describe('cold authorized dynamic links', () => {
  it('rematches the original local URL after installing authorized routes', async () => {
    const next = vi.fn(); await callbacks[0]({ path: '/404', fullPath: '/404', meta: {}, redirectedFrom: { fullPath: '/content/sync?source=go#history' }}, {}, next);
    expect(generateRoutes).toHaveBeenCalledWith(['admin']); expect(addRoute).toHaveBeenCalledWith({ path: '/content/sync' });
    expect(next).toHaveBeenCalledWith({ path: '/content/sync', query: { source: 'go' }, hash: '#history', replace: true });
  });
  it('never forwards an external or protocol-relative redirected target', async () => {
    for (const fullPath of ['https://example.invalid', '//example.invalid']) {
      const next = vi.fn(); await callbacks[0]({ path: '/404', fullPath: '/404', meta: {}, redirectedFrom: { fullPath }}, {}, next);
      expect(next).toHaveBeenCalledWith({ path: '/404', query: {}, hash: '', replace: true });
    }
  });
});

describe('login return context', () => {
  it('retains the requested query and hash before login', async () => {
    authState.token = ''; const next = vi.fn();
    await callbacks[0]({ path: '/404', fullPath: '/404', meta: {}, redirectedFrom: { fullPath: '/content/workbench?module_code=go&collect=1#source' }}, {}, next);
    expect(next).toHaveBeenCalledWith({ path: '/login', query: { redirect: '/content/workbench?module_code=go&collect=1#source' }});
  });
  it('retains exact article destination when profile lookup fails', async () => {
    getInfo.mockRejectedValueOnce(new Error('expired')); const next = vi.fn();
    await callbacks[0]({ path: '/article/edit/9007199254740993', fullPath: '/article/edit/9007199254740993?tab=source#details', meta: {}}, {}, next);
    expect(next).toHaveBeenCalledWith({ path: '/login', query: { redirect: '/article/edit/9007199254740993?tab=source#details' }});
  });
});
