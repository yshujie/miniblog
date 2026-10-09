import { mount, flushPromises } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { defineComponent, h, reactive } from 'vue';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import Workbench from './workbench.vue';
const { openCollect, state } = vi.hoisted(() => ({ openCollect: vi.fn(), state: { route: { query: {} as Record<string, string>, hash: '#retained' }}}));
vi.mock('@/components/content/SyncHealthBar.vue', () => ({ default: { template: '<div />' }}));
vi.mock('@/components/content/CatalogContextPanel.vue', () => ({ default: { template: '<div />' }}));
vi.mock('@/components/content/ArticleListPanel.vue', () => ({ default: { name: 'ArticleListPanel', template: '<div />' }}));
vi.mock('vue-router', () => ({ useRoute: () => state.route }));
vi.mock('@/composables/useCatalog', () => ({ useCatalog: () => ({
  tree: { value: [] }, load: async () => {}, valid: () => true, label: () => 'Go / 基础', siblings: () => [], move: vi.fn()
}) }));
beforeEach(() => { localStorage.clear(); setActivePinia(createPinia()); openCollect.mockClear(); state.route = reactive({ query: { module_code: 'go', section_code: 'base', collect: '1', unrelated: 'kept' }, hash: '#retained' }); });
function open() {
  return mount(Workbench, { global: { stubs: { CatalogContextPanel: true, SyncHealthBar: true, RouterLink: true, ElAlert: true, ElButton: true, ElTree: true, ElDrawer: true, ArticleListPanel: defineComponent({ props: ['context'], setup(props, { expose }) { expose({ openCollect }); return () => h('div', { 'data-test': 'list-context' }, JSON.stringify(props.context)); } }) }, directives: { loading: () => {} }}});
}
describe('contextual collection route', () => {
  it('opens after the catalog loads and preserves unrelated query and hash', async () => {
    const wrapper = open(); await flushPromises(); expect(openCollect).toHaveBeenCalledOnce(); expect(state.route.query.unrelated).toBe('kept'); expect(state.route.hash).toBe('#retained'); wrapper.unmount();
  });
  it('reacts to a new directory entry without clearing an open draft for unrelated query changes', async () => {
    const wrapper = open(); await flushPromises();
    state.route.query.unrelated = 'changed'; await flushPromises(); expect(openCollect).toHaveBeenCalledOnce();
    state.route.query.section_code = 'advanced'; await flushPromises();
    expect(wrapper.get('[data-test="list-context"]').text()).toContain('advanced'); expect(openCollect).toHaveBeenCalledTimes(2);
    delete state.route.query.collect; await flushPromises(); state.route.query.collect = '1'; await flushPromises(); expect(openCollect).toHaveBeenCalledTimes(3); wrapper.unmount();
  });
});
