import { mount, flushPromises } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { defineComponent, h } from 'vue';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import Workbench from './workbench.vue';
const { openCollect } = vi.hoisted(() => ({ openCollect: vi.fn() }));
vi.mock('@/components/content/ArticleListPanel.vue', () => ({ default: { name: 'ArticleListPanel', template: '<div />' }}));
vi.mock('vue-router', () => ({ useRoute: () => ({ query: { module_code: 'go', section_code: 'base', collect: '1' }}) }));
vi.mock('@/composables/useCatalog', () => ({ useCatalog: () => ({
  tree: { value: [] }, load: async () => {}, valid: () => true, label: () => 'Go / 基础', siblings: () => [], move: vi.fn()
}) }));
beforeEach(() => { localStorage.clear(); setActivePinia(createPinia()); });
describe('contextual collection route', () => {
  it('opens collection after the catalog loads without replacing the route and remounting the drawer', async () => {
    const wrapper = mount(Workbench, { global: { stubs: { RouterLink: true, ElAlert: true, ElButton: true, ElTree: true, ArticleListPanel: defineComponent({ setup(_, { expose }) { expose({ openCollect }); return () => h('div'); } }) }, directives: { loading: () => {} }}});
    await flushPromises(); expect(openCollect).toHaveBeenCalledOnce(); wrapper.unmount();
  });
});
