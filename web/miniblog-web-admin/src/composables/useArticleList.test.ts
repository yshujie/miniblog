import { beforeEach, expect, it, vi } from 'vitest';
import { useArticleList } from './useArticleList';
import { fetchArticles } from '@/api/content';
vi.mock('@/api/content', () => ({ fetchArticles: vi.fn() }));
beforeEach(() => { vi.mocked(fetchArticles).mockReset(); });
it('returns the correct server total and resets a changed filter to page one', async () => {
  const list = useArticleList(); list.filters.page = 3; list.filters.title = '新筛选'; vi.mocked(fetchArticles).mockResolvedValue({ articles: [{ id: '1' }], total: 120 } as never);
  await list.search(true); expect(list.filters.page).toBe(1); expect(list.total.value).toBe(120); expect(fetchArticles).toHaveBeenCalledWith(expect.objectContaining({ title: '新筛选', page: 1 }));
});
it('ignores slow results from previous directory selections', async () => {
  const list = useArticleList(); let old!: (value: unknown) => void; vi.mocked(fetchArticles).mockReturnValueOnce(new Promise(resolve => { old = resolve; }) as never).mockResolvedValueOnce({ articles: [{ id: 'new' }], total: 1 } as never);
  const first = list.search(); list.filters.section_code = 'new'; await list.search(true); old({ articles: [{ id: 'old' }], total: 1 }); await first;
  expect(list.articles.value[0].id).toBe('new'); expect(list.loading.value).toBe(false);
});
it('shows an explicit failed read state without displaying stale rows', async () => {
  const list = useArticleList(); list.articles.value = [{ id: 'old' }] as never; vi.mocked(fetchArticles).mockRejectedValue(new Error('数据库不可用'));
  await list.search(); expect(list.error.value).toBe('数据库不可用'); expect(list.articles.value).toEqual([]); expect(list.total.value).toBe(0);
});
