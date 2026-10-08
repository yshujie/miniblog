import { describe, expect, it, beforeEach, vi } from 'vitest';
import { useArticleEditor } from './useArticleEditor';
import { getArticle, saveArticle, changeArticleStatus } from '@/api/content';
vi.mock('@/api/content', () => ({ getArticle: vi.fn(), saveArticle: vi.fn(), changeArticleStatus: vi.fn() }));
const published = { id: '9007199254740993', title: '标题', author: '作者', tags: [], external_link: 'https://notion.so/a', content: '历史正文', module: { code: 'go' }, section: { code: 'base' }, status: 'Published', pos: 1 };
beforeEach(() => { vi.mocked(getArticle).mockReset().mockResolvedValue({ article: published } as never); vi.mocked(saveArticle).mockReset(); vi.mocked(changeArticleStatus).mockReset(); });
describe('published article metadata and status actions', () => {
  it('saves metadata without sending content or downgrading the published state', async () => {
    const editor = useArticleEditor(); await editor.load(published.id); editor.form.title = '改标题';
    vi.mocked(saveArticle).mockResolvedValue({ article: { ...published, title: '改标题' }} as never);
    expect(await editor.save()).toBe(true); expect(editor.article.value?.status).toBe('Published');
    expect(vi.mocked(saveArticle).mock.calls[0][0]).not.toHaveProperty('content'); expect(editor.dirty.value).toBe(false);
  });
  it('allows saving metadata with an empty optional author', async () => {
    const editor = useArticleEditor(); await editor.load(published.id); editor.form.author = '';
    vi.mocked(saveArticle).mockResolvedValue({ article: { ...published, author: '' }} as never);
    expect(await editor.save()).toBe(true); expect(vi.mocked(saveArticle).mock.calls[0][0].author).toBe('');
  });
  it('refuses to discard unsaved inputs during status changes', async () => {
    const editor = useArticleEditor(); await editor.load(published.id); editor.form.title = '未保存';
    expect(await editor.changeStatus('unpublish')).toBe(false); expect(changeArticleStatus).not.toHaveBeenCalled(); expect(editor.form.title).toBe('未保存');
  });
  it('does not publish if saving modified inputs fails', async () => {
    const editor = useArticleEditor(); await editor.load(published.id); editor.form.title = '未保存'; vi.mocked(saveArticle).mockRejectedValue(new Error('保存失败'));
    expect(await editor.changeStatus('publish', true)).toBe(false); expect(changeArticleStatus).not.toHaveBeenCalled(); expect(editor.form.title).toBe('未保存');
  });
  it('keeps the restored command DTO when a following read fails', async () => {
    vi.mocked(getArticle).mockResolvedValueOnce({ article: { ...published, status: 'Deleted' }} as never);
    const editor = useArticleEditor(); await editor.load(published.id);
    vi.mocked(changeArticleStatus).mockResolvedValue({ article: { ...published, status: 'Draft' }} as never); vi.mocked(getArticle).mockRejectedValue(new Error('读取500'));
    expect(await editor.changeStatus('restore')).toBe(true); expect(editor.article.value?.status).toBe('Draft'); expect(editor.form.title).toBe('标题');
    expect(editor.error.value).toContain('状态已更新'); expect(editor.error.value).toContain('无需重复'); expect(changeArticleStatus).toHaveBeenCalledOnce();
  });
  it('keeps saved inputs and a confirmed legacy publish state when the follow-up read fails', async () => {
    const editor = useArticleEditor(); await editor.load(published.id); editor.form.title = '修改并发布';
    vi.mocked(saveArticle).mockResolvedValue({ article: { ...published, title: '修改并发布', status: 'Draft' }} as never);
    vi.mocked(changeArticleStatus).mockResolvedValue({}); vi.mocked(getArticle).mockRejectedValue(new Error('读取500'));
    expect(await editor.changeStatus('publish', true)).toBe(true); expect(editor.article.value?.status).toBe('Published'); expect(editor.form.title).toBe('修改并发布'); expect(editor.dirty.value).toBe(false);
  });
  it('saves first when approved, then changes status and reloads', async () => {
    const editor = useArticleEditor(); await editor.load(published.id); editor.form.title = '修改';
    vi.mocked(saveArticle).mockResolvedValue({ article: { ...published, title: '修改' }} as never);
    vi.mocked(getArticle).mockResolvedValue({ article: { ...published, title: '修改', status: 'Unpublished' }} as never);
    expect(await editor.changeStatus('unpublish', true)).toBe(true); expect(saveArticle).toHaveBeenCalledOnce(); expect(changeArticleStatus).toHaveBeenCalledWith(published.id, 'unpublish'); expect(editor.form.title).toBe('修改');
  });
});
