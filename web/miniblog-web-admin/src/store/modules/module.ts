import { defineStore } from 'pinia';
import { fetchModules, createModule, updateModule, publishModule, unpublishModule, deleteModule } from '@/api/module';
import type { ModuleItem } from '@/types/content';
import { shareFetch, bySort } from '@/stores/shared-fetch';
export type { ModuleItem } from '@/types/content';
export default defineStore('moduleStore', {
  state: () => ({ modules: [] as ModuleItem[], loading: false, loadingVersion: -1, loaded: false, revision: 0 }),
  getters: {
    moduleOptions: state => state.modules,
    getModuleByCode: state => (code: string) => state.modules.find(item => item.code === code)
  },
  actions: {
    fetchModules(force = false): Promise<void> {
      if (this.loaded && !force) return Promise.resolve();
      const version = this.revision;
      return shareFetch(this, `modules:${version}`, async () => {
        this.loading = true; this.loadingVersion = version;
        try {
          const response = await fetchModules() as unknown as { modules?: ModuleItem[] };
          if (version === this.revision) { this.modules = bySort(response.modules || []); this.loaded = true; }
        } finally { if (version === this.loadingVersion) this.loading = false; }
      });
    },
    ensureLoaded(force = false) { return this.fetchModules(force); },
    invalidate() { this.loaded = false; this.revision += 1; },
    upsertModule(item?: ModuleItem) {
      if (!item) return;
      this.revision += 1;
      this.modules = bySort([...this.modules.filter(existing => existing.code !== item.code), item]);
    },
    async createNewModule(payload: { code: string; title: string; sort?: number }) {
      const response = await createModule(payload) as unknown as { module?: ModuleItem }; this.upsertModule(response.module);
    },
    async updateExistingModule(code: string, payload: { title: string; sort?: number }) {
      const response = await updateModule(code, payload) as unknown as { module?: ModuleItem }; this.upsertModule(response.module);
    },
    async publishExistingModule(code: string) { const response = await publishModule(code) as unknown as { module?: ModuleItem }; this.upsertModule(response.module); },
    async unpublishExistingModule(code: string) { const response = await unpublishModule(code) as unknown as { module?: ModuleItem }; this.upsertModule(response.module); },
    async deleteExistingModule(code: string) { await deleteModule(code); this.revision += 1; this.modules = this.modules.filter(item => item.code !== code); }
  }
});
