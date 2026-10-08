import { createPinia, setActivePinia } from 'pinia';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useCatalog } from './useCatalog';
import { reorderCatalog } from '@/api/content';
vi.mock('@/api/module', () => ({ fetchModules: vi.fn(), createModule: vi.fn(), updateModule: vi.fn(), publishModule: vi.fn(), unpublishModule: vi.fn(), deleteModule: vi.fn() }));
vi.mock('@/api/section', () => ({ fetchSections: vi.fn(), createSection: vi.fn(), updateSection: vi.fn(), publishSection: vi.fn(), unpublishSection: vi.fn(), deleteSection: vi.fn() }));
vi.mock('@/api/subsection', () => ({ fetchSubsections: vi.fn(), createSubsection: vi.fn(), updateSubsection: vi.fn(), publishSubsection: vi.fn(), unpublishSubsection: vi.fn(), deleteSubsection: vi.fn() }));
vi.mock('@/api/content', () => ({ reorderCatalog: vi.fn() }));
beforeEach(() => { setActivePinia(createPinia()); });
describe('catalog context and sibling order', () => {
  it('requires the whole target chain to be normal, including recent contexts', () => {
    const catalog = useCatalog(); const context = { module_code: 'go', section_code: 'base', subsection_code: 'first' };
    catalog.modules.modules = [{ code: 'go', title: 'Go', status: 1 }];
    catalog.sections.setSections('go', [{ code: 'base', module_code: 'go', title: '基础', status: 1 }]);
    catalog.subsections.setSubsections('base', [{ code: 'first', section_code: 'base', title: '第一节', status: 1 }]);
    expect(catalog.valid(context)).toBe(true); catalog.modules.modules[0].status = 2; expect(catalog.valid(context)).toBe(false);
    catalog.modules.modules[0].status = 1; catalog.subsections.subsectionsBySection.base[0].status = 2; expect(catalog.valid(context)).toBe(false);
    expect(catalog.valid({ ...context, subsection_code: '' })).toBe(true);
  });
  it('retains hidden siblings in the complete catalog reorder request', async () => {
    const catalog = useCatalog(); catalog.modules.modules = [{ code: 'go', title: 'Go', status: 1 }, { code: 'hidden', title: '隐藏', status: 2 }];
    vi.mocked(reorderCatalog).mockRejectedValue(new Error('stop before reload'));
    await expect(catalog.move(catalog.tree.value[0], 1)).rejects.toThrow('stop before reload');
    expect(reorderCatalog).toHaveBeenCalledWith({ kind: 'module', parent_code: undefined, codes: ['hidden', 'go'] });
  });
});
