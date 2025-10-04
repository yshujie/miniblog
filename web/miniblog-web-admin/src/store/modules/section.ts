import { defineStore } from 'pinia';
import { fetchSections, createSection, updateSection, publishSection, unpublishSection } from '@/api/section';

export interface SectionItem {
  code: string;
  title: string;
  module_code: string;
  sort?: number;
  status?: number;
}

interface FetchSectionsResponse {
  sections?: SectionItem[];
}

interface SectionResponse {
  section?: SectionItem;
}

interface SectionState {
  sectionsByModule: Record<string, SectionItem[]>;
  loadingModules: Record<string, boolean>;
}

const normalizeSection = (section?: SectionItem) => {
  if (!section) {
    return undefined;
  }
  return {
    sort: 0,
    status: 0,
    ...section
  } as SectionItem;
};

export default defineStore({
  id: 'sectionStore',
  state: (): SectionState => ({
    sectionsByModule: {},
    loadingModules: {}
  }),
  getters: {
    getSectionsByModule: (state) => (moduleCode: string) => state.sectionsByModule[moduleCode] || []
  },
  actions: {
    setSections(moduleCode: string, sections: SectionItem[]) {
      this.sectionsByModule = {
        ...this.sectionsByModule,
        [moduleCode]: sections.map((item) => normalizeSection(item) as SectionItem)
      };
    },
    upsertSection(section?: SectionItem) {
      const normalized = normalizeSection(section);
      if (!normalized) {
        return;
      }
      const list = [...(this.sectionsByModule[normalized.module_code] || [])];
      const index = list.findIndex((item) => item.code === normalized.code);
      if (index >= 0) {
        list.splice(index, 1, { ...list[index], ...normalized });
      } else {
        list.push(normalized);
      }
      this.sectionsByModule = {
        ...this.sectionsByModule,
        [normalized.module_code]: list
      };
    },
    async fetchSections(moduleCode: string, force = false) {
      if (!moduleCode) {
        return;
      }
      if (this.loadingModules[moduleCode]) {
        return;
      }
      const hasCached = Boolean(this.sectionsByModule[moduleCode]);
      if (hasCached && !force) {
        return;
      }

      this.loadingModules = { ...this.loadingModules, [moduleCode]: true };
      try {
        const response = await fetchSections(moduleCode) as FetchSectionsResponse;
        this.setSections(moduleCode, response.sections ?? []);
      } catch (error) {
        throw error;
      } finally {
        const { [moduleCode]: _removed, ...rest } = this.loadingModules;
        this.loadingModules = rest;
      }
    },
    async createSection(payload: { module_code: string; code: string; title: string }) {
      const response = await createSection(payload) as SectionResponse;
      this.upsertSection(response.section);
    },
    async updateSection(code: string, payload: { title: string; sort?: number }) {
      const response = await updateSection(code, payload) as SectionResponse;
      this.upsertSection(response.section);
    },
    async publishSection(code: string) {
      const response = await publishSection(code) as SectionResponse;
      this.upsertSection(response.section);
    },
    async unpublishSection(code: string) {
      const response = await unpublishSection(code) as SectionResponse;
      this.upsertSection(response.section);
    }
  }
});
