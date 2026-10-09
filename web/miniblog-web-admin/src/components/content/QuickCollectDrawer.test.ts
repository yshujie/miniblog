import { mount, flushPromises, type VueWrapper } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import ElementPlus from 'element-plus';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import QuickCollectDrawer from './QuickCollectDrawer.vue';
import useWorkspace from '@/store/modules/contentWorkspace';
import { previewSource, registerArticle, changeArticleStatus, moveArticle } from '@/api/content';
import { ApiError } from '@/utils/api-error';
import type { ArticleInfo, DirectoryContext, SourcePreview } from '@/types/content';
const { push } = vi.hoisted(() => ({ push: vi.fn() }));
vi.mock('vue-router', () => ({ useRouter: () => ({ push }) }));
vi.mock('@/store/modules/user', () => ({ default: () => ({ name: '昵称作者' }) }));
vi.mock('@/api/content', () => ({ previewSource: vi.fn(), registerArticle: vi.fn(), changeArticleStatus: vi.fn(), moveArticle: vi.fn() }));
vi.mock('@/composables/useCatalog', () => ({ useCatalog: () => ({
  load: async () => {},
  valid: (value?: DirectoryContext) => value?.module_code === 'go' && ['base', 'advanced'].includes(value.section_code) && !value.subsection_code,
  label: (value: DirectoryContext) => `Go / ${value.section_code}`,
  modules: { modules: [{ code: 'go', title: 'Go', status: 1 }] },
  sections: { getSectionsByModule: () => [{ code: 'base', title: '基础', status: 1 }, { code: 'advanced', title: '进阶', status: 1 }], fetchSections: async () => {} },
  subsections: { getSubsectionsBySection: () => [], fetchSubsections: async () => {} }
}) }));
const context = { module_code: 'go', section_code: 'base', subsection_code: '' };
const article = { id: '9007199254740993', title: '现有标题', status: 'Deleted', module: { code: 'go', title: 'Go' }, section: { code: 'base', title: '基础' }} as ArticleInfo;
let wrapper: VueWrapper;
beforeEach(() => {
  localStorage.clear(); setActivePinia(createPinia());
  vi.mocked(previewSource).mockReset().mockResolvedValue({ provider: 'feishu', canonical_url: 'https://example.feishu.cn/docx/a', title: '', metadata_status: 'manual_required' });
  vi.mocked(registerArticle).mockReset(); vi.mocked(changeArticleStatus).mockReset(); vi.mocked(moveArticle).mockReset();
});
afterEach(() => { wrapper?.unmount(); });
async function open(value: DirectoryContext = context) {
  wrapper = mount(QuickCollectDrawer, { props: { modelValue: true, context: value }, global: { plugins: [ElementPlus], stubs: { ElDrawer: { template: '<section><slot /><slot name="footer" /></section>' }}}});
  await flushPromises(); return wrapper;
}
async function fill(title = '手填标题') {
  await wrapper.get('input[data-test="collect-link"]').setValue('https://example.feishu.cn/docx/a');
  await wrapper.get('input[data-test="collect-title"]').setValue(title);
}
describe('quick collection drawer', () => {
  it('offers view only for an already managed source, without move or restore', async () => {
    const managed = { ...article, management: { mode: 'notion_sync' as const, managed_fields: ['title'] }, allowed_actions: ['view_source', 'reorder', 'release_hold'] };
    vi.mocked(previewSource).mockResolvedValue({ provider: 'notion', canonical_url: 'https://example.feishu.cn/docx/a', title: '来源标题', metadata_status: 'resolved', existing_article: managed });
    await open(); await fill();
    await wrapper.findAll('button').find(button => button.text() === '重新获取')!.trigger('click'); await flushPromises();
    expect(wrapper.get('[data-test="collect-duplicate"]').text()).toContain('Notion 同步管理');
    expect(wrapper.findAll('button').some(button => button.text() === '移动到所选目录')).toBe(false);
    expect(wrapper.findAll('button').some(button => button.text() === '恢复为草稿')).toBe(false);
    expect(moveArticle).not.toHaveBeenCalled(); expect(changeArticleStatus).not.toHaveBeenCalled();
  });
  it('prefers the current directory, defaults nickname, and preserves directory/author after publish and continue', async () => {
    useWorkspace().remember({ ...context, section_code: 'advanced' }, '');
    await open(); await fill();
    expect(wrapper.get('input[data-test="collect-author"]').element).toHaveProperty('value', '昵称作者');
    vi.mocked(registerArticle).mockResolvedValue({ outcome: 'created', article });
    await wrapper.get('[data-test="collect-continue"]').trigger('click'); await flushPromises();
    expect(registerArticle).toHaveBeenCalledWith(expect.objectContaining({ section_code: 'base', author: '昵称作者', tags: [], publish: true }), expect.any(String));
    expect(wrapper.get('input[data-test="collect-link"]').element).toHaveProperty('value', '');
    expect(wrapper.get('input[data-test="collect-title"]').element).toHaveProperty('value', '');
    expect(wrapper.get('input[data-test="collect-author"]').element).toHaveProperty('value', '昵称作者');
    expect(wrapper.emitted('collected')).toHaveLength(1); expect(wrapper.emitted('update:modelValue')).toBeUndefined();
  });
  it('uses a recent valid directory when context is absent and keeps inputs after the registration gate rejects', async () => {
    useWorkspace().remember({ ...context, section_code: 'advanced' });
    await open({ module_code: '', section_code: '', subsection_code: '' }); await fill();
    vi.mocked(registerArticle).mockRejectedValue(new ApiError('migration pending', 'ContentRegistrationUnavailable', 503));
    await wrapper.get('[data-test="collect-publish"]').trigger('click'); await flushPromises();
    expect(registerArticle).toHaveBeenCalledWith(expect.objectContaining({ section_code: 'advanced' }), expect.any(String));
    expect(wrapper.get('[data-test="collect-error"]').text()).toContain('本次输入已保留');
    expect(wrapper.get('input[data-test="collect-title"]').element).toHaveProperty('value', '手填标题');
    expect(wrapper.get('input[data-test="collect-title"]').attributes('disabled')).toBeUndefined();
    expect(wrapper.emitted('update:modelValue')).toBeUndefined();
  });
  it('keeps a manual title when Notion metadata arrives late and offers explicit duplicate recovery', async () => {
    let finish!: (value: SourcePreview) => void;
    vi.mocked(previewSource).mockReturnValue(new Promise(resolve => { finish = resolve; }));
    await open(); await fill('手改标题');
    const previewButton = wrapper.findAll('button').find(button => button.text() === '重新获取')!;
    await previewButton.trigger('click');
    finish({ provider: 'notion', canonical_url: 'https://example.feishu.cn/docx/a', title: '自动标题', metadata_status: 'resolved', existing_article: article }); await flushPromises();
    expect(wrapper.get('input[data-test="collect-title"]').element).toHaveProperty('value', '手改标题');
    expect(wrapper.get('[data-test="collect-duplicate"]').text()).toContain('现有标题');
    vi.mocked(changeArticleStatus).mockResolvedValue({ article: { ...article, status: 'Draft' }});
    vi.mocked(previewSource).mockRejectedValue(new Error('preview unavailable after confirmed restore'));
    const restore = wrapper.findAll('button').find(button => button.text() === '恢复为草稿')!;
    await restore.trigger('click'); await flushPromises();
    expect(changeArticleStatus).toHaveBeenCalledWith(article.id, 'restore'); expect(registerArticle).not.toHaveBeenCalled(); expect(moveArticle).not.toHaveBeenCalled();
    expect(wrapper.get('[data-test="collect-duplicate"]').text()).toContain('草稿');
  });
  it('retries an uncertain publish-and-continue with the same payload/key, then keeps the sheet open', async () => {
    await open(); await fill();
    vi.mocked(registerArticle).mockRejectedValueOnce(new Error('connection lost')).mockResolvedValueOnce({ outcome: 'created', article });
    await wrapper.get('[data-test="collect-continue"]').trigger('click'); await flushPromises();
    expect(wrapper.get('input[data-test="collect-title"]').attributes('disabled')).toBeDefined();
    const first = vi.mocked(registerArticle).mock.calls[0];
    await wrapper.findAll('button').find(button => button.text() === '原样重试')!.trigger('click'); await flushPromises();
    expect(vi.mocked(registerArticle).mock.calls[1]).toEqual(first);
    expect(wrapper.get('input[data-test="collect-link"]').element).toHaveProperty('value', '');
    expect(wrapper.get('input[data-test="collect-title"]').attributes('disabled')).toBeUndefined();
    expect(wrapper.emitted('update:modelValue')).toBeUndefined();
  });
});
