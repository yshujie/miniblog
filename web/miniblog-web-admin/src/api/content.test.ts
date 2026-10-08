import { beforeEach, expect, it, vi } from 'vitest';
import request from '@/utils/request';
import { fetchArticles, fetchPositionArticles, normalizeArticle, previewSource, registerArticle } from './content';
vi.mock('@/utils/request', () => ({ default: vi.fn() }));
beforeEach(() => { vi.mocked(request).mockReset(); });
it('prefers the exact legacy id_text instead of a rounded numeric ID', () => {
  expect(normalizeArticle({ id: 9007199254740992, id_text: '9007199254740993' } as never).id).toBe('9007199254740993');
});
it('preserves all filters and the server total', async () => {
  vi.mocked(request).mockResolvedValue({ articles: [], total: 305 });
  const filters = { module_code: 'go', section_code: 'base', subsection_code: 'types', title: 'title', status: 'Published' as const, page: 3, limit: 20 };
  expect((await fetchArticles(filters)).total).toBe(305); expect(request).toHaveBeenCalledWith(expect.objectContaining({ params: filters }));
});
it('loads every page of a direct position and excludes archived articles for reorder', async () => {
  vi.mocked(request).mockResolvedValueOnce({ articles: [{ id: '1', status: 'Published' }], total: 2 }).mockResolvedValueOnce({ articles: [{ id: '2', status: 'Deleted' }], total: 2 });
  expect((await fetchPositionArticles('base')).map(item => item.id)).toEqual(['1']);
  expect(request).toHaveBeenLastCalledWith(expect.objectContaining({ params: expect.objectContaining({ direct_only: true, page: 2 }) }));
});
it('uses a ten-second preview and sends the frozen request key', async () => {
  vi.mocked(request).mockResolvedValueOnce({ title: 'Notion', metadata_status: 'resolved' }).mockResolvedValueOnce({ outcome: 'created', article: { id: '1' }});
  await previewSource('https://notion.so/a'); expect(request).toHaveBeenLastCalledWith(expect.objectContaining({ timeout: 10000 }));
  await registerArticle({ external_link: 'https://notion.so/a', title: 'title', section_code: 'base', author: '作者', tags: [], publish: true }, 'same-key');
  expect(request).toHaveBeenLastCalledWith(expect.objectContaining({ headers: { 'Idempotency-Key': 'same-key' }}));
});
