import { mount, flushPromises, type VueWrapper } from '@vue/test-utils';
import { ref, reactive } from 'vue';
import { createRouter, createMemoryHistory, type Router } from 'vue-router';
import ElementPlus, { ElMessage, ElMessageBox, ElSelect } from 'element-plus';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import SyncView from './sync.vue';
import { useNotionSync } from '@/composables/useNotionSync';
import type { SyncSource, SyncStatus } from '@/types/notion-sync';
vi.mock('@/composables/useNotionSync', () => ({ useNotionSync: vi.fn() }));
vi.mock('@/composables/useCatalog', () => ({ useCatalog: () => ({ load: vi.fn().mockResolvedValue(undefined), modules: { modules: [{ code: 'go', title: 'Go', status: 1 }, { code: 'python', title: 'Python', status: 1 }] }, sections: { getSectionsByModule: (code: string) => code === 'go' ? [{ code: 'base', title: '基础', status: 1 }, { code: 'target', title: '目标章节', status: 1 }, { code: 'disabled', title: '停用章节', status: 2 }] : [{ code: 'python-base', title: 'Python 基础', status: 1 }] }, subsections: { getSubsectionsBySection: () => [] }}) }));
const source = (): SyncSource => ({ source_id: 'source', label: 'Go 文档', module_code: 'go', enabled: true, health: 'healthy', config_revision: 7, catalog_bindings: [], config: { title_property_id: 'title', state_property_id: 'state', topic_property_id: 'topic', tags_property_id: 'tags', state_option_ids: { draft: 'd', published: 'p', unpublished: 'u', archived: 'a' }}});
function setup() {
  const value = source();
  return {
    latestSourceLoading: ref(false), loading: ref(false), actionBusy: ref(false), error: ref(''), detailError: ref(''), detailLoading: ref(false),
    status: ref<SyncStatus>({ enabled: true, paused: false, baseline_frozen: true, health: 'healthy', pending_count: 2, blocked_count: 3, error_count: 4, sources: [value] }),
    sources: ref({ items: [value], total: 1, page: 1, limit: 20 }), pages: ref({ items: [{ page_id: 'page', source_id: 'source', title: '受限文章', desired_state: 'published', local_state: 'published', effective_visibility: false, needs_revalidation: true, visibility_reason: 'needs_revalidation', management_state: 'managed', publication_hold: false }], total: 1, page: 1, limit: 20 }),
    runs: ref({ items: [{ run_id: 'finished-with-pending', mode: 'dry_run', status: 'completed', phase: 'finished', finished_at: '2026-10-08T00:00:00Z', counts: { seen: 5, created: 0, updated: 0, unchanged: 0, blocked: 0, failed: 0, pending: 2, frozen: 3 }}], total: 1, page: 1, limit: 20 }), items: ref({ items: [], total: 0, page: 1, limit: 20 }), selectedRun: ref(undefined), selectedRunID: ref(''),
    sourceQuery: reactive({ page: 1, limit: 20 }), pageFilterDraft: reactive({ title: '', source_id: '', management_state: '' }), pageFilters: reactive({ page: 1, limit: 20, title: '', source_id: '', management_state: '' }), runQuery: reactive({ page: 1, limit: 20 }), itemQuery: reactive({ page: 1, limit: 20 }),
    refresh: vi.fn().mockResolvedValue(undefined), readSource: vi.fn(), searchPages: vi.fn(), loadPages: vi.fn(), loadRuns: vi.fn(), selectRun: vi.fn(), closeRun: vi.fn(), control: vi.fn(), start: vi.fn(), saveSource: vi.fn().mockResolvedValue(true), saveBinding: vi.fn()
  };
}
let sync: ReturnType<typeof setup>; let wrapper: VueWrapper | undefined; let router: Router;
beforeEach(() => {
  sync = setup(); vi.mocked(useNotionSync).mockReturnValue(sync as unknown as ReturnType<typeof useNotionSync>);
  vi.spyOn(ElMessageBox, 'confirm').mockResolvedValue('confirm' as Awaited<ReturnType<typeof ElMessageBox.confirm>>);
});
afterEach(() => { wrapper?.unmount(); wrapper = undefined; ElMessage.closeAll(); vi.restoreAllMocks(); });
async function open(location = '/content/sync') {
  router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/content/sync', component: { template: '<div />' }}] });
  await router.push(location); await router.isReady();
  wrapper = mount(SyncView, { global: { plugins: [ElementPlus, router], stubs: {
    ElDialog: { props: ['modelValue'], template: '<section v-if="modelValue"><slot /><slot name="footer" /></section>' },
    ElDrawer: { props: ['modelValue', 'title', 'size'], template: '<section v-if="modelValue" role="dialog" :aria-label="title" :data-size="size"><h2>{{ title }}</h2><slot /><slot name="footer" /></section>' },
    RouterLink: { props: ['to'], template: '<a :href="to"><slot /></a>' }
  }}});
  await flushPromises(); return wrapper;
}
async function tab(name: string) { await wrapper!.get('#tab-' + name).trigger('click'); await flushPromises(); }
async function configure() { await tab('sources'); await wrapper!.findAll('button').find(button => button.text() === '配置')!.trigger('click'); await flushPromises(); }
describe('sync management page', () => {
  it('keeps missing run counts unknown and distinguishes preview plans from applied updates', async () => {
    sync.runs.value.items[0].counts.seen = undefined as unknown as number;
    sync.runs.value.items[0].counts.updated = undefined as unknown as number;
    sync.runs.value.items[0].counts.failed = undefined as unknown as number;
    sync.status.value!.health = 'disabled';
    const view = await open();
    const summary = view.get('.run-summary'); expect(summary.text()).toContain('发现 —'); expect(summary.text()).toContain('拟更新 —'); expect(summary.text()).toContain('失败 —');
    expect(view.get('.health-dot').classes()).not.toContain('is-healthy');
  });

  it('presents four tabs and preserves unrelated legacy query and hash during tab changes', async () => {
    const view = await open('/content/sync?legacy=keep&source_id=source#reading');
    expect(view.findAll('[role="tab"]').map(node => node.text())).toEqual(['同步概览', '来源与绑定', '页面状态', '运行记录']);
    expect(view.find('[data-test="sync-overview"]').exists()).toBe(true);
    expect(view.find('[data-test="sync-pages"]').exists()).toBe(false);
    await tab('pages');
    expect(router.currentRoute.value.fullPath).toBe('/content/sync?legacy=keep&source_id=source&tab=pages#reading');
    expect(view.find('[data-test="sync-pages"]').exists()).toBe(true);
    expect(view.find('[data-test="sync-overview"]').exists()).toBe(false);
    await router.replace({ path: '/content/sync', query: { tab: 'runs', legacy: 'keep' }, hash: '#reading' }); await flushPromises();
    expect(view.find('[data-test="sync-runs"]').exists()).toBe(true);
    await router.replace({ path: '/content/sync', query: { tab: 'unknown', legacy: 'keep' }}); await flushPromises();
    expect(view.find('[data-test="sync-overview"]').exists()).toBe(true);
    expect(router.currentRoute.value.query.legacy).toBe('keep');
  });
  it('opens a nonempty saved run query as a string without starting another run', async () => {
    const view = await open('/content/sync?tab=runs&run_id=9007199254740993&legacy=keep#detail');
    expect(sync.selectRun).toHaveBeenCalledWith('9007199254740993');
    expect(view.get('[data-test="run-panel"]').attributes('aria-label')).toBe('同步运行详情');
    expect(sync.start).not.toHaveBeenCalled();
    await router.replace({ path: '/content/sync', query: { tab: 'runs', run_id: '' }}); await flushPromises();
    expect(sync.selectRun).toHaveBeenCalledOnce();
  });
  it('keeps unsubmitted filters, current pages and source drafts while the route tab changes', async () => {
    const view = await open('/content/sync?tab=sources'); await configure();
    await view.get('input[data-test="source-label"]').setValue('未保存来源名称');
    sync.pageFilterDraft.title = '尚未查询'; sync.pageFilters.page = 3; sync.sourceQuery.page = 2; sync.runQuery.page = 4;
    await router.replace({ path: '/content/sync', query: { tab: 'pages' }}); await flushPromises();
    expect(view.get('input[data-test="source-label"]').element).toHaveProperty('value', '未保存来源名称');
    expect(sync.pageFilterDraft.title).toBe('尚未查询'); expect(sync.pageFilters.page).toBe(3); expect(sync.sourceQuery.page).toBe(2); expect(sync.runQuery.page).toBe(4);
    expect(sync.searchPages).not.toHaveBeenCalled(); expect(sync.start).not.toHaveBeenCalled();
  });
  it('uses one titled side panel for source and binding and makes the source panel full screen on touch widths', async () => {
    const view = await open(); await configure();
    expect(view.findAll('[role="dialog"]')).toHaveLength(1);
    expect(view.get('[data-test="source-panel"]').attributes('aria-label')).toBe('来源配置');
    expect(view.get('[data-test="source-panel"]').findAll('h2').filter(node => node.text() === '来源配置')).toHaveLength(1);
    const initial = window.innerWidth; Object.defineProperty(window, 'innerWidth', { value: 390, configurable: true }); window.dispatchEvent(new Event('resize')); await flushPromises();
    expect(view.get('[data-test="source-panel"]').attributes('data-size')).toBe('100%');
    Object.defineProperty(window, 'innerWidth', { value: initial, configurable: true }); window.dispatchEvent(new Event('resize'));
  });
  it('keeps source pause distinct from local takedown and opens saved run IDs without executing them', async () => {
    sync.sources.value.items[0].enabled = false;
    const view = await open('/content/sync?tab=sources');
    expect(view.text()).toContain('停用来源不等于本地紧急下架');
    sync.runs.value.items[0].run_id = '9007199254740993'; await tab('runs');
    await view.findAll('button').find(button => button.text() === '9007199254740993')!.trigger('click'); await flushPromises();
    expect(sync.selectRun).toHaveBeenCalledWith('9007199254740993'); expect(sync.start).not.toHaveBeenCalled();
    expect(view.get('[data-test="run-panel"]').attributes('aria-label')).toBe('同步运行详情');
    sync.detailError.value = '详情读取失败'; await flushPromises();
    await view.findAll('button').find(button => button.text() === '重试读取')!.trigger('click');
    expect(sync.selectRun).toHaveBeenLastCalledWith('9007199254740993', false); expect(sync.start).not.toHaveBeenCalled();
  });

  it('shows source intent, local state and actual public restriction separately', async () => {
    const view = await open();
    expect(view.text()).toContain('历史待核对 2'); expect(view.text()).toContain('公开受限 3'); expect(view.text()).toContain('读取 / 执行失败 4');
    await tab('pages');
    expect(view.text()).toContain('来源期望'); expect(view.text()).toContain('本站状态'); expect(view.text()).toContain('前台不可见'); expect(view.text()).toContain('等待重新核验');
    expect(view.text()).toContain('等待重新同步核验来源'); await tab('runs'); expect(view.text()).toContain('待处理 2'); expect(view.text()).toContain('冻结 3'); expect(view.text()).toContain('失败 0'); expect(view.text()).not.toContain('部分项目失败或被阻止');
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
  it('allows preview with the runtime gate off but requires every sync gate and no active run', async () => {
    const view = await open();
    sync.status.value = { ...sync.status.value, enabled: false, paused: true, source_writes_paused: true, baseline_frozen: false };
    await flushPromises();
    expect(view.get('[data-test="run-preview"]').attributes('disabled')).toBeUndefined();
    await view.get('[data-test="run-preview"]').trigger('click'); expect(sync.start).toHaveBeenCalledWith('dry_run');
    expect(view.get('[data-test="run-sync"]').attributes('disabled')).toBeDefined();
    for (const blocked of [{ enabled: false }, { paused: true }, { source_writes_paused: true }, { baseline_frozen: false }, { baseline_frozen: undefined }, { current_run_id: 'active' }]) {
      sync.status.value = { ...sync.status.value, enabled: true, paused: false, source_writes_paused: false, baseline_frozen: true, current_run_id: undefined, ...blocked };
      await flushPromises(); expect(view.get('[data-test="run-sync"]').attributes('disabled')).toBeDefined();
    }
    expect(view.get('[data-test="run-preview"]').attributes('disabled')).toBeDefined();
    sync.status.value = { ...sync.status.value, current_run_id: undefined }; await flushPromises();
    expect(view.get('[data-test="run-sync"]').attributes('disabled')).toBeUndefined();
  });
  it('keeps a failed configuration draft and revision until latest comparison is explicitly accepted', async () => {
    const view = await open(); await configure(); await view.get('input[data-test="source-label"]').setValue('我的草稿');
    sync.saveSource.mockResolvedValueOnce(false); await view.get('[data-test="source-save"]').trigger('click'); await flushPromises();
    sync.readSource.mockResolvedValue({ ...source(), label: '远端名称', config_revision: 8 });
    await view.get('[data-test="source-read-latest"]').trigger('click'); await flushPromises();
    expect(view.get('input[data-test="source-label"]').element).toHaveProperty('value', '我的草稿');
    expect(view.get('[data-test="source-comparison"]').text()).toContain('编辑依据版本 7 · 最新版本 8');
    expect(view.get('[data-test="source-comparison"]').text()).toContain('远端名称');
    expect(view.get('[data-test="source-save"]').attributes('disabled')).toBeDefined(); expect(sync.saveSource).toHaveBeenCalledOnce();
    await view.get('[data-test="source-accept-latest"]').trigger('click'); await view.get('[data-test="source-save"]').trigger('click'); await flushPromises();
    expect(sync.saveSource).toHaveBeenLastCalledWith(expect.objectContaining({ config_revision: 8 }), { label: '我的草稿', expected_config_revision: 8 });
  });
  it('preserves a binding choice across conflict comparison, validates enabled same-module sections and requires explicit review', async () => {
    const binding = { id: '9007199254740993', source_id: 'source', option_id: 'topic', option_name: '主题', section_code: 'base', status: 'bound' };
    sync.sources.value.items[0].catalog_bindings = [binding]; const view = await open(); await tab('sources');
    await view.findAll('button').find(button => button.text() === '调整绑定')!.trigger('click'); await flushPromises();
    const choose = async (code: string) => { const selects = view.findAllComponents(ElSelect); selects[selects.length - 1].vm.$emit('update:modelValue', code); await flushPromises(); };
    await choose('disabled'); expect(view.get('[data-test="binding-save"]').attributes('disabled')).toBeDefined();
    await choose('python-base'); expect(view.get('[data-test="binding-save"]').attributes('disabled')).toBeDefined();
    await choose('target'); sync.saveBinding.mockResolvedValueOnce(false); await view.get('[data-test="binding-save"]').trigger('click'); await flushPromises();
    expect(sync.saveBinding).toHaveBeenLastCalledWith(binding, 'target', 7);
    sync.readSource.mockResolvedValue({ ...source(), config_revision: 8, catalog_bindings: [{ ...binding, section_code: 'remote' }] });
    await view.get('[data-test="binding-read-latest"]').trigger('click'); await flushPromises();
    expect(view.get('[data-test="binding-comparison"]').text()).toContain('本次草稿：target'); expect(view.get('[data-test="binding-comparison"]').text()).toContain('go / remote');
    expect(view.get('[data-test="binding-save"]').attributes('disabled')).toBeDefined(); expect(sync.saveBinding).toHaveBeenCalledOnce();
    await view.get('[data-test="binding-accept-latest"]').trigger('click'); sync.saveBinding.mockResolvedValueOnce(true); await view.get('[data-test="binding-save"]').trigger('click'); await flushPromises();
    expect(sync.saveBinding).toHaveBeenLastCalledWith(expect.objectContaining({ id: '9007199254740993', section_code: 'remote' }), 'target', 8);
  });
  it('does not adopt a binding that disappeared while the draft was open', async () => {
    sync.sources.value.items[0].catalog_bindings = [{ id: 'binding', source_id: 'source', option_id: 'topic', option_name: '主题', section_code: 'base', status: 'bound' }];
    const view = await open(); await tab('sources'); await view.findAll('button').find(button => button.text() === '调整绑定')!.trigger('click');
    sync.readSource.mockResolvedValue({ ...source(), config_revision: 8, catalog_bindings: [] });
    await view.get('[data-test="binding-read-latest"]').trigger('click'); await flushPromises();
    expect(view.text()).toContain('此主题绑定已不存在'); expect(view.find('[data-test="binding-accept-latest"]').exists()).toBe(false);
    expect(view.get('[data-test="binding-save"]').attributes('disabled')).toBeDefined(); expect(sync.saveBinding).not.toHaveBeenCalled();
  });
});
