import { defineStore } from 'pinia';
import { fetchModules, createModule, updateModule, publishModule, unpublishModule, deleteModule } from '@/api/module';

export interface ModuleItem {
  id?: number | string;
  code: string;
  title: string;
  status?: number;
}

interface FetchModulesResponse {
  modules?: ModuleItem[];
}

interface ModuleResponse {
  module?: ModuleItem;
}

interface ModuleState {
  modules: ModuleItem[];
  loading: boolean;
  loaded: boolean;
}

export default defineStore({
  id: 'moduleStore',
  state: (): ModuleState => ({
    modules: [],
    loading: false,
    loaded: false
  }),
  getters: {
    moduleOptions: (state) => state.modules,
    getModuleByCode: (state) => (code: string) => state.modules.find((item) => item.code === code)
  },
  actions: {
    async fetchModules(force = false) {
      if (this.loading) {
        return;
      }
      if (this.loaded && !force) {
        return;
      }

      this.loading = true;
      try {
        const response = await fetchModules() as FetchModulesResponse;
        this.modules = response.modules ?? [];
        this.loaded = true;
      } catch (error) {
        throw error;
      } finally {
        this.loading = false;
      }
    },
    async ensureLoaded(force = false) {
      if (force) {
        await this.fetchModules(true);
        return;
      }
      if (!this.loaded) {
        await this.fetchModules();
      }
    },
    upsertModule(module?: ModuleItem) {
      if (!module) {
        return;
      }
      const index = this.modules.findIndex((item) => item.code === module.code);
      if (index >= 0) {
        this.modules.splice(index, 1, { ...this.modules[index], ...module });
      } else {
        this.modules.push(module);
      }
      this.loaded = true;
    },
    async createNewModule(payload: { code: string; title: string }) {
      const response = await createModule(payload) as ModuleResponse;
      this.upsertModule(response.module);
    },
    async updateExistingModule(code: string, payload: { title: string }) {
      await this.ensureLoaded();
      const response = await updateModule(code, payload) as ModuleResponse;
      this.upsertModule(response.module);
    },
    async publishExistingModule(code: string) {
      await this.ensureLoaded();
      const response = await publishModule(code) as ModuleResponse;
      this.upsertModule(response.module);
    },
    async unpublishExistingModule(code: string) {
      await this.ensureLoaded();
      const response = await unpublishModule(code) as ModuleResponse;
      this.upsertModule(response.module);
    },
    async deleteExistingModule(code: string) {
      await this.ensureLoaded();
      await deleteModule(code);
      // remove from local list
      this.modules = this.modules.filter((item) => item.code !== code);
    }
  }
});
