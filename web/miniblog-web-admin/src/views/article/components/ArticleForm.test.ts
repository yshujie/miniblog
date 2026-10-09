import { mount, flushPromises, type VueWrapper } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import ElementPlus, { ElMessageBox } from 'element-plus';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import ArticleForm from './ArticleForm.vue';
import { getArticle, saveArticle, saveArticleLocalFields, setPublicationHold } from '@/api/content';
import type { ArticleInfo } from '@/types/content';
const { guards } = vi.hoisted(() => ({ guards: { leave: undefined as undefined | (() => Promise<boolean>), update: undefined as undefined | ((to: { params: { id: string } }) => Promise<boolean>) }}));
vi.mock('vue-router', () => ({ useRoute: () => ({ params: { id: '9007199254740993' }}), onBeforeRouteLeave: (guard: typeof guards.leave) => { guards.leave = guard; }, onBeforeRouteUpdate: (guard: typeof guards.update) => { guards.update = guard; } }));
vi.mock('@/api/content', () => ({ getArticle: vi.fn(), saveArticle: vi.fn(), saveArticleLocalFields: vi.fn(), changeArticleStatus: vi.fn(), setPublicationHold: vi.fn() }));
vi.mock('@/composables/useCatalog', () => ({ useCatalog: () => ({ load: async () => {}, valid: () => true, label: () => 'Go / 基础' }) }));
vi.mock('@/components/content/DirectoryPicker.vue', () => ({ default: { template: '<div>目录选择</div>' }}));
const article: ArticleInfo = { id: '9007199254740993', title: '来源标题', author: '原作者', tags: ['Go'], external_link: 'https://www.notion.so/page', content: '历史正文', module: { code: 'go', title: 'Go' }, section: { code: 'base', module_code: 'go', title: '基础' }, pos: 1, status: 'Published' };
let wrapper: VueWrapper;
beforeEach(() => { localStorage.clear(); setActivePinia(createPinia()); vi.clearAllMocks(); });
afterEach(() => { wrapper?.unmount(); vi.restoreAllMocks(); });
async function open(value = article) { vi.mocked(getArticle).mockResolvedValue({ article: value }); wrapper = mount(ArticleForm, { props: { isEdit: true }, global: { plugins: [ElementPlus], stubs: { RouterLink: { template: '<a><slot /></a>' }, ElDrawer: { template: '<section><slot /><slot name="footer" /></section>' }}}}); await flushPromises(); }
describe('article detail ownership and navigation', () => {
  it('renders managed source fields as facts, then saves only author while preserving Published', async () => {
    const managed = { ...article, management: { mode: 'notion_sync' as const, managed_fields: ['title', 'catalog', 'tags'] }, allowed_actions: ['edit_local_fields', 'view_source', 'hold'] }; await open(managed);
    expect(wrapper.get('[data-test="source-title"]').text()).toBe('来源标题'); expect(wrapper.find('[data-test="article-title"]').exists()).toBe(false); expect(wrapper.text()).toContain('尚未提供');
    await wrapper.get('input[data-test="article-author"]').setValue('本站作者'); vi.mocked(saveArticleLocalFields).mockResolvedValue({ article: { ...managed, author: '本站作者' }}); await wrapper.get('[data-test="article-save"]').trigger('click'); await flushPromises();
    expect(saveArticleLocalFields).toHaveBeenCalledWith(article.id, { author: '本站作者' }); expect(saveArticle).not.toHaveBeenCalled(); expect(wrapper.text()).toContain('已发布');
  });
  it('rejects leaving and switching article IDs with unsaved manual inputs, while allowing unrelated query updates', async () => {
    await open(); await wrapper.get('input[data-test="article-title"]').setValue('未保存标题'); vi.spyOn(ElMessageBox, 'confirm').mockRejectedValue('cancel');
    expect(await guards.leave!()).toBe(false); expect(await guards.update!({ params: { id: '9007199254740994' }})).toBe(false); expect(await guards.update!({ params: { id: article.id }})).toBe(true); expect(wrapper.get('input[data-test="article-title"]').element).toHaveProperty('value', '未保存标题');
  });
  it('submits an independent hold and keeps an unsaved local author draft', async () => {
    const managed = { ...article, management: { mode: 'notion_sync' as const, managed_fields: ['title'] }, allowed_actions: ['edit_local_fields', 'hold'] }; await open(managed); await wrapper.get('input[data-test="article-author"]').setValue('作者草稿'); await wrapper.findAll('button').find(value => value.text() === '紧急下架')!.trigger('click');
    vi.mocked(setPublicationHold).mockResolvedValue({ article: { ...managed, publication_hold: { held: true }, allowed_actions: ['edit_local_fields', 'release_hold'] }}); await wrapper.findAll('button').find(value => value.text() === '确认下架')!.trigger('click'); await flushPromises(); expect(setPublicationHold).toHaveBeenCalledWith(article.id, { held: true, reason: '' }); expect(wrapper.get('input[data-test="article-author"]').element).toHaveProperty('value', '作者草稿'); expect(saveArticleLocalFields).not.toHaveBeenCalled();
  });
});
