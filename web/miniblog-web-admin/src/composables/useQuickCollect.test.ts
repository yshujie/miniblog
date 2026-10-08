import { effectScope } from 'vue';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { useQuickCollect } from './useQuickCollect';
import { ApiError } from '@/utils/api-error';
import { previewSource, registerArticle } from '@/api/content';
vi.mock('@/api/content', () => ({ previewSource: vi.fn(), registerArticle: vi.fn() }));
const article = { id: '9007199254740993', title: '现有标题', status: 'Published' };
const preview = (title: string) => ({ provider: 'notion', canonical_url: 'https://notion.so/a', title, metadata_status: 'resolved' as const });
function deferred<T>() { let resolve!: (value: T) => void; const promise = new Promise<T>(done => { resolve = done; }); return { promise, resolve }; }
let scope: ReturnType<typeof effectScope>;
function setup() { scope = effectScope(); const quick = scope.run(useQuickCollect)!; quick.reset({ module_code: 'go', section_code: 'base', subsection_code: '' }, '作者'); return quick; }
beforeEach(() => { vi.useFakeTimers(); vi.mocked(previewSource).mockReset(); vi.mocked(registerArticle).mockReset(); });
afterEach(() => { scope?.stop(); vi.useRealTimers(); });
describe('quick collection', () => {
  it('ignores an old link response and preserves a manually edited title', async () => {
    const quick = setup(); const old = deferred<ReturnType<typeof preview>>(); const latest = deferred<ReturnType<typeof preview>>();
    vi.mocked(previewSource).mockReturnValueOnce(old.promise).mockReturnValueOnce(latest.promise);
    quick.setLink('https://notion.so/old'); const first = quick.suggestTitle();
    quick.setLink('https://notion.so/new'); const second = quick.suggestTitle(); quick.setTitle('手改标题');
    latest.resolve(preview('新建议')); await second; old.resolve(preview('旧建议')); await first;
    expect(quick.form.title).toBe('手改标题'); expect(quick.preview.value?.title).toBe('新建议');
  });
  it('allows a manually titled Feishu link with empty tags', async () => {
    const quick = setup(); quick.setLink('https://example.feishu.cn/docx/test'); quick.setTitle('手填标题');
    vi.mocked(registerArticle).mockResolvedValue({ outcome: 'created', article } as never);
    const result = await quick.submit(); expect(result?.outcome).toBe('created');
    expect(vi.mocked(registerArticle).mock.calls[0][0]).toMatchObject({ tags: [], title: '手填标题', author: '作者', section_code: 'base', publish: true });
  });
  it('publishes once with an intentionally empty optional author and tags', async () => {
    const quick = setup(); quick.setLink('https://example.feishu.cn/docx/no-author'); quick.setTitle('无作者文章'); quick.form.author = '';
    vi.mocked(registerArticle).mockResolvedValue({ outcome: 'created', article } as never);
    expect((await quick.submit())?.outcome).toBe('created'); expect(registerArticle).toHaveBeenCalledOnce();
    expect(vi.mocked(registerArticle).mock.calls[0][0]).toMatchObject({ author: '', tags: [], publish: true });
  });
  it('freezes a payload and key across an uncertain retry, preserving inputs', async () => {
    const quick = setup(); quick.setLink('https://notion.so/a'); quick.setTitle('原始标题');
    vi.mocked(registerArticle).mockRejectedValueOnce(new ApiError('超时')).mockResolvedValueOnce({ outcome: 'created', article } as never);
    await quick.submit(); expect(quick.form.title).toBe('原始标题'); expect(quick.frozen.value).toBeDefined();
    quick.form.title = '外部修改'; await quick.submit();
    const calls = vi.mocked(registerArticle).mock.calls; expect(calls[1]).toEqual(calls[0]); expect(calls[1][0].title).toBe('原始标题');
  });
  it('does not overwrite an already registered article or clear the form', async () => {
    const quick = setup(); quick.setLink('https://notion.so/a'); quick.setTitle('新输入');
    vi.mocked(registerArticle).mockResolvedValue({ outcome: 'already_registered', article } as never);
    await quick.submit(true, true); expect(quick.form.title).toBe('新输入'); expect(quick.duplicate.value?.title).toBe('现有标题'); expect(quick.error.value).toContain('未修改');
  });
  it('continues with the same directory/author and clears per-article fields', async () => {
    const quick = setup(); quick.setLink('https://notion.so/a'); quick.setTitle('标题'); quick.form.tags = ['标签'];
    vi.mocked(registerArticle).mockResolvedValue({ outcome: 'created', article } as never);
    await quick.submit(true, true); expect(quick.form).toMatchObject({ section_code: 'base', author: '作者', external_link: '', title: '', tags: [] });
  });
  it('blocks double clicks and a late preview while submitting', async () => {
    const quick = setup(); const title = deferred<ReturnType<typeof preview>>(); const response = deferred<never>();
    vi.mocked(previewSource).mockReturnValue(title.promise); vi.mocked(registerArticle).mockReturnValue(response.promise);
    quick.setLink('https://notion.so/a'); const previewing = quick.suggestTitle(); quick.setTitle('提交标题');
    const first = quick.submit(); await quick.submit(); title.resolve(preview('迟到标题')); await previewing;
    expect(quick.form.title).toBe('提交标题'); expect(registerArticle).toHaveBeenCalledTimes(1);
    response.resolve({ outcome: 'created', article } as never); await first;
  });
});
