import { describe, expect, it, beforeEach, vi } from 'vitest';
import { ApiError } from '@/utils/api-error';
import { useArticleEditor } from './useArticleEditor';
import { getArticle, saveArticle, changeArticleStatus, saveArticleLocalFields, setPublicationHold } from '@/api/content';
vi.mock('@/api/content', () => ({ getArticle: vi.fn(), saveArticle: vi.fn(), changeArticleStatus: vi.fn(), saveArticleLocalFields: vi.fn(), setPublicationHold: vi.fn() }));
const published = { id: '9007199254740993', title: '标题', author: '作者', tags: [], external_link: 'https://notion.so/a', content: '历史正文', module: { code: 'go' }, section: { code: 'base' }, status: 'Published', pos: 1 };
beforeEach(() => { vi.mocked(getArticle).mockReset().mockResolvedValue({ article: published } as never); vi.mocked(saveArticle).mockReset(); vi.mocked(changeArticleStatus).mockReset(); vi.mocked(saveArticleLocalFields).mockReset(); vi.mocked(setPublicationHold).mockReset(); });
describe('published article metadata and status actions', () => {
  it('recognizes takeover after a stale manual PUT and retains both local author and old source drafts', async () => {
    const editor = useArticleEditor(); await editor.load(published.id); editor.form.author = '作者草稿'; editor.form.title = '旧标题草稿';
    vi.mocked(saveArticle).mockRejectedValue(new ApiError('来源已接管', 'SourceManaged', 409));
    vi.mocked(getArticle).mockResolvedValue({ article: { ...published, title: 'Notion 新标题', management: { mode: 'notion_sync', managed_fields: ['title'] }, allowed_actions: ['edit_local_fields'] }} as never);
    expect(await editor.save()).toBe(false); expect(editor.article.value?.management?.mode).toBe('notion_sync');
    expect(editor.form.title).toBe('Notion 新标题'); expect(editor.form.author).toBe('作者草稿'); expect(editor.conflictDraft.value?.title).toBe('旧标题草稿'); expect(editor.dirty.value).toBe(true);
    vi.mocked(saveArticleLocalFields).mockResolvedValue({ article: { ...editor.article.value, author: '作者草稿' }} as never);
    expect(await editor.save()).toBe(true); expect(saveArticleLocalFields).toHaveBeenCalledWith(published.id, { author: '作者草稿' });
  });
  it('saves managed author only and preserves inputs on failure, even when the source directory is unavailable', async () => {
    const managed = { ...published, section: undefined, management: { mode: 'notion_sync', managed_fields: ['title', 'catalog'] }, allowed_actions: ['edit_local_fields', 'reorder', 'hold'] };
    vi.mocked(getArticle).mockResolvedValueOnce({ article: managed } as never);
    const editor = useArticleEditor(); await editor.load(published.id); editor.form.author = '本地作者'; editor.form.title = '旧表单不能送出';
    vi.mocked(saveArticleLocalFields).mockRejectedValueOnce(new Error('failed'));
    expect(await editor.save()).toBe(false); expect(editor.form.author).toBe('本地作者'); expect(editor.dirty.value).toBe(true);
    vi.mocked(saveArticleLocalFields).mockResolvedValueOnce({ article: { ...managed, author: '本地作者' }} as never);
    expect(await editor.save()).toBe(true); expect(saveArticleLocalFields).toHaveBeenLastCalledWith(published.id, { author: '本地作者' }); expect(saveArticle).not.toHaveBeenCalled();
    expect(await editor.changeStatus('unpublish')).toBe(false); expect(changeArticleStatus).not.toHaveBeenCalled();
  });
  it('keeps a local author draft after a confirmed hold response', async () => {
    const managed = { ...published, management: { mode: 'notion_sync', managed_fields: ['title'] }, allowed_actions: ['edit_local_fields', 'hold'] };
    vi.mocked(getArticle).mockResolvedValueOnce({ article: managed } as never);
    const editor = useArticleEditor(); await editor.load(published.id); editor.form.author = '未保存作者';
    vi.mocked(setPublicationHold).mockResolvedValueOnce({ article: { ...managed, publication_hold: { held: true }, allowed_actions: ['edit_local_fields', 'release_hold'] }} as never);
    expect(await editor.changeHold(true, '紧急')).toBe(true); expect(editor.article.value?.publication_hold?.held).toBe(true); expect(editor.form.author).toBe('未保存作者'); expect(editor.dirty.value).toBe(true);
    expect(setPublicationHold).toHaveBeenCalledWith(published.id, { held: true, reason: '紧急' }); expect(changeArticleStatus).not.toHaveBeenCalled();
  });
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
