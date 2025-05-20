import { defineStore } from 'pinia'
import type { Module } from '@/types/module'
import { fetchModules } from '@/api/module'

// module store
export const useModuleStore = defineStore('module', {
  state: () => ({
    modules: [] as Module[],
  }),

  getters: {
    // 获取所有模块
    getAllModules: (state): Module[] => state.modules,

    // 根据 code 获取模块
    getModuleByCode: (state) => (code: string): Module | undefined => {
      return state.modules.find(module => module.code === code)
    },
  },

  actions: {
    // 加载所有模块
    async loadModules() {
      if (this.modules.length > 0) {
        return // 如果已经加载过，直接返回
      }

      this.modules = await fetchModules()
    },

    // 加载模块下的所有章节
    async loadSections(code: string) {
      const module = this.getModuleByCode(code)
      if (!module) {
        throw new Error(`Module with code ${code} not found`)
      }
      if (module.sections.length > 0) {
        return // 如果已经加载过，直接返回
      }

      const sections = await module.loadSections()
      module.sections = sections
    },

    // 加载模块下的所有文章
    async loadArticles(code: string) {
      const module = this.getModuleByCode(code)
      if (!module) {
        throw new Error(`Module with code ${code} not found`)
      }
      if (module.sections.length === 0) {
        throw new Error(`Module with code ${code} has no sections`)
      }
      if (module.sections.some(section => section.articles.length > 0)) {
        return // 如果已经加载过，直接返回
      }

      for (const section of module.sections) {
        const articles = await section.loadArticles()
        section.articles = articles
      }
    },

    // 清除模块数据
    clearModules() {
      this.modules = []
    }
  }
}) 