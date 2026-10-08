import { createPinia, setActivePinia } from 'pinia';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import useSubsections from './subsection';
import { fetchSubsections } from '@/api/subsection';
vi.mock('@/api/subsection', () => ({ fetchSubsections: vi.fn(), createSubsection: vi.fn(), updateSubsection: vi.fn(), publishSubsection: vi.fn(), unpublishSubsection: vi.fn(), deleteSubsection: vi.fn() }));
beforeEach(() => { setActivePinia(createPinia()); vi.mocked(fetchSubsections).mockReset(); });
describe('subsection cache parent codes', () => {
  it.each(['constructor', 'toString', '__proto__'])('treats %s as a parent code without inheriting or changing object prototypes', async parent => {
    const store = useSubsections(); vi.mocked(fetchSubsections).mockResolvedValue({ subsections: [{ code: 'child', section_code: parent, title: '子章节', status: 1, sort: 1 }] } as never);
    await store.fetchSubsections(parent); expect(fetchSubsections).toHaveBeenCalledWith(parent); expect(store.getSubsectionsBySection(parent)[0].code).toBe('child');
    expect(store.loadedSections[parent]).toBe(true); expect(store.loadingSections[parent]).toBe(false);
    store.upsertSubsection({ code: 'second', section_code: parent, title: '新增', sort: 2 }); expect(store.getSubsectionsBySection(parent)).toHaveLength(2);
    store.invalidate(parent); expect(store.loadedSections[parent]).toBe(false); expect(store.getSubsectionsBySection('unloaded')).toEqual([]);
  });
});
