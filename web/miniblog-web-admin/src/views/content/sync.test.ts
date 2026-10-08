import { mount, flushPromises, type VueWrapper } from '@vue/test-utils';
import { ref, reactive } from 'vue';
import ElementPlus, { ElMessage, ElMessageBox } from 'element-plus';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import SyncView from './sync.vue';
import { useNotionSync } from '@/composables/useNotionSync';
import type { SyncSource, SyncStatus } from '@/types/notion-sync';
vi.mock('@/composables/useNotionSync', () => ({ useNotionSync: vi.fn() }));
vi.mock('@/composables/useCatalog', () => ({ useCatalog: () => ({ load: vi.fn().mockResolvedValue(undefined), modules: { modules: [{ code: 'go', title: 'Go', status: 1 }, { code: 'python', title: 'Python', status: 1 }] }, sections: { getSectionsByModule: () => [{ code: 'base', title: '基础', status: 1 }] }, subsections: { getSubsectionsBySection: () => [] }}) }));
const source = (): SyncSource => ({ source_id: 'source', label: 'Go 文档', module_code: 'go', enabled: true, health: 'healthy', config_revision: 7, catalog_bindings: [], config: { title_property_id: 'title', state_property_id: 'state', topic_property_id: 'topic', tags_property_id: 'tags', state_option_ids: { draft: 'd', published: 'p', unpublished: 'u', archived: 'a' }}});
function setup() {
  const value = source();
  return {
    loading: ref(false), actionBusy: ref(false), error: ref(''), detailError: ref(''), detailLoading: ref(false),
    status: ref<SyncStatus>({ enabled: true, paused: false, health: 'healthy', pending_count: 2, blocked_count: 3, error_count: 4, sources: [value] }),
    sources: ref({ items: [value], total: 1, page: 1, limit: 20 }), pages: ref({ items: [{ page_id: 'page', source_id: 'source', title: '受限文章', desired_state: 'published', local_state: 'published', effective_visibility: false, needs_revalidation: true, visibility_reason: 'needs_revalidation', management_state: 'managed', publication_hold: false }], total: 1, page: 1, limit: 20 }),
    runs: ref({ items: [{ run_id: 'finished-with-pending', mode: 'dry_run', status: 'completed', phase: 'finished', finished_at: '2026-10-08T00:00:00Z', counts: { seen: 5, created: 0, updated: 0, unchanged: 0, blocked: 0, failed: 0, pending: 2, frozen: 3 }}], total: 1, page: 1, limit: 20 }), items: ref({ items: [], total: 0, page: 1, limit: 20 }), selectedRun: ref(undefined), selectedRunID: ref(''),
    sourceQuery: reactive({ page: 1, limit: 20 }), pageFilters: reactive({ page: 1, limit: 20, title: '', source_id: '', management_state: '' }), runQuery: reactive({ page: 1, limit: 20 }), itemQuery: reactive({ page: 1, limit: 20 }),
    refresh: vi.fn().mockResolvedValue(undefined), loadPages: vi.fn(), loadRuns: vi.fn(), selectRun: vi.fn(), closeRun: vi.fn(), control: vi.fn(), start: vi.fn(), saveSource: vi.fn().mockResolvedValue(true), saveBinding: vi.fn()
  };
}
let sync: ReturnType<typeof setup>; let wrapper: VueWrapper | undefined;
beforeEach(() => {
  sync = setup(); vi.mocked(useNotionSync).mockReturnValue(sync as unknown as ReturnType<typeof useNotionSync>);
  vi.spyOn(ElMessageBox, 'confirm').mockResolvedValue('confirm' as Awaited<ReturnType<typeof ElMessageBox.confirm>>);
});
afterEach(() => { wrapper?.unmount(); wrapper = undefined; ElMessage.closeAll(); vi.restoreAllMocks(); });
async function open() {
  wrapper = mount(SyncView, { global: { plugins: [ElementPlus], stubs: {
    ElDialog: { props: ['modelValue'], template: '<section v-if="modelValue"><slot /><slot name="footer" /></section>' },
    ElDrawer: { props: ['modelValue'], template: '<section v-if="modelValue"><slot /></section>' },
    RouterLink: { props: ['to'], template: '<a :href="to"><slot /></a>' }
  }}});
  await flushPromises(); return wrapper;
}
async function configure() { await wrapper!.findAll('button').find(button => button.text() === '配置')!.trigger('click'); await flushPromises(); }
describe('sync management page', () => {
  it('shows source intent, local state and actual public restriction separately', async () => {
    const view = await open();
    expect(view.text()).toContain('来源期望'); expect(view.text()).toContain('本站状态'); expect(view.text()).toContain('前台不可见'); expect(view.text()).toContain('等待重新核验');
    expect(view.text()).toContain('历史待核对 2'); expect(view.text()).toContain('公开受限 3'); expect(view.text()).toContain('读取 / 执行失败 4');
    expect(view.text()).toContain('等待重新同步核验来源'); expect(view.text()).toContain('待处理 2'); expect(view.text()).toContain('冻结 3'); expect(view.text()).toContain('失败 0'); expect(view.text()).not.toContain('部分项目失败或被阻止');
  });
  it('retains an unsaved draft during refresh and submits only the changed label at the editing revision', async () => {
    const view = await open(); await configure(); await view.get('input[data-test="source-label"]').setValue('我的新名称');
    sync.refresh.mockImplementation(async () => { sync.sources.value.items = [{ ...source(), label: '别人修改的名称', config_revision: 8 }]; });
    await view.findAll('button').find(button => button.text() === '刷新状态')!.trigger('click'); await flushPromises();
    expect(view.get('input[data-test="source-label"]').element).toHaveProperty('value', '我的新名称');
    await view.get('[data-test="source-save"]').trigger('click'); await flushPromises();
    expect(sync.saveSource).toHaveBeenCalledWith(expect.objectContaining({ config_revision: 7 }), { label: '我的新名称', expected_config_revision: 7 }); expect(ElMessageBox.confirm).not.toHaveBeenCalled();
  });
  it('does not submit unchanged mappings and keeps changed inputs after cancelling the public-impact confirmation', async () => {
    const view = await open(); await configure(); await view.get('[data-test="source-save"]').trigger('click'); await flushPromises();
    expect(sync.saveSource).not.toHaveBeenCalled(); await configure(); await view.get('input[data-test="source-topic_property_id"]').setValue('changed-topic');
    expect(view.get('[data-test="source-publication-impact"]').text()).toContain('可能暂时影响前台阅读');
    vi.mocked(ElMessageBox.confirm).mockRejectedValueOnce('cancel'); await view.get('[data-test="source-save"]').trigger('click'); await flushPromises();
    expect(sync.saveSource).not.toHaveBeenCalled(); expect(view.get('input[data-test="source-topic_property_id"]').element).toHaveProperty('value', 'changed-topic');
    expect(ElMessageBox.confirm).toHaveBeenCalledWith(expect.stringContaining('重新核验'), '确认公开影响', expect.any(Object));
  });
});
