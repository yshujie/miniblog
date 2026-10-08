import { defineStore } from 'pinia';
import { fetchSections, createSection, updateSection, publishSection, unpublishSection, deleteSection } from '@/api/section';
import { shareFetch, bySort } from '@/stores/shared-fetch';
import type { SectionItem } from '@/types/content';
export type { SectionItem } from '@/types/content';
export default defineStore('sectionStore', {
  state: () => ({ sectionsByModule: Object.create(null) as Record<string, SectionItem[]>, loadingModules: Object.create(null) as Record<string, boolean>, loadingVersions: Object.create(null) as Record<string, number>, loadedModules: Object.create(null) as Record<string, boolean>, revisionModules: Object.create(null) as Record<string, number> }),
  getters: { getSectionsByModule: state => (code: string) => state.sectionsByModule[code] || [] },
  actions: {
    setSections(code: string, items: SectionItem[]) { this.sectionsByModule[code] = bySort(items); this.loadedModules[code] = true; },
    invalidate(code: string) { this.loadedModules[code] = false; this.revisionModules[code] = (this.revisionModules[code] || 0) + 1; },
    upsertSection(item?: SectionItem) {
      if (!item) return;
      const parent = item.module_code;
      this.revisionModules[parent] = (this.revisionModules[parent] || 0) + 1;
      this.sectionsByModule[parent] = bySort([...(this.sectionsByModule[parent] || []).filter(existing => existing.code !== item.code), item]);
    },
    fetchSections(code: string, force = false): Promise<void> {
      if (!code || (this.loadedModules[code] && !force)) return Promise.resolve();
      const version = this.revisionModules[code] || 0;
      return shareFetch(this, `${code}:${version}`, async () => {
        this.loadingModules[code] = true; this.loadingVersions[code] = version;
        try {
          const response = await fetchSections(code) as unknown as { sections?: SectionItem[] };
          if (version === (this.revisionModules[code] || 0)) this.setSections(code, response.sections || []);
        } finally { if (version === this.loadingVersions[code]) this.loadingModules[code] = false; }
      });
    },
    async createSection(payload: { module_code: string; code: string; title: string; sort?: number }) { const response = await createSection(payload) as unknown as { section?: SectionItem }; this.upsertSection(response.section); },
    async updateSection(code: string, payload: { title: string; sort?: number }) { const response = await updateSection(code, payload) as unknown as { section?: SectionItem }; this.upsertSection(response.section); },
    async publishSection(code: string) { const response = await publishSection(code) as unknown as { section?: SectionItem }; this.upsertSection(response.section); },
    async unpublishSection(code: string) { const response = await unpublishSection(code) as unknown as { section?: SectionItem }; this.upsertSection(response.section); },
    async deleteSection(code: string) {
      await deleteSection(code);
      for (const parent of Object.keys(this.sectionsByModule)) { this.revisionModules[parent] = (this.revisionModules[parent] || 0) + 1; this.sectionsByModule[parent] = this.sectionsByModule[parent].filter(item => item.code !== code); }
    }
  }
});
