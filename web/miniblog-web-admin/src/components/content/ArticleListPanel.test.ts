import { mount, flushPromises, type VueWrapper } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import ElementPlus from 'element-plus';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import ArticleListPanel from './ArticleListPanel.vue';
import { fetchArticles, fetchPositionArticles, reorderArticles } from '@/api/content';
import type { ArticleInfo } from '@/types/content';
vi.mock('vue-router', () => ({ useRouter: () => ({ push: vi.fn() }) }));
vi.mock('@/api/content', () => ({ fetchArticles: vi.fn(), fetchPositionArticles: vi.fn(), reorderArticles: vi.fn(), changeArticleStatus: vi.fn(), moveArticle: vi.fn(), setPublicationHold: vi.fn() }));
vi.mock('@/composables/useCatalog', () => ({ useCatalog: () => ({ load: async () => {}, valid: () => true, label: () => '', modules: { modules: [] }, sections: { getSectionsByModule: () => [] }, subsections: { getSubsectionsBySection: () => [] }}) }));
vi.mock('./QuickCollectDrawer.vue', () => ({ default: { template: '<div />' }}));
const article = { id: '9007199254740993', title: '完整标题', author: '', tags: [], status: 'Published', module: { code: 'go', title: 'Go' }, section: { code: 'base', title: '基础', module_code: 'go' }, external_link: '', content: '', pos: 1 } as ArticleInfo;
let wrapper: VueWrapper;
beforeEach(() => { localStorage.clear(); setActivePinia(createPinia()); vi.clearAllMocks(); vi.mocked(fetchArticles).mockResolvedValue({ articles: [article], total: 25 }); });
afterEach(() => { wrapper?.unmount(); });
async function open() { wrapper = mount(ArticleListPanel, { props: { context: { module_code: 'go', section_code: 'base', subsection_code: '' }}, global: { plugins: [ElementPlus], stubs: { RouterLink: { template: '<a><slot /></a>' }, ElDropdown: { template: '<div><slot /><slot name="dropdown" /></div>' }, ElDropdownMenu: { template: '<div><slot /></div>' }, ElDropdownItem: { props: ['command'], emits: ['click'], template: '<button @click="$emit(\'click\')"><slot /></button>' }}}}); await flushPromises(); }
describe('shared article list', () => {
  it('shows server total and an unknown visibility without inferring from published status', async () => { await open(); expect(wrapper.text()).toContain('25'); expect(wrapper.text()).toContain('尚未提供'); });
  it('reorders the complete position list and retains string IDs outside the displayed page', async () => {
    vi.mocked(fetchPositionArticles).mockResolvedValue([{ ...article, id: '9007199254740992' }, article, { ...article, id: '9007199254740994' }]);
    await open(); const button = wrapper.findAll('button').find(item => item.text() === '上移'); expect(button).toBeDefined(); await button!.trigger('click'); await flushPromises();
    expect(fetchPositionArticles).toHaveBeenCalledWith('base', ''); expect(reorderArticles).toHaveBeenCalledWith({ section_code: 'base', subsection_code: undefined, article_ids: ['9007199254740993', '9007199254740992', '9007199254740994'] });
  });
});
