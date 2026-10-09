import { mount, flushPromises, type VueWrapper } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import ElementPlus, { ElMessageBox } from 'element-plus';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import CatalogContextPanel from './CatalogContextPanel.vue';
import useModules from '@/store/modules/module';
import useSections from '@/store/modules/section';
import { updateSection, createSection, deleteSection } from '@/api/section';
import type { DirectoryNode } from '@/composables/useCatalog';
vi.mock('@/api/module', () => ({ fetchModules: vi.fn(), createModule: vi.fn(), updateModule: vi.fn(), publishModule: vi.fn(), unpublishModule: vi.fn(), deleteModule: vi.fn() }));
vi.mock('@/api/section', () => ({ fetchSections: vi.fn(), createSection: vi.fn(), updateSection: vi.fn(), publishSection: vi.fn(), unpublishSection: vi.fn(), deleteSection: vi.fn() }));
vi.mock('@/api/subsection', () => ({ fetchSubsections: vi.fn(), createSubsection: vi.fn(), updateSubsection: vi.fn(), publishSubsection: vi.fn(), unpublishSubsection: vi.fn(), deleteSubsection: vi.fn() }));
const section = { code: 'base', title: '基础', module_code: 'go', sort: 1 };
const node: DirectoryNode = { key: 'section:base', code: 'base', label: '基础', kind: 'section', module_code: 'go', section_code: 'base', subsection_code: '' };
let wrapper: VueWrapper;
beforeEach(() => { setActivePinia(createPinia()); vi.clearAllMocks(); useModules().modules = [{ code: 'go', title: 'Go', status: 1 }]; useSections().setSections('go', [section]); vi.spyOn(ElMessageBox, 'confirm').mockResolvedValue('confirm' as never); });
afterEach(() => { wrapper?.unmount(); vi.restoreAllMocks(); });
async function open(value = node) { wrapper = mount(CatalogContextPanel, { props: { node: value }, global: { plugins: [ElementPlus], stubs: { ElDrawer: { template: '<section><slot /><slot name="footer" /></section>' }, ElDropdown: { template: '<div><slot /><slot name="dropdown" /></div>' }, ElDropdownMenu: { template: '<div><slot /></div>' }, ElDropdownItem: { emits: ['click'], template: '<button @click="$emit(\'click\')"><slot /></button>' }}}}); await flushPromises(); }
async function click(label: string) { await wrapper.findAll('button').find(value => value.text() === label)!.trigger('click'); await flushPromises(); }
describe('catalog contextual operations', () => {
  it('keeps unknown status explicit and updates the title without changing code or parent', async () => {
    await open(); expect(wrapper.text()).toContain('状态待确认'); await click('编辑资料');
    expect(wrapper.get('input[data-test="catalog-code"]').attributes('disabled')).toBeDefined(); await wrapper.get('input[data-test="catalog-title"]').setValue('新的基础标题');
    vi.mocked(updateSection).mockResolvedValue({ section: { ...section, title: '新的基础标题' }} as never); await click('保存资料');
    expect(updateSection).toHaveBeenCalledWith('base', { title: '新的基础标题', sort: 1 }); expect(useSections().getSectionsByModule('go')[0].title).toBe('新的基础标题'); expect(wrapper.emitted('changed')?.[0][0]).toMatchObject({ module_code: 'go', section_code: 'base', subsection_code: '' });
  });
  it('creates a chapter under its fixed selected topic and emits the actual new context', async () => {
    await open({ ...node, key: 'module:go', code: 'go', kind: 'module', section_code: '' }); await click('新增章节'); await wrapper.get('input[data-test="catalog-title"]').setValue('工程化'); await wrapper.get('input[data-test="catalog-code"]').setValue('engineering');
    vi.mocked(createSection).mockResolvedValue({ section: { code: 'engineering', title: '工程化', module_code: 'go', sort: 0, status: 1 }} as never); await click('创建目录');
    expect(createSection).toHaveBeenCalledWith({ title: '工程化', code: 'engineering', module_code: 'go', sort: 0 }); expect(wrapper.emitted('changed')?.[0][0]).toEqual({ module_code: 'go', section_code: 'engineering', subsection_code: '' });
  });
  it('retains context and shows a server dependency rejection instead of deleting locally', async () => {
    await open(); vi.mocked(deleteSection).mockRejectedValue(new Error('章节仍有文章')); await click('永久删除目录'); expect(wrapper.text()).toContain('章节仍有文章'); expect(useSections().getSectionsByModule('go')).toHaveLength(1); expect(wrapper.emitted('changed')).toBeUndefined();
  });
});
