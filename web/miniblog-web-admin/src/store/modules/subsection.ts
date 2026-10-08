import { defineStore } from 'pinia';
import { fetchSubsections, createSubsection, updateSubsection, publishSubsection, unpublishSubsection, deleteSubsection } from '@/api/subsection';
import { shareFetch, bySort } from '@/stores/shared-fetch';
import type { SubsectionItem } from '@/types/content';
export type { SubsectionItem } from '@/types/content';
export default defineStore('subsectionStore', {
  state: () => ({ subsectionsBySection: Object.create(null) as Record<string, SubsectionItem[]>, loadingSections: Object.create(null) as Record<string, boolean>, loadingVersions: Object.create(null) as Record<string, number>, loadedSections: Object.create(null) as Record<string, boolean>, revisionSections: Object.create(null) as Record<string, number> }),
  getters: { getSubsectionsBySection: state => (code: string) => state.subsectionsBySection[code] || [] },
  actions: {
    setSubsections(code: string, items: SubsectionItem[]) { this.subsectionsBySection[code] = bySort(items); this.loadedSections[code] = true; },
    invalidate(code: string) { this.loadedSections[code] = false; this.revisionSections[code] = (this.revisionSections[code] || 0) + 1; },
    upsertSubsection(item?: SubsectionItem) {
      if (!item) return;
      const parent = item.section_code;
      this.revisionSections[parent] = (this.revisionSections[parent] || 0) + 1;
      this.subsectionsBySection[parent] = bySort([...(this.subsectionsBySection[parent] || []).filter(existing => existing.code !== item.code), item]);
    },
    fetchSubsections(code: string, force = false): Promise<void> {
      if (!code || (this.loadedSections[code] && !force)) return Promise.resolve();
      const version = this.revisionSections[code] || 0;
      return shareFetch(this, `${code}:${version}`, async () => {
        this.loadingSections[code] = true; this.loadingVersions[code] = version;
        try {
          const response = await fetchSubsections(code) as unknown as { subsections?: SubsectionItem[] };
          if (version === (this.revisionSections[code] || 0)) this.setSubsections(code, response.subsections || []);
        } finally { if (version === this.loadingVersions[code]) this.loadingSections[code] = false; }
      });
    },
    async createSubsection(payload: { section_code: string; code: string; title: string; sort?: number }) { const response = await createSubsection(payload) as unknown as { subsection?: SubsectionItem }; this.upsertSubsection(response.subsection); },
    async updateSubsection(code: string, payload: { title: string; sort?: number }) { const response = await updateSubsection(code, payload) as unknown as { subsection?: SubsectionItem }; this.upsertSubsection(response.subsection); },
    async publishSubsection(code: string) { const response = await publishSubsection(code) as unknown as { subsection?: SubsectionItem }; this.upsertSubsection(response.subsection); },
    async unpublishSubsection(code: string) { const response = await unpublishSubsection(code) as unknown as { subsection?: SubsectionItem }; this.upsertSubsection(response.subsection); },
    async deleteSubsection(code: string) {
      await deleteSubsection(code);
      for (const parent of Object.keys(this.subsectionsBySection)) { this.revisionSections[parent] = (this.revisionSections[parent] || 0) + 1; this.subsectionsBySection[parent] = this.subsectionsBySection[parent].filter(item => item.code !== code); }
    }
  }
});
