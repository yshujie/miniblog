import { beforeEach, describe, expect, it, vi } from 'vitest';
import request from '@/utils/request';
import { getSyncPages, getSyncSources, getSyncItems, bindSyncCatalog, updateSyncSource, triggerSyncRun } from './notion-sync';
vi.mock('@/utils/request', () => ({ default: vi.fn() }));
beforeEach(() => vi.mocked(request).mockReset());
describe('Notion sync HTTP contract', () => {
  it('normalizes entries while preserving large article IDs', async () => {
    vi.mocked(request).mockResolvedValue({ entries: [{ page_id: 'page', article_id: '9007199254740993' }], total: 1 } as never);
    const result = await getSyncPages({ page: 2, limit: 20 });
    expect(result).toEqual({ items: [{ page_id: 'page', article_id: '9007199254740993' }], total: 1, page: 2, limit: 20 });
    expect(request).toHaveBeenCalledWith(expect.objectContaining({ url: '/notion-sync/pages', method: 'get', params: { page: 2, limit: 20 }}));
  });
  it('preserves distinct catalog item identities without page IDs', async () => {
    const items = [{ item_id: 'catalog:source:one', page_id: '', outcome: 'created' }, { item_id: 'catalog:source:two', page_id: '', outcome: 'blocked' }];
    vi.mocked(request).mockResolvedValue({ items, total: 2, page: 1, limit: 20 } as never);
    expect((await getSyncItems('run')).items).toEqual(items.map(item => ({ ...item, article_id: undefined })));
  });
  it('refuses rounded numeric IDs instead of navigating to another article', async () => {
    vi.mocked(request).mockResolvedValue({ items: [{ article_id: 9007199254740992 }] } as never);
    await expect(getSyncPages({ page: 1, limit: 20 })).rejects.toThrow('ID');
  });
  it('uses the confirmed PATCH and binding version contracts', async () => {
    vi.mocked(request).mockResolvedValue({} as never);
    await updateSyncSource('source', { label: 'Go', module_code: 'go', expected_config_revision: 2 });
    expect(request).toHaveBeenLastCalledWith({ url: '/notion-sync/sources/source', method: 'patch', data: { label: 'Go', module_code: 'go', expected_config_revision: 2 }});
    await bindSyncCatalog({ id: '9007199254740993', source_id: 'source', option_id: 'topic', option_name: '主题', status: 'conflict' }, 'section', 2);
    expect(request).toHaveBeenLastCalledWith(expect.objectContaining({ url: '/notion-sync/catalog-bindings/9007199254740993', method: 'put', data: { source_id: 'source', option_id: 'topic', section_code: 'section', expected_config_revision: 2 }}));
    await triggerSyncRun('dry_run'); expect(request).toHaveBeenLastCalledWith(expect.objectContaining({ url: '/notion-sync/runs', method: 'post', data: { mode: 'dry_run' }}));
  });
  it('accepts empty sources and fills absent bindings', async () => {
    vi.mocked(request).mockResolvedValue({ items: [{ source_id: 's' }], total: 1, page: 1, limit: 20 } as never);
    expect((await getSyncSources()).items[0].catalog_bindings).toEqual([]);
  });
});
