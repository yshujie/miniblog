import { createPinia, setActivePinia } from 'pinia';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import useSections from './section';
import { fetchSections, createSection } from '@/api/section';
vi.mock('@/api/section', () => ({ fetchSections: vi.fn(), createSection: vi.fn(), updateSection: vi.fn(), publishSection: vi.fn(), unpublishSection: vi.fn(), deleteSection: vi.fn() }));
const row = (code: string, sort = 1) => ({ module_code: 'go', code, title: code, sort });
beforeEach(() => { setActivePinia(createPinia()); vi.mocked(fetchSections).mockReset(); });
describe('directory cache completeness', () => {
  it.each(['constructor', 'toString', '__proto__'])('treats %s as an ordinary parent code in every cache', async parent => {
    const store = useSections(); vi.mocked(fetchSections).mockResolvedValue({ sections: [{ ...row('child'), module_code: parent }] } as never);
    await store.fetchSections(parent); expect(fetchSections).toHaveBeenCalledWith(parent); expect(store.getSectionsByModule(parent)[0].code).toBe('child');
    expect(store.loadedModules[parent]).toBe(true); expect(store.loadingModules[parent]).toBe(false);
    store.upsertSection({ ...row('created'), module_code: parent }); expect(store.getSectionsByModule(parent)).toHaveLength(2);
    store.invalidate(parent); expect(store.loadedModules[parent]).toBe(false); expect(store.getSectionsByModule('unloaded')).toEqual([]);
  });
  it('does not treat a create result as the full parent collection', async () => {
    const store = useSections(); vi.mocked(createSection).mockResolvedValue({ section: row('new') } as never);
    await store.createSection({ module_code: 'go', code: 'new', title: 'new', sort: 2 });
    expect(store.loadedModules.go).not.toBe(true);
    vi.mocked(fetchSections).mockResolvedValue({ sections: [row('existing'), row('new', 2)] } as never);
    await store.fetchSections('go'); expect(store.getSectionsByModule('go')).toHaveLength(2);
  });
  it('shares in-flight fetches and remembers an empty complete collection', async () => {
    const store = useSections(); let finish!: (value: unknown) => void;
    vi.mocked(fetchSections).mockReturnValue(new Promise(resolve => { finish = resolve; }) as never);
    const first = store.fetchSections('go'); const second = store.fetchSections('go'); await Promise.resolve();
    expect(fetchSections).toHaveBeenCalledTimes(1); finish({ sections: [] }); await Promise.all([first, second]);
    await store.fetchSections('go'); expect(fetchSections).toHaveBeenCalledTimes(1); expect(store.loadedModules.go).toBe(true);
  });
  it('lets an invalidation start a fresh request while the obsolete request is still pending', async () => {
    const store = useSections(); let finish!: (value: unknown) => void;
    vi.mocked(fetchSections).mockReturnValueOnce(new Promise(resolve => { finish = resolve; }) as never).mockResolvedValueOnce({ sections: [row('fresh')] } as never);
    const old = store.fetchSections('go'); await Promise.resolve(); store.invalidate('go'); await store.fetchSections('go');
    finish({ sections: [row('stale')] }); await old; expect(store.sectionsByModule.go.map(item => item.code)).toEqual(['fresh']);
  });
  it('keeps a mutation from being overwritten by an earlier fetch', async () => {
    const store = useSections(); let finish!: (value: unknown) => void;
    vi.mocked(fetchSections).mockReturnValueOnce(new Promise(resolve => { finish = resolve; }) as never);
    const old = store.fetchSections('go'); await Promise.resolve(); store.upsertSection(row('created')); finish({ sections: [row('stale')] }); await old;
    expect(store.sectionsByModule.go.map(item => item.code)).toEqual(['created']); expect(store.loadedModules.go).not.toBe(true); expect(store.loadingModules.go).toBe(false);
  });
  it('ignores invalidated responses and sorts updated items immediately', async () => {
    const store = useSections(); let finish!: (value: unknown) => void;
    vi.mocked(fetchSections).mockReturnValue(new Promise(resolve => { finish = resolve; }) as never);
    const pending = store.fetchSections('go'); await Promise.resolve(); store.invalidate('go'); finish({ sections: [row('stale')] }); await pending;
    expect(store.loadedModules.go).not.toBe(true); expect(store.sectionsByModule.go).toBeUndefined();
    store.setSections('go', [row('a', 1), row('b', 2)]); store.upsertSection(row('b', 0)); expect(store.sectionsByModule.go.map(item => item.code)).toEqual(['b', 'a']);
  });
});
