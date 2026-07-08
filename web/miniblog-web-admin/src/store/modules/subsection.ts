import { defineStore } from 'pinia';
import {
  fetchSubsections,
  createSubsection,
  updateSubsection,
  publishSubsection,
  unpublishSubsection,
  deleteSubsection
} from '@/api/subsection';

export interface SubsectionItem {
  code: string;
  title: string;
  section_code: string;
  sort?: number;
  status?: number;
}

interface FetchSubsectionsResponse {
  subsections?: SubsectionItem[];
}

interface SubsectionResponse {
  subsection?: SubsectionItem;
}

interface SubsectionState {
  subsectionsBySection: Record<string, SubsectionItem[]>;
  loadingSections: Record<string, boolean>;
}

const normalizeSubsection = (subsection?: SubsectionItem) => {
  if (!subsection) {
    return undefined;
  }
  return {
    sort: 0,
    status: 0,
    ...subsection
  } as SubsectionItem;
};

export default defineStore({
  id: 'subsectionStore',
  state: (): SubsectionState => ({
    subsectionsBySection: {},
    loadingSections: {}
  }),
  getters: {
    getSubsectionsBySection: (state) => (sectionCode: string) => state.subsectionsBySection[sectionCode] || []
  },
  actions: {
    setSubsections(sectionCode: string, subsections: SubsectionItem[]) {
      this.subsectionsBySection = {
        ...this.subsectionsBySection,
        [sectionCode]: subsections.map((item) => normalizeSubsection(item) as SubsectionItem)
      };
    },
    upsertSubsection(subsection?: SubsectionItem) {
      const normalized = normalizeSubsection(subsection);
      if (!normalized) {
        return;
      }
      const list = [...(this.subsectionsBySection[normalized.section_code] || [])];
      const index = list.findIndex((item) => item.code === normalized.code);
      if (index >= 0) {
        list.splice(index, 1, { ...list[index], ...normalized });
      } else {
        list.push(normalized);
      }
      this.subsectionsBySection = {
        ...this.subsectionsBySection,
        [normalized.section_code]: list
      };
    },
    async fetchSubsections(sectionCode: string, force = false) {
      if (!sectionCode) {
        return;
      }
      if (this.loadingSections[sectionCode]) {
        return;
      }
      const hasCached = Boolean(this.subsectionsBySection[sectionCode]);
      if (hasCached && !force) {
        return;
      }

      this.loadingSections = { ...this.loadingSections, [sectionCode]: true };
      try {
        const response = await fetchSubsections(sectionCode) as FetchSubsectionsResponse;
        this.setSubsections(sectionCode, response.subsections ?? []);
      } finally {
        const { [sectionCode]: _removed, ...rest } = this.loadingSections;
        this.loadingSections = rest;
      }
    },
    async createSubsection(payload: { section_code: string; code: string; title: string }) {
      const response = await createSubsection(payload) as SubsectionResponse;
      this.upsertSubsection(response.subsection);
    },
    async updateSubsection(code: string, payload: { title: string; sort?: number }) {
      const response = await updateSubsection(code, payload) as SubsectionResponse;
      this.upsertSubsection(response.subsection);
    },
    async publishSubsection(code: string) {
      const response = await publishSubsection(code) as SubsectionResponse;
      this.upsertSubsection(response.subsection);
    },
    async unpublishSubsection(code: string) {
      const response = await unpublishSubsection(code) as SubsectionResponse;
      this.upsertSubsection(response.subsection);
    },
    async deleteSubsection(code: string) {
      await deleteSubsection(code);
      const sectionCode = Object.keys(this.subsectionsBySection).find((key) =>
        (this.subsectionsBySection[key] || []).some((item) => item.code === code)
      );
      if (sectionCode) {
        this.subsectionsBySection = {
          ...this.subsectionsBySection,
          [sectionCode]: (this.subsectionsBySection[sectionCode] || []).filter((item) => item.code !== code)
        };
      }
    }
  }
});
